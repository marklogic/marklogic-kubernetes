package e2e

import (
	"crypto/tls"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/imroc/req/v3"
	"github.com/marklogic/marklogic-kubernetes/test/testUtil"
	"github.com/tidwall/gjson"

	"github.com/gruntwork-io/terratest/modules/helm"
	http_helper "github.com/gruntwork-io/terratest/modules/http-helper"
	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/gruntwork-io/terratest/modules/random"
)

// 	"github.com/stretchr/testify/assert"

func TestTLSEnabledWithSelfSigned(t *testing.T) {
	// Path to the helm chart we will test
	helmChartPath, e := filepath.Abs("../../charts")
	if e != nil {
		t.Fatalf(e.Error())
	}
	imageRepo := "marklogicdb/marklogic-db"
	imageTag := "11.2.0-centos-1.1.2"
	username := "admin"
	password := "admin"

	namespaceName := "marklogic-" + strings.ToLower(random.UniqueId())
	kubectlOptions := k8s.NewKubectlOptions("", "", namespaceName)
	options := &helm.Options{
		KubectlOptions: kubectlOptions,
		SetValues: map[string]string{
			"persistence.enabled":           "false",
			"replicaCount":                  "2",
			"image.repository":              imageRepo,
			"image.tag":                     imageTag,
			"auth.adminUsername":            username,
			"auth.adminPassword":            password,
			"logCollection.enabled":         "false",
			"tls.enableOnDefaultAppServers": "true",
		},
	}

	t.Logf("====Creating namespace: " + namespaceName)
	k8s.CreateNamespace(t, kubectlOptions, namespaceName)

	defer t.Logf("====Deleting namespace: " + namespaceName)
	defer k8s.DeleteNamespace(t, kubectlOptions, namespaceName)

	t.Logf("====Installing Helm Chart")
	releaseName := "test-join"
	helm.Install(t, options, helmChartPath, releaseName)

	podName := releaseName + "-0"
	tlsConfig := tls.Config{InsecureSkipVerify: true}

	// wait until the pod is in Ready status
	k8s.WaitUntilPodAvailable(t, kubectlOptions, podName, 10, 20*time.Second)
	podOneName := releaseName + "-1"
	k8s.WaitUntilPodAvailable(t, kubectlOptions, podOneName, 10, 20*time.Second)

	// verify MarkLogic is ready
	_, err := testUtil.MLReadyCheck(t, kubectlOptions, podName, &tlsConfig)
	if err != nil {
		t.Fatal("MarkLogic failed to start")
	}

	tunnel := k8s.NewTunnel(
		kubectlOptions, k8s.ResourceTypePod, podName, 8002, 8002)
	defer tunnel.Close()
	tunnel.ForwardPort(t)
	endpointManage := fmt.Sprintf("https://%s/manage/v2", tunnel.Endpoint())
	t.Logf(`Endpoint: %s`, endpointManage)

	client := req.C().EnableInsecureSkipVerify()

	resp, err := client.R().
		SetDigestAuth(username, password).
		Get("https://localhost:8002/manage/v2")

	if err != nil {
		t.Fatalf(err.Error())
	}

	fmt.Println("StatusCode: ", resp.GetStatusCode())
}

func GenerateCACertificate(caPath string) error {
	var err error
	fmt.Println("====Generating CA Certificates")
	genKeyCmd := strings.Replace("openssl genrsa -out caPath/ca-private-key.pem 2048", "caPath", caPath, -1)
	genCACertCmd := strings.Replace("openssl req -new -x509 -days 3650 -key caPath/ca-private-key.pem -out caPath/cacert.pem -subj '/CN=TlsTest/C=US/ST=California/L=RedwoodCity/O=Progress/OU=MarkLogic'", "caPath", caPath, -1)
	rvariable := []string{genKeyCmd, genCACertCmd}
	for _, j := range rvariable {
		cmd := exec.Command("bash", "-c", j)
		err = cmd.Run()
	}
	return err
}

