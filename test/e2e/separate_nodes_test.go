package e2e

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/gruntwork-io/terratest/modules/random"
	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
	digestAuth "github.com/xinsnake/go-http-digest-auth-client"
)

func TestSeparateEDnode(t *testing.T) {
	username := "admin"
	password := "admin"
	imageRepo, repoPres := os.LookupEnv("dockerRepository")
	imageTag, tagPres := os.LookupEnv("dockerVersion")
	namespaceName := "marklogic-" + strings.ToLower(random.UniqueId())
	kubectlOptions := k8s.NewKubectlOptions("", "", namespaceName)
	dnodeReleaseName := "test-dnode-group"
	enodeReleaseName := "test-enode-group"
	dnodePodName := dnodeReleaseName + "-marklogic-0"
	enodePodName0 := enodeReleaseName + "-marklogic-0"
	enodePodName1 := enodeReleaseName + "-marklogic-1"

	// Path to the helm chart we will test
	helmChartPath, e := filepath.Abs("../../charts")
	if e != nil {
		t.Fatalf(e.Error())
	}

	if !repoPres {
		imageRepo = "marklogic-centos/marklogic-server-centos"
		t.Logf("No imageRepo variable present, setting to default value: " + imageRepo)
	}

	if !tagPres {
		imageTag = "10-internal"
		t.Logf("No imageTag variable present, setting to default value: " + imageTag)
	}

	options := &helm.Options{
		KubectlOptions: kubectlOptions,
		SetValues: map[string]string{
			"persistence.enabled":   "false",
			"replicaCount":          "1",
			"image.repository":      imageRepo,
			"image.tag":             imageTag,
			"auth.adminUsername":    username,
			"auth.adminPassword":    password,
			"group.name":            "dnode",
			"logCollection.enabled": "false",
		},
	}

	t.Logf("====Creating namespace: " + namespaceName)
	k8s.CreateNamespace(t, kubectlOptions, namespaceName)

	defer t.Logf("====Deleting namespace: " + namespaceName)
	defer k8s.DeleteNamespace(t, kubectlOptions, namespaceName)

	t.Logf("====Installing Helm Chart " + dnodeReleaseName)
	helm.Install(t, options, helmChartPath, dnodeReleaseName)

	// wait until the pod is in ready status
	k8s.WaitUntilPodAvailable(t, kubectlOptions, dnodePodName, 10, 20*time.Second)

	tunnel := k8s.NewTunnel(
		kubectlOptions, k8s.ResourceTypePod, dnodePodName, 8002, 8002)
	defer tunnel.Close()
	tunnel.ForwardPort(t)

	hostsEndpoint := fmt.Sprintf("http://%s/manage/v2/hosts?format=json", tunnel.Endpoint())
	t.Logf(`Endpoint: %s`, hostsEndpoint)

	getHostsDR := digestAuth.NewRequest(username, password, "GET", hostsEndpoint, "")

	resp, err := getHostsDR.Execute()
	if err != nil {
		t.Fatalf(err.Error())
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf(err.Error())
	}
	t.Logf("Get hosts response:\n" + string(body))

	bootstrapHostJSON := gjson.Get(string(body), `host-default-list.list-items.list-item.#(roleref="bootstrap").nameref`)
	t.Logf(`BootstrapHost: = %s`, bootstrapHostJSON)
	// verify bootstrap host exists on the cluster
	if bootstrapHostJSON.Str == "" {
		t.Errorf("Bootstrap does not exists on cluster")
	}

	enodeOptions := &helm.Options{
		KubectlOptions: kubectlOptions,
		SetValues: map[string]string{
			"persistence.enabled":   "false",
			"replicaCount":          "2",
			"image.repository":      imageRepo,
			"image.tag":             imageTag,
			"auth.adminUsername":    username,
			"auth.adminPassword":    password,
			"group.name":            "enode",
			"bootstrapHostName":     bootstrapHostJSON.Str,
			"logCollection.enabled": "false",
		},
	}
	t.Logf("====Installing Helm Chart " + enodeReleaseName)
	helm.Install(t, enodeOptions, helmChartPath, enodeReleaseName)

	// wait until the first enode pod is in Ready status
	k8s.WaitUntilPodAvailable(t, kubectlOptions, enodePodName0, 45, 20*time.Second)

	groupEndpoint := fmt.Sprintf("http://%s/manage/v2/groups", tunnel.Endpoint())
	t.Logf(`Endpoint: %s`, groupEndpoint)

	getGroupsDR := digestAuth.NewRequest(username, password, "GET", groupEndpoint, "")

	if resp, err = getGroupsDR.Execute(); err != nil {
		t.Fatalf(err.Error())
	}
	defer resp.Body.Close()
	if body, err = ioutil.ReadAll(resp.Body); err != nil {
		t.Fatalf(err.Error())
	}
	t.Logf("Groups status response:\n" + string(body))

	// verify groups dnode, enode exists on the cluster
	if !strings.Contains(string(body), "<nameref>dnode</nameref>") && !strings.Contains(string(body), "<nameref>enode</nameref>") {
		t.Errorf("Groups does not exists on cluster")
	}

	// wait until the second enode pod is in Ready status
	k8s.WaitUntilPodAvailable(t, kubectlOptions, enodePodName1, 45, 20*time.Second)

	enodeEndpoint := fmt.Sprintf("http://%s/manage/v2/groups/enode?format=json", tunnel.Endpoint())
	t.Logf(`Endpoint: %s`, enodeEndpoint)

	getEnodeDR := digestAuth.NewRequest(username, password, "GET", enodeEndpoint, "")

	resp, err = getEnodeDR.Execute()
	if err != nil {
		t.Fatalf(err.Error())
	}
	defer resp.Body.Close()

	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf(err.Error())
	}
	t.Logf("Get enode group response:\n" + string(body))

	enodeHostCountJSON := gjson.Get(string(body), `group-default.relations.relation-group.#(typeref="hosts").relation-count.value`)
	t.Logf(`enodeHostCount: = %s`, enodeHostCountJSON)

	// verify bootstrap host exists on the cluster
	if enodeHostCountJSON.Num != 2 {
		t.Errorf("enode hosts does not exists on cluster")
	}
}

