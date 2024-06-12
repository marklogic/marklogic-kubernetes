package e2e

import (
	"fmt"
	"io"
	"io/ioutil"
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
	digestAuth "github.com/xinsnake/go-http-digest-auth-client"
)

func TestSeparateEDnode(t *testing.T) {
	username := "admin"
	password := "admin"
	imageRepo, repoPres := os.LookupEnv("dockerRepository")
	imageTag, tagPres := os.LookupEnv("dockerVersion")
	upgradeHelm, upgradeHelmTestPres := os.LookupEnv("upgradeTest")
	initialChartVersion, _ := os.LookupEnv("initialChartVersion")
	namespaceName := "ml-" + strings.ToLower(random.UniqueId())
	kubectlOptions := k8s.NewKubectlOptions("", "", namespaceName)
	dnodeReleaseName := "dnode"
	enodeReleaseName := "enode"
	dnodePodName := dnodeReleaseName + "-0"
	enodePodName0 := enodeReleaseName + "-0"
	enodePodName1 := enodeReleaseName + "-1"

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
		imageTag = "latest"
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
			"group.name":            "dnode",
			"group.enableXdqpSsl":   "true",
			"logCollection.enabled": "false",
		},
	}

	t.Logf("====Creating namespace: " + namespaceName)
	k8s.CreateNamespace(t, kubectlOptions, namespaceName)

	defer t.Logf("====Deleting namespace: " + namespaceName)
	defer k8s.DeleteNamespace(t, kubectlOptions, namespaceName)

	//add the helm chart repo and install the last helm chart release from repository
	//to test and upgrade this chart to the latest one to be released
	if upgradeHelmTestPres {
		helm.RemoveRepo(t, options, "marklogic")
		helm.AddRepo(t, options, "marklogic", "https://marklogic.github.io/marklogic-kubernetes/")
		helmChartPath = "marklogic/marklogic"
	}

	t.Logf("====Installing Helm Chart " + dnodeReleaseName)
	dnodePodName = testUtil.HelmInstall(t, options, dnodeReleaseName, kubectlOptions, helmChartPath)

	// wait until the pod is in ready status
	k8s.WaitUntilPodAvailable(t, kubectlOptions, dnodePodName, 15, 20*time.Second)

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

	bootstrapHostJSON := gjson.Get(string(body), `host-default-list.list-items.list-item.#(roleref="bootstrap").nameref`)
	t.Logf(`BootstrapHost: = %s`, bootstrapHostJSON)
	// verify bootstrap host exists on the cluster
	if bootstrapHostJSON.Str == "" {
		t.Errorf("Bootstrap does not exists on cluster")
	}

	t.Log("====Verify xdqp-ssl-enabled is set to true")
	endpoint := fmt.Sprintf("http://%s/manage/v2/groups/dnode/properties?format=json", tunnel.Endpoint())
	t.Logf(`Endpoint for group properties: %s`, endpoint)

	request := digestAuth.NewRequest(username, password, "GET", endpoint, "")
	resp, err = request.Execute()
	if err != nil {
		t.Fatalf(err.Error())
	}
	defer resp.Body.Close()
	body, err = io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf(err.Error())
	}
	t.Log(string(body))
	xdqpSSLEnabled := gjson.Get(string(body), `xdqp-ssl-enabled`)
	// verify xdqp-ssl-enabled is set to trues
	assert.Equal(t, true, xdqpSSLEnabled.Bool(), "xdqp-ssl-enabled should be set to true")

	enodeOptions := &helm.Options{
		KubectlOptions: kubectlOptions,
		SetValues: map[string]string{
			"persistence.enabled":   "true",
			"replicaCount":          "2",
			"image.repository":      imageRepo,
			"image.tag":             imageTag,
			"auth.adminUsername":    username,
			"auth.adminPassword":    password,
			"group.name":            "enode",
			"bootstrapHostName":     bootstrapHostJSON.Str,
			"group.enableXdqpSsl":   "false",
			"logCollection.enabled": "false",
		},
	}
	t.Logf("====Installing Helm Chart " + enodeReleaseName)
	enodePodName0 = testUtil.HelmInstall(t, enodeOptions, enodeReleaseName, kubectlOptions, helmChartPath)

	// wait until the first enode pod is in Ready status
	k8s.WaitUntilPodAvailable(t, kubectlOptions, enodePodName0, 45, 20*time.Second)

	t.Log("====Verify xdqp-ssl-enabled is set to false on Enode")
	endpoint = fmt.Sprintf("http://%s/manage/v2/groups/enode/properties?format=json", tunnel.Endpoint())
	t.Logf(`Endpoint for group properties: %s`, endpoint)

	request = digestAuth.NewRequest(username, password, "GET", endpoint, "")
	resp, err = request.Execute()
	if err != nil {
		t.Fatalf(err.Error())
	}
	defer resp.Body.Close()
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf(err.Error())
	}

	xdqpSSLEnabled = gjson.Get(string(body), `xdqp-ssl-enabled`)
	// verify xdqp-ssl-enabled is set to false
	assert.Equal(t, false, xdqpSSLEnabled.Bool())

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

	// verify groups dnode, enode exists on the cluster
	if !strings.Contains(string(body), "<nameref>dnode</nameref>") && !strings.Contains(string(body), "<nameref>enode</nameref>") {
		t.Errorf("Groups does not exists on cluster")
	}

	// wait until the second enode pod is in Ready status
	k8s.WaitUntilPodAvailable(t, kubectlOptions, enodePodName1, 45, 20*time.Second)

	//set helm options for upgrading E/D node releases
	helmUpgradeOptions := &helm.Options{
		KubectlOptions: kubectlOptions,
		SetValues: map[string]string{
			"persistence.enabled":   "true",
			"logCollection.enabled": "false",
			"useLegacyHostnames":    "true",
			"allowLongHostnames":    "true",
		},
	}

	if upgradeHelmTestPres {
		t.Logf("UpgradeHelmTest is set to %s. Running helm upgrade test" + upgradeHelm)
		testUtil.HelmUpgrade(t, helmUpgradeOptions, dnodeReleaseName, kubectlOptions, []string{dnodePodName}, initialChartVersion)
		testUtil.HelmUpgrade(t, helmUpgradeOptions, enodeReleaseName, kubectlOptions, []string{enodePodName0, enodePodName1}, initialChartVersion)
	}

	enodeHostCountJSON := 0
	client := req.C()
	_, reqErr := client.R().
		SetDigestAuth(username, password).
		SetRetryCount(3).
		SetRetryFixedInterval(10 * time.Second).
		AddRetryCondition(func(resp *req.Response, err error) bool {
			if resp == nil || err != nil {
				t.Logf("error in AddRetryCondition: %s", err.Error())
				return true
			}
			if resp.Response == nil {
				t.Log("Could not get the Response Object, Retrying...")
				return true
			}
			if resp.Body == nil {
				t.Log("Could not get the body for the response, Retrying...")
				return true
			}
			body, err := io.ReadAll(resp.Body)
			if body == nil || err != nil {
				t.Logf("error in read response body: %s", err.Error())
				return true
			}
			totalHosts := gjson.Get(string(body), `group-default.relations.relation-group.#(typeref="hosts").relation-count.value`)
			enodeHostCountJSON = int(totalHosts.Num)
			if enodeHostCountJSON != 2 {
				t.Log("Waiting for hosts to join enode group")
			}
			return enodeHostCountJSON != 2
		}).
		Get("http://localhost:8002/manage/v2/groups/enode?format=json")

	if reqErr != nil {
		t.Fatalf(reqErr.Error())
	}

	// verify bootstrap host exists on the cluster
	if enodeHostCountJSON != 2 {
		t.Errorf("enode hosts does not exists on cluster")
	}
}

func TestIncorrectBootsrapHostname(t *testing.T) {
	username := "admin"
	password := "admin"
	imageRepo, repoPres := os.LookupEnv("dockerRepository")
	imageTag, tagPres := os.LookupEnv("dockerVersion")
	namespaceName := "ml-" + strings.ToLower(random.UniqueId())
	kubectlOptions := k8s.NewKubectlOptions("", "", namespaceName)
	dnodeReleaseName := "dnode"
	enodeReleaseName := "enode"
	dnodePodName := dnodeReleaseName + "-0"

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
	k8s.WaitUntilPodAvailable(t, kubectlOptions, dnodePodName, 15, 20*time.Second)

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
