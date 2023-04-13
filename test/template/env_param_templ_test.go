package template_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"

	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/gruntwork-io/terratest/modules/random"
)

func TestChartTemplateEnableConvertersEnabled(t *testing.T) {

	// Path to the helm chart we will test
	helmChartPath, err := filepath.Abs("../../charts")
	releaseName := "marklogic-container-env-test"
	t.Log(helmChartPath, releaseName)
	require.NoError(t, err)

	// Set up the namespace; confirm that the template renders the expected value for the namespace.
	namespaceName := "marklogic-" + strings.ToLower(random.UniqueId())
	t.Logf("Namespace: %s\n", namespaceName)

	// Setup the args for helm install
	options := &helm.Options{
		SetValues: map[string]string{
			"image.repository":    "marklogicdb/marklogic-db",
			"image.tag":           "latest",
			"persistence.enabled": "false",
			"enableConverters":    "true",
		},
		KubectlOptions: k8s.NewKubectlOptions("", "", namespaceName),
	}

	// render the tempate
	output := helm.RenderTemplate(t, options, helmChartPath, releaseName, []string{"templates/statefulset.yaml"})

	var statefulset appsv1.StatefulSet
	helm.UnmarshalK8SYaml(t, output, &statefulset)

	// Verify the name and namespace matches
	require.Equal(t, namespaceName, statefulset.Namespace)

	// Verify the value of enableConverters parameter
	expectedEnableConverters := "true"
	actualEnableConverters := statefulset.Spec.Template.Spec.Containers[0].Env[3].Value
	require.Equal(t, expectedEnableConverters, actualEnableConverters)
}

func TestChartTemplateEnableConvertersDisabled(t *testing.T) {

	// Path to the helm chart we will test
	helmChartPath, err := filepath.Abs("../../charts")
	releaseName := "marklogic-container-env-test"
	t.Log(helmChartPath, releaseName)
	require.NoError(t, err)

	// Set up the namespace; confirm that the template renders the expected value for the namespace.
	namespaceName := "marklogic-" + strings.ToLower(random.UniqueId())
	t.Logf("Namespace: %s\n", namespaceName)

	// Setup the args for helm install
	options := &helm.Options{
		SetValues: map[string]string{
			"image.repository":    "marklogicdb/marklogic-db",
			"image.tag":           "latest",
			"persistence.enabled": "false",
			"enableConverters":    "false",
		},
		KubectlOptions: k8s.NewKubectlOptions("", "", namespaceName),
	}

	// render the tempate
	output := helm.RenderTemplate(t, options, helmChartPath, releaseName, []string{"templates/statefulset.yaml"})

	var statefulset appsv1.StatefulSet
	helm.UnmarshalK8SYaml(t, output, &statefulset)

	// Verify the name and namespace matches
	require.Equal(t, namespaceName, statefulset.Namespace)

	// Verify the value of enableConverters parameter
	expectedEnableConverters := "false"
	actualEnableConverters := statefulset.Spec.Template.Spec.Containers[0].Env[3].Value
	require.Equal(t, actualEnableConverters, expectedEnableConverters)

}

func TestChartTemplateLicenseValues(t *testing.T) {

	// Path to the helm chart we will test
	helmChartPath, err := filepath.Abs("../../charts")
	releaseName := "marklogic-container-env-test"
	t.Log(helmChartPath, releaseName)
	require.NoError(t, err)

	// Set up the namespace; confirm that the template renders the expected value for the namespace.
	namespaceName := "marklogic-" + strings.ToLower(random.UniqueId())
	t.Logf("Namespace: %s\n", namespaceName)

	// Setup the args for helm install
	options := &helm.Options{
		SetValues: map[string]string{
			"image.repository":    "marklogicdb/marklogic-db",
			"image.tag":           "latest",
			"persistence.enabled": "false",
			"license.key":         "Test License Key",
			"license.licensee":    "Test Licensee",
		},
		KubectlOptions: k8s.NewKubectlOptions("", "", namespaceName),
	}

	// render the tempate
	output := helm.RenderTemplate(t, options, helmChartPath, releaseName, []string{"templates/statefulset.yaml"})

	var statefulset appsv1.StatefulSet
	helm.UnmarshalK8SYaml(t, output, &statefulset)

	// Verify the name and namespace matches
	require.Equal(t, namespaceName, statefulset.Namespace)

	// Verify the value of licenseKey and licensee parameters
	expectedLicenseKey := "Test License Key"
	expectedLicensee := "Test Licensee"
	actualLicenseKey := statefulset.Spec.Template.Spec.Containers[0].Env[4].Value
	actualLicensee := statefulset.Spec.Template.Spec.Containers[0].Env[5].Value

	require.Equal(t, actualLicenseKey, expectedLicenseKey)
	require.Equal(t, actualLicensee, expectedLicensee)
}

func TestChartTemplateAuthRealmValue(t *testing.T) {

	// Path to the helm chart we will test
	helmChartPath, err := filepath.Abs("../../charts")
	releaseName := "marklogic-container-env-test"
	t.Log(helmChartPath, releaseName)
	require.NoError(t, err)

	// Set up the namespace; confirm that the template renders the expected value for the namespace.
	namespaceName := "marklogic-" + strings.ToLower(random.UniqueId())
	t.Logf("Namespace: %s\n", namespaceName)

	// Setup the args for helm install
	options := &helm.Options{
		SetValues: map[string]string{
			"image.repository":    "marklogicdb/marklogic-db",
			"image.tag":           "latest",
			"persistence.enabled": "false",
			"auth.realm":          "public",
		},
		KubectlOptions: k8s.NewKubectlOptions("", "", namespaceName),
	}

	// render the tempate
	output := helm.RenderTemplate(t, options, helmChartPath, releaseName, []string{"templates/statefulset.yaml"})

	var statefulset appsv1.StatefulSet
	helm.UnmarshalK8SYaml(t, output, &statefulset)

	// Verify the name and namespace matches
	require.Equal(t, namespaceName, statefulset.Namespace)

	// Verify the value of security realm
	expectedRealm := "public"
	actualRealm := statefulset.Spec.Template.Spec.Containers[0].Env[6].Value
	require.Equal(t, actualRealm, expectedRealm)
}
