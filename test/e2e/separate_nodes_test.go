package e2e

import (
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
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

var username = "admin"
var password = "admin"

func VerifyDnodeConfig(t *testing.T, dnodePodName string, kubectlOptions *k8s.KubectlOptions, protocol string) (string, error) {
	tunnel := k8s.NewTunnel(
		kubectlOptions, k8s.ResourceTypePod, dnodePodName, 8002, 8002)
	defer tunnel.Close()
	tunnel.ForwardPort(t)

	hostManageEndpoint := fmt.Sprintf("%s://%s/manage/v2/hosts?format=json", protocol, tunnel.Endpoint())
	totalHosts := 0
	bootstrapHost := ""
	client := req.C().
		EnableInsecureSkipVerify().
		SetCommonDigestAuth("admin", "admin").
		SetCommonRetryCount(10).
		SetCommonRetryFixedInterval(10 * time.Second)

	resp, err := client.R().
		AddRetryCondition(func(resp *req.Response, err error) bool {
			if err != nil {
				t.Logf("===Error from retryFunc : %s", err.Error())
			}
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Logf("error: %s", err.Error())
			}
			totalHosts = int(gjson.Get(string(body), `host-default-list.list-items.list-count.value`).Num)
			bootstrapHost = (gjson.Get(string(body), `host-default-list.list-items.list-item.#(roleref="bootstrap").nameref`)).Str
			if totalHosts != 1 {
				t.Log("Waiting for host to configure")
			}
			return totalHosts != 1
		}).
		Get(hostManageEndpoint)
	defer resp.Body.Close()
	if err != nil {
		t.Fatalf(err.Error())
	}

	// verify bootstrap host exists on the cluster
	t.Log("====Verifying bootstrap host exists on the cluster")
	if bootstrapHost == "" {
		t.Errorf("Bootstrap does not exists on cluster")
	}

	dnodeEndpoint := fmt.Sprintf("%s://%s/manage/v2/groups/dnode/properties?format=json", protocol, tunnel.Endpoint())
	t.Log("====Verifying xdqp-ssl-enabled is set to true for dnode group")
	resp, err = client.R().
		Get(dnodeEndpoint)
	defer resp.Body.Close()
	if err != nil {
		t.Fatalf(err.Error())
	}
	body, err := io.ReadAll(resp.Body)
	xdqpSSLEnabled := gjson.Get(string(body), `xdqp-ssl-enabled`).Bool()

	// verify xdqp-ssl-enabled is set to true
	assert.Equal(t, true, xdqpSSLEnabled, "xdqp-ssl-enabled should be set to true")
	return bootstrapHost, err
}

func VerifyEnodeConfig(t *testing.T, dnodePodName string, kubectlOptions *k8s.KubectlOptions, protocol string) {
	tunnel := k8s.NewTunnel(
		kubectlOptions, k8s.ResourceTypePod, dnodePodName, 8002, 8002)
	defer tunnel.Close()
	tunnel.ForwardPort(t)

	t.Log("====Verify xdqp-ssl-enabled is set to false on Enode")
	endpoint := fmt.Sprintf("%s://%s/manage/v2/groups/enode/properties?format=json", protocol, tunnel.Endpoint())
	t.Logf(`Endpoint for group properties: %s`, endpoint)
	client := req.C().
		EnableInsecureSkipVerify().
		SetCommonDigestAuth("admin", "admin").
		SetCommonRetryCount(10).
		SetCommonRetryFixedInterval(10 * time.Second)
	resp, err := client.R().
		Get(endpoint)
	defer resp.Body.Close()
	if err != nil {
		t.Fatalf(err.Error())
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf(err.Error())
	}
	xdqpSSLEnabled := gjson.Get(string(body), `xdqp-ssl-enabled`)
	// verify xdqp-ssl-enabled is set to false
	assert.Equal(t, false, xdqpSSLEnabled.Bool())

	t.Log("====Verify both dnode and enode groups exist")
	groupEndpoint := fmt.Sprintf("%s://%s/manage/v2/groups", protocol, tunnel.Endpoint())
	t.Logf(`Endpoint: %s`, groupEndpoint)
	resp, err = client.R().
		Get(groupEndpoint)
	defer resp.Body.Close()
	if body, err = io.ReadAll(resp.Body); err != nil {
		t.Fatalf(err.Error())
	}
	// verify groups dnode, enode exists on the cluster
	if !strings.Contains(string(body), "<nameref>dnode</nameref>") && !strings.Contains(string(body), "<nameref>enode</nameref>") {
		t.Errorf("Groups does not exists on cluster")
	}

	enodeEndpoint := fmt.Sprintf("%s://%s/manage/v2/groups/enode?format=json", protocol, tunnel.Endpoint())
	enodeHostCount := 0
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
			enodeHostCount = int(totalHosts.Num)
			t.Logf("====Response: %d", resp.GetStatusCode())
			t.Logf("====enodeHostCount: %d", enodeHostCount)
			if enodeHostCount != 2 {
				t.Log("Waiting for hosts to join enode group")
			}
			return enodeHostCount != 2
		}).
		Get(enodeEndpoint)

	if reqErr != nil {
		t.Fatalf(reqErr.Error())
	}
	// verify two host exists on the cluster
	if enodeHostCount != 2 {
		t.Errorf("enode hosts does not exists on cluster")
	}
}

