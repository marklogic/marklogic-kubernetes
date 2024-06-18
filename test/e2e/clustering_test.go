package e2e

import (
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

func TestClusterJoin(t *testing.T) {
	// Path to the helm chart we will test
	helmChartPath, e := filepath.Abs("../../charts")
	if e != nil {
		t.Fatalf(e.Error())
	}
	username := "admin"
	password := "admin"
	var initialChartVersion string
	upgradeHelm, _ := os.LookupEnv("upgradeTest")
	runUpgradeTest, err := strconv.ParseBool(upgradeHelm)
	if runUpgradeTest {
		initialChartVersion, _ = os.LookupEnv("initialChartVersion")
		t.Logf("====Setting initial Helm chart version: %s", initialChartVersion)
	}
	imageRepo, repoPres := os.LookupEnv("dockerRepository")
	imageTag, tagPres := os.LookupEnv("dockerVersion")

	if !repoPres {
		imageRepo = "marklogicdb/marklogic-db"
		t.Logf("No imageRepo variable present, setting to default value: " + imageRepo)
	}

	if !tagPres {
		imageTag = "latest"
		t.Logf("No imageTag variable present, setting to default value: " + imageTag)
	}

	namespaceName := "ml-" + strings.ToLower(random.UniqueId())
	kubectlOptions := k8s.NewKubectlOptions("", "", namespaceName)
	options := &helm.Options{
		KubectlOptions: kubectlOptions,
		SetValues: map[string]string{
			"persistence.enabled":   "true",
			"replicaCount":          "2",
			"image.repository":      imageRepo,
			"image.tag":             imageTag,
			"auth.adminUsername":    username,
			"auth.adminPassword":    password,
			"logCollection.enabled": "false",
		},
		Version: initialChartVersion,
	}

	t.Logf("====Creating namespace: " + namespaceName)
	k8s.CreateNamespace(t, kubectlOptions, namespaceName)

	defer t.Logf("====Deleting namespace: " + namespaceName)
	defer k8s.DeleteNamespace(t, kubectlOptions, namespaceName)

	releaseName := "test-join"
	//add the helm chart repo and install the last helm chart release from repository
	//to test and upgrade this chart to the latest one to be released
	if runUpgradeTest {
		helm.AddRepo(t, options, "marklogic", "https://marklogic.github.io/marklogic-kubernetes/")
		defer helm.RemoveRepo(t, options, "marklogic")
		helmChartPath = "marklogic/marklogic"
	}

	t.Logf("====Setting helm chart path to %s", helmChartPath)
	t.Logf("====Installing Helm Chart")
	podName := testUtil.HelmInstall(t, options, releaseName, kubectlOptions, helmChartPath)
	podOneName := releaseName + "-1"

	// wait until the pod is in Ready status
	k8s.WaitUntilPodAvailable(t, kubectlOptions, podName, 15, 20*time.Second)

	if runUpgradeTest {
		upgradeOptionsMap := map[string]string{
			"persistence.enabled":   "true",
			"replicaCount":          "2",
			"auth.adminUsername":    username,
			"auth.adminPassword":    password,
			"logCollection.enabled": "false",
			"allowLongHostnames":    "true",
		}
		if strings.HasPrefix(initialChartVersion, "1.0") {
			podName = releaseName + "-marklogic-0"
			podOneName = releaseName + "-marklogic-1"
			upgradeOptionsMap["useLegacyHostnames"] = "true"
		}
		//set helm options for upgrading helm chart version
		helmUpgradeOptions := &helm.Options{
			KubectlOptions: kubectlOptions,
			SetValues:      upgradeOptionsMap,
		}
		t.Logf("UpgradeHelmTest is enabled. Running helm upgrade test")
		testUtil.HelmUpgrade(t, helmUpgradeOptions, releaseName, kubectlOptions, []string{podName, podOneName}, initialChartVersion)
	}
	tunnel := k8s.NewTunnel(
		kubectlOptions, k8s.ResourceTypePod, podName, 8002, 8002)
	defer tunnel.Close()
	tunnel.ForwardPort(t)

	numOfHosts := 0
	client := req.C()
	_, err = client.R().
		SetDigestAuth(username, password).
		SetRetryCount(5).
		SetRetryFixedInterval(10 * time.Second).
		AddRetryCondition(func(resp *req.Response, err error) bool {
			if resp == nil || err != nil {
				t.Logf("error in AddRetryCondition: %s", err.Error())
				return true
			}
			if resp.Response == nil {
				t.Log("Could not get the Response Object, Retrying...")
				return true
			}
			if resp.Body == nil {
				t.Log("Could not get the body for the response, Retrying...")
				return true
			}
			body, err := io.ReadAll(resp.Body)
			if body == nil || err != nil {
				t.Logf("error in read response body: %s", err.Error())
				return true
			}
			totalHosts := gjson.Get(string(body), `host-default-list.list-items.list-count.value`)
			numOfHosts = int(totalHosts.Num)
			if numOfHosts != 2 {
				t.Log("Number of hosts: " + string(totalHosts.Raw))
				t.Log("Waiting for MarkLogic count of MarkLogic hosts to be 2")
			}
			return numOfHosts != 2
		}).
		Get("http://localhost:8002/manage/v2/hosts?format=json")

	if err != nil {
		t.Error("Error getting hosts")
		t.Fatalf(err.Error())
	}

	if numOfHosts != 2 {
		t.Errorf("Wrong number of hosts")
	}
}
