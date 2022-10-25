package e2e

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/gruntwork-io/terratest/modules/random"
	"github.com/tidwall/gjson"
	digestAuth "github.com/xinsnake/go-http-digest-auth-client"
)

func TestSeparateEDnode(t *testing.T) {
	// Path to the helm chart we will test
	helmChartPath, e := filepath.Abs("../../charts")
	if e != nil {
		t.Fatalf(e.Error())
	}
	username := "admin"
	password := "admin"
	var resp *http.Response
	var body []byte
	var err error

	imageRepo, repoPres := os.LookupEnv("dockerRepository")
	imageTag, tagPres := os.LookupEnv("dockerVersion")

	if !repoPres {
		imageRepo = "marklogic-centos/marklogic-server-centos"
		t.Logf("No imageRepo variable present, setting to default value: " + imageRepo)
	}

	if !tagPres {
		imageTag = "10-internal"
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

	dnodeReleaseName := "test-dnode-group"
	t.Logf("====Installing Helm Chart" + dnodeReleaseName)
	helm.Install(t, options, helmChartPath, dnodeReleaseName)

	dnodePodName := dnodeReleaseName + "-marklogic-0"

	// wait until the pod is in Ready status
	k8s.WaitUntilPodAvailable(t, kubectlOptions, dnodePodName, 10, 20*time.Second)

	time.Sleep(10 * time.Second)
	tunnel := k8s.NewTunnel(
		kubectlOptions, k8s.ResourceTypePod, dnodePodName, 8002, 8002)
	defer tunnel.Close()
	tunnel.ForwardPort(t)
	hostsEndpoint := fmt.Sprintf("http://%s/manage/v2/hosts?format=json", tunnel.Endpoint())
	t.Logf(`Endpoint: %s`, hostsEndpoint)

	dr := digestAuth.NewRequest(username, password, "GET", hostsEndpoint, "")

	if resp, err = dr.Execute(); err != nil {
		t.Fatalf(err.Error())
	}
	defer resp.Body.Close()

	if body, err = ioutil.ReadAll(resp.Body); err != nil {
		t.Fatalf(err.Error())
	}
	t.Logf("Response:\n" + string(body))
	bootstrapHost := gjson.Get(string(body), `host-default-list.list-items.list-item.#(roleref="bootstrap").nameref`)
	t.Logf(`BootstrapHost: = %s`, bootstrapHost)

	// verify bootstrap host exists on the cluster
	if bootstrapHost.String() == "" {
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
			"bootstrapHostName":     bootstrapHost.String(),
			"logCollection.enabled": "false",
		},
	}
	enodeReleaseName := "test-enode-group"
	t.Logf("====Installing Helm Chart " + enodeReleaseName)
	helm.Install(t, enodeOptions, helmChartPath, enodeReleaseName)

	enodePodName0 := enodeReleaseName + "-marklogic-0"

	// wait until the first enode pod is in Ready status
	k8s.WaitUntilPodAvailable(t, kubectlOptions, enodePodName0, 45, 20*time.Second)

	groupEndpoint := fmt.Sprintf("http://%s/manage/v2/groups", tunnel.Endpoint())
	t.Logf(`Endpoint: %s`, groupEndpoint)

	drGroups := digestAuth.NewRequest(username, password, "GET", groupEndpoint, "")

	if resp, err = drGroups.Execute(); err != nil {
		t.Fatalf(err.Error())
	}
	defer resp.Body.Close()

	if body, err = ioutil.ReadAll(resp.Body); err != nil {
		t.Fatalf(err.Error())
	}
	t.Logf("Response:\n" + string(body))

	// verify groups dnode, enode exists on the cluster
	if !strings.Contains(string(body), "<nameref>dnode</nameref>") && !strings.Contains(string(body), "<nameref>enode</nameref>") {
		t.Errorf("Groups does not exists on cluster")
	}

	enodePodName1 := enodeReleaseName + "-marklogic-1"

	// wait until the second enode pod is in Ready status
	k8s.WaitUntilPodAvailable(t, kubectlOptions, enodePodName1, 45, 20*time.Second)

	enodeEndpoint := fmt.Sprintf("http://%s/manage/v2/groups/enode?format=json", tunnel.Endpoint())
	t.Logf(`Endpoint: %s`, enodeEndpoint)

	drEnode := digestAuth.NewRequest(username, password, "GET", enodeEndpoint, "")

	if resp, err = drEnode.Execute(); err != nil {
		t.Fatalf(err.Error())
	}
	defer resp.Body.Close()

	if body, err = ioutil.ReadAll(resp.Body); err != nil {
		t.Fatalf(err.Error())
	}
	t.Logf("Response:\n" + string(body))

	enodeHostCount := gjson.Get(string(body), `group-default.relations.relation-group.#(typeref="hosts").relation-count.value`)
	t.Logf(`enodeHostCount: = %s`, enodeHostCount)

	// verify bootstrap host exists on the cluster
	if !strings.Contains(enodeHostCount.String(), "2") {
		t.Errorf("enode hosts does not exists on cluster")
	}
}

