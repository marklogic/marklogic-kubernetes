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
	"github.com/stretchr/testify/assert"
	digestAuth "github.com/xinsnake/go-http-digest-auth-client"
)

func TestEnableConvertersAndLicense(t *testing.T) {

	// Path to the helm chart we will test
	helmChartPath, e := filepath.Abs("../../charts")
	if e != nil {
		t.Fatalf(e.Error())
	}
	imageRepo, repoPres := os.LookupEnv("dockerRepository")
	imageTag, tagPres := os.LookupEnv("dockerVersion")
	username := "admin"
	password := "AdminPa$s_with@!#%^&*()"
	var resp *http.Response
	var body []byte
	var err error

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
			"persistence.enabled":   "true",
			"replicaCount":          "1",
			"image.repository":      imageRepo,
			"image.tag":             imageTag,
			"auth.adminUsername":    username,
			"auth.adminPassword":    password,
			"logCollection.enabled": "false",
			"enableConverters":      "true",
			"license.key":           "3981-CE27-75BB-9D3C-B81C-E067-1B39-DDFE-0875-C37E-D3F0-A76C-34E5-2F86-76BB-ADDD-E677-CB3F-D5FE-4773-C3CD-5EE8-87BC-36E5-3F71-0C15",
			"license.licensee":      "MarkLogic - Version 9 QA Test License",
		},
	}

	t.Logf("====Creating namespace: " + namespaceName)
	k8s.CreateNamespace(t, kubectlOptions, namespaceName)

	defer t.Logf("====Deleting namespace: " + namespaceName)
	defer k8s.DeleteNamespace(t, kubectlOptions, namespaceName)

	t.Logf("====Installing Helm Chart")
	releaseName := "test"
	helm.Install(t, options, helmChartPath, releaseName)

	podName := releaseName + "-marklogic-0"
	// wait until the pod is in Ready status
	k8s.WaitUntilPodAvailable(t, kubectlOptions, podName, 15, 15*time.Second)
	tunnel := k8s.NewTunnel(
		kubectlOptions, k8s.ResourceTypePod, podName, 8001, 8001)
	defer tunnel.Close()
	tunnel.ForwardPort(t)
	endpoint := fmt.Sprintf("http://%s/admin/v1/timestamp", tunnel.Endpoint())
	t.Logf(`Endpoint: %s`, endpoint)

	// Make request to server as soon as it is ready
	timestamp := digestAuth.NewRequest(username, password, "GET", endpoint, "")

	if resp, err = timestamp.Execute(); err != nil {
		t.Fatalf(err.Error())
	}
	if body, err = ioutil.ReadAll(resp.Body); err != nil {
		t.Fatalf(err.Error())
	}

	t.Logf("Timestamp response:\n" + string(body))

	// Get logs from a running container
	podConfig := k8s.GetPod(t, kubectlOptions, podName)
	logs, err := k8s.GetPodLogsE(t, kubectlOptions, podConfig, "")
	if err != nil {
		t.Errorf("Failed to get logs for pod %s in namespace %s: %v", podName, namespaceName, err)
	}

	// Verify that the license is getting installed
	assert.Contains(t, logs, "LICENSE_KEY and LICENSEE are defined")
	// Verify that converters are getting installed
	assert.Contains(t, logs, "converters.rpm to be installed")
}
