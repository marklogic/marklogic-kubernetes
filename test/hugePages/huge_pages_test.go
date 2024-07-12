package hugepages

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
	"github.com/marklogic/marklogic-kubernetes/test/testUtil"
	"github.com/stretchr/testify/assert"
	digestAuth "github.com/xinsnake/go-http-digest-auth-client"
)

func TestHugePagesSettings(t *testing.T) {
	// var resp *http.Response
	var body []byte
	var err error
	var podName string
	var helmChartPath string
	var initialChartVersion string
	imageRepo, repoPres := os.LookupEnv("dockerRepository")
	imageTag, tagPres := os.LookupEnv("dockerVersion")
	upgradeHelm, _ := os.LookupEnv("upgradeTest")
	runUpgradeTest, err := strconv.ParseBool(upgradeHelm)
	if runUpgradeTest {
		initialChartVersion, _ = os.LookupEnv("initialChartVersion")
		t.Logf("====Setting initial Helm chart version: %s", initialChartVersion)
	}

	if !repoPres {
		imageRepo = "progressofficial/marklogic-db"
		t.Logf("No imageRepo variable present, setting to default value: " + imageRepo)
	}

	if !tagPres {
		imageTag = "latest-11"
		t.Logf("No imageTag variable present, setting to default value: " + imageTag)
	}

	username := "admin"
	password := "admin"

	namespaceName := "ml-" + strings.ToLower(random.UniqueId())
	kubectlOptions := k8s.NewKubectlOptions("", "", namespaceName)
	options := &helm.Options{
		KubectlOptions: kubectlOptions,
		SetValues: map[string]string{
			"persistence.enabled":            "true",
			"replicaCount":                   "1",
			"image.repository":               imageRepo,
			"image.tag":                      imageTag,
			"auth.adminUsername":             username,
			"auth.adminPassword":             password,
			"logCollection.enabled":          "false",
			"hugepages.enabled":              "true",
			"hugepages.mountPath":            "/dev/hugepages",
			"resources.limits.hugepages-2Mi": "1Gi",
			"resources.limits.memory":        "8Gi",
			"resources.requests.memory":      "8Gi",
		},
		Version: initialChartVersion,
	}
	t.Logf("====Installing Helm Chart")
	releaseName := "hugepages"

	t.Logf("====Creating namespace: " + namespaceName)
	k8s.CreateNamespace(t, kubectlOptions, namespaceName)

	defer t.Logf("====Deleting namespace: " + namespaceName)
	defer k8s.DeleteNamespace(t, kubectlOptions, namespaceName)

	helmChartPath, err = filepath.Abs("../../charts")
	if err != nil {
		t.Fatalf(err.Error())
	}

	//add the helm chart repo and install the last helm chart release from repository
	//to test and upgrade this chart to the latest one to be released
	if runUpgradeTest {
		helm.AddRepo(t, options, "marklogic", "https://marklogic.github.io/marklogic-kubernetes/")
		defer helm.RemoveRepo(t, options, "marklogic")
		helmChartPath = "marklogic/marklogic"
	}

	t.Logf("====Setting helm chart path to %s", helmChartPath)
	t.Logf("====Installing Helm Chart")
	podName = testUtil.HelmInstall(t, options, releaseName, kubectlOptions, helmChartPath)

	t.Logf("====Describe pod for verifying huge pages")
	k8s.RunKubectl(t, kubectlOptions, "describe", "pod", podName)

	tlsConfig := tls.Config{}
	// wait until the pod is in Ready status
	k8s.WaitUntilPodAvailable(t, kubectlOptions, podName, 10, 15*time.Second)

	// verify MarkLogic is ready
	_, err = testUtil.MLReadyCheck(t, kubectlOptions, podName, &tlsConfig)
	if err != nil {
		t.Fatal("MarkLogic failed to start")
	}

	if runUpgradeTest {
		upgradeOptionsMap := map[string]string{
			"persistence.enabled":                               "true",
			"replicaCount":                                      "1",
			"hugepages.enabled":                                 "true",
			"hugepages.mountPath":                               "/dev/hugepages",
			"resources.limits.hugepages-2Mi":                    "1Gi",
			"resources.limits.memory":                           "8Gi",
			"resources.requests.memory":                         "8Gi",
			"allowLongHostnames":                                "true",
			"rootToRootlessUpgrade":                             "true",
			"containerSecurityContext.allowPrivilegeEscalation": "true",
		}
		if strings.HasPrefix(initialChartVersion, "1.0") {
			podName = releaseName + "-marklogic-0"
			upgradeOptionsMap["useLegacyHostnames"] = "true"
		}
		//set helm options for upgrading helm chart version
		helmUpgradeOptions := &helm.Options{
			KubectlOptions: kubectlOptions,
			SetValues:      upgradeOptionsMap,
		}
		t.Logf("UpgradeHelmTest is enabled. Running helm upgrade test")
		testUtil.HelmUpgrade(t, helmUpgradeOptions, releaseName, kubectlOptions, []string{podName}, initialChartVersion)
	}

	tunnel8002 := k8s.NewTunnel(kubectlOptions, k8s.ResourceTypePod, podName, 8002, 8002)
	defer tunnel8002.Close()
	tunnel8002.ForwardPort(t)
	endpointManage := fmt.Sprintf("http://%s/manage/v2/logs?format=text&filename=ErrorLog.txt", tunnel8002.Endpoint())
	request := digestAuth.NewRequest(username, password, "GET", endpointManage, "")
	response, err := request.Execute()
	if err != nil {
		t.Fatalf(err.Error())
	}
	defer response.Body.Close()
	assert.Equal(t, 200, response.StatusCode)

	body, err = io.ReadAll(response.Body)
	t.Log(string(body))

	// Verify if Huge pages are configured on the MarkLogic node
	if !strings.Contains(string(body), "Linux Huge Pages: detected 1280") {
		t.Errorf("Huge Pages not configured for the node")
	}

	// restart pod in the cluster and verify its ready and MarkLogic server is healthy
	testUtil.RestartPodAndVerify(t, false, []string{podName}, namespaceName, kubectlOptions, &tlsConfig)
}
