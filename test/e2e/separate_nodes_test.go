package e2e

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/gruntwork-io/terratest/modules/random"
	digest_auth "github.com/xinsnake/go-http-digest-auth-client"
	"github.com/tidwall/gjson"
)

func TestSeparateEDnode(t *testing.T) {
	// Path to the helm chart we will test 
	helmChartPath, e := filepath.Abs("../../charts")
	if (e != nil) {
		t.Fatalf(e.Error())
	}
	username := "admin"
	password := "admin"
	var resp *http.Response
	var body []byte
	var err error

	namespaceName := "marklogic-" + strings.ToLower(random.UniqueId())
	kubectlOptions := k8s.NewKubectlOptions("", "", namespaceName)
	options := &helm.Options{
		KubectlOptions: kubectlOptions,
		SetValues: map[string]string{
			"persistence.enabled": "false",
			"replicaCount":        "1",
			"image.repository":    "marklogic-centos/marklogic-server-centos",
			"image.tag":           "10-internal",
			"auth.adminUsername":  username,
			"auth.adminPassword":  password,
			"group.name": 		   "dnode",
			"logCollection.enabled":    "false",
		},
	}

	t.Logf("====Creating namespace: " + namespaceName)
	k8s.CreateNamespace(t, kubectlOptions, namespaceName)

	defer t.Logf("====Deleting namespace: " + namespaceName)
	defer k8s.DeleteNamespace(t, kubectlOptions, namespaceName)
	
	releaseName := "test-dnode-group"
	t.Logf("====Installing Helm Chart" + releaseName)
	helm.Install(t, options, helmChartPath, releaseName)

	podName := releaseName + "-marklogic-0"

	// wait until the pod is in Ready status
	k8s.WaitUntilPodAvailable(t, kubectlOptions, podName, 10, 20*time.Second)

	time.Sleep(10 * time.Second)
	tunnel := k8s.NewTunnel(
		kubectlOptions, k8s.ResourceTypePod, podName, 8002, 8002)
	defer tunnel.Close()
	tunnel.ForwardPort(t)
	hosts_endpoint := fmt.Sprintf("http://%s/manage/v2/hosts?format=json", tunnel.Endpoint())
	t.Logf(`Endpoint: %s`, hosts_endpoint)

	dr := digest_auth.NewRequest(username, password, "GET", hosts_endpoint, "")

	if resp, err = dr.Execute(); err != nil {
		t.Fatalf(err.Error())
	}
	defer resp.Body.Close()

	if body, err = ioutil.ReadAll(resp.Body); err != nil {
		t.Fatalf(err.Error())
	}
	t.Logf("Response:\n" + string(body))
	bootstrapHost := gjson.Get(string(body), `host-default-list.list-items.list-item.#(roleref="bootstrap").nameref`)
	t.Logf(`BootstrapHost: = %s` , bootstrapHost)

	enodeOptions := &helm.Options{
		KubectlOptions: kubectlOptions,
		SetValues: map[string]string{
			"persistence.enabled": "false",
			"replicaCount":        "2",
			"image.repository":    "marklogic-centos/marklogic-server-centos",
			"image.tag":           "10-internal",
			"auth.adminUsername":  username,
			"auth.adminPassword":  password,
			"group.name": 		   "enode",
			"bootstrapHostName":   bootstrapHost.String(),
			"logCollection.enabled":    "false",
		},
	}
	releaseName2 := "test-enode-group"
	t.Logf("====Installing Helm Chart " + releaseName2)
	helm.Install(t, enodeOptions, helmChartPath, releaseName2)

	enodePodName := releaseName2 + "-marklogic-0"

	// wait until the pod is in Ready status
	k8s.WaitUntilPodAvailable(t, kubectlOptions, enodePodName, 15, 20*time.Second)

	group_endpoint := fmt.Sprintf("http://%s/manage/v2/groups", tunnel.Endpoint())
	t.Logf(`Endpoint: %s`, group_endpoint)

	dr_groups := digest_auth.NewRequest(username, password, "GET", group_endpoint, "")

	if resp, err = dr_groups.Execute(); err != nil {
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

}