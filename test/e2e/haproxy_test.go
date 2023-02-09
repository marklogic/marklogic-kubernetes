package e2e

import (
	"fmt"
	"testing"
	"time"

	"github.com/gruntwork-io/terratest/modules/k8s"
	digestAuth "github.com/xinsnake/go-http-digest-auth-client"
)

func TestHAProxy(t *testing.T) {
	// Path to the helm chart we will test
	// helmChartPath, e := filepath.Abs("../../charts")
	// if e != nil {
	// 	t.Fatalf(e.Error())
	// }
	// imageRepo, repoPres := os.LookupEnv("dockerRepository")
	// imageTag, tagPres := os.LookupEnv("dockerVersion")

	// if !repoPres {
	// 	imageRepo = "marklogic-centos/marklogic-server-centos"
	// 	t.Logf("No imageRepo variable present, setting to default value: " + imageRepo)
	// }

	// if !tagPres {
	// 	imageTag = "10-internal"
	// 	t.Logf("No imageTag variable present, setting to default value: " + imageTag)
	// }

	username := "admin"
	password := "admin"

	// namespaceName := "marklogic-" + strings.ToLower(random.UniqueId())
	namespaceName := "default"
	kubectlOptions := k8s.NewKubectlOptions("", "", namespaceName)
	// options := &helm.Options{
	// 	KubectlOptions: kubectlOptions,
	// 	SetValues: map[string]string{
	// 		"persistence.enabled": "false",
	// 		"replicaCount":        "1",
	// 		"image.repository":    imageRepo,
	// 		"image.tag":           imageTag,
	// 		"auth.adminUsername":  username,
	// 		"auth.adminPassword":  username,
	// 		"haproxy.enabled":     "true",
	// 	},
	// }

	// k8s.CreateNamespace(t, kubectlOptions, namespaceName)

	// defer k8s.DeleteNamespace(t, kubectlOptions, namespaceName)

	releaseName := "test"
	// helm.Install(t, options, helmChartPath, releaseName)
	// tlsConfig := tls.Config{}
	podName := releaseName + "-marklogic-0"
	// wait until the pod is in Ready status
	k8s.WaitUntilPodAvailable(t, kubectlOptions, podName, 10, 15*time.Second)

	// defer helm.Delete(t, options, releaseName, true)

	// service for haproxy pod
	serviceName := releaseName + "-haproxy"
	_, err := k8s.GetServiceE(t, kubectlOptions, serviceName)
	if err != nil {
		t.Log("could not find HAProxy service")
		t.Fatalf(err.Error())
		return
	}
	k8s.WaitUntilServiceAvailable(t, kubectlOptions, serviceName, 10, 15*time.Second)
	tunnel8002 := k8s.NewTunnel(
		kubectlOptions, k8s.ResourceTypeService, serviceName, 8002, 8002)
	defer tunnel8002.Close()
	tunnel8002.ForwardPort(t)
	endpoint8002 := fmt.Sprintf("http://%s/manage/v2/hosts", tunnel8002.Endpoint())
	url8002 := fmt.Sprintf("http://%s/manage/v2/hosts", endpoint8002)
	t.Log(url8002)
	request := digestAuth.NewRequest(username, password, "GET", url8002, "")

	response, err := request.Execute()
	if err != nil {
		t.Fatalf(err.Error())
		return
	}
	defer response.Body.Close()

	t.Log(response)

	// verify if 7997 health check endpoint returns 200
	// http_helper.HttpGetWithRetryWithCustomValidation(
	// 	t,
	// 	endpoint7997,
	// 	&tlsConfig,
	// 	10,
	// 	15*time.Second,
	// 	func(statusCode int, body string) bool {
	// 		return statusCode == 200
	// 	},
	// )

}
