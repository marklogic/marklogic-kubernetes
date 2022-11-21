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
			"group.enableXdqpSsl":   "true",
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

	// test to verify XDQPSSL is enabled
	dnodePropEndpoint := fmt.Sprintf("http://%s/manage/v2/groups/dnode/properties?format=json", tunnel.Endpoint())
	t.Logf(`Endpoint: %s`, dnodePropEndpoint)

	drDnodeProp := digestAuth.NewRequest(username, password, "GET", dnodePropEndpoint, "")

	if resp, err = drDnodeProp.Execute(); err != nil {
		t.Fatalf(err.Error())
	}
	defer resp.Body.Close()

	if body, err = ioutil.ReadAll(resp.Body); err != nil {
		t.Fatalf(err.Error())
	}
	t.Logf("Response:\n" + string(body))

	dnodeXdqpSSL := gjson.Get(string(body), `xdqp-ssl-enabled`)
	t.Logf(`dnodeXdqpSSL: = %s`, dnodeXdqpSSL)

	// verify xdqpssl is enabled on the host
	if !strings.Contains(dnodeXdqpSSL.String(), "true") {
		t.Errorf("xdqp-ssl-enabled is disabled on dnode")
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
			"group.enableXdqpSsl":   "true",
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

	// test to verify XDQPSSL is enabled
	enodePropEndpoint := fmt.Sprintf("http://%s/manage/v2/groups/enode/properties?format=json", tunnel.Endpoint())
	t.Logf(`Endpoint: %s`, enodePropEndpoint)

	drEnodeProp := digestAuth.NewRequest(username, password, "GET", enodePropEndpoint, "")

	if resp, err = drEnodeProp.Execute(); err != nil {
		t.Fatalf(err.Error())
	}
	defer resp.Body.Close()

	if body, err = ioutil.ReadAll(resp.Body); err != nil {
		t.Fatalf(err.Error())
	}
	t.Logf("Response:\n" + string(body))

	enodeXdqpSSL := gjson.Get(string(body), `xdqp-ssl-enabled`)
	t.Logf(`enodeXdqpSSL: = %s`, enodeXdqpSSL)

	// verify xdqpssl is enabled on the host
	if !strings.Contains(enodeXdqpSSL.String(), "true") {
		t.Errorf("xdqp-ssl-enabled is disabled on enode")
	}
}