func GenerateCertificates(path string, caPath string) error {
	var err error
	fmt.Println("====Generating TLS Certificates")
	genTLSKeyCmd := strings.Replace("openssl genpkey -algorithm RSA -out path/tls.key", "path", path, -1)
	genCsrCmd := strings.Replace("openssl req -new -key path/tls.key -config path/server.cnf -out path/tls.csr", "path", path, -1)
	genCrtCmd := strings.Replace(strings.Replace("openssl x509 -req -CA caPath/cacert.pem -CAkey caPath/ca-private-key.pem -CAcreateserial -CAserial path/cacert.srl -in path/tls.csr -out path/tls.crt -days 365", "path", path, -1), "caPath", caPath, -1)
	rvariable := []string{genTLSKeyCmd, genCsrCmd, genCrtCmd}
	for _, j := range rvariable {
		cmd := exec.Command("bash", "-c", j)
		err = cmd.Run()
	}
	return err
}

func TestTLSEnabledWithNamedCert(t *testing.T) {
	// Path to the helm chart we will test
	releaseName := "marklogic"
	namespaceName := "marklogic-" + "tlsnamed"
	kubectlOptions := k8s.NewKubectlOptions("", "", namespaceName)
	var err error
	imageRepo := "marklogicdb/marklogic-db"
	imageTag := "11.2.0-centos-1.1.2"

	// Setup the args for helm install using custom values.yaml file
	options := &helm.Options{
		ValuesFiles: []string{"../test_data/values/tls_twonode_values.yaml"},
		SetValues: map[string]string{
			"image.repository": imageRepo,
			"image.tag":        imageTag,
		},
		KubectlOptions: k8s.NewKubectlOptions("", "", namespaceName),
	}

	helmChartPath, e := filepath.Abs("../../charts")
	if e != nil {
		t.Fatalf(e.Error())
	}

	t.Logf("====Creating namespace: " + namespaceName)
	k8s.CreateNamespace(t, kubectlOptions, namespaceName)

	defer t.Logf("====Deleting namespace: " + namespaceName)
	defer k8s.DeleteNamespace(t, kubectlOptions, namespaceName)

	// generate CA certificates for pods
	err = GenerateCACertificate("../test_data/ca_certs")
	if err != nil {
		t.Log("====Error: ", err)
	}

	//generate certificates for pod zero
	err = GenerateCertificates("../test_data/pod_zero_certs", "../test_data/ca_certs")
	if err != nil {
		t.Log("====Error: ", err)
	}

	//generate certificates for pod one
	err = GenerateCertificates("../test_data/pod_one_certs", "../test_data/ca_certs")
	if err != nil {
		t.Log("====Error: ", err)
	}

	// create secret for ca certificate
	t.Logf("====Creating secret for ca certificate")
	k8s.RunKubectl(t, kubectlOptions, "create", "secret", "generic", "ca-cert", "--from-file=../test_data/ca_certs/cacert.pem")

	// create secret for named certificate for pod-0
	t.Logf("====Creating secret for pod-0 certificate")
	k8s.RunKubectl(t, kubectlOptions, "create", "secret", "generic", "marklogic-0-cert", "--from-file=../test_data/pod_zero_certs/tls.crt", "--from-file=../test_data/pod_zero_certs/tls.key")

	// create secret for named certificate for pod-1
	t.Logf("====Creating secret for pod-1 certificate")
	k8s.RunKubectl(t, kubectlOptions, "create", "secret", "generic", "marklogic-1-cert", "--from-file=../test_data/pod_one_certs/tls.crt", "--from-file=../test_data/pod_one_certs/tls.key")

	t.Logf("====Installing Helm Chart")
	helm.Install(t, options, helmChartPath, releaseName)

	podName := releaseName + "-0"
	podOneName := releaseName + "-1"

	tlsConfig := tls.Config{InsecureSkipVerify: true}

	// wait until the pod is in Ready status
	k8s.WaitUntilPodAvailable(t, kubectlOptions, podName, 10, 20*time.Second)
	tunnel8001 := k8s.NewTunnel(kubectlOptions, k8s.ResourceTypePod, podName, 8001, 8001)
	defer tunnel8001.Close()
	tunnel8001.ForwardPort(t)
	endpoint7997 := fmt.Sprintf("http://%s", tunnel8001.Endpoint())

	// verify if 7997 health check endpoint returns 200
	http_helper.HttpGetWithRetryWithCustomValidation(
		t,
		endpoint7997,
		&tlsConfig,
		10,
		15*time.Second,
		func(statusCode int, body string) bool {
			return statusCode == 403
		},
	)

	t.Log("Getting error message from marklogic-0")
	client := req.C().
		EnableInsecureSkipVerify().
		SetCommonDigestAuth("admin", "admin").
		SetCommonRetryCount(10).
		SetCommonRetryFixedInterval(10 * time.Second)

	res, err := client.R().
		Get("http://localhost:8001/")
	defer res.Body.Close()

	body1, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf(err.Error())
	}
	t.Log("======localhost:8001")
	t.Log(string(body1))

	// wait until pods are in Ready status
	k8s.RunKubectl(t, kubectlOptions, "get", "pods")
	k8s.RunKubectl(t, kubectlOptions, "get", "ns")
	k8s.RunKubectl(t, kubectlOptions, "get", "secrets")
	k8s.RunKubectl(t, kubectlOptions, "get", "pvc")
	k8s.RunKubectl(t, kubectlOptions, "get", "cm", "marklogic", "-o", "yaml")
	err = k8s.RunKubectlE(t, kubectlOptions, "logs", podName)
	if err != nil {
		t.Logf("Error: %s", err.Error())
	}
	k8s.RunKubectl(t, kubectlOptions, "describe", "pods", podOneName)

	isPodOneAvailable := false
	counter := 0
	for isPodOneAvailable == false {
		k8s.RunKubectl(t, kubectlOptions, "get", "pod", podOneName)
		// k8s.RunKubectl(t, kubectlOptions, "describe", "pod", podOneName)
		podOne := k8s.GetPod(t, kubectlOptions, podOneName)
		err := k8s.RunKubectlE(t, kubectlOptions, "logs", "-p", podOneName)
		if err != nil {
			t.Logf("ml-1 log not ready")
		}
		err = k8s.RunKubectlE(t, kubectlOptions, "logs", "-c", "copy-certs", podOneName)
		if err != nil {
			t.Logf("copy-certs log not ready")
		}
		if !k8s.IsPodAvailable(podOne) {
			counter++
			t.Logf("Pod is not available, retrying %d times", counter)
			time.Sleep(20 * time.Second)
			if counter > 15 {
				t.Fatalf("Pod is not available after 5 minutes")
			}
		} else {
			isPodOneAvailable = true
		}
	}

	// k8s.WaitUntilPodAvailable(t, kubectlOptions, podOneName, 15, 30*time.Second)

	tunnel := k8s.NewTunnel(
		kubectlOptions, k8s.ResourceTypePod, podName, 8002, 8002)
	defer tunnel.Close()
	tunnel.ForwardPort(t)

	totalHosts := 1
	// client := req.C().
	// 	EnableInsecureSkipVerify().
	// 	SetCommonDigestAuth("admin", "admin").
	// 	SetCommonRetryCount(10).
	// 	SetCommonRetryFixedInterval(10 * time.Second)

	resp, err := client.R().
		AddRetryCondition(func(resp *req.Response, err error) bool {
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Logf("error: %s", err.Error())
			}
			totalHosts = int(gjson.Get(string(body), `host-status-list.status-list-summary.total-hosts.value`).Num)
			if totalHosts != 2 {
				t.Log("Waiting for second host to join MarkLogic cluster")
			}
			return totalHosts != 2
		}).
		Get("https://localhost:8002/manage/v2/hosts?view=status&format=json")
	defer resp.Body.Close()
	if err != nil {
		t.Fatalf(err.Error())
	}

	if totalHosts != 2 {
		t.Errorf("Incorrect number of MarkLogic hosts")
	}

	resp, err = client.R().
		Get("https://localhost:8002/manage/v2/certificate-templates/defaultTemplate?format=json")
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf(err.Error())
	}
	defaultCertTemplID := gjson.Get(string(body), `certificate-template-default.id`)

	resp, err = client.R().
		Get("https://localhost:8002/manage/v2/certificates?format=json")
	defer resp.Body.Close()

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf(err.Error())
	}
	certID := (gjson.Get(string(body), `certificate-default-list.list-items.list-item.1.idref`))

	endpoint := strings.Replace("https://localhost:8002/manage/v2/certificates/certId?format=json", "certId", certID.Str, -1)
	resp, err = client.R().
		Get(endpoint)
	defer resp.Body.Close()

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf(err.Error())
	}

	certTemplID := gjson.Get(string(body), `certificate-default.template-id`)
	isCertTemporary := gjson.Get(string(body), `certificate-default.temporary`)
	certHostName := gjson.Get(string(body), `certificate-default.host-name`)

	//verify named certificate is configured for default certificate template
	if defaultCertTemplID.Str != certTemplID.Str {
		t.Errorf("Named certificates not configured for defaultTemplate")
	}

	//verify temporary certificate is not used
	if isCertTemporary.Str != "false" {
		t.Errorf("Named certificate is not configured for host")
	}

	//verify correct hostname is set for named certificate
	t.Log("Verifying hostname is set for named certificate", certHostName.Str)

	if certHostName.Str != "marklogic-1.marklogic.marklogic-tlsnamed.svc.cluster.local" && certHostName.Str != "marklogic-0.marklogic.marklogic-tlsnamed.svc.cluster.local" {
		t.Errorf("Incorrect hostname configured for Named certificate")
	}
}

