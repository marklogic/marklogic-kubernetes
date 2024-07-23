package e2e

import (
	"crypto/tls"
	"fmt"
	"io"
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
)

func TestHelmInstall(t *testing.T) {
	var err error
	var podZeroName string
	var helmChartPath string
	var initialChartVersion string
	releaseName := "test-install"
	secretName := releaseName + "-admin"
	imageRepo, repoPres := os.LookupEnv("dockerRepository")
	imageTag, tagPres := os.LookupEnv("dockerVersion")
	upgradeHelm, _ := os.LookupEnv("upgradeTest")
	runUpgradeTest, _ := strconv.ParseBool(upgradeHelm)
	if runUpgradeTest {
		initialChartVersion, _ = os.LookupEnv("initialChartVersion")
		t.Logf("====Setting initial Helm chart version: %s", initialChartVersion)
	}

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
	valuesMap := map[string]string{"persistence.enabled": "true",
		"replicaCount":          "2",
		"image.repository":      imageRepo,
		"image.tag":             imageTag,
		"logCollection.enabled": "false",
	}
	options := &helm.Options{
		KubectlOptions: kubectlOptions,
		SetValues:      valuesMap,
		Version:        initialChartVersion,
	}

	t.Logf("====Creating namespace: " + namespaceName)
	k8s.CreateNamespace(t, kubectlOptions, namespaceName)

	defer t.Logf("====Deleting namespace: " + namespaceName)
	defer k8s.DeleteNamespace(t, kubectlOptions, namespaceName)
	helmChartPath, err = filepath.Abs("../../charts")
	if err != nil {
		t.Fatalf(err.Error())
	}

	//add the helm chart repo and install the last helm chart release from repository
	//to test and upgrade this chart to the latest one to be released
	if runUpgradeTest {
		delete(valuesMap, "image.repository")
		delete(valuesMap, "image.tag")
		helm.AddRepo(t, options, "marklogic", "https://marklogic.github.io/marklogic-kubernetes/")
		defer helm.RemoveRepo(t, options, "marklogic")
		helmChartPath = "marklogic/marklogic"
	}

	t.Logf("====Setting helm chart path to %s", helmChartPath)
	t.Logf("====Installing Helm Chart")
	podZeroName = testUtil.HelmInstall(t, options, releaseName, kubectlOptions, helmChartPath)
	tlsConfig := tls.Config{}

	// wait until the pod is in Ready status
	k8s.WaitUntilPodAvailable(t, kubectlOptions, podZeroName, 15, 15*time.Second)

	// verify MarkLogic is ready
	_, err = testUtil.MLReadyCheck(t, kubectlOptions, podZeroName, &tlsConfig)
	if err != nil {
		t.Fatal("MarkLogic failed to start")
	}

	podOneName := releaseName + "-1"

	if runUpgradeTest {
		upgradeOptionsMap := map[string]string{
			"persistence.enabled":   "true",
			"replicaCount":          "2",
			"logCollection.enabled": "false",
			"allowLongHostnames":    "true",
			"rootToRootlessUpgrade": "true",
		}
		if strings.HasPrefix(initialChartVersion, "1.0") {
			podZeroName = releaseName + "-marklogic-0"
			podOneName = releaseName + "-marklogic-1"
			secretName = releaseName + "-marklogic-admin"
			upgradeOptionsMap["useLegacyHostnames"] = "true"
		}
		//set helm options for upgrading helm chart version
		helmUpgradeOptions := &helm.Options{
			KubectlOptions: kubectlOptions,
			SetValues:      upgradeOptionsMap,
		}
		t.Logf("UpgradeHelmTest is enabled. Running helm upgrade test")
		testUtil.HelmUpgrade(t, helmUpgradeOptions, releaseName, kubectlOptions, []string{podZeroName, podOneName}, initialChartVersion)
	}

	t.Log("====Testing Generated Random Password====")
	secret := k8s.GetSecret(t, kubectlOptions, secretName)
	passwordArr := secret.Data["password"]
	password := string(passwordArr[:])
	// the generated random password should have length of 10
	assert.Equal(t, 10, len(password))
	usernameArr := secret.Data["username"]
	username := string(usernameArr[:])
	// the random generated username should have length of 11"
	assert.Equal(t, 11, len(username))

	tunnel8002 := k8s.NewTunnel(kubectlOptions, k8s.ResourceTypePod, podZeroName, 8002, 8002)
	defer tunnel8002.Close()
	tunnel8002.ForwardPort(t)

	client := req.C().
		SetCommonDigestAuth(username, password).
		SetCommonRetryCount(5).
		SetCommonRetryFixedInterval(10 * time.Second)

	t.Log("====Verify xdqp-ssl-enabled is set to true by default")
	endpoint := fmt.Sprintf("http://%s/manage/v2/groups/Default/properties?format=json", tunnel8002.Endpoint())
	t.Logf(`Endpoint for group properties: %s`, endpoint)

	xdqpEnabledValue := false
	_, err = client.R().
		AddRetryCondition(func(resp *req.Response, err error) bool {
			if err != nil {
				t.Logf("error in getting group property: %s", err.Error())
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
			xdqpSSLEnabled := gjson.Get(string(body), `xdqp-ssl-enabled`)
			t.Logf("xdqpSSLEnabled: %s", xdqpSSLEnabled)
			if xdqpSSLEnabled.Bool() != true {
				t.Logf("xdqpSSLEnabled is not set to true yet. retrying...")
				return true
			} else {
				xdqpEnabledValue = true
				return false
			}
		}).
		Get(endpoint)
	if err != nil {
		t.Fatalf("Error in getting xdqpSSLEnabled: %s", err.Error())
	}

	// verify xdqp-ssl-enabled is set to trues
	assert.Equal(t, true, xdqpEnabledValue, "xdqp-ssl-enabled should be set to true")

	// restart all pods in the cluster and verify its ready and MarkLogic server is healthy
	testUtil.RestartPodAndVerify(t, true, []string{podZeroName, podOneName}, namespaceName, kubectlOptions, &tlsConfig)
}