func TestSeparateEDnode(t *testing.T) {
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
			"group.name":            "dnode",
			"logCollection.enabled": "false",
		},
		Version: initialChartVersion,
	}

	t.Logf("====Creating namespace: " + namespaceName)
	k8s.CreateNamespace(t, kubectlOptions, namespaceName)

	defer t.Logf("====Deleting namespace: " + namespaceName)
	defer k8s.DeleteNamespace(t, kubectlOptions, namespaceName)

	//add the helm chart repo and install the last helm chart release from repository
	//to test and upgrade this chart to the latest one to be released
	if runUpgradeTest {
		helm.AddRepo(t, options, "marklogic", "https://marklogic.github.io/marklogic-kubernetes/")
		defer helm.RemoveRepo(t, options, "marklogic")
		helmChartPath = "marklogic/marklogic"
	}

	t.Logf("====Setting helm chart path to %s", helmChartPath)
	t.Logf("====Installing Helm Chart " + dnodeReleaseName)
	dnodePodName = testUtil.HelmInstall(t, options, dnodeReleaseName, kubectlOptions, helmChartPath)

	// wait until the pod is in ready status
	k8s.WaitUntilPodAvailable(t, kubectlOptions, dnodePodName, 15, 20*time.Second)
	bootstrapHost, err := VerifyDnodeConfig(t, dnodePodName, kubectlOptions, "http")
	if err != nil {
		t.Errorf(err.Error())
	}

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
			"bootstrapHostName":     bootstrapHost,
			"group.enableXdqpSsl":   "false",
			"logCollection.enabled": "false",
		},
	}
	t.Logf("====Installing Helm Chart " + enodeReleaseName)
	enodePodName0 = testUtil.HelmInstall(t, enodeOptions, enodeReleaseName, kubectlOptions, helmChartPath)

	// wait until the first enode pod is in Ready status
	k8s.WaitUntilPodAvailable(t, kubectlOptions, enodePodName0, 45, 20*time.Second)

	// wait until the second enode pod is in Ready status
	k8s.WaitUntilPodAvailable(t, kubectlOptions, enodePodName1, 45, 20*time.Second)

	VerifyEnodeConfig(t, dnodePodName, kubectlOptions, "http")

	if runUpgradeTest {
		dnodeUpgradeOptionsMap := map[string]string{
			"persistence.enabled":   "true",
			"logCollection.enabled": "false",
			"replicaCount":          "1",
			"group.name":            "dnode",
			"group.enableXdqpSsl":   "true",
			"auth.adminUsername":    username,
			"auth.adminPassword":    password,
			"allowLongHostnames":    "true",
		}
		enodeUpgradeOptionsMap := map[string]string{
			"persistence.enabled":   "true",
			"logCollection.enabled": "false",
			"replicaCount":          "2",
			"auth.adminUsername":    username,
			"auth.adminPassword":    password,
			"group.name":            "enode",
			"group.enableXdqpSsl":   "false",
			"allowLongHostnames":    "true",
		}
		if strings.HasPrefix(initialChartVersion, "1.0") {
			dnodePodName = dnodeReleaseName + "-marklogic-0"
			enodePodName0 = enodeReleaseName + "-marklogic-0"
			enodePodName1 = enodeReleaseName + "-marklogic-1"
			dnodeUpgradeOptionsMap["useLegacyHostnames"] = "true"
			enodeUpgradeOptionsMap["useLegacyHostnames"] = "true"
		}
		t.Logf("UpgradeHelmTest is enabled. Running helm upgrade test")
		//set helm options for upgrading Dnode release
		dnodeUpgradeOptions := &helm.Options{
			KubectlOptions: kubectlOptions,
			SetValues:      dnodeUpgradeOptionsMap,
		}

		testUtil.HelmUpgrade(t, dnodeUpgradeOptions, dnodeReleaseName, kubectlOptions, []string{dnodePodName}, initialChartVersion)
		output, err := testUtil.WaitUntilPodRunning(t, kubectlOptions, dnodePodName, 20, 20*time.Second)
		if err != nil {
			t.Error(err.Error())
		}
		if output != "Running" {
			t.Error(output)
		}
		bootstrapHost, err = VerifyDnodeConfig(t, dnodePodName, kubectlOptions, "http")
		enodeUpgradeOptionsMap["bootstrapHostName"] = bootstrapHost

		//set helm options for upgrading Enode releases
		enodeUpgradeOptions := &helm.Options{
			KubectlOptions: kubectlOptions,
			SetValues:      enodeUpgradeOptionsMap,
		}
		testUtil.HelmUpgrade(t, enodeUpgradeOptions, enodeReleaseName, kubectlOptions, []string{enodePodName0, enodePodName1}, initialChartVersion)
		output, err = testUtil.WaitUntilPodRunning(t, kubectlOptions, enodePodName0, 10, 15*time.Second)
		if err != nil {
			t.Error(err.Error())
		}
		if output != "Running" {
			t.Error(output)
		}
		VerifyEnodeConfig(t, dnodePodName, kubectlOptions, "http")
	}

	tlsConfig := tls.Config{}
	// restart all pods at once in the cluster and verify its ready and MarkLogic server is healthy
	testUtil.RestartPodAndVerify(t, true, []string{dnodePodName, enodePodName0, enodePodName1}, namespaceName, kubectlOptions, &tlsConfig)

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
		imageRepo = "progressofficial/marklogic-db"
		t.Logf("No imageRepo variable present, setting to default value: " + imageRepo)
	}

	if !tagPres {
		imageTag = "latest-11"
		t.Logf("No imageTag variable present, setting to default value: " + imageTag)
	}

	// Helm options for dnode creation
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
			"persistence.enabled":   "true",
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

	tlsConfig := tls.Config{}
	// restart pods in the cluster and verify its ready and MarkLogic server is healthy
	testUtil.RestartPodAndVerify(t, false, []string{dnodePodName}, namespaceName, kubectlOptions, &tlsConfig)

}