// func TestTlsOnEDnode(t *testing.T) {

// 	imageRepo, repoPres := os.LookupEnv("dockerRepository")
// 	imageTag, tagPres := os.LookupEnv("dockerVersion")
// 	namespaceName := "marklogic-tlsednode"
// 	kubectlOptions := k8s.NewKubectlOptions("", "", namespaceName)
// 	dnodeReleaseName := "dnode"
// 	enodeReleaseName := "enode"
// 	dnodePodName := dnodeReleaseName + "-0"
// 	enodePodName0 := enodeReleaseName + "-0"
// 	enodePodName1 := enodeReleaseName + "-1"
// 	var err error

// 	// Path to the helm chart we will test
// 	helmChartPath, e := filepath.Abs("../../charts")
// 	if e != nil {
// 		t.Fatalf(e.Error())
// 	}

// 	if !repoPres {
// 		imageRepo = "marklogicdb/marklogic-db"
// 		t.Logf("No imageRepo variable present, setting to default value: " + imageRepo)
// 	}

// 	if !tagPres {
// 		imageTag = "latest"
// 		t.Logf("No imageTag variable present, setting to default value: " + imageTag)
// 	}

// 	options := &helm.Options{
// 		ValuesFiles: []string{"../test_data/values/tls_dnode_values.yaml"},
// 		SetValues: map[string]string{
// 			"image.repository": imageRepo,
// 			"image.tag":        imageTag,
// 		},
// 		KubectlOptions: k8s.NewKubectlOptions("", "", namespaceName),
// 	}

