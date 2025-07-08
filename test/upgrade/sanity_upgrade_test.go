package e2e

/********************************************************************************
*
* Test: MlImageUpgrade
* This test verifies the upgrade of MarkLogic Docker images in a Kubernetes environment.
* 
********************************************************************************/
import (
    "io"
    "github.com/tidwall/gjson"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/gruntwork-io/terratest/modules/random"
	"github.com/marklogic/marklogic-kubernetes/test/testUtil"
    "github.com/imroc/req/v3"

)

type DockerImage struct {
    Repo string
    Tag  string
    Version string
    Type  string
    Description string
}

func TestMlImageUpgrade(t *testing.T) {
	var helmChartPath string
	var initialChartVersion string

    adminUsername := "admin"
    adminPassword := "admin"

    // Path to the helm chart we will test
	helmChartPath, err := filepath.Abs("../../charts")
	if err != nil {
		t.Fatalf(err.Error())
	}

    upgradeImageList := [][]DockerImage{
        {
            DockerImage{Repo: "progressofficial/marklogic-db", Tag: "11.3.1-ubi-rootless-2.1.3", Version: "11030000", Type: "rootless", Description: "MarkLogic 11.3.1 UBI8 Rootless"},
            DockerImage{Repo: "ml-docker-db-dev-tierpoint.bed-artifactory.bedford.progress.com/marklogic/marklogic-server-ubi-rootless", Tag: "latest-12-ubi-rootless", Version: "12000000", Type: "rootless", Description: "MarkLogic 12 UBI8 Rootless"},
        },
        {
            DockerImage{Repo: "progressofficial/marklogic-db", Tag: "11.3.1-ubi9-rootless-2.1.3", Version: "11030000", Type: "rootless", Description: "MarkLogic 11.3.1 UBI9 Rootless"},
            DockerImage{Repo: "ml-docker-db-dev-tierpoint.bed-artifactory.bedford.progress.com/marklogic/marklogic-server-ubi9-rootless", Tag: "latest-12-ubi9-rootless", Version: "12000000", Type: "rootless", Description: "MarkLogic 12 UBI9 Rootless"},
        },
        {
            DockerImage{Repo: "progressofficial/marklogic-db", Tag: "11.3.1-ubi-rootless-2.1.3", Version: "11030000", Type: "rootless", Description: "MarkLogic 11.3.1 UBI8 Rootless"},
            DockerImage{Repo: "ml-docker-db-dev-tierpoint.bed-artifactory.bedford.progress.com/marklogic/marklogic-server-ubi9-rootless", Tag: "latest-12-ubi9-rootless", Version: "12000000", Type: "rootless", Description: "MarkLogic 12 UBI9 Rootless"},
        },
        {
            DockerImage{Repo: "progressofficial/marklogic-db", Tag: "11.3.1-ubi-2.1.3", Version: "11030000", Type: "root", Description: "MarkLogic 11.3.1 UBI8 Root"},
            DockerImage{Repo: "ml-docker-db-dev-tierpoint.bed-artifactory.bedford.progress.com/marklogic/marklogic-server-ubi", Tag: "latest-12", Version: "12000000", Type: "root", Description: "MarkLogic 12 UBI8 Root"},
        },
        {
            DockerImage{Repo: "progressofficial/marklogic-db", Tag: "11.3.1-ubi9-2.1.3", Version: "11030000", Type: "root", Description: "MarkLogic 11.3.1 UBI9 Root"},
            DockerImage{Repo: "ml-docker-db-dev-tierpoint.bed-artifactory.bedford.progress.com/marklogic/marklogic-server-ubi9", Tag: "latest-12", Version: "12000000", Type: "root", Description: "MarkLogic 12 UBI9 Root"},
        },
        {
            DockerImage{Repo: "progressofficial/marklogic-db", Tag: "11.3.1-ubi-2.1.3", Version: "11030000", Type: "root", Description: "MarkLogic 11.3.1 UBI8 Root"},
            DockerImage{Repo: "ml-docker-db-dev-tierpoint.bed-artifactory.bedford.progress.com/marklogic/marklogic-server-ubi9", Tag: "latest-12", Version: "12000000", Type: "root", Description: "MarkLogic 12 UBI9 Root"},
        },
        {
            DockerImage{Repo: "progressofficial/marklogic-db", Tag: "11.3.1-ubi-2.1.3", Version: "11030000", Type: "root", Description: "MarkLogic 11.3.1 UBI8 Root"},
            DockerImage{Repo: "ml-docker-db-dev-tierpoint.bed-artifactory.bedford.progress.com/marklogic/marklogic-server-ubi-rootless", Tag: "latest-12-ubi-rootless", Version: "12000000", Type: "rootless", Description: "MarkLogic 12 UBI8 Rootless"},
        },
        {
            DockerImage{Repo: "progressofficial/marklogic-db", Tag: "11.3.1-ubi9-2.1.3", Version: "11030000", Type: "root", Description: "MarkLogic 11.3.1 UBI9 Root"},
            DockerImage{Repo: "ml-docker-db-dev-tierpoint.bed-artifactory.bedford.progress.com/marklogic/marklogic-server-ubi9-rootless", Tag: "latest-12-ubi9-rootless", Version: "12000000", Type: "rootless", Description: "MarkLogic 12 UBI9 Rootless"},
        },
        {
            DockerImage{Repo: "progressofficial/marklogic-db", Tag: "11.3.1-ubi-2.1.3", Version: "11030000", Type: "root", Description: "MarkLogic 11.3.1 UBI8 Root"},
            DockerImage{Repo: "ml-docker-db-dev-tierpoint.bed-artifactory.bedford.progress.com/marklogic/marklogic-server-ubi9-rootless", Tag: "latest-12-ubi9-rootless", Version: "12000000", Type: "rootless", Description: "MarkLogic 12 UBI9 Rootless"},
        },
    }

    for i, image := range upgradeImageList {
        originalImage := image[0]
        upgradeImage := image[1]
        t.Logf("Running upgrade test %d with images: %s -> %s", i+1, originalImage.Description, upgradeImage.Description)

        namespaceName := "ml-" + strings.ToLower(random.UniqueId())
        kubectlOptions := k8s.NewKubectlOptions("", "", namespaceName)
        valuesMap := map[string]string{
            "persistence.enabled": "true",
            "replicaCount":        "1",
            "image.repository":    originalImage.Repo,
            "image.tag":           originalImage.Tag,
            "auth.adminUsername":  adminUsername,
            "auth.adminPassword":  adminPassword,
            "auth.walletPassword": "admin",
        }
        if originalImage.Type == "root" {
            valuesMap["containerSecurityContext.allowPrivilegeEscalation"] = "true"
        }

        options := &helm.Options{
            KubectlOptions: kubectlOptions,
            SetValues:      valuesMap,
            Version:        initialChartVersion,
            ExtraArgs: map[string][]string{
            "install": {"--hide-notes"},
            },
        }

        t.Logf("====Creating namespace: " + namespaceName)
        k8s.CreateNamespace(t, kubectlOptions, namespaceName)


        t.Logf("====Setting helm chart path to %s", helmChartPath)
        t.Logf("====Installing Helm Chart")
        releaseName := "test-ml-upgrade"
        podName := testUtil.HelmInstall(t, options, releaseName, kubectlOptions, helmChartPath)

        tunnel := k8s.NewTunnel(
		kubectlOptions, k8s.ResourceTypePod, podName, 8002, 8002)
        tunnel.ForwardPort(t)
        client := req.C().EnableInsecureSkipVerify()

        currentMLVersion := getMLVersion(t, client, adminUsername, adminPassword)
        if currentMLVersion != originalImage.Version {
            t.Fatalf("Expected ML version to start with %s, but got %s",
                originalImage.Version, currentMLVersion)
        } else {
            t.Logf("ML version successfully installed: %s", currentMLVersion)
        }
        tunnel.Close()

        // create options for helm upgrade
        upgradeOptionsMap := map[string]string{
            "image.repository":    upgradeImage.Repo,
            "image.tag":           upgradeImage.Tag,
            "persistence.enabled":   "true",
            "replicaCount":          "1",
            "logCollection.enabled": "false",
        }

        if upgradeImage.Type == "root" {
            upgradeOptionsMap["containerSecurityContext.allowPrivilegeEscalation"] = "true"
        }

        if originalImage.Type == "root" && upgradeImage.Type == "rootless" {
            t.Logf("====Performing root to rootless upgrade")
            upgradeOptionsMap["rootToRootlessUpgrade"] = "true"
        }

        //set helm options for upgrading helm chart version
        helmUpgradeOptions := &helm.Options{
            KubectlOptions: kubectlOptions,
            SetValues:      upgradeOptionsMap,
            ExtraArgs: map[string][]string{
            "upgrade": {"--hide-notes"},
            },
        }
        
        helm.Upgrade(t, helmUpgradeOptions, helmChartPath, releaseName)
        t.Logf("====Waiting for pod to be available after upgrade")

        k8s.RunKubectl(t, kubectlOptions, "delete", "pod", podName)
        k8s.WaitUntilPodAvailable(t, kubectlOptions, podName, 15, 15*time.Second)

        t.Logf("====Checking if the pod is running after upgrade")
        pod := k8s.GetPod(t, kubectlOptions, podName)
        if pod.Status.Phase != "Running" {
            t.Fatalf("Pod %s is not running after upgrade, status: %s", podName, pod.Status.Phase)
        }

        tunnel8001 := k8s.NewTunnel(
		kubectlOptions, k8s.ResourceTypePod, podName, 8001, 8001)
        tunnel8001.ForwardPort(t)

        tunnel8002 := k8s.NewTunnel(
		kubectlOptions, k8s.ResourceTypePod, podName, 8002, 8002)
        tunnel8002.ForwardPort(t)   

        resp, err := client.R().
            SetDigestAuth(adminUsername, adminPassword).
            Post("http://localhost:8001/security-upgrade-go.xqy")

        if err != nil {
            t.Fatalf("Error in upgrading ML Security DB: %s", err.Error())
        }
        body, err := io.ReadAll(resp.Body)
        if err != nil {
            t.Logf("error in reading the response: %s", err.Error())
        }
        if resp.StatusCode != 200 {
            t.Logf("Expected status code 200, got %d. Response: %s",
                resp.StatusCode, string(body))
        }

        t.Logf("====Checking if the image is updated after upgrade")
        upgradedMLVersion := getMLVersion(t, client, adminUsername, adminPassword)
        t.Logf("====Current ML Version after upgrade: %s", upgradedMLVersion)

        if upgradedMLVersion != upgradeImage.Version {
            t.Fatalf("Expected ML version to start with %s, but got %s",
                upgradeImage.Version, upgradedMLVersion)
        } else {
            t.Logf("ML version successfully upgraded to %s", upgradedMLVersion)
        }
        t.Logf("====Deleting namespace: " + namespaceName)
        tunnel8001.Close()
        tunnel8002.Close()
        k8s.DeleteNamespace(t, kubectlOptions, namespaceName)
    }
}

func getMLVersion(t *testing.T, client *req.Client, adminUsername, adminPassword string) string {
    resp, err := client.R().
        SetDigestAuth(adminUsername, adminPassword).
        Get("http://localhost:8002/manage/v2?format=json")

    if err != nil {
        t.Fatalf(err.Error())
    }
    body, err := io.ReadAll(resp.Body)
    if err != nil {
        t.Logf("error in reading the response: %s", err.Error())
    }
    if resp.StatusCode != 200 {
        t.Fatalf("Expected status code 200, got %d. Response: %s",
            resp.StatusCode, string(body))
    }
    mlVersion := gjson.Get(string(body), `local-cluster-default.effective-version`).String()
    return mlVersion
}
