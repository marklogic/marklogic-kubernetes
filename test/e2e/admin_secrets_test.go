package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/gruntwork-io/terratest/modules/random"
)

func TestMlAdminSecrets(t *testing.T) {
	// Path to the helm chart we will test
	helmChartPath, e := filepath.Abs("../../charts")
	if e != nil {
		t.Fatalf(e.Error())
	}
	imageRepo, repoPres := os.LookupEnv("dockerRepository")
	imageTag, tagPres := os.LookupEnv("dockerVersion")

	if !repoPres {
		imageRepo = "ml-docker-db-dev-tierpoint.bed-artifactory.bedford.progress.com/marklogic/marklogic-server-centos"
		t.Logf("No imageRepo variable present, setting to default value: " + imageRepo)
	}

	if !tagPres {
		imageTag = "11.0.nightly-centos-1.0.2"
		t.Logf("No imageTag variable present, setting to default value: " + imageTag)
	}

	namespaceName := "marklogic-" + strings.ToLower(random.UniqueId())
	kubectlOptions := k8s.NewKubectlOptions("", "", namespaceName)
	options := &helm.Options{
		KubectlOptions: kubectlOptions,
		SetValues: map[string]string{
			"persistence.enabled": "false",
			"replicaCount":        "1",
			"image.repository":    imageRepo,
			"image.tag":           imageTag,
			"auth.adminUsername":  "admin",
			"auth.adminPassword":  "admin",
			"auth.walletPassword": "admin",
		},
	}

	t.Logf("====Creating namespace: " + namespaceName)
	k8s.CreateNamespace(t, kubectlOptions, namespaceName)

	defer t.Logf("====Deleting namespace: " + namespaceName)
	defer k8s.DeleteNamespace(t, kubectlOptions, namespaceName)

	t.Logf("====Installing Helm Chart")
	releaseName := "test-ml-secrets"
	helm.Install(t, options, helmChartPath, releaseName)

	podName := releaseName + "-marklogic-0"
	// wait until the pod is in Ready status
	k8s.WaitUntilPodAvailable(t, kubectlOptions, podName, 10, 15*time.Second)

	// get corev1.Pod to get logs of a pod
	pod := k8s.GetPod(t, kubectlOptions, podName)

	// get pod logs to verify wallet password is set as docker secret
	t.Logf("====Getting pod logs")
	podLogs := k8s.GetPodLogs(t, kubectlOptions, pod, "")

	// verify logs if wallet password is set as secret
	if !strings.Contains(podLogs, "MARKLOGIC_WALLET_PASSWORD_FILE is set, using file as secret for wallet-password.") {
		t.Errorf("wallet password not set as secret")
	}
}
