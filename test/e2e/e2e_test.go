package e2e

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	
	"github.com/gruntwork-io/terratest/modules/helm"
	http_helper "github.com/gruntwork-io/terratest/modules/http-helper"
	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/gruntwork-io/terratest/modules/random"
	digest_auth "github.com/judgegregg/go-http-digest-auth-client"
	"github.com/stretchr/testify/require"
)

// Path to the helm chart we will test
var helmChartPath, err = filepath.Abs("../../charts")
var releaseName string = "test"
var namespaceName string = "marklogic-" + strings.ToLower(random.UniqueId())
var kubectlOptions = k8s.NewKubectlOptions("", "", namespaceName)
var options = &helm.Options{
	KubectlOptions: kubectlOptions,
	SetValues: map[string]string{
		"persistence.enabled": "false",
		"replicaCount":        "1",
		"image.repository":    "marklogic-centos/marklogic-server-centos",
		"image.tag":           "10-internal",
	},
}

func TestMain(m *testing.M) {
	t := &testing.T{}
	require.NoError(t, err)
	log.Println("====Creating namespace: " + namespaceName)

	// create a new namespace for testing
	k8s.CreateNamespace(t, kubectlOptions, namespaceName)

	// anything before this runs before the tests run
	exitVal := m.Run()
	// anything after this runs after the tests run
	log.Println("====Deleting Helm Releases: " + namespaceName)
	helm.Delete(t, options, releaseName+"-upgrade", true)
	helm.Delete(t, options, releaseName+"-install", true)
	helm.Delete(t, options, releaseName+"-join", true)
	log.Println("====Deleting namespace: " + namespaceName)
	k8s.DeleteNamespace(t, kubectlOptions, namespaceName)

	os.Exit(exitVal)
}

func TestHelmInstall(t *testing.T) {
	t.Logf("====Installing Helm Chart")
	releaseName := releaseName + "-install"
	helm.Install(t, options, helmChartPath, releaseName)

	tlsConfig := tls.Config{}
	podName := releaseName + "-marklogic-0"
	// wait until the pod is in Ready status
	k8s.WaitUntilPodAvailable(t, kubectlOptions, podName, 10, 15*time.Second)
	tunnel := k8s.NewTunnel(
		kubectlOptions, k8s.ResourceTypePod, podName, 7997, 7997)
	defer tunnel.Close()
	tunnel.ForwardPort(t)
	endpoint := fmt.Sprintf("http://%s", tunnel.Endpoint())
	t.Logf(`Endpoint: %s`, endpoint)

	http_helper.HttpGetWithRetryWithCustomValidation(
		t,
		endpoint,
		&tlsConfig,
		10,
		15*time.Second,
		func(statusCode int, body string) bool {
			return statusCode == 200
		},
	)
}

func TestHelmUpgrade(t *testing.T) {
	t.Logf("====Installing Helm Chart")
	releaseName := releaseName + "-upgrade"
	helm.Install(t, options, helmChartPath, releaseName)

	newOptions := &helm.Options{
		KubectlOptions: kubectlOptions,
		SetValues: map[string]string{
			"persistence.enabled": "false",
			"replicaCount":        "2",
			"image.repository":    "marklogic-centos/marklogic-server-centos",
			"image.tag":           "10-internal",
		},
	}

	t.Logf("====Upgrading Helm Chart")
	helm.Upgrade(t, newOptions, helmChartPath, releaseName)

	tlsConfig := tls.Config{}
	podName := releaseName + "-marklogic-1"

	// wait until the pod is in Ready status
	k8s.WaitUntilPodAvailable(t, kubectlOptions, podName, 10, 20*time.Second)
	tunnel := k8s.NewTunnel(
		kubectlOptions, k8s.ResourceTypePod, podName, 7997, 7997)
	defer tunnel.Close()
	tunnel.ForwardPort(t)
	endpoint := fmt.Sprintf("http://%s", tunnel.Endpoint())
	t.Logf(`Endpoint: %s`, endpoint)

	http_helper.HttpGetWithRetryWithCustomValidation(
		t,
		endpoint,
		&tlsConfig,
		10,
		15*time.Second,
		func(statusCode int, body string) bool {
			return statusCode == 200
		},
	)
}

func TestClusterJoin(t *testing.T) {
	var username string = "admin"
	var password string = "admin"
	var resp *http.Response
	var body []byte
	var err error

	options := &helm.Options{
		KubectlOptions: kubectlOptions,
		SetValues: map[string]string{
			"persistence.enabled": "false",
			"replicaCount":        "2",
			"image.repository":    "marklogic-centos/marklogic-server-centos",
			"image.tag":           "10-internal",
			"auth.adminUsername":  username,
			"auth.adminPassword":  password,
		},
	}
	t.Logf("====Installing Helm Chart")
	releaseName := releaseName + "-join"
	helm.Install(t, options, helmChartPath, releaseName)

	podName := releaseName + "-marklogic-1"

	// wait until the pod is in Ready status
	k8s.WaitUntilPodAvailable(t, kubectlOptions, podName, 10, 20*time.Second)
	tunnel := k8s.NewTunnel(
		kubectlOptions, k8s.ResourceTypePod, podName, 8002, 8002)
	defer tunnel.Close()
	tunnel.ForwardPort(t)
	endpoint := fmt.Sprintf("http://%s/manage/v2/hosts", tunnel.Endpoint())
	t.Logf(`Endpoint: %s`, endpoint)

	dr := digest_auth.NewRequest(username, password, "GET", endpoint, "")
	
	if resp, err = dr.Execute(); err != nil {
		log.Fatalln(err)
	}
	defer resp.Body.Close()

	if body, err = ioutil.ReadAll(resp.Body); err != nil {
		log.Fatalln(err)
	}

	t.Logf("Response:\n" + string(body))
	if !strings.Contains(string(body), "<list-count units=\"quantity\">2</list-count>") {
		t.Errorf("Wrong number of hosts")
	}
}
