package e2e

import (
	"crypto/tls"
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
)

func TestMlAdminSecrets(t *testing.T) {
	var helmChartPath string
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
		"replicaCount":        "1",
		"image.repository":    imageRepo,
		"image.tag":           imageTag,
		"auth.adminUsername":  "admin",
		"auth.adminPassword":  "admin",
		"auth.walletPassword": "admin",
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

	// Path to the helm chart we will test
	helmChartPath, err = filepath.Abs("../../charts")
	if err != nil {
		t.Fatalf(err.Error())
	}

	//add the helm chart repo and install the last helm chart release from repository
	//to test and upgrade this chart to the latest one to be released
	if runUpgradeTest {
		delete(valuesMap, "image.repository")
		delete(valuesMap, "image.tag")
		helm.AddRepo(t, options, "marklogic", "https://marklogic.github.io/marklogic-kubernetes/")
		defer helm.RemoveRepo(t, options, "marklogic")
		helmChartPath = "marklogic/marklogic"
	}

	t.Logf("====Setting helm chart path to %s", helmChartPath)
	t.Logf("====Installing Helm Chart")
	releaseName := "test-ml-secrets"
	podName := testUtil.HelmInstall(t, options, releaseName, kubectlOptions, helmChartPath)

	// wait until the pod is in Ready status
	k8s.WaitUntilPodAvailable(t, kubectlOptions, podName, 15, 15*time.Second)

	if runUpgradeTest {
		// create options for helm upgrade
		upgradeOptionsMap := map[string]string{
			"persistence.enabled":   "true",
			"replicaCount":          "1",
			"logCollection.enabled": "false",
			"allowLongHostnames":    "true",
			"rootToRootlessUpgrade": "true",
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

	// get corev1.Pod to get logs of a pod
	pod := k8s.GetPod(t, kubectlOptions, podName)

	// get pod logs to verify wallet password is set as docker secret
	t.Logf("====Getting pod logs")
	podLogs := k8s.GetPodLogs(t, kubectlOptions, pod, "")

	// verify logs if wallet password is set as secret
	if !strings.Contains(podLogs, "MARKLOGIC_WALLET_PASSWORD_FILE is set, using file as secret for wallet-password.") {
		t.Errorf("wallet password not set as secret")
	}

	tlsConfig := tls.Config{}
	// restart pods in the cluster and verify its ready and MarkLogic server is healthy
	testUtil.RestartPodAndVerify(t, false, []string{podName}, namespaceName, kubectlOptions, &tlsConfig)
}
