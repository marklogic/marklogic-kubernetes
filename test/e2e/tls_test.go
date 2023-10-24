package e2e

import (
	"crypto/tls"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/imroc/req/v3"
	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"

	"github.com/gruntwork-io/terratest/modules/helm"
	http_helper "github.com/gruntwork-io/terratest/modules/http-helper"
	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/gruntwork-io/terratest/modules/random"
)

func TestTLSEnabledWithSelfSigned(t *testing.T) {
	// Path to the helm chart we will test
	helmChartPath, e := filepath.Abs("../../charts")
	if e != nil {
		t.Fatalf(e.Error())
	}
	username := "admin"
	password := "admin"

	namespaceName := "marklogic-" + strings.ToLower(random.UniqueId())
	kubectlOptions := k8s.NewKubectlOptions("", "", namespaceName)
	options := &helm.Options{
		KubectlOptions: kubectlOptions,
		SetValues: map[string]string{
			"persistence.enabled":           "false",
			"replicaCount":                  "1",
			"image.repository":              "marklogicdb/marklogic-db",
			"image.tag":                     "latest",
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
		func(statusCode int, body string) bool {
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

	resp, err := client.R().
		SetDigestAuth(username, password).
		Get("https://localhost:8002/manage/v2")

	if err != nil {
		t.Fatalf(err.Error())
	}

	fmt.Println("StatusCode: ", resp.GetStatusCode())
}

func GenerateCertificates(command string) error {
	cmd := exec.Command("bash", "-c", command)
	err := cmd.Run()
	if err != nil {
		fmt.Println(err)
	}
	return err
}

func TestTLSEnabledWithNamedCert(t *testing.T) {
	// Path to the helm chart we will test
	releaseName := "marklogic"
	namespaceName := "marklogic-" + "tlsnamed"
	kubectlOptions := k8s.NewKubectlOptions("", "", namespaceName)
	var err error

	// Setup the args for helm install using custom values.yaml file
	options := &helm.Options{
		ValuesFiles:    []string{"../test_data/values/tls_twonode_values.yaml"},
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

	//generate certificates for testing tls using named certificates
	err = GenerateCertificates("openssl genrsa -out ../test_data/ca_certs/ca-private-key.pem 2048")
	if err != nil {
		t.Log("====Error: ", err)
	}

	err = GenerateCertificates("openssl req -new -x509 -days 3650 -key ../test_data/ca_certs/ca-private-key.pem -out ../test_data/ca_certs/cacert.pem -subj '/CN=TlsTest/C=US/ST=California/L=RedwoodCity/O=Progress/OU=MarkLogic'")
	if err != nil {
		t.Log("====Error: ", err)
	}

	err = GenerateCertificates("openssl genpkey -algorithm RSA -out ../test_data/pod_zero_certs/tls.key")
	if err != nil {
		t.Log("====Error: ", err)
	}

	err = GenerateCertificates("openssl req -new -key ../test_data/pod_zero_certs/tls.key -config ../test_data/pod_zero_certs/server.cnf -out ../test_data/pod_zero_certs/tls.csr")
	if err != nil {
		t.Log("====Error: ", err)
	}

	err = GenerateCertificates("openssl x509 -req -CA ../test_data/ca_certs/cacert.pem -CAkey ../test_data/ca_certs/ca-private-key.pem -CAcreateserial -CAserial ../test_data/pod_zero_certs/cacert.srl -in ../test_data/pod_zero_certs/tls.csr -out ../test_data/pod_zero_certs/tls.crt -days 365")
	if err != nil {
		t.Log("====Error: ", err)
	}

	err = GenerateCertificates("openssl genpkey -algorithm RSA -out ../test_data/pod_one_certs/tls.key")
	if err != nil {
		t.Log("====Error: ", err)
	}

	err = GenerateCertificates("openssl req -new -key ../test_data/pod_one_certs/tls.key -config ../test_data/pod_one_certs/server.cnf -out ../test_data/pod_one_certs/tls.csr")
	if err != nil {
		t.Log("====Error: ", err)
	}

	err = GenerateCertificates("openssl x509 -req -CA ../test_data/ca_certs/cacert.pem -CAkey ../test_data/ca_certs/ca-private-key.pem -CAcreateserial -CAserial ../test_data/pod_one_certs/cacert.srl -in ../test_data/pod_one_certs/tls.csr -out ../test_data/pod_one_certs/tls.crt -days 365")
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
		func(statusCode int, body string) bool {
			return statusCode == 200
		},
	)

	// wait until pods are in Ready status
	k8s.WaitUntilPodAvailable(t, kubectlOptions, podName, 15, 30*time.Second)
	k8s.WaitUntilPodAvailable(t, kubectlOptions, podOneName, 15, 30*time.Second)

	// get corev1.Pod to get logs of a pod
	pod := k8s.GetPod(t, kubectlOptions, podName)

	// get pod logs to verify named certificate is used
	t.Logf("====Getting pod logs")
	podLogs := k8s.GetPodLogs(t, kubectlOptions, pod, "")

	// verify logs if named certificate is used
	if !strings.Contains(podLogs, "Info: [poststart] certType in postStart: named") {
		t.Errorf("TLS configuration failed")
	}

	// get corev1.Pod to get logs of a pod
	podOne := k8s.GetPod(t, kubectlOptions, podName)

	// get pod logs to verify pod-1 joins the cluster using tls and certificates
	t.Logf("====Getting podOne logs")
	podOneLogs := k8s.GetPodLogs(t, kubectlOptions, podOne, "")

	// verify logs if wallet password is set as secret
	if (!strings.Contains(podOneLogs, "Info: [poststart] MARKLOGIC_JOIN_TLS_ENABLED is set to true, configuring SSL")) && (!strings.Contains(podOneLogs, "creating named certificate")) {
		t.Errorf("TLS configuration failed")
	}

	tunnel := k8s.NewTunnel(
		kubectlOptions, k8s.ResourceTypePod, podName, 8002, 8002)
	defer tunnel.Close()
	tunnel.ForwardPort(t)

	totalHosts := 1
	client := req.C().
		EnableInsecureSkipVerify().
		SetCommonDigestAuth("admin", "admin").
		SetCommonRetryCount(10).
		SetCommonRetryFixedInterval(10 * time.Second)

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

	if totalHosts != 2 {
		t.Errorf("Incorrect number of MarkLogic hosts")
	}

	if err != nil {
		t.Fatalf(err.Error())
	}
}

func TestTlsOnEDnode(t *testing.T) {

	imageRepo, repoPres := os.LookupEnv("dockerRepository")
	imageTag, tagPres := os.LookupEnv("dockerVersion")
	namespaceName := "marklogic-tlsednode"
	kubectlOptions := k8s.NewKubectlOptions("", "", namespaceName)
	dnodeReleaseName := "dnode"
	enodeReleaseName := "enode"
	dnodePodName := dnodeReleaseName + "-0"
	enodePodName0 := enodeReleaseName + "-0"
	enodePodName1 := enodeReleaseName + "-1"
	var err error

	// Path to the helm chart we will test
	helmChartPath, e := filepath.Abs("../../charts")
	if e != nil {
		t.Fatalf(e.Error())
	}

	if !repoPres {
		imageRepo = "marklogic-centos/marklogic-server-centos"
		t.Logf("No imageRepo variable present, setting to default value: " + imageRepo)
	}

	if !tagPres {
		imageTag = "10-internal"
		t.Logf("No imageTag variable present, setting to default value: " + imageTag)
	}

	options := &helm.Options{
		ValuesFiles:    []string{"../test_data/values/tls_dnode_values.yaml"},
		KubectlOptions: k8s.NewKubectlOptions("", "", namespaceName),
	}

	t.Logf("====Creating namespace: " + namespaceName)
	k8s.CreateNamespace(t, kubectlOptions, namespaceName)

	defer t.Logf("====Deleting namespace: " + namespaceName)
	defer k8s.DeleteNamespace(t, kubectlOptions, namespaceName)

	// generate certificates for testing tls using named certificates
	err = GenerateCertificates("openssl genrsa -out ../test_data/ca_certs/ca-private-key.pem 2048")
	if err != nil {
		t.Log("====Error: ", err)
	}

	err = GenerateCertificates("openssl req -new -x509 -days 3650 -key ../test_data/ca_certs/ca-private-key.pem -out ../test_data/ca_certs/cacert.pem -subj '/CN=TlsTest/C=US/ST=California/L=RedwoodCity/O=Progress/OU=MarkLogic'")
	if err != nil {
		t.Log("====Error: ", err)
	}

	err = GenerateCertificates("openssl genpkey -algorithm RSA -out ../test_data/dnode_zero_certs/tls.key")
	if err != nil {
		t.Log("====Error: ", err)
	}

	err = GenerateCertificates("openssl req -new -key ../test_data/dnode_zero_certs/tls.key -config ../test_data/dnode_zero_certs/server.cnf -out ../test_data/dnode_zero_certs/tls.csr")
	if err != nil {
		t.Log("====Error: ", err)
	}

	err = GenerateCertificates("openssl x509 -req -CA ../test_data/ca_certs/cacert.pem -CAkey ../test_data/ca_certs/ca-private-key.pem -CAcreateserial -CAserial ../test_data/dnode_zero_certs/cacert.srl -in ../test_data/dnode_zero_certs/tls.csr -out ../test_data/dnode_zero_certs/tls.crt -days 365")
	if err != nil {
		t.Log("====Error: ", err)
	}

	t.Logf("====Creating secret for ca certificate")
	k8s.RunKubectl(t, kubectlOptions, "create", "secret", "generic", "ca-cert", "--from-file=../test_data/ca_certs/cacert.pem")

	t.Logf("====Creating secret for pod-0 certificate")
	k8s.RunKubectl(t, kubectlOptions, "create", "secret", "generic", "dnode-0-cert", "--from-file=../test_data/dnode_zero_certs/tls.crt", "--from-file=../test_data/dnode_zero_certs/tls.key")

	t.Logf("====Installing Helm Chart " + dnodeReleaseName)
	helm.Install(t, options, helmChartPath, dnodeReleaseName)

	// wait until the pod is in ready status
	k8s.WaitUntilPodAvailable(t, kubectlOptions, dnodePodName, 10, 20*time.Second)

	tunnel := k8s.NewTunnel(
		kubectlOptions, k8s.ResourceTypePod, dnodePodName, 8002, 8002)
	defer tunnel.Close()
	tunnel.ForwardPort(t)

	totalHosts := 0
	bootstrapHost := ""
	client := req.C().
		EnableInsecureSkipVerify().
		SetCommonDigestAuth("admin", "admin").
		SetCommonRetryCount(10).
		SetCommonRetryFixedInterval(10 * time.Second)

	resp, err := client.R().
		AddRetryCondition(func(resp *req.Response, err error) bool {
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Logf("error: %s", err.Error())
			}
			totalHosts = int(gjson.Get(string(body), `host-default-list.list-items.list-count.value`).Num)
			bootstrapHost = (gjson.Get(string(body), `host-default-list.list-items.list-item.#(roleref="bootstrap").nameref`)).Str
			if totalHosts != 1 {
				t.Log("Waiting for host to configure")
			}
			return totalHosts != 1
		}).
		Get("https://localhost:8002/manage/v2/hosts?format=json")
	defer resp.Body.Close()

	if err != nil {
		t.Fatalf(err.Error())
	}

	// verify bootstrap host exists on the cluster
	t.Log("====Verifying bootstrap host exists on the cluster")
	if bootstrapHost == "" {
		t.Errorf("Bootstrap does not exists on cluster")
	}

	t.Log("====Verifying xdqp-ssl-enabled is set to true for dnode group")
	resp, err = client.R().
		Get("https://localhost:8002/manage/v2/groups/dnode/properties?format=json")
	defer resp.Body.Close()

	if err != nil {
		t.Fatalf(err.Error())
	}
	body, err := io.ReadAll(resp.Body)
	xdqpSSLEnabled := gjson.Get(string(body), `xdqp-ssl-enabled`).Bool()

	// verify xdqp-ssl-enabled is set to true
	assert.Equal(t, true, xdqpSSLEnabled, "xdqp-ssl-enabled should be set to true")

	enodeOptions := &helm.Options{
		KubectlOptions: kubectlOptions,
		ValuesFiles:    []string{"../test_data/values/tls_enode_values.yaml"},
	}

	//generate certificates for enodes
	err = GenerateCertificates("openssl genpkey -algorithm RSA -out ../test_data/enode_zero_certs/tls.key")
	if err != nil {
		t.Log("====Error: ", err)
	}

	err = GenerateCertificates("openssl req -new -key ../test_data/enode_zero_certs/tls.key -config ../test_data/enode_zero_certs/server.cnf -out ../test_data/enode_zero_certs/tls.csr")
	if err != nil {
		t.Log("====Error: ", err)
	}

	err = GenerateCertificates("openssl x509 -req -CA ../test_data/ca_certs/cacert.pem -CAkey ../test_data/ca_certs/ca-private-key.pem -CAcreateserial -CAserial ../test_data/enode_zero_certs/cacert.srl -in ../test_data/enode_zero_certs/tls.csr -out ../test_data/enode_zero_certs/tls.crt -days 365")
	if err != nil {
		t.Log("====Error: ", err)
	}

	err = GenerateCertificates("openssl genpkey -algorithm RSA -out ../test_data/enode_one_certs/tls.key")
	if err != nil {
		t.Log("====Error: ", err)
	}

	err = GenerateCertificates("openssl req -new -key ../test_data/enode_one_certs/tls.key -config ../test_data/enode_one_certs/server.cnf -out ../test_data/enode_one_certs/tls.csr")
	if err != nil {
		t.Log("====Error: ", err)
	}

	err = GenerateCertificates("openssl x509 -req -CA ../test_data/ca_certs/cacert.pem -CAkey ../test_data/ca_certs/ca-private-key.pem -CAcreateserial -CAserial ../test_data/enode_one_certs/cacert.srl -in ../test_data/enode_one_certs/tls.csr -out ../test_data/enode_one_certs/tls.crt -days 365")
	if err != nil {
		t.Log("====Error: ", err)
	}

	t.Logf("====Creating secret for enode-0 certificates")
	k8s.RunKubectl(t, kubectlOptions, "create", "secret", "generic", "enode-0-cert", "--from-file=../test_data/enode_zero_certs/tls.crt", "--from-file=../test_data/enode_zero_certs/tls.key")

	t.Logf("====Creating secret for enode-1 certificates")
	k8s.RunKubectl(t, kubectlOptions, "create", "secret", "generic", "enode-1-cert", "--from-file=../test_data/enode_one_certs/tls.crt", "--from-file=../test_data/enode_one_certs/tls.key")

	t.Logf("====Installing Helm Chart " + enodeReleaseName)
	helm.Install(t, enodeOptions, helmChartPath, enodeReleaseName)

	// wait until the first enode pod is in Ready status
	k8s.WaitUntilPodAvailable(t, kubectlOptions, enodePodName0, 45, 20*time.Second)

	t.Log("====Verify xdqp-ssl-enabled is set to false on Enode")
	resp, err = client.R().
		Get("https://localhost:8002/manage/v2/hosts?format=json")

	if err != nil {
		t.Fatalf(err.Error())
	}
	defer resp.Body.Close()
	body, err = io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf(err.Error())
	}

	xdqpSSLEnabled = gjson.Get(string(body), `xdqp-ssl-enabled`).Bool()
	// verify xdqp-ssl-enabled is set to false
	assert.Equal(t, false, xdqpSSLEnabled)

	resp, err = client.R().
		Get("https://localhost:8002/manage/v2/groups")

	defer resp.Body.Close()
	if body, err = io.ReadAll(resp.Body); err != nil {
		t.Fatalf(err.Error())
	}

	// verify groups dnode, enode exists on the cluster
	if !strings.Contains(string(body), "<nameref>dnode</nameref>") && !strings.Contains(string(body), "<nameref>enode</nameref>") {
		t.Errorf("Groups does not exists on cluster")
	}

	// wait until the second enode pod is in Ready status
	k8s.WaitUntilPodAvailable(t, kubectlOptions, enodePodName1, 45, 20*time.Second)

	t.Log("====Verifying two hosts joined enode group")
	enodeHostCount := 0
	resp, err = client.R().
		AddRetryCondition(func(resp *req.Response, err error) bool {
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Logf("error: %s", err.Error())
			}
			enodeHostCount = int((gjson.Get(string(body), `group-default.relations.relation-group.#(typeref="hosts").relation-count.value`)).Num)
			if enodeHostCount != 2 {
				t.Log("Waiting for second host to join MarkLogic cluster")
			}
			return enodeHostCount != 2
		}).
		Get("https://localhost:8002/manage/v2/groups/enode?format=json")

	if err != nil {
		t.Fatalf(err.Error())
	}
	defer resp.Body.Close()

	t.Log(`enodeHostCount:= `, enodeHostCount)

	// verify enode hosts exists on the cluster
	if enodeHostCount != 2 {
		t.Errorf("enode hosts does not exists on cluster")
	}
}