func TestIncorrectBootsrapHostname(t *testing.T) {
	var resp *http.Response
	var body []byte
	var err error

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
	bootstrapHost := "Incorrect Host Name"

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

	t.Logf("====Installing D Node Helm Chart" + dnodeReleaseName)
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

	if resp, err = getHostsRequest.Execute(); err != nil {
		t.Fatalf(err.Error())
	}

	defer resp.Body.Close()

	if body, err = ioutil.ReadAll(resp.Body); err != nil {
		t.Fatalf(err.Error())
	}

	t.Logf("Response:\n" + string(body))
	t.Logf(`BootstrapHost: = %s`, bootstrapHost)
	
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
			"bootstrapHostName":     bootstrapHost,
			"logCollection.enabled": "false",
		},
	}

	t.Logf("====Installing E Node Helm Chart" + enodeReleaseName)
	helm.Install(t, enodeOptions, helmChartPath, enodeReleaseName)

	// Give pod time to fail before checking if it did
	time.Sleep(20 * time.Second)

	// Verify clustering failed given incorrect hostname 
	clusterStatusEndpoint := fmt.Sprintf("http://%s/manage/v2?view=status", tunnel.Endpoint())
	clusterStatus := digestAuth.NewRequest(username, password, "GET", clusterStatusEndpoint, "")
	t.Logf(`clusterStatusEndpoint: %s`, clusterStatusEndpoint)
	if resp, err = clusterStatus.Execute(); err != nil {
		t.Fatalf(err.Error())
	}
	totalHostsJson := gjson.Get(string(body), "host-default-list.list-items.list-count.value")
	// Total hosts be one as second host should have failed to create
	if (totalHostsJson.Num != 1) {
		t.Errorf("Wrong number of hosts: %v instead of 1", totalHostsJson.Num)
	}
	t.Logf("\nCluster Status Response:\n\n" + string(body))

	// Verify enode group creation failed given incorrect hostname 
	enodeGroupStatusEndpoint := fmt.Sprintf("http://%s/manage/v2/groups/enode", tunnel.Endpoint())
	groupStatus := digestAuth.NewRequest(username, password, "GET", enodeGroupStatusEndpoint, "")
	t.Logf(`enodeGroupStatusEndpoint: %s`, enodeGroupStatusEndpoint)
	if resp, err = groupStatus.Execute(); err != nil {
		t.Fatalf(err.Error())
	}
	if body, err = ioutil.ReadAll(resp.Body); err != nil {
		t.Fatalf(err.Error())
	}
	if (!strings.Contains(string(body), "404")) {
		t.Errorf("Enode group should not exist")
	}
	t.Logf("\nEnode Group Status Response:\n\n" + string(body))
}
