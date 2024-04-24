package hugePages

import (
	"crypto/tls"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/gruntwork-io/terratest/modules/random"
	"github.com/marklogic/marklogic-kubernetes/test/testUtil"
	"github.com/stretchr/testify/assert"
	digestAuth "github.com/xinsnake/go-http-digest-auth-client"
)

func TestHugePagesSettings(t *testing.T) {
	// var resp *http.Response
	var body []byte
	var err error
	var podName string
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

	username := "admin"
	password := "admin"

	options := map[string]string{
		"persistence.enabled":            "false",
		"replicaCount":                   "1",
		"image.repository":               imageRepo,
		"image.tag":                      imageTag,
		"auth.adminUsername":             username,
		"auth.adminPassword":             password,
		"logCollection.enabled":          "false",
		"hugepages.enabled":              "true",
		"hugepages.mountPath":            "/dev/hugepages",
		"resources.limits.hugepages-2Mi": "1Gi",
		"resources.limits.memory":        "8Gi",
		"resources.requests.memory":      "8Gi",
	}
	t.Logf("====Installing Helm Chart")
	releaseName := "hugepages"

	namespaceName := "ml-" + strings.ToLower(random.UniqueId())
	kubectlOptions := k8s.NewKubectlOptions("", "", namespaceName)

	t.Logf("====Creating namespace: " + namespaceName)
	k8s.CreateNamespace(t, kubectlOptions, namespaceName)

	defer t.Logf("====Deleting namespace: " + namespaceName)
	defer k8s.DeleteNamespace(t, kubectlOptions, namespaceName)

	podName = testUtil.HelmInstall(t, options, releaseName, kubectlOptions)

	t.Logf("====Describe pod for verifying huge pages")
	k8s.RunKubectl(t, kubectlOptions, "describe", "pod", podName)

	tlsConfig := tls.Config{}
	// wait until the pod is in Ready status
	k8s.WaitUntilPodAvailable(t, kubectlOptions, podName, 10, 15*time.Second)

	// verify MarkLogic is ready
	_, err = testUtil.MLReadyCheck(t, kubectlOptions, podName, tlsConfig)
	if err != nil {
		t.Fatal("MarkLogic failed to start")
	}

	tunnel8002 := k8s.NewTunnel(kubectlOptions, k8s.ResourceTypePod, podName, 8002, 8002)
	defer tunnel8002.Close()
	tunnel8002.ForwardPort(t)
	endpointManage := fmt.Sprintf("http://%s/manage/v2/logs?format=text&filename=ErrorLog.txt", tunnel8002.Endpoint())
	request := digestAuth.NewRequest(username, password, "GET", endpointManage, "")
	response, err := request.Execute()
	if err != nil {
		t.Fatalf(err.Error())
	}
	defer response.Body.Close()
	assert.Equal(t, 200, response.StatusCode)

	body, err = io.ReadAll(response.Body)
	t.Log(string(body))

	// Verify if Huge pages are configured on the MarkLogic node
	if !strings.Contains(string(body), "Linux Huge Pages: detected 1280") {
		t.Errorf("Huge Pages not configured for the node")
	}
}
