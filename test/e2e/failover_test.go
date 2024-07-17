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
	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/gruntwork-io/terratest/modules/random"
	"github.com/imroc/req/v3"
	"github.com/marklogic/marklogic-kubernetes/test/testUtil"
	"github.com/tidwall/gjson"
	// "github.com/tidwall/gjson"
)

type Forest struct {
	ForestName string `json:"forest-name"`
	Host       string `json:"host"`
}

type ForestProperties struct {
	ForestReplica []ForestReplica `json:"forest-replica"`
}

type ForestReplica struct {
	ReplicaName string `json:"replica-name"`
	Host        string `json:"host"`
}

func TestFailover(t *testing.T) {
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
		imageRepo = "progressofficial/marklogic-db"
		t.Logf("No imageRepo variable present, setting to default value: " + imageRepo)
	}

	if !tagPres {
		imageTag = "latest-11"
		t.Logf("No imageTag variable present, setting to default value: " + imageTag)
	}

	namespaceName := "ml-" + strings.ToLower(random.UniqueId())
	kubectlOptions := k8s.NewKubectlOptions("", "", namespaceName)
	options := &helm.Options{
		KubectlOptions: kubectlOptions,
		SetValues: map[string]string{
			"persistence.enabled": "true",
			"replicaCount":        "3",
			"image.repository":    imageRepo,
			"image.tag":           imageTag,
			"auth.adminUsername":  username,
			"auth.adminPassword":  password,
		},
	}

	t.Logf("====Creating namespace: " + namespaceName)
	k8s.CreateNamespace(t, kubectlOptions, namespaceName)

	defer k8s.DeleteNamespace(t, kubectlOptions, namespaceName)

	releaseName := "failover"
	hostName1 := fmt.Sprintf("%s-1.%s.%s.svc.cluster.local", releaseName, releaseName, namespaceName)
	forestName := "security1"

	t.Logf("====Setting helm chart path to %s", helmChartPath)
	t.Logf("====Installing Helm Chart")
	testUtil.HelmInstall(t, options, releaseName, kubectlOptions, helmChartPath)
	podZeroName := releaseName + "-0"
	podOneName := releaseName + "-1"

	// wait until the pod is in Ready status
	k8s.WaitUntilPodAvailable(t, kubectlOptions, podZeroName, 15, 20*time.Second)

	tunnel := k8s.NewTunnel(
		kubectlOptions, k8s.ResourceTypePod, podZeroName, 8002, 8002)
	tunnel.ForwardPort(t)

	client := req.C().DevMode()

	// Create a new forest
	resp, err := client.R().
		SetDigestAuth(username, password).
		SetBody(&Forest{ForestName: forestName, Host: hostName1}).
		SetRetryCount(5).
		SetRetryFixedInterval(10 * time.Second).
		AddRetryCondition(func(resp *req.Response, err error) bool {
			if err != nil {
				t.Logf("error in AddRetryCondition: %s", err.Error())
				return true
			}
			if resp == nil {
				t.Logf("error getting response")
				return true
			}
			return resp.StatusCode != 201
		}).
		Post("http://localhost:8002/manage/v2/forests")

	if err != nil {
		t.Errorf("Error creating forest %s", forestName)
		t.Fatalf(err.Error())
	}

	if resp.StatusCode != 201 {
		t.Error("Response code is not 201 when creating forest. Actual response code", resp.Status)
	}

	t.Logf("Forest %s created successfully", forestName)

	// Set replica forest security1 for Security
	resp, err = client.R().
		SetDigestAuth(username, password).
		SetBody(&ForestProperties{ForestReplica: []ForestReplica{{ReplicaName: forestName, Host: hostName1}}}).
		SetRetryCount(5).
		SetRetryFixedInterval(10 * time.Second).
		AddRetryCondition(func(resp *req.Response, err error) bool {
			if err != nil {
				t.Logf("error in AddRetryCondition: %s", err.Error())
				return true
			}
			if resp == nil {
				t.Logf("error getting response")
				return true
			}
			return resp.StatusCode != 204
		}).
		Put("http://localhost:8002/manage/v2/forests/Security/properties")

	if err != nil {
		t.Error("Error setting replica forest for Security")
		t.Fatalf(err.Error())
	}

	if resp.StatusCode != 204 {
		t.Error("Response code is not 204 when updating forest replica. Actual response code", resp.Status)
	}

	t.Log("Replica forest set for Security")

	// Make sure the security1 forestg is in sync replicating state
	_, err = client.R().
		SetDigestAuth(username, password).
		SetRetryCount(5).
		SetRetryFixedInterval(10 * time.Second).
		AddRetryCondition(func(resp *req.Response, err error) bool {
			if err != nil {
				t.Logf("error in AddRetryCondition: %s", err.Error())
				return true
			}
			if resp == nil || resp.Body == nil {
				t.Logf("error in getting response body")
				return true
			}
			body, err := io.ReadAll(resp.Body)
			if body == nil || err != nil {
				t.Logf("error in read response body")
				return true
			}
			forestStatus := gjson.Get(string(body), `forest-status.status-properties.state.value`)
			t.Logf("Forest status waiting to be sync replicating, current status: %s", forestStatus.String())
			return forestStatus.String() != "sync replicating"
		}).
		Get("http://localhost:8002/manage/v2/forests/" + forestName + "?view=status&format=json")

	if err != nil {
		t.Errorf("Error getting forest status for %s and waiting for sync replicating", forestName)
		t.Fatalf(err.Error())
	}

	// delete the pod 0 to trigger Security forest failover to security1
	k8s.RunKubectl(t, kubectlOptions, "delete", "pod", podZeroName)

	k8s.WaitUntilPodAvailable(t, kubectlOptions, podZeroName, 15, 20*time.Second)
	tunnel.Close()
	tunnel = k8s.NewTunnel(
		kubectlOptions, k8s.ResourceTypePod, podZeroName, 8002, 8002)
	defer tunnel.Close()
	tunnel.ForwardPort(t)

	// Make sure the security1 forest1 is primary forest now and status is open
	_, err = client.R().
		SetDigestAuth(username, password).
		SetRetryCount(5).
		SetRetryFixedInterval(10 * time.Second).
		AddRetryCondition(func(resp *req.Response, err error) bool {
			if err != nil {
				t.Logf("error in AddRetryCondition: %s", err.Error())
				return true
			}
			if resp == nil || resp.Body == nil {
				t.Logf("error in getting response body")
				return true
			}
			body, err := io.ReadAll(resp.Body)
			if body == nil || err != nil {
				t.Logf("error in read response body: %s", err.Error())
				return true
			}
			forestStatus := gjson.Get(string(body), `forest-status.status-properties.state.value`)
			t.Logf("Forest status waiting to be open, current status: %s", forestStatus.String())
			return forestStatus.String() != "open"
		}).
		Get("http://localhost:8002/manage/v2/forests/" + forestName + "?view=status&format=json")

	if err != nil {
		t.Error("Error getting forest status for security1 and waiting for open")
		t.Fatalf(err.Error())
	}

	tlsConfig := tls.Config{}
	// restart all pods in the cluster and verify its ready and MarkLogic server is healthy
	testUtil.RestartPodAndVerify(t, true, []string{podZeroName, podOneName}, namespaceName, kubectlOptions, &tlsConfig)
}