// 	t.Logf("====Creating namespace: " + namespaceName)
// 	k8s.CreateNamespace(t, kubectlOptions, namespaceName)

// 	defer t.Logf("====Deleting namespace: " + namespaceName)
// 	defer k8s.DeleteNamespace(t, kubectlOptions, namespaceName)

// 	// generate CA certificates for pods
// 	err = GenerateCACertificate("../test_data/ca_certs")
// 	if err != nil {
// 		t.Log("====Error: ", err)
// 	}

// 	//generate certificates for dnode pod zero
// 	err = GenerateCertificates("../test_data/dnode_zero_certs", "../test_data/ca_certs")
// 	if err != nil {
// 		t.Log("====Error: ", err)
// 	}

// 	t.Logf("====Creating secret for ca certificate")
// 	k8s.RunKubectl(t, kubectlOptions, "create", "secret", "generic", "ca-cert", "--from-file=../test_data/ca_certs/cacert.pem")

// 	t.Logf("====Creating secret for pod-0 certificate")
// 	k8s.RunKubectl(t, kubectlOptions, "create", "secret", "generic", "dnode-0-cert", "--from-file=../test_data/dnode_zero_certs/tls.crt", "--from-file=../test_data/dnode_zero_certs/tls.key")

// 	t.Logf("====Installing Helm Chart " + dnodeReleaseName)
// 	helm.Install(t, options, helmChartPath, dnodeReleaseName)

