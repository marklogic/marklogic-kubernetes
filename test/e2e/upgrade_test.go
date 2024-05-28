package e2e

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/client"
	"github.com/gruntwork-io/terratest/modules/helm"
	http_helper "github.com/gruntwork-io/terratest/modules/http-helper"
	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/gruntwork-io/terratest/modules/random"
	"github.com/imroc/req/v3"
	"github.com/marklogic/marklogic-kubernetes/test/testUtil"
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
		imageTag = "11.0.nightly-centos-1.0.2"
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
	secretName := releaseName + "-admin"
	secret := k8s.GetSecret(t, kubectlOptions, secretName)
	usernameArr := secret.Data["username"]
	username := string(usernameArr)
	passwordArr := secret.Data["password"]
	passwordAfterInstall := string(passwordArr[:])

	newOptions := &helm.Options{
		KubectlOptions: kubectlOptions,
		SetValues: map[string]string{
			"persistence.enabled":   "true",
			"replicaCount":          "2",
			"image.repository":      "marklogicdb/marklogic-db",
			"image.tag":             "latest",
			"logCollection.enabled": "false",
		},
	}

	t.Logf("====Upgrading Helm Chart")
	helm.Upgrade(t, newOptions, helmChartPath, releaseName)

	tlsConfig := tls.Config{}
	podOneName := releaseName + "-1"
	podZeroName := releaseName + "-0"

	// wait until the pod is in Ready status
	k8s.WaitUntilPodAvailable(t, kubectlOptions, podOneName, 20, 20*time.Second)

	t.Log("====Test password in secret should not change after upgrade====")
	secret = k8s.GetSecret(t, kubectlOptions, secretName)
	passwordArr = secret.Data["password"]
	passwordAfterUpgrade := string(passwordArr[:])
	assert.Equal(t, passwordAfterUpgrade, passwordAfterInstall)

	tunnel := k8s.NewTunnel(
		kubectlOptions, k8s.ResourceTypePod, podZeroName, 7997, 7997)

	defer tunnel.Close()
	tunnel.ForwardPort(t)
	endpoint := fmt.Sprintf("http://%s", tunnel.Endpoint())
	t.Logf(`Endpoint: %s`, endpoint)

	http_helper.HttpGetWithRetryWithCustomValidation(
		t,
		endpoint,
		&tlsConfig,
		15,
		20*time.Second,
		func(statusCode int, body string) bool {
			return statusCode == 200
		},
	)

	tunnel8002 := k8s.NewTunnel(
		kubectlOptions, k8s.ResourceTypePod, podZeroName, 8002, 8002)
	defer tunnel8002.Close()
	tunnel8002.ForwardPort(t)

	hostsEndpoint := fmt.Sprintf("http://%s/manage/v2/hosts?view=status&format=json", tunnel8002.Endpoint())
	t.Logf(`Endpoint: %s`, hostsEndpoint)

	totalHosts := 1
	client := req.C().
		SetCommonDigestAuth(username, passwordAfterUpgrade).
		SetCommonRetryCount(10).
		SetCommonRetryFixedInterval(10 * time.Second)

	resp, err := client.R().
		AddRetryCondition(func(resp *req.Response, err error) bool {
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Logf("error: %s", err.Error())
			}
			totalHosts = int(gjson.Get(string(body), `host-status-list.status-list-summary.total-hosts.value`).Num)
			if totalHosts != 2 {
				t.Log("Waiting for second host to join MarkLogic cluster")
			}
			return totalHosts != 2
		}).
		Get(hostsEndpoint)

	if err != nil {
		t.Fatalf(err.Error())
	}
	defer resp.Body.Close()

	if totalHosts != 2 {
		t.Errorf("Incorrect number of MarkLogic hosts found after helm upgrade")
	}

	// restart all pods at once in the cluster and verify its ready and MarkLogic server is healthy
	testUtil.RestartPodAndVerify(t, true, []string{podZeroName, podOneName}, namespaceName, kubectlOptions, &tlsConfig)

	// restart 1 pod at a time in the cluster and verify its ready and MarkLogic server is healthy
	testUtil.RestartPodAndVerify(t, false, []string{podZeroName, podOneName}, namespaceName, kubectlOptions, &tlsConfig)
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
		imageTag = "11.0.nightly-centos-1.0.2"
		t.Logf("No imageTag variable present, setting to default value: " + imageTag)
	}
	if !prevTagPres {
		prevImageTag = "10.0-nightly-centos-1.0.2"
		t.Logf("No imageTag variable present, setting to default value: " + prevImageTag)
	}

	username := "admin"
	password := "admin"

	namespaceName := "ml-" + strings.ToLower(random.UniqueId())
	kubectlOptions := k8s.NewKubectlOptions("", "", namespaceName)
	options := &helm.Options{
		KubectlOptions: kubectlOptions,
		SetValues: map[string]string{
			"persistence.enabled": "true",
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
	releaseName := "ml-upgrade"
	helm.Install(t, options, helmChartPath, releaseName)

	podName := releaseName + "-0"

	// wait until second pod is in Ready status
	k8s.WaitUntilPodAvailable(t, kubectlOptions, podName, 20, 20*time.Second)

	k8s.RunKubectl(t, kubectlOptions, "describe", "pod", podName)

	newOptions := &helm.Options{
		KubectlOptions: kubectlOptions,
		SetValues: map[string]string{
			"persistence.enabled":   "true",
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

	// Get MarkLogic version for a running instance
	clusterEndpoint := fmt.Sprintf("http://%s/manage/v2?format=json", tunnel.Endpoint())
	t.Logf(`Endpoint: %s`, clusterEndpoint)

	getMLversion := digestAuth.NewRequest(username, password, "GET", clusterEndpoint, "")

	resp, err := getMLversion.Execute()
	if err != nil {
		t.Fatalf(err.Error())
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf(err.Error())
	}
	mlVersionResp := gjson.Get(string(body), `local-cluster-default.version`)
	t.Logf("MarkLogic version: %s", mlVersionResp.Str)

	// Get MarkLogic version from the image metadata
	// Connect to Docker
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}
	// Get image details
	imageDetails, _, err := cli.ImageInspectWithRaw(ctx, imageRepo+":"+imageTag)
	if err != nil {
		panic(err)
	}
	mlVersionInImage := imageDetails.Config.Labels["com.marklogic.release-version"]

	// extract ML version from server response (actual) and image metadata (expected)
	mlVersionPattern := regexp.MustCompile(`(\d+\.\d+)`)
	actualMlVersion := mlVersionPattern.FindStringSubmatch(mlVersionResp.Str)
	expectedMlVersion := mlVersionPattern.FindStringSubmatch(mlVersionInImage)

	// verify latest MarkLogic version after upgrade
	assert.Equal(t, actualMlVersion, expectedMlVersion)

	tlsConfig := tls.Config{}
	// restart all pods at once in the cluster and verify its ready and MarkLogic server is healthy
	testUtil.RestartPodAndVerify(t, true, []string{podName}, namespaceName, kubectlOptions, &tlsConfig)
}
