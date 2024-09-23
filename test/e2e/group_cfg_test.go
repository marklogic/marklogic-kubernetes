package e2e

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
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
)

func VerifyGroupChange(t *testing.T, groupEndpoint string, newGroupName string) (bool, error) {
	client := req.C().
		SetCommonBasicAuth("admin", "admin").
		SetCommonRetryCount(5).
		SetCommonRetryFixedInterval(10 * time.Second)

	t.Logf(`Endpoint: %s`, groupEndpoint)

	groupChanged := false

	_, err := client.R().
		SetContentType("application/json").
		AddRetryCondition(func(resp *req.Response, err error) bool {
			if err != nil {
				t.Logf("error in getting group config: %s", err.Error())
				return true
			}
			if resp == nil || resp.Body == nil {
				t.Logf("error in getting response body")
				return true
			}
			body, err := io.ReadAll(resp.Body)
			if body == nil || err != nil {
				t.Logf("error in read response body")
				return true
			}
			groupName := gjson.Get(string(body), `group-name`)
			t.Logf("current group name: %s", groupName)
			if groupName.Str != newGroupName {
				t.Logf("group name is not updated yet. retrying...")
				return true
			}
			groupChanged = true
			return false
		}).
		Get(groupEndpoint)
	if err != nil {
		t.Fatal(err.Error())
		return false, err
	}
	return groupChanged, nil
}

func TestSingleGroupChange(t *testing.T) {
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
		imageRepo = "progressofficial/marklogic-db"
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

	k8s.WaitUntilPodAvailable(t, kubectlOptions, podZeroName, 15, 20*time.Second)

	newGroupName := "new_group"

	helmUpgradeOptions := &helm.Options{
		KubectlOptions: kubectlOptions,
		SetValues: map[string]string{
			"group.name": newGroupName,
		},
	}
	helm.Upgrade(t, helmUpgradeOptions, helmChartPath, releaseName)

	k8s.RunKubectl(t, kubectlOptions, "delete", "pod", podZeroName)

	// for debugging on jenkins
	k8s.RunKubectl(t, kubectlOptions, "describe", "pod", podZeroName)

	k8s.WaitUntilPodAvailable(t, kubectlOptions, podZeroName, 15, 20*time.Second)
	// wait until the pod is in Running status
	output, err := testUtil.WaitUntilPodRunning(t, kubectlOptions, podZeroName, 10, 15*time.Second)
	if err != nil {
		t.Error(err.Error())
	}
	if output != "Running" {
		t.Error(output)
	}

	// wait until the pod is in Ready status
	tunnel := k8s.NewTunnel(
		kubectlOptions, k8s.ResourceTypePod, podZeroName, 8002, 8002)
	defer tunnel.Close()
	tunnel.ForwardPort(t)

	// change the group name for dnode and verify it passes
	t.Logf("====Test updating group name for %s to %s", groupName, newGroupName)
	groupEndpoint := fmt.Sprintf("http://%s/manage/v2/groups/%s/properties?format=json", tunnel.Endpoint(), newGroupName)
	groupChangedResult, err := VerifyGroupChange(t, groupEndpoint, newGroupName)
	if err != nil {
		t.Fatalf("Error in changing group name: %s", err.Error())
	}
	assert.Equal(t, true, groupChangedResult, "Group name change failed")
}

