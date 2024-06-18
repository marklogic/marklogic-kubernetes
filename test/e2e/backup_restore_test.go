package e2e

import (
	"encoding/json"
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

type BackupRestoreReq struct {
	Operation       string `json:"operation"`
	BackupDir       string `json:"backup-dir"`
	IncludeReplicas string `json:"include-replicas"`
	Incremental     string `json:"incremental,omitempty"`
	IncrementalDir  string `json:"incremental-dir,omitempty"`
}
type BackupRestoreStatusReq struct {
	Operation string `json:"operation"`
	JobID     string `json:"job-id"`
	HostName  string `json:"host-name,omitempty"`
}

func PutDocs(docPath string, docName string, client *req.Client, qConsoleEndpoint string) (string, error) {
	result := ""
	xmlData, err := os.ReadFile(docPath + docName)
	if err != nil {
		return result, err
	}
	strXMLData := string(xmlData)

	resp, err := client.R().
		SetContentType("application/json").
		SetBodyString(strXMLData).
		Put(qConsoleEndpoint)
	if err != nil {
		return result, err
	}
	if resp.GetStatusCode() == 201 {
		result = "Created"
	}
	return result, err
}
func GetDocs(client *req.Client, getEndpoint string, acceptHeader string) (string, error) {
	resp, err := client.R().
		SetContentType("application/json").
		SetHeader("Accept", acceptHeader).
		Get(getEndpoint)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	bodyXML, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(bodyXML), err
}

func DeleteDocs(client *req.Client, deleteEndpoint string) (string, error) {
	result := ""
	resp, err := client.R().
		SetContentType("application/json").
		Delete(deleteEndpoint)
	if err != nil {
		return result, err
	}
	defer resp.Body.Close()

	if resp.GetStatusCode() == 204 {
		result = "Deleted"
	}
	return result, err
}

func RunRequests(client *req.Client, dbReq string, hostsEndpoint string) (string, error) {
	var err error
	var body []byte
	headerMap := map[string]string{
		"Content-Type": "application/json",
		"Accept":       "application/json",
	}
	result := ""
	status := ""
	operation := (gjson.Get(dbReq, `operation`)).Str
	var retryFn = (func(resp *req.Response, err error) bool {
		if err != nil {
			fmt.Println(err.Error())
		}
		body, err = io.ReadAll(resp.Body)
		if err != nil {
			fmt.Println(err.Error())
		}
		result = (string(body))
		return true
	})

	if operation == "backup-status" {
		retryFn = (func(resp *req.Response, err error) bool {
			if err != nil {
				fmt.Printf("error: %s", err.Error())
			}
			body, _ := io.ReadAll(resp.Body)
			status = (gjson.Get(string(body), `status`)).Str
			if status != "completed" {
				fmt.Println("Waiting for backup to be completed")
			}
			result = (string(body))
			return status != "completed"
		})
	}

	resp, err := client.R().
		AddRetryCondition(retryFn).
		SetHeaders(headerMap).
		SetBodyString(dbReq).
		Post(hostsEndpoint)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	return result, err
}

func TestMlDbBackupRestore(t *testing.T) {
	// var resp *http.Response
	var helmChartPath string
	var err error
	var podName string
	var initialChartVersion string
	upgradeHelm, _ := os.LookupEnv("upgradeTest")
	runUpgradeTest, err := strconv.ParseBool(upgradeHelm)
	if runUpgradeTest {
		initialChartVersion, _ = os.LookupEnv("initialChartVersion")
		t.Logf("====Setting initial Helm chart version: %s", initialChartVersion)
	}
	imageRepo, repoPres := os.LookupEnv("dockerRepository")
	imageTag, tagPres := os.LookupEnv("dockerVersion")

	if !repoPres {
		imageRepo = "ml-docker-db-dev-tierpoint.bed-artifactory.bedford.progress.com/marklogic/marklogic-server-centos"
		t.Logf("No imageRepo variable present, setting to default value: " + imageRepo)
	}

	if !tagPres {
		imageTag = "11.1.0-centos-1.1.2"
		t.Logf("No imageTag variable present, setting to default value: " + imageTag)
	}

	username := "admin"
	password := "admin"

	namespaceName := "ml-" + strings.ToLower(random.UniqueId())
	kubectlOptions := k8s.NewKubectlOptions("", "", namespaceName)
	options := &helm.Options{
		KubectlOptions: kubectlOptions,
		SetValues: map[string]string{
			"persistence.enabled":   "true",
			"replicaCount":          "1",
			"image.repository":      "ml-docker-db-dev-tierpoint.bed-artifactory.bedford.progress.com/marklogic/marklogic-server-ubi-rootless",
			"image.tag":             "latest-11",
			"auth.adminUsername":    username,
			"auth.adminPassword":    password,
			"logCollection.enabled": "false",
		},
		Version: initialChartVersion,
	}

	t.Logf("====Installing Helm Chart")
	releaseName := "bkuprestore"

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
		helm.AddRepo(t, options, "marklogic", "https://marklogic.github.io/marklogic-kubernetes/")
		defer helm.RemoveRepo(t, options, "marklogic")
		helmChartPath = "marklogic/marklogic"
	}

	t.Logf("====Setting helm chart path to %s", helmChartPath)
	t.Logf("====Installing Helm Chart")
	podName = testUtil.HelmInstall(t, options, releaseName, kubectlOptions, helmChartPath)

	t.Logf("====Describe pod for backup restore test")
	k8s.RunKubectl(t, kubectlOptions, "describe", "pod", podName)

	// wait until the pod is in Ready status
	k8s.WaitUntilPodAvailable(t, kubectlOptions, podName, 10, 15*time.Second)

	if runUpgradeTest {
		// create options for helm upgrade
		upgradeOptionsMap := map[string]string{
			"persistence.enabled":   "true",
			"replicaCount":          "1",
			"logCollection.enabled": "false",
			"allowLongHostnames":    "true",
		}
		if strings.HasPrefix(initialChartVersion, "1.0") {
			podName = releaseName + "-marklogic-0"
			upgradeOptionsMap["useLegacyHostnames"] = "true"
		}
		//set helm options for upgrading helm chart version
		helmUpgradeOptions := &helm.Options{
			KubectlOptions: kubectlOptions,
			SetValues:      upgradeOptionsMap,
		}

		t.Logf("UpgradeHelmTest is enabled. Running helm upgrade test")
		testUtil.HelmUpgrade(t, helmUpgradeOptions, releaseName, kubectlOptions, []string{podName}, initialChartVersion)
	}

	//create backup directories and setup permissions
	k8s.RunKubectl(t, kubectlOptions, "exec", podName, "--", "/bin/bash", "-c", "cd /tmp && mkdir backup && chmod 777 backup && mkdir backup/incrBackup && chmod 777 backup/incrBackup")

	//set test data path and documents for tests
	docPath := "../test_data/bkup_restore_reqs/"
	docs := []string{"testOne.xml", "testTwo.xml"}

	tunnel8000 := k8s.NewTunnel(kubectlOptions, k8s.ResourceTypePod, podName, 8000, 8000)
	defer tunnel8000.Close()
	tunnel8000.ForwardPort(t)

	client := req.C().
		SetCommonDigestAuth(username, password).
		SetCommonRetryCount(10).
		SetCommonRetryFixedInterval(10 * time.Second)

	//creating documents in the Documents DB
	for _, doc := range docs {
		qConsoleEndpoint := fmt.Sprintf("http://%s/v1/documents?database=Documents&uri=%s", tunnel8000.Endpoint(), doc)
		fmt.Println(qConsoleEndpoint)
		result, err := PutDocs(docPath, doc, client, qConsoleEndpoint)
		if err != nil {
			t.Fatalf(err.Error())
		}
		assert.Equal(t, "Created", result)
	}

	getEndpoint := fmt.Sprintf("http://%s/v1/documents?database=Documents&uri=%s&uri=%s", tunnel8000.Endpoint(), docs[0], docs[1])
	fmt.Println(getEndpoint)

	// verify both docs are loaded in Documents DB
	result, err := GetDocs(client, getEndpoint, "multipart/mixed")
	if err != nil {
		t.Fatalf(err.Error())
	}
	if !strings.Contains(string(result), "<b>two</b>") && !strings.Contains(string(result), "<a>one</a>") {
		t.Errorf("Both docs are loaded")
	}

	tunnel8002 := k8s.NewTunnel(kubectlOptions, k8s.ResourceTypePod, podName, 8002, 8002)
	defer tunnel8002.Close()
	tunnel8002.ForwardPort(t)
	manageEndpoint := fmt.Sprintf("http://%s/manage/v2/databases/Documents", tunnel8002.Endpoint())
	fmt.Println(manageEndpoint)

	bkupReq := &BackupRestoreReq{
		Operation:       "backup-database",
		BackupDir:       "/tmp/backup",
		IncludeReplicas: "true"}
	bkupReqRes, _ := json.Marshal(bkupReq)

	//full backup for Documents DB
	result, err = RunRequests(client, string(bkupReqRes), manageEndpoint)
	if err != nil {
		t.Fatalf(err.Error())
	}
	jobID := (gjson.Get(result, `job-id`))
	hostName := (gjson.Get(result, `host-name`))

	bkupStatusReq := &BackupRestoreStatusReq{
		Operation: "backup-status",
		JobID:     jobID.String(),
		HostName:  hostName.String()}
	bkupStatusReqRes, _ := json.Marshal(bkupStatusReq)

	//get status of full backup job
	result, err = RunRequests(client, string(bkupStatusReqRes), manageEndpoint)
	if err != nil {
		t.Fatalf(err.Error())
	}
	bkupStatus := (gjson.Get(result, `status`)).Str

	//verify full backup is completed
	assert.Equal(t, "completed", bkupStatus)

	deleteEndpoint := fmt.Sprintf("http://%s/v1/documents?database=Documents&uri=%s", tunnel8000.Endpoint(), docs[1])
	fmt.Println(deleteEndpoint)

	//incremental backup
	incrBkupReq := &BackupRestoreReq{
		Operation:       "backup-database",
		BackupDir:       "/tmp/backup",
		IncludeReplicas: "true",
		Incremental:     "true",
		IncrementalDir:  "/tmp/backup/incrBackup"}
	incrBkupReqRes, _ := json.Marshal(incrBkupReq)

	//incremnetal backup for Documents DB
	result, err = RunRequests(client, string(incrBkupReqRes), manageEndpoint)
	if err != nil {
		t.Fatalf(err.Error())
	}
	jobID = (gjson.Get(result, `job-id`))
	hostName = (gjson.Get(result, `host-name`))

	incrBkupStatusReq := &BackupRestoreStatusReq{
		Operation: "backup-status",
		JobID:     jobID.String(),
		HostName:  hostName.String()}
	incrBkupStatusReqRes, _ := json.Marshal(incrBkupStatusReq)

	//get status of backup job
	result, err = RunRequests(client, string(incrBkupStatusReqRes), manageEndpoint)
	if err != nil {
		t.Fatalf(err.Error())
	}
	incrBkupStatus := (gjson.Get(result, `status`)).Str

	//verify backup is completed
	assert.Equal(t, "completed", incrBkupStatus)

	//delete a document from Documents DB
	result, err = DeleteDocs(client, deleteEndpoint)
	if err != nil {
		t.Fatalf(err.Error())
	}
	assert.Equal(t, "Deleted", result)

	rstrReq := &BackupRestoreReq{
		Operation:       "restore-database",
		BackupDir:       "/tmp/backup",
		IncludeReplicas: "true"}
	brstrReqRes, _ := json.Marshal(rstrReq)

	//restore Documents DB from incremental backup
	result, err = RunRequests(client, string(brstrReqRes), manageEndpoint)
	if err != nil {
		t.Fatalf(err.Error())
	}
	restoreJobID := (gjson.Get(result, `job-id`)).Str
	assert.NotEqual(t, "", restoreJobID)

	result, err = GetDocs(client, getEndpoint, "multipart/mixed")
	if err != nil {
		t.Fatalf(err.Error())
	}
	// verify both docs are restored
	if !strings.Contains(string(result), "<b>two</b>") && !strings.Contains(string(result), "<a>one</a>") {
		t.Errorf("Both docs are restored")
	}
}
