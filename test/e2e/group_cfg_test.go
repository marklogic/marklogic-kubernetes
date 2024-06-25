package e2e

import (
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
	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
	digestAuth "github.com/xinsnake/go-http-digest-auth-client"
)

func VerifyGrpNameChng(t *testing.T, groupEndpoint string, kubectlOptions *k8s.KubectlOptions, groupName string, newGroupName string) (int, error) {
	client := req.C().
		SetCommonDigestAuth("admin", "admin").
		SetCommonRetryCount(10).
		SetCommonRetryFixedInterval(10 * time.Second)

	t.Log("====Updating group name")

	t.Logf(`Endpoint: %s`, groupEndpoint)
	strJsonData := fmt.Sprintf(`{"group-name":"%s"}`, newGroupName)
	t.Logf("New group name: %s", strJsonData)

	resp, err := client.R().
		SetContentType("application/json").
		SetBodyJsonString(strJsonData).
		Put(groupEndpoint)
	if err != nil {
		t.Fatal(err.Error())
		return (resp.GetStatusCode()), err
	}
	return resp.GetStatusCode(), resp.Err
}

func TestSingleGrpNameChng(t *testing.T) {
	// Path to the helm chart we will test
	helmChartPath, e := filepath.Abs("../../charts")
	if e != nil {
		t.Fatalf(e.Error())
	}
	username := "admin"
	password := "admin"
	releaseName := "test-grp"
	groupName := "Default"
	imageRepo, repoPres := os.LookupEnv("dockerRepository")
	imageTag, tagPres := os.LookupEnv("dockerVersion")

	if !repoPres {
		imageRepo = "marklogicdb/marklogic-db"
		t.Logf("No imageRepo variable present, setting to default value: " + imageRepo)
	}

	if !tagPres {
		imageTag = "latest-11"
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
		},
	}

	t.Logf("====Creating namespace: " + namespaceName)
	k8s.CreateNamespace(t, kubectlOptions, namespaceName)

	defer t.Logf("====Deleting namespace: " + namespaceName)
	defer k8s.DeleteNamespace(t, kubectlOptions, namespaceName)

	t.Logf("====Setting helm chart path to %s", helmChartPath)
	t.Logf("====Installing Helm Chart")
	podZeroName := testUtil.HelmInstall(t, options, releaseName, kubectlOptions, helmChartPath)

	// wait until the pod is in Ready status
	k8s.WaitUntilPodAvailable(t, kubectlOptions, podZeroName, 15, 20*time.Second)
	tunnel := k8s.NewTunnel(
		kubectlOptions, k8s.ResourceTypePod, podZeroName, 8002, 8002)
	defer tunnel.Close()
	tunnel.ForwardPort(t)

	// change the group name for dnode and verify it passes
	groupEndpoint := fmt.Sprintf("http://%s/manage/v2/groups/%s/properties", tunnel.Endpoint(), groupName)
	responseCode, err := VerifyGrpNameChng(t, groupEndpoint, kubectlOptions, groupName, "newDefault")
	if err != nil {
		t.Fatalf(err.Error())
	}
	if responseCode != 204 {
		t.Fatal("Failed to change group name")
	}
}