// 	// wait until the pod is in ready status
// 	k8s.WaitUntilPodAvailable(t, kubectlOptions, dnodePodName, 10, 20*time.Second)

// 	tunnel := k8s.NewTunnel(
// 		kubectlOptions, k8s.ResourceTypePod, dnodePodName, 8002, 8002)
// 	defer tunnel.Close()
// 	tunnel.ForwardPort(t)

// 	totalHosts := 0
// 	bootstrapHost := ""
// 	client := req.C().
// 		EnableInsecureSkipVerify().
// 		SetCommonDigestAuth("admin", "admin").
// 		SetCommonRetryCount(10).
// 		SetCommonRetryFixedInterval(10 * time.Second)

// 	resp, err := client.R().
// 		AddRetryCondition(func(resp *req.Response, err error) bool {
// 			body, err := io.ReadAll(resp.Body)
// 			if err != nil {
// 				t.Logf("error: %s", err.Error())
// 			}
// 			totalHosts = int(gjson.Get(string(body), `host-default-list.list-items.list-count.value`).Num)
// 			bootstrapHost = (gjson.Get(string(body), `host-default-list.list-items.list-item.#(roleref="bootstrap").nameref`)).Str
// 			if totalHosts != 1 {
// 				t.Log("Waiting for host to configure")
// 			}
// 			return totalHosts != 1
// 		}).
// 		Get("https://localhost:8002/manage/v2/hosts?format=json")
// 	defer resp.Body.Close()

// 	if err != nil {
// 		t.Fatalf(err.Error())
// 	}

// 	// verify bootstrap host exists on the cluster
// 	t.Log("====Verifying bootstrap host exists on the cluster")
// 	if bootstrapHost == "" {
// 		t.Errorf("Bootstrap does not exists on cluster")
// 	}

// 	t.Log("====Verifying xdqp-ssl-enabled is set to true for dnode group")
// 	resp, err = client.R().
// 		Get("https://localhost:8002/manage/v2/groups/dnode/properties?format=json")
// 	defer resp.Body.Close()

// 	if err != nil {
// 		t.Fatalf(err.Error())
// 	}
// 	body, err := io.ReadAll(resp.Body)
// 	xdqpSSLEnabled := gjson.Get(string(body), `xdqp-ssl-enabled`).Bool()

// 	// verify xdqp-ssl-enabled is set to true
// 	assert.Equal(t, true, xdqpSSLEnabled, "xdqp-ssl-enabled should be set to true")

// 	enodeOptions := &helm.Options{
// 		KubectlOptions: kubectlOptions,
// 		SetValues: map[string]string{
// 			"image.repository": imageRepo,
// 			"image.tag":        imageTag,
// 		},
// 		ValuesFiles: []string{"../test_data/values/tls_enode_values.yaml"},
// 	}