func TestMultipleGroupChange(t *testing.T) {
	username := "admin"
	password := "admin"
	imageRepo, repoPres := os.LookupEnv("dockerRepository")
	imageTag, tagPres := os.LookupEnv("dockerVersion")
	var initialChartVersion string
	namespaceName := "ml-" + strings.ToLower(random.UniqueId())
	kubectlOptions := k8s.NewKubectlOptions("", "", namespaceName)
	dnodeGrpName := "dnode"
	enodeGrpName := "enode"
	dnodeReleaseName := "dnode"
	enodeReleaseName := "enode"

	// Path to the helm chart we will test
	helmChartPath, e := filepath.Abs("../../charts")
	if e != nil {
		t.Fatalf(e.Error())
	}

	if !repoPres {
		imageRepo = "progressofficial/marklogic-db"
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
	dnodePodName := testUtil.HelmInstall(t, options, dnodeReleaseName, kubectlOptions, helmChartPath)

	// for debugging on jenkins
	k8s.RunKubectl(t, kubectlOptions, "describe", "pod", dnodePodName)

	k8s.WaitUntilPodAvailable(t, kubectlOptions, dnodePodName, 15, 20*time.Second)
	// wait until the pod is in Running status
	output, err := testUtil.WaitUntilPodRunning(t, kubectlOptions, dnodePodName, 10, 15*time.Second)
	if err != nil {
		t.Error(err.Error())
	}
	if output != "Running" {
		t.Error(output)
	}

	bootstrapHost := fmt.Sprintf("%s-0.%s.%s.svc.cluster.local", dnodeReleaseName, dnodeReleaseName, namespaceName)
	enodeOptions := &helm.Options{
		KubectlOptions: kubectlOptions,
		SetValues: map[string]string{
			"persistence.enabled":   "true",
			"replicaCount":          "1",
			"image.repository":      imageRepo,
			"image.tag":             imageTag,
			"auth.adminUsername":    username,
			"auth.adminPassword":    password,
			"group.name":            enodeGrpName,
			"bootstrapHostName":     bootstrapHost,
			"group.enableXdqpSsl":   "false",
			"logCollection.enabled": "false",
		},
	}
	t.Logf("====Installing Helm Chart " + enodeReleaseName)
	enodePodName0 := testUtil.HelmInstall(t, enodeOptions, enodeReleaseName, kubectlOptions, helmChartPath)

	// for debugging on jenkins
	k8s.RunKubectl(t, kubectlOptions, "describe", "pod", enodePodName0)

	// wait until the first enode pod is in Ready status
	k8s.WaitUntilPodAvailable(t, kubectlOptions, enodePodName0, 15, 20*time.Second)

	k8s.WaitUntilPodAvailable(t, kubectlOptions, enodePodName0, 15, 20*time.Second)
	// wait until the pod is in Running status
	output, err = testUtil.WaitUntilPodRunning(t, kubectlOptions, enodePodName0, 10, 15*time.Second)
	if err != nil {
		t.Error(err.Error())
	}
	if output != "Running" {
		t.Error(output)
	}

	newDnodeGroupName := "newDnode"
	newEnodeGroupName := "newEnode"

	helmUpgradeOptionsDnode := &helm.Options{
		KubectlOptions: kubectlOptions,
		SetValues: map[string]string{
			"group.name": newDnodeGroupName,
		},
	}
	helm.Upgrade(t, helmUpgradeOptionsDnode, helmChartPath, dnodeReleaseName)

	k8s.RunKubectl(t, kubectlOptions, "delete", "pod", dnodePodName)

	// for debugging on jenkins
	k8s.RunKubectl(t, kubectlOptions, "describe", "pod", dnodePodName)

	k8s.WaitUntilPodAvailable(t, kubectlOptions, dnodePodName, 15, 20*time.Second)
	// wait until the pod is in Running status
	output, err = testUtil.WaitUntilPodRunning(t, kubectlOptions, dnodePodName, 10, 15*time.Second)
	if err != nil {
		t.Error(err.Error())
	}
	if output != "Running" {
		t.Error(output)
	}

	tunnel := k8s.NewTunnel(
		kubectlOptions, k8s.ResourceTypePod, dnodePodName, 8002, 8002)
	defer tunnel.Close()
	tunnel.ForwardPort(t)

	// change the group name for dnode and verify it passes
	t.Logf("====Test updating group name for %s to %s", dnodeGrpName, newDnodeGroupName)
	groupDnodeEndpoint := fmt.Sprintf("http://%s/manage/v2/groups/%s/properties?format=json", tunnel.Endpoint(), newDnodeGroupName)
	groupChangedResult, err := VerifyGroupChange(t, groupDnodeEndpoint, newDnodeGroupName)
	if err != nil {
		t.Fatalf("Error in changing group name: %s", err.Error())
	}
	assert.Equal(t, true, groupChangedResult, "dnode Group name change failed")

	helmUpgradeOptionsEnode := &helm.Options{
		KubectlOptions: kubectlOptions,
		SetValues: map[string]string{
			"group.name": newEnodeGroupName,
		},
	}
	helm.Upgrade(t, helmUpgradeOptionsEnode, helmChartPath, enodeReleaseName)

	k8s.RunKubectl(t, kubectlOptions, "delete", "pod", enodePodName0)

	// for debugging on jenkins
	k8s.RunKubectl(t, kubectlOptions, "describe", "pod", enodePodName0)

	k8s.WaitUntilPodAvailable(t, kubectlOptions, enodePodName0, 15, 20*time.Second)
	// wait until the pod is in Running status
	output, err = testUtil.WaitUntilPodRunning(t, kubectlOptions, enodePodName0, 10, 15*time.Second)
	if err != nil {
		t.Error(err.Error())
	}
	if output != "Running" {
		t.Error(output)
	}

	// change the group name for dnode and verify it passes
	t.Logf("====Test updating group name for %s to %s", enodeGrpName, newEnodeGroupName)
	groupEnodeEndpoint := fmt.Sprintf("http://%s/manage/v2/groups/%s/properties?format=json", tunnel.Endpoint(), newEnodeGroupName)
	groupChangedResult, err = VerifyGroupChange(t, groupEnodeEndpoint, newEnodeGroupName)
	if err != nil {
		t.Fatalf("Error in changing group name: %s", err.Error())
	}
	assert.Equal(t, true, groupChangedResult, "enode Group name change failed")

}
