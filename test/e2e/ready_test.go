package e2e

import (
	"crypto/tls"
	"fmt"
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
)

func TestMarklogicReady(t *testing.T) {
	// Path to the helm chart we will test
	helmChartPath, e := filepath.Abs("../../charts")
	if e != nil {
		t.Fatalf(e.Error())
	}
	var body []byte
	var err error
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
		"replicaCount":          "2",
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

	t.Logf("====Installing Helm Chart")
	releaseName := "test-install"
	//add the helm chart repo and install the last helm chart release from repository
	//to test and upgrade this chart to the latest one to be released
	if runUpgradeTest {
		delete(valuesMap, "image.repository")
		delete(valuesMap, "image.tag")
		helm.AddRepo(t, options, "marklogic", "https://marklogic.github.io/marklogic-kubernetes/")
		defer helm.RemoveRepo(t, options, "marklogic")
		helmChartPath = "marklogic/marklogic"
	}

	podZeroName := testUtil.HelmInstall(t, options, releaseName, kubectlOptions, helmChartPath)

	// wait until the pod is in Ready status
	k8s.WaitUntilPodAvailable(t, kubectlOptions, podZeroName, 15, 15*time.Second)

	podOneName := releaseName + "-1"

	if runUpgradeTest {
		upgradeOptionsMap := map[string]string{
			"persistence.enabled":   "true",
			"replicaCount":          "2",
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
		testUtil.HelmUpgrade(t, helmUpgradeOptions, releaseName, kubectlOptions, []string{podZeroName, podOneName}, initialChartVersion)
	}

	tunnel := k8s.NewTunnel(
		kubectlOptions, k8s.ResourceTypePod, podOneName, 8001, 8001)
	defer tunnel.Close()
	tunnel.ForwardPort(t)
	endpoint := fmt.Sprintf("http://%s/admin/v1/timestamp", tunnel.Endpoint())
	t.Logf(`Endpoint: %s`, endpoint)

	// Make request to server as soon as it is ready
	client := req.C().
		SetCommonDigestAuth(username, password).
		SetCommonRetryCount(10).
		SetCommonRetryFixedInterval(10 * time.Second)

	// Make request to server as soon as it is ready
	resp, err := client.R().
		Get(endpoint)

	if err != nil {
		t.Fatalf(err.Error())
	}
	defer resp.Body.Close()

	if body, err = io.ReadAll(resp.Body); err != nil {
		t.Fatalf(err.Error())
	}

	t.Logf("Timestamp response:\n" + string(body))

	tlsConfig := tls.Config{}
	// restart all pods at once in the cluster and verify its ready and MarkLogic server is healthy
	testUtil.RestartPodAndVerify(t, true, []string{podZeroName, podOneName}, namespaceName, kubectlOptions, &tlsConfig)
}
