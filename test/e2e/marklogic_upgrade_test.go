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
	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/gruntwork-io/terratest/modules/random"
	"github.com/imroc/req/v3"
	"github.com/marklogic/marklogic-kubernetes/test/testUtil"
	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
)

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
		imageRepo = "ml-docker-db-dev-tierpoint.bed-artifactory.bedford.progress.com/marklogic/marklogic-server-ubi-rootless"
		t.Logf("No imageRepo variable present, setting to default value: " + imageRepo)
	}
	if !tagPres {
		imageTag = "latest-11"
		t.Logf("No imageTag variable present, setting to default value: " + imageTag)
	}
	if !prevTagPres {
		prevImageTag = "latest-10"
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
			"auth.adminUsername":    username,
			"auth.adminPassword":    password,
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

	reqClient := req.C().
		SetCommonDigestAuth(username, password).
		SetCommonRetryCount(10).
		SetCommonRetryFixedInterval(10 * time.Second)

	resp, err := reqClient.R().
		Get(clusterEndpoint)

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
	// restart pod in the cluster and verify its ready and MarkLogic server is healthy
	testUtil.RestartPodAndVerify(t, false, []string{podName}, namespaceName, kubectlOptions, &tlsConfig)
}