func TestGroupNameChng(t *testing.T) {
	username := "admin"
	password := "admin"
	imageRepo, repoPres := os.LookupEnv("dockerRepository")
	imageTag, tagPres := os.LookupEnv("dockerVersion")
	var initialChartVersion string
	upgradeHelm, _ := os.LookupEnv("upgradeTest")
	runUpgradeTest, _ := strconv.ParseBool(upgradeHelm)
	if runUpgradeTest {
		initialChartVersion, _ = os.LookupEnv("initialChartVersion")
		t.Logf("====Setting initial Helm chart version: %s", initialChartVersion)
	}
	namespaceName := "ml-" + strings.ToLower(random.UniqueId())
	kubectlOptions := k8s.NewKubectlOptions("", "", namespaceName)
	dnodeGrpName := "dnode"
	enodeGrpName := "enode"
	dnodeReleaseName := "dnode"
	enodeReleaseName := "enode"
	dnodePodName := dnodeReleaseName + "-0"
	enodePodName0 := enodeReleaseName + "-0"

	// Path to the helm chart we will test
	helmChartPath, e := filepath.Abs("../../charts")
	if e != nil {
		t.Fatalf(e.Error())
	}

	if !repoPres {
		imageRepo = "marklogicdb/marklogic-db"
		t.Logf("No imageRepo variable present, setting to default value: " + imageRepo)
	}

	if !tagPres {
		imageTag = "latest-11"
		t.Logf("No imageTag variable present, setting to default value: " + imageTag)
	}

	options := &helm.Options{
		KubectlOptions: kubectlOptions,
		SetValues: map[string]string{
			"persistence.enabled":   "true",
			"replicaCount":          "1",
			"image.repository":      imageRepo,
			"image.tag":             imageTag,
			"auth.adminUsername":    username,
			"auth.adminPassword":    password,
			"group.name":            dnodeGrpName,
			"logCollection.enabled": "false",
		},
		Version: initialChartVersion,
	}

	t.Logf("====Creating namespace: " + namespaceName)
	k8s.CreateNamespace(t, kubectlOptions, namespaceName)

	defer t.Logf("====Deleting namespace: " + namespaceName)
	defer k8s.DeleteNamespace(t, kubectlOptions, namespaceName)

	t.Logf("====Setting helm chart path to %s", helmChartPath)
	t.Logf("====Installing Helm Chart " + dnodeReleaseName)
	dnodePodName = testUtil.HelmInstall(t, options, dnodeReleaseName, kubectlOptions, helmChartPath)

	// wait until the pod is in ready status
	k8s.WaitUntilPodAvailable(t, kubectlOptions, dnodePodName, 15, 20*time.Second)
	tunnel := k8s.NewTunnel(
		kubectlOptions, k8s.ResourceTypePod, dnodePodName, 8002, 8002)
	defer tunnel.Close()
	tunnel.ForwardPort(t)

	// change the group name for dnode and verify it passes
	groupEndpoint := fmt.Sprintf("http://%s/manage/v2/groups/%s/properties", tunnel.Endpoint(), dnodeGrpName)
	responseCode, err := VerifyGrpNameChng(t, groupEndpoint, kubectlOptions, dnodeGrpName, "newDnode")
	if err != nil {
		t.Fatalf(err.Error())
	}
	if responseCode != 204 {
		t.Fatal("Failed to change group name")
	}

	hostsEndpoint := fmt.Sprintf("http://%s/manage/v2/hosts?format=json", tunnel.Endpoint())
	t.Logf(`Endpoint: %s`, hostsEndpoint)

	getHostsDR := digestAuth.NewRequest(username, password, "GET", hostsEndpoint, "")

	resp, err := getHostsDR.Execute()
	if err != nil {
		t.Fatalf(err.Error())
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf(err.Error())
	}
	bootstrapHost := gjson.Get(string(body), `host-default-list.list-items.list-item.#(roleref="bootstrap").nameref`)
	t.Logf("bootstrapHost: %s", bootstrapHost)

	enodeOptions := &helm.Options{
		KubectlOptions: kubectlOptions,
		SetValues: map[string]string{
			"persistence.enabled":   "true",
			"replicaCount":          "2",
			"image.repository":      imageRepo,
			"image.tag":             imageTag,
			"auth.adminUsername":    username,
			"auth.adminPassword":    password,
			"group.name":            enodeGrpName,
			"bootstrapHostName":     bootstrapHost.Str,
			"group.enableXdqpSsl":   "false",
			"logCollection.enabled": "false",
		},
	}
	t.Logf("====Installing Helm Chart " + enodeReleaseName)
	enodePodName0 = testUtil.HelmInstall(t, enodeOptions, enodeReleaseName, kubectlOptions, helmChartPath)

	// wait until the first enode pod is in Ready status
	k8s.WaitUntilPodAvailable(t, kubectlOptions, enodePodName0, 15, 20*time.Second)

	// change the enode group name to a existing group name in the cluster and verify it fails
	groupEndpoint = fmt.Sprintf("http://%s/manage/v2/groups/%s/properties", tunnel.Endpoint(), enodeGrpName)
	responseCode, err = VerifyGrpNameChng(t, groupEndpoint, kubectlOptions, enodeGrpName, "newDnode")
	if err != nil {
		t.Fatalf(err.Error())
	}
	assert.Equal(t, 400, responseCode)

	// change the enode group name to a new group name and verify it passes
	groupEndpoint = fmt.Sprintf("http://%s/manage/v2/groups/%s/properties", tunnel.Endpoint(), enodeGrpName)
	responseCode, err = VerifyGrpNameChng(t, groupEndpoint, kubectlOptions, enodeGrpName, "newEnode")
	if err != nil {
		t.Fatalf(err.Error())
	}
	assert.Equal(t, 204, responseCode)
}