func TestIncorrectBootsrapHostname(t *testing.T) {
	username := "admin"
	password := "admin"
	imageRepo, repoPres := os.LookupEnv("dockerRepository")
	imageTag, tagPres := os.LookupEnv("dockerVersion")
	namespaceName := "marklogic-" + strings.ToLower(random.UniqueId())
	kubectlOptions := k8s.NewKubectlOptions("", "", namespaceName)
	dnodeReleaseName := "test-dnode-group"
	enodeReleaseName := "test-enode-group"
	dnodePodName := dnodeReleaseName + "-marklogic-0"

	// Incorrect boostrap hostname for negative test
	incorrectBootstrapHost := "Incorrect Host Name"

	// Path to the helm chart we will test
	helmChartPath, e := filepath.Abs("../../charts")

	if e != nil {
		t.Fatalf(e.Error())
	}

	if !repoPres {
		imageRepo = "marklogic-centos/marklogic-server-centos"
		t.Logf("No imageRepo variable present, setting to default value: " + imageRepo)
	}

	if !tagPres {
		imageTag = "10-internal"
		t.Logf("No imageTag variable present, setting to default value: " + imageTag)
	}

	// Helm options for dnode creation
	options := &helm.Options{
		KubectlOptions: kubectlOptions,
		SetValues: map[string]string{
			"persistence.enabled":   "false",
			"replicaCount":          "1",
			"image.repository":      imageRepo,
			"image.tag":             imageTag,
			"auth.adminUsername":    username,
			"auth.adminPassword":    password,
			"group.name":            "dnode",
			"logCollection.enabled": "false",
		},
	}

	t.Logf("====Creating namespace: " + namespaceName)
	k8s.CreateNamespace(t, kubectlOptions, namespaceName)

	defer t.Logf("====Deleting namespace: " + namespaceName)
	defer k8s.DeleteNamespace(t, kubectlOptions, namespaceName)

	t.Logf("====Installing D Node Helm Chart " + dnodeReleaseName)
	helm.Install(t, options, helmChartPath, dnodeReleaseName)

	// wait until the pod is in ready status
	k8s.WaitUntilPodAvailable(t, kubectlOptions, dnodePodName, 10, 20*time.Second)

	tunnel := k8s.NewTunnel(
		kubectlOptions, k8s.ResourceTypePod, dnodePodName, 8002, 8002)

	defer tunnel.Close()

	tunnel.ForwardPort(t)
	hostsEndpoint := fmt.Sprintf("http://%s/manage/v2/hosts?format=json", tunnel.Endpoint())
	t.Logf(`Endpoint: %s`, hostsEndpoint)

	getHostsRequest := digestAuth.NewRequest(username, password, "GET", hostsEndpoint, "")
	resp, err := getHostsRequest.Execute()
	if err != nil {
		t.Fatalf(err.Error())
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf(err.Error())
	}

	t.Logf(`BootstrapHost: = %s`, incorrectBootstrapHost)

	// Helm options for enode creation
	enodeOptions := &helm.Options{
		KubectlOptions: kubectlOptions,
		SetValues: map[string]string{
			"persistence.enabled":   "false",
			"replicaCount":          "1",
			"image.repository":      imageRepo,
			"image.tag":             imageTag,
			"auth.adminUsername":    username,
			"auth.adminPassword":    password,
			"group.name":            "enode",
			"bootstrapHostName":     incorrectBootstrapHost,
			"logCollection.enabled": "false",
		},
	}

	t.Logf("====Installing E Node Helm Chart " + enodeReleaseName)
	helm.Install(t, enodeOptions, helmChartPath, enodeReleaseName)

	// Give pod time to fail before checking if it did
	time.Sleep(20 * time.Second)

	totalHostsJSON := gjson.Get(string(body), "host-default-list.list-items.list-count.value")

	// Total hosts be one as second host should have failed to create
	if totalHostsJSON.Num != 1 {
		t.Errorf("Wrong number of hosts: %v instead of 1", totalHostsJSON.Num)
	}

	// Verify clustering failed given incorrect hostname
	clusterStatusEndpoint := fmt.Sprintf("http://%s/manage/v2?view=status", tunnel.Endpoint())
	clusterStatus := digestAuth.NewRequest(username, password, "GET", clusterStatusEndpoint, "")
	t.Logf(`clusterStatusEndpoint: %s`, clusterStatusEndpoint)
	resp, err = clusterStatus.Execute()
	if err != nil {
		t.Fatalf(err.Error())
	}
	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf(err.Error())
	}

	// Verify enode group creation failed given incorrect hostname
	enodeGroupStatusEndpoint := fmt.Sprintf("http://%s/manage/v2/groups/enode", tunnel.Endpoint())
	groupStatus := digestAuth.NewRequest(username, password, "GET", enodeGroupStatusEndpoint, "")
	t.Logf(`enodeGroupStatusEndpoint: %s`, enodeGroupStatusEndpoint)
	resp, err = groupStatus.Execute()
	if err != nil {
		t.Fatalf(err.Error())
	}
	// the request for enode should be 404
	assert.Equal(t, 404, resp.StatusCode)
}
