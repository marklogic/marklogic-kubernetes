package e2e

import (
	"crypto/tls"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gruntwork-io/terratest/modules/helm"
	http_helper "github.com/gruntwork-io/terratest/modules/http-helper"
	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/gruntwork-io/terratest/modules/random"
	"github.com/imroc/req/v3"
	"github.com/tidwall/gjson"
)

func TestPathBasedRouting(t *testing.T) {
	// Path to the helm chart we will test
	helmChartPath, e := filepath.Abs("../../charts")
	if e != nil {
		t.Fatalf(e.Error())
	}
	username := "admin"
	password := "admin"

	imageRepo, repoPres := os.LookupEnv("dockerRepository")
	imageTag, tagPres := os.LookupEnv("dockerVersion")

	if !repoPres {
		imageRepo = "marklogicdb/marklogic-db"
		t.Logf("No imageRepo variable present, setting to default value: " + imageRepo)
	}

	if !tagPres {
		imageTag = "latest"
		t.Logf("No imageTag variable present, setting to default value: " + imageTag)
	}

	namespaceName := "ml-" + strings.ToLower(random.UniqueId())
	kubectlOptions := k8s.NewKubectlOptions("", "", namespaceName)
	options := &helm.Options{
		KubectlOptions: kubectlOptions,
		SetValues: map[string]string{
			"persistence.enabled":   "false",
			"replicaCount":          "2",
			"image.repository":      "marklogicdb/marklogic-db",
			"image.tag":             "11.1.0-centos-1.1.2",
			"auth.adminUsername":    imageRepo,
			"auth.adminPassword":    imageTag,
			"logCollection.enabled": "false",
			"pathbased.enabled":     "true",
		},
	}

	t.Logf("====Creating namespace: " + namespaceName)
	k8s.CreateNamespace(t, kubectlOptions, namespaceName)

	defer t.Logf("====Deleting namespace: " + namespaceName)
	defer k8s.DeleteNamespace(t, kubectlOptions, namespaceName)

	t.Logf("====Installing Helm Chart")
	releaseName := "test-path"
	helm.Install(t, options, helmChartPath, releaseName)

	podZeroName := releaseName + "-0"
	podOneName := releaseName + "-1"

	// wait until the pod is in Ready status
	k8s.WaitUntilPodAvailable(t, kubectlOptions, podOneName, 15, 20*time.Second)
	tunnel := k8s.NewTunnel(
		kubectlOptions, k8s.ResourceTypePod, podZeroName, 8002, 8002)
	defer tunnel.Close()
	tunnel.ForwardPort(t)

	numOfHosts := 0
	client := req.C()

	//verify manage AppServer is acessible when pathbased routing is enabled
	_, err := client.R().
		SetBasicAuth(username, password).
		SetRetryCount(5).
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
			totalHosts := gjson.Get(string(body), `host-default-list.list-items.list-count.value`)
			numOfHosts = int(totalHosts.Num)
			if numOfHosts != 2 {
				t.Log("Number of hosts: " + string(totalHosts.Raw))
				t.Log("Waiting for MarkLogic count of MarkLogic hosts to be 2")
			}
			return numOfHosts != 2
		}).
		Get("http://localhost:8002/manage/v2/hosts?format=json")

	if err != nil {
		t.Error("Error getting hosts")
		t.Fatalf(err.Error())
	}

	if numOfHosts != 2 {
		t.Errorf("Wrong number of hosts")
	}

	resp, err := client.R().
		SetBasicAuth(username, password).
		SetRetryCount(5).
		SetRetryFixedInterval(10 * time.Second).
		Get("http://localhost:8002/manage/v2/servers/Manage/properties?group-id=Default&format=json")

	if err != nil {
		t.Errorf("Error getting AppServer properties")
		t.Fatalf(err.Error())
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf(err.Error())
	}
	serverAuthentication := gjson.Get(string(body), `authentication`)

	//verify basic authentication is configured for AppServer
	if serverAuthentication.Str != "basic" {
		t.Errorf("basic authentication not configured for AppServer")
	}
}

func TestPathBasedRoutingWithTLS(t *testing.T) {
	// Path to the helm chart we will test
	helmChartPath, e := filepath.Abs("../../charts")
	if e != nil {
		t.Fatalf(e.Error())
	}
	username := "admin"
	password := "admin"

	imageRepo, repoPres := os.LookupEnv("dockerRepository")
	imageTag, tagPres := os.LookupEnv("dockerVersion")

	if !repoPres {
		imageRepo = "marklogicdb/marklogic-db"
		t.Logf("No imageRepo variable present, setting to default value: " + imageRepo)
	}

	if !tagPres {
		imageTag = "latest"
		t.Logf("No imageTag variable present, setting to default value: " + imageTag)
	}

	namespaceName := "marklogic-" + strings.ToLower(random.UniqueId())
	kubectlOptions := k8s.NewKubectlOptions("", "", namespaceName)
	options := &helm.Options{
		KubectlOptions: kubectlOptions,
		SetValues: map[string]string{
			"persistence.enabled":           "false",
			"replicaCount":                  "1",
			"image.repository":              imageRepo,
			"image.tag":                     imageTag,
			"auth.adminUsername":            username,
			"auth.adminPassword":            password,
			"logCollection.enabled":         "false",
			"tls.enableOnDefaultAppServers": "true",
			"pathbased.enabled":             "true",
		},
	}

	t.Logf("====Creating namespace: " + namespaceName)
	k8s.CreateNamespace(t, kubectlOptions, namespaceName)

	defer t.Logf("====Deleting namespace: " + namespaceName)
	defer k8s.DeleteNamespace(t, kubectlOptions, namespaceName)

	t.Logf("====Installing Helm Chart")
	releaseName := "test-pb-tls"
	helm.Install(t, options, helmChartPath, releaseName)

	podName := releaseName + "-0"
	tlsConfig := tls.Config{InsecureSkipVerify: true}

	// wait until the pod is in Ready status
	k8s.WaitUntilPodAvailable(t, kubectlOptions, podName, 10, 20*time.Second)
	tunnel7997 := k8s.NewTunnel(kubectlOptions, k8s.ResourceTypePod, podName, 7997, 7997)
	defer tunnel7997.Close()
	tunnel7997.ForwardPort(t)
	endpoint7997 := fmt.Sprintf("http://%s", tunnel7997.Endpoint())

	// verify if 7997 health check endpoint returns 200
	http_helper.HttpGetWithRetryWithCustomValidation(
		t,
		endpoint7997,
		&tlsConfig,
		10,
		15*time.Second,
		func(statusCode int, _ string) bool {
			return statusCode == 200
		},
	)

	tunnel := k8s.NewTunnel(
		kubectlOptions, k8s.ResourceTypePod, podName, 8002, 8002)
	defer tunnel.Close()
	tunnel.ForwardPort(t)
	endpointManage := fmt.Sprintf("https://%s/manage/v2", tunnel.Endpoint())
	t.Logf(`Endpoint: %s`, endpointManage)

	client := req.C().EnableInsecureSkipVerify()

	// verify https is working when pathbased is enabled
	resp, err := client.R().
		SetBasicAuth(username, password).
		Get("https://localhost:8002/manage/v2")

	if err != nil {
		t.Fatalf(err.Error())
	}

	if resp.GetStatusCode() != 200 {
		t.Errorf("error returned")
	}
}
