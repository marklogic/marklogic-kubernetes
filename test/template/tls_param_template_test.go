package template_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"

	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/gruntwork-io/terratest/modules/k8s"
)

func TestChartTemplateTLSEnabled(t *testing.T) {

	// Path to the helm chart we will test
	helmChartPath, err := filepath.Abs("../../charts")
	releaseName := "marklogic"
	t.Log(helmChartPath, releaseName)
	require.NoError(t, err)

	// Set up the namespace; confirm that the template renders the expected value for the namespace.
	namespaceName := "marklogic-templ"
	t.Logf("Namespace: %s\n", namespaceName)

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

	// Setup the args for helm install using custom values.yaml file
	options := &helm.Options{
		ValuesFiles: []string{"../test_data/values/tls_template_values.yaml"},
		SetValues: map[string]string{
			"image.repository": imageRepo,
			"image.tag":        imageTag,
		},
		KubectlOptions: k8s.NewKubectlOptions("", "", namespaceName),
	}

	// render the tempate
	output := helm.RenderTemplate(t, options, helmChartPath, releaseName, []string{"templates/statefulset.yaml"})

	var statefulset appsv1.StatefulSet
	helm.UnmarshalK8SYaml(t, output, &statefulset)

	// Verify the name and namespace matches
	require.Equal(t, namespaceName, statefulset.Namespace)

	//Verify InitContainer: copy-certs is created when enableOnDefaultAppServers is true
	expectedInitContainer := "copy-certs"
	require.Equal(t, expectedInitContainer, statefulset.Spec.Template.Spec.InitContainers[0].Name)

	//Verify Volumes are created for CACerts and CertSecrets and values are mapped to volume secrets
	expectedVolumeForCACerts := "ca-cert-secret"
	expectedVolumeForCertSecrets := "server-cert-secrets"
	expectedSecretForVolumeCACerts := "ca-secret"
	expectedSecretForVolumeCertSecrets := "marklogic-0-cert"
	expectedCRTForVolumeCertSecrets := "tls_0.crt"
	require.Equal(t, expectedVolumeForCACerts, statefulset.Spec.Template.Spec.Volumes[1].Name)
	require.Equal(t, expectedVolumeForCertSecrets, statefulset.Spec.Template.Spec.Volumes[2].Name)
	require.Equal(t, expectedSecretForVolumeCACerts, statefulset.Spec.Template.Spec.Volumes[1].Secret.SecretName)
	require.Equal(t, expectedSecretForVolumeCertSecrets, statefulset.Spec.Template.Spec.Volumes[2].Projected.Sources[0].Secret.Name)
	require.Equal(t, expectedCRTForVolumeCertSecrets, statefulset.Spec.Template.Spec.Volumes[2].Projected.Sources[0].Secret.Items[0].Path)
}

func TestChartTemplateTLSDisabled(t *testing.T) {

	// Path to the helm chart we will test
	helmChartPath, err := filepath.Abs("../../charts")
	releaseName := "marklogic"
	t.Log(helmChartPath, releaseName)
	require.NoError(t, err)

	// Set up the namespace; confirm that the template renders the expected value for the namespace.
	namespaceName := "marklogic-templ"
	t.Logf("Namespace: %s\n", namespaceName)

	// Setup the args for helm install
	options := &helm.Options{
		SetValues: map[string]string{
			"image.repository":              "progressofficial/marklogic-db",
			"image.tag":                     "latest",
			"persistence.enabled":           "false",
			"tls.enableOnDefaultAppServers": "false",
		},
		KubectlOptions: k8s.NewKubectlOptions("", "", namespaceName),
	}

	// render the tempate
	output := helm.RenderTemplate(t, options, helmChartPath, releaseName, []string{"templates/statefulset.yaml"})

	var statefulset appsv1.StatefulSet
	helm.UnmarshalK8SYaml(t, output, &statefulset)

	// Verify the name and namespace matches
	require.Equal(t, namespaceName, statefulset.Namespace)

	//Verify InitContainer is not created when enableOnDefaultAppServers is false
	numinit := len(statefulset.Spec.Template.Spec.InitContainers)
	require.Equal(t, 0, numinit)
}
