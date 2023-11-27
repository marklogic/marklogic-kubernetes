package e2e

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gruntwork-io/terratest/modules/helm"
	http_helper "github.com/gruntwork-io/terratest/modules/http-helper"
	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/gruntwork-io/terratest/modules/random"
	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
	digestAuth "github.com/xinsnake/go-http-digest-auth-client"
)

func TestHelmUpgrade(t *testing.T) {
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
		imageTag = "11.0.20230307-centos-1.0.2"
		t.Logf("No imageTag variable present, setting to default value: " + imageTag)
	}

	namespaceName := "marklogic-" + strings.ToLower(random.UniqueId())
	kubectlOptions := k8s.NewKubectlOptions("", "", namespaceName)
	options := &helm.Options{
		KubectlOptions: kubectlOptions,
		SetValues: map[string]string{
			"persistence.enabled":   "false",
			"replicaCount":          "1",
			"image.repository":      imageRepo,
			"image.tag":             imageTag,
			"logCollection.enabled": "false",
		},
	}

	t.Logf("====Creating namespace: " + namespaceName)
	k8s.CreateNamespace(t, kubectlOptions, namespaceName)
	defer t.Logf("====Deleting namespace: " + namespaceName)
	defer k8s.DeleteNamespace(t, kubectlOptions, namespaceName)

	t.Logf("====Installing Helm Chart")
	releaseName := "test-upgrade"
	helm.Install(t, options, helmChartPath, releaseName)

	// save the generated password from first installation
	secretName := releaseName + "-marklogic-admin"
	secret := k8s.GetSecret(t, kubectlOptions, secretName)
	passwordArr := secret.Data["password"]
	passwordAfterInstall := string(passwordArr[:])

	newOptions := &helm.Options{
		KubectlOptions: kubectlOptions,
		SetValues: map[string]string{
			"persistence.enabled":   "false",
			"replicaCount":          "2",
			"image.repository":      imageRepo,
			"image.tag":             imageTag,
			"logCollection.enabled": "false",
		},
	}

	t.Logf("====Upgrading Helm Chart")
	helm.Upgrade(t, newOptions, helmChartPath, releaseName)

	tlsConfig := tls.Config{}
	podName := releaseName + "-marklogic-1"

	// wait until the pod is in Ready status
	k8s.WaitUntilPodAvailable(t, kubectlOptions, podName, 20, 20*time.Second)
	tunnel := k8s.NewTunnel(
		kubectlOptions, k8s.ResourceTypePod, podName, 7997, 7997)
	defer tunnel.Close()
	tunnel.ForwardPort(t)
	endpoint := fmt.Sprintf("http://%s", tunnel.Endpoint())
	t.Logf(`Endpoint: %s`, endpoint)

	http_helper.HttpGetWithRetryWithCustomValidation(
		t,
		endpoint,
		&tlsConfig,
		10,
		15*time.Second,
		func(statusCode int, body string) bool {
			return statusCode == 200
		},
	)

	t.Log("====Test password in secret should not change after upgrade====")
	secret = k8s.GetSecret(t, kubectlOptions, secretName)
	passwordArr = secret.Data["password"]
	passwordAfterUpgrade := string(passwordArr[:])
	assert.Equal(t, passwordAfterUpgrade, passwordAfterInstall)
}

func TestMLupgrade(t *testing.T) {
	// Path to the helm chart we will test
	helmChartPath, e := filepath.Abs("../../charts")
	if e != nil {
		t.Fatalf(e.Error())
	}
	imageRepo, repoPres := os.LookupEnv("dockerRepository")
	imageTag, tagPres := os.LookupEnv("dockerVersion")
	prevImageTag, prevTagPres := os.LookupEnv("dockerVersion")

	if !repoPres {
		imageRepo = "ml-docker-db-dev-tierpoint.bed-artifactory.bedford.progress.com/marklogic/marklogic-server-centos"
		t.Logf("No imageRepo variable present, setting to default value: " + imageRepo)
	}
	if !tagPres {
		imageTag = "11.0.20230307-centos-1.0.2"
		t.Logf("No imageTag variable present, setting to default value: " + imageTag)
	}
	if !prevTagPres {
		prevImageTag = "10.0-20230307-centos-1.0.2"
		t.Logf("No imageTag variable present, setting to default value: " + prevImageTag)
	}

	username := "admin"
	password := "admin"

	namespaceName := "marklogic-" + strings.ToLower(random.UniqueId())
	kubectlOptions := k8s.NewKubectlOptions("", "", namespaceName)
	options := &helm.Options{
		KubectlOptions: kubectlOptions,
		SetValues: map[string]string{
			"persistence.enabled": "false",
			"replicaCount":        "1",
			"updateStrategy.type": "OnDelete",
			"image.repository":    imageRepo,
			"image.tag":           prevImageTag,
			"auth.adminUsername":  username,
			"auth.adminPassword":  password,
		},
	}

	t.Logf("====Creating namespace: " + namespaceName)
	k8s.CreateNamespace(t, kubectlOptions, namespaceName)
	defer t.Logf("====Deleting namespace: " + namespaceName)
	defer k8s.DeleteNamespace(t, kubectlOptions, namespaceName)

	t.Logf("====Installing Helm Chart")
	releaseName := "test-ml-upgrade"
	helm.Install(t, options, helmChartPath, releaseName)

	podName := releaseName + "-marklogic-0"

	// wait until second pod is in Ready status
	k8s.WaitUntilPodAvailable(t, kubectlOptions, podName, 20, 20*time.Second)

	k8s.RunKubectl(t, kubectlOptions, "describe", "pod", podName)

	newOptions := &helm.Options{
		KubectlOptions: kubectlOptions,
		SetValues: map[string]string{
			"persistence.enabled":   "false",
			"image.repository":      imageRepo,
			"image.tag":             imageTag,
			"logCollection.enabled": "false",
		},
	}

	t.Logf("====Upgrading Helm Chart")
	helm.Upgrade(t, newOptions, helmChartPath, releaseName)

	// delete pods to allow upgrades
	k8s.RunKubectl(t, kubectlOptions, "delete", "pod", podName)

	// wait until pod is in Ready status with new configuration
	k8s.WaitUntilPodAvailable(t, kubectlOptions, podName, 15, 30*time.Second)

	tunnel := k8s.NewTunnel(
		kubectlOptions, k8s.ResourceTypePod, podName, 8002, 8002)
	defer tunnel.Close()
	tunnel.ForwardPort(t)

	clusterEndpoint := fmt.Sprintf("http://%s/manage/v2?format=json", tunnel.Endpoint())
	t.Logf(`Endpoint: %s`, clusterEndpoint)

	getMLversion := digestAuth.NewRequest(username, password, "GET", clusterEndpoint, "")

	resp, err := getMLversion.Execute()
	if err != nil {
		t.Fatalf(err.Error())
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf(err.Error())
	}
	mlVersion := gjson.Get(string(body), `local-cluster-default.version`)
	expectedMlVersion := strings.Split(imageTag, "-centos")[0]
	// verify latest MarkLogic version after upgrade
	assert.Equal(t, mlVersion.Str, expectedMlVersion)
}
