package e2e

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/gruntwork-io/terratest/modules/random"
	"github.com/marklogic/marklogic-kubernetes/test/testUtil"
	"github.com/stretchr/testify/assert"
	digestAuth "github.com/xinsnake/go-http-digest-auth-client"
)

func TestEnableConvertersAndLicense(t *testing.T) {
	var resp *http.Response
	var body []byte
	var err error
	// Path to the helm chart we will test
	helmChartPath, e := filepath.Abs("../../charts")
	if e != nil {
		t.Fatalf(e.Error())
	}
	imageRepo, repoPres := os.LookupEnv("dockerRepository")
	imageTag, tagPres := os.LookupEnv("dockerVersion")
	var initialChartVersion string
	upgradeHelm, _ := os.LookupEnv("upgradeTest")
	runUpgradeTest, err := strconv.ParseBool(upgradeHelm)
	if runUpgradeTest {
		initialChartVersion, _ = os.LookupEnv("initialChartVersion")
		t.Logf("====Setting initial Helm chart version: %s", initialChartVersion)
	}
	username := "admin"
	password := "AdminPa$s_with@!#%^&*()"

	if !repoPres {
		imageRepo = "marklogic-centos/marklogic-server-centos"
		t.Logf("No imageRepo variable present, setting to default value: " + imageRepo)
	}

	if !tagPres {
		imageTag = "10-internal"
		t.Logf("No imageTag variable present, setting to default value: " + imageTag)
	}

	namespaceName := "ml-" + strings.ToLower(random.UniqueId())
	kubectlOptions := k8s.NewKubectlOptions("", "", namespaceName)
	options := &helm.Options{
		KubectlOptions: kubectlOptions,
		SetValues: map[string]string{
			"persistence.enabled":   "true",
			"replicaCount":          "1",
			"image.repository":      imageRepo,
			"image.tag":             imageTag,
			"auth.adminUsername":    username,
			"auth.adminPassword":    password,
			"logCollection.enabled": "false",
			"enableConverters":      "true",
		},
		Version: initialChartVersion,
	}

	t.Logf("====Creating namespace: " + namespaceName)
	k8s.CreateNamespace(t, kubectlOptions, namespaceName)

	defer t.Logf("====Deleting namespace: " + namespaceName)
	defer k8s.DeleteNamespace(t, kubectlOptions, namespaceName)

	t.Logf("====Installing Helm Chart")
	releaseName := "test"
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
	k8s.WaitUntilPodAvailable(t, kubectlOptions, podName, 15, 15*time.Second)
	if runUpgradeTest {
		upgradeOptionsMap := map[string]string{
			"persistence.enabled":   "true",
			"replicaCount":          "2",
			"auth.adminUsername":    username,
			"auth.adminPassword":    password,
			"enableConverters":      "true",
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
		kubectlOptions, k8s.ResourceTypePod, podName, 8001, 8001)
	defer tunnel.Close()
	tunnel.ForwardPort(t)
	endpoint := fmt.Sprintf("http://%s/admin/v1/timestamp", tunnel.Endpoint())
	t.Logf(`Endpoint: %s`, endpoint)

	// Make request to server as soon as it is ready
	timestamp := digestAuth.NewRequest(username, password, "GET", endpoint, "")

	if resp, err = timestamp.Execute(); err != nil {
		t.Fatalf(err.Error())
	}
	if body, err = io.ReadAll(resp.Body); err != nil {
		t.Fatalf(err.Error())
	}

	t.Logf("Timestamp response:\n" + string(body))

	// Get logs from a running container
	podConfig := k8s.GetPod(t, kubectlOptions, podName)
	logs, err := k8s.GetPodLogsE(t, kubectlOptions, podConfig, "")
	if err != nil {
		t.Errorf("Failed to get logs for pod %s in namespace %s: %v", podName, namespaceName, err)
	}

	// Verify that converters are getting installed
	assert.Contains(t, logs, "INSTALL_CONVERTERS is true, installing converters.")
}
