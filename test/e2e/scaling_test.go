package e2e

import (
	"crypto/tls"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/gruntwork-io/terratest/modules/random"
	"github.com/imroc/req/v3"
	"github.com/marklogic/marklogic-kubernetes/test/testUtil"
	"github.com/tidwall/gjson"
)

func TestHelmScaleUp(t *testing.T) {
	// Path to the helm chart we will test
	helmChartPath, e := filepath.Abs("../../charts")
	if e != nil {
		t.Fatalf(e.Error())
	}
	var initialChartVersion string
	imageRepo, repoPres := os.LookupEnv("dockerRepository")
	imageTag, tagPres := os.LookupEnv("dockerVersion")
	upgradeHelm, _ := os.LookupEnv("upgradeTest")
	runUpgradeTest, _ := strconv.ParseBool(upgradeHelm)
	if runUpgradeTest {
		initialChartVersion, _ = os.LookupEnv("initialChartVersion")
		t.Logf("====Setting initial Helm chart version: %s", initialChartVersion)
	}
	username := "admin"
	password := "admin"

	if !repoPres {
		imageRepo = "progressofficial/marklogic-db"
		t.Logf("No imageRepo variable present, setting to default value: " + imageRepo)
	}

	if !tagPres {
		imageTag = "latest-11"
		t.Logf("No imageTag variable present, setting to default value: " + imageTag)
	}

	namespaceName := "ml-" + strings.ToLower(random.UniqueId())
	kubectlOptions := k8s.NewKubectlOptions("", "", namespaceName)
	valuesMap := map[string]string{"persistence.enabled": "true",
		"replicaCount":          "1",
		"image.repository":      imageRepo,
		"image.tag":             imageTag,
		"auth.adminUsername":    username,
		"auth.adminPassword":    password,
		"logCollection.enabled": "false",
	}
	options := &helm.Options{
		KubectlOptions: kubectlOptions,
		SetValues:      valuesMap,
		Version:        initialChartVersion,
	}

	t.Logf("====Creating namespace: " + namespaceName)
	k8s.CreateNamespace(t, kubectlOptions, namespaceName)
	defer t.Logf("====Deleting namespace: " + namespaceName)
	defer k8s.DeleteNamespace(t, kubectlOptions, namespaceName)

	//add the helm chart repo and install the last helm chart release from repository
	//to test and upgrade this chart to the latest one to be released
	if runUpgradeTest {
		delete(valuesMap, "image.repository")
		delete(valuesMap, "image.tag")
		helm.AddRepo(t, options, "marklogic", "https://marklogic.github.io/marklogic-kubernetes/")
		defer helm.RemoveRepo(t, options, "marklogic")
		helmChartPath = "marklogic/marklogic"
	}

	releaseName := "test-scale-up"
	t.Logf("====Setting helm chart path to %s", helmChartPath)
	t.Logf("====Installing Helm Chart")
	podZeroName := testUtil.HelmInstall(t, options, releaseName, kubectlOptions, helmChartPath)

	// wait until first pod is in Ready status
	k8s.WaitUntilPodAvailable(t, kubectlOptions, podZeroName, 30, 10*time.Second)

	newOptions := &helm.Options{
		KubectlOptions: kubectlOptions,
		SetValues: map[string]string{
			"persistence.enabled":   "true",
			"replicaCount":          "2",
			"image.repository":      imageRepo,
			"image.tag":             imageTag,
			"logCollection.enabled": "false",
			"auth.adminUsername":    username,
			"auth.adminPassword":    password,
		},
	}

	t.Logf("====Scaling up pods using helm upgrade")
	helm.Upgrade(t, newOptions, helmChartPath, releaseName)

	podOneName := releaseName + "-1"
	// wait until second pod is in Ready status
	k8s.WaitUntilPodAvailable(t, kubectlOptions, podOneName, 30, 10*time.Second)

	if runUpgradeTest {
		upgradeOptionsMap := map[string]string{
			"persistence.enabled":   "true",
			"replicaCount":          "2",
			"auth.adminUsername":    username,
			"auth.adminPassword":    password,
			"logCollection.enabled": "false",
			"allowLongHostnames":    "true",
			"rootToRootlessUpgrade": "true",
		}
		if strings.HasPrefix(initialChartVersion, "1.0") {
			podZeroName = releaseName + "-marklogic-0"
			podOneName = releaseName + "-marklogic-1"
			upgradeOptionsMap["useLegacyHostnames"] = "true"
		}

		//set helm options for upgrading helm chart version
		helmUpgradeOptions := &helm.Options{
			KubectlOptions: kubectlOptions,
			SetValues:      upgradeOptionsMap,
		}
		t.Logf("UpgradeHelmTest is enabled. Running helm upgrade test")
		t.Logf("====Upgrading Helm Chart")
		testUtil.HelmUpgrade(t, helmUpgradeOptions, releaseName, kubectlOptions, []string{podZeroName, podOneName}, initialChartVersion)
	}

	output, err := testUtil.WaitUntilPodRunning(t, kubectlOptions, podOneName, 20, 20*time.Second)
	if err != nil {
		t.Error(err.Error())
	}
	if output != "Running" {
		t.Error(output)
	}
	tlsConfig := tls.Config{}
	_, err = testUtil.MLReadyCheck(t, kubectlOptions, podOneName, &tlsConfig)
	if err != nil {
		t.Fatal(err.Error())
	}
	tunnel := k8s.NewTunnel(
		kubectlOptions, k8s.ResourceTypePod, podZeroName, 8002, 8002)
	defer tunnel.Close()
	tunnel.ForwardPort(t)

	numOfHosts := 1
	client := req.C()
	_, err = client.R().
		SetDigestAuth(username, password).
		SetRetryCount(5).
		SetRetryFixedInterval(10 * time.Second).
		AddRetryCondition(func(resp *req.Response, err error) bool {
			if err != nil {
				t.Logf("error in AddRetryCondition: %s", err.Error())
				return true
			}
			if resp == nil || resp.Body == nil {
				t.Log("Could not get the Response Body, Retrying...")
				return true
			}
			body, err := io.ReadAll(resp.Body)
			if body == nil || err != nil {
				t.Logf("error in read response body: %s", err.Error())
				return true
			}
			totalHosts := gjson.Get(string(body), `host-status-list.status-list-summary.total-hosts.value`)
			numOfHosts = int(totalHosts.Num)
			if numOfHosts != 2 {
				t.Log("Waiting for second host to join MarkLogic cluster")
			}
			return numOfHosts != 2
		}).
		Get("http://localhost:8002/manage/v2/hosts?view=status&format=json")

	if err != nil {
		t.Fatalf(err.Error())
	}
	// verify total number of hosts on the clsuter after scaling up
	if numOfHosts != 2 {
		t.Errorf("Incorrect number of MarkLogic hosts")
	}

	// restart all pods at once in the cluster and verify its ready and MarkLogic server is healthy
	testUtil.RestartPodAndVerify(t, true, []string{podZeroName, podOneName}, namespaceName, kubectlOptions, &tlsConfig)
}
