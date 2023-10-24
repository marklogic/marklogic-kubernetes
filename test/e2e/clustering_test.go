package e2e

import (
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
	"github.com/tidwall/gjson"
)

func TestClusterJoin(t *testing.T) {
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

	t.Logf("====Installing Helm Chart")
	releaseName := "test-join"
	helm.Install(t, options, helmChartPath, releaseName)

	podName := releaseName + "-1"

	// wait until the pod is in Ready status
	k8s.WaitUntilPodAvailable(t, kubectlOptions, podName, 10, 20*time.Second)
	tunnel := k8s.NewTunnel(
		kubectlOptions, k8s.ResourceTypePod, podName, 8002, 8002)
	defer tunnel.Close()
	tunnel.ForwardPort(t)

	numOfHosts := 0
	client := req.C()
	resp, err := client.R().
		SetDigestAuth(username, password).
		SetRetryCount(5).
		SetRetryFixedInterval(10 * time.Second).
		AddRetryCondition(func(resp *req.Response, err error) bool {
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Logf("error: %s", err.Error())
			}
			totalHosts := gjson.Get(string(body), `host-default-list.list-items.list-count.value`)
			numOfHosts = int(totalHosts.Num)
			if numOfHosts != 2 {
				t.Log("Waiting for MarkLogic hosts")
			}
			return numOfHosts != 2
		}).
		Get("http://localhost:8002/manage/v2/hosts?format=json")
	defer resp.Body.Close()

	if err != nil {
		t.Fatalf(err.Error())
	}

	if numOfHosts != 2 {
		t.Errorf("Wrong number of hosts")
	}
}