// 	//generate certificates for enode pod zero
// 	err = GenerateCertificates("../test_data/enode_zero_certs", "../test_data/ca_certs")
// 	if err != nil {
// 		t.Log("====Error: ", err)
// 	}

// 	//generate certificates for enode pod one
// 	err = GenerateCertificates("../test_data/enode_one_certs", "../test_data/ca_certs")
// 	if err != nil {
// 		t.Log("====Error: ", err)
// 	}

// 	t.Logf("====Creating secret for enode-0 certificates")
// 	k8s.RunKubectl(t, kubectlOptions, "create", "secret", "generic", "enode-0-cert", "--from-file=../test_data/enode_zero_certs/tls.crt", "--from-file=../test_data/enode_zero_certs/tls.key")

// 	t.Logf("====Creating secret for enode-1 certificates")
// 	k8s.RunKubectl(t, kubectlOptions, "create", "secret", "generic", "enode-1-cert", "--from-file=../test_data/enode_one_certs/tls.crt", "--from-file=../test_data/enode_one_certs/tls.key")

// 	t.Logf("====Installing Helm Chart " + enodeReleaseName)
// 	helm.Install(t, enodeOptions, helmChartPath, enodeReleaseName)

// 	// wait until the first enode pod is in Ready status
// 	k8s.WaitUntilPodAvailable(t, kubectlOptions, enodePodName0, 20, 20*time.Second)

// 	t.Log("====Verify xdqp-ssl-enabled is set to false on Enode")
// 	resp, err = client.R().
// 		Get("https://localhost:8002/manage/v2/hosts?format=json")

// 	if err != nil {
// 		t.Fatalf(err.Error())
// 	}
// 	defer resp.Body.Close()
// 	body, err = io.ReadAll(resp.Body)
// 	if err != nil {
// 		t.Fatalf(err.Error())
// 	}

// 	xdqpSSLEnabled = gjson.Get(string(body), `xdqp-ssl-enabled`).Bool()
// 	// verify xdqp-ssl-enabled is set to false
// 	assert.Equal(t, false, xdqpSSLEnabled)

// 	resp, err = client.R().
// 		Get("https://localhost:8002/manage/v2/groups")

// 	defer resp.Body.Close()
// 	if body, err = io.ReadAll(resp.Body); err != nil {
// 		t.Fatalf(err.Error())
// 	}

// 	// verify groups dnode, enode exists on the cluster
// 	if !strings.Contains(string(body), "<nameref>dnode</nameref>") && !strings.Contains(string(body), "<nameref>enode</nameref>") {
// 		t.Errorf("Groups does not exists on cluster")
// 	}

// 	// wait until the second enode pod is in Ready status
// 	k8s.WaitUntilPodAvailable(t, kubectlOptions, enodePodName1, 20, 20*time.Second)

// 	t.Log("====Verifying two hosts joined enode group")
// 	enodeHostCount := 0
// 	resp, err = client.R().
// 		AddRetryCondition(func(resp *req.Response, err error) bool {
// 			body, err := io.ReadAll(resp.Body)
// 			if err != nil {
// 				t.Logf("error: %s", err.Error())
// 			}
// 			enodeHostCount = int((gjson.Get(string(body), `group-default.relations.relation-group.#(typeref="hosts").relation-count.value`)).Num)
// 			if enodeHostCount != 2 {
// 				t.Log("Waiting for second host to join MarkLogic cluster")
// 			}
// 			return enodeHostCount != 2
// 		}).
// 		Get("https://localhost:8002/manage/v2/groups/enode?format=json")

// 	if err != nil {
// 		t.Fatalf(err.Error())
// 	}
// 	defer resp.Body.Close()

// 	t.Log(`enodeHostCount:= `, enodeHostCount)

// 	// verify enode hosts exists on the cluster
// 	if enodeHostCount != 2 {
// 		t.Errorf("enode hosts does not exists on cluster")
// 	}
// }
