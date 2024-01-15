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

func TestChartTemplateHugePagesConfig(t *testing.T) {

	// Path to the helm chart we will test
	helmChartPath, err := filepath.Abs("../../charts")
	releaseName := "hugepages"
	t.Log(helmChartPath, releaseName)
	require.NoError(t, err)

	// Set up the namespace; confirm that the template renders the expected value for the namespace.
	namespaceName := "ml-" + strings.ToLower(random.UniqueId())

	// Setup the args for helm install
	options := &helm.Options{
		SetValues: map[string]string{
			"image.repository":               "marklogicdb/marklogic-db",
			"image.tag":                      "latest",
			"persistence.enabled":            "true",
			"logCollection.enabled":          "true",
			"resources.limits.hugepages-2Mi": "1Gi",
		},
		KubectlOptions: k8s.NewKubectlOptions("", "", namespaceName),
	}

	// render the tempate
	output := helm.RenderTemplate(t, options, helmChartPath, releaseName, []string{"templates/statefulset.yaml"})

	var statefulset appsv1.StatefulSet
	helm.UnmarshalK8SYaml(t, output, &statefulset)

	// Verify the name and namespace matches
	require.Equal(t, namespaceName, statefulset.Namespace)

	expectedHugePages := "1Gi"

	statefulSetContainers := statefulset.Spec.Template.Spec.Containers
	resourceLimits := statefulSetContainers[0].Resources.Limits

	var actualHugePages string
	if value, exist := resourceLimits["hugepages-2Mi"]; exist {
		t.Log("ActualHugePages: ", value.String())
		actualHugePages = value.String()
	} else {
		t.Errorf("hugepages-2Mi not found")
	}
	// Verify the huge pages is configured
	require.Equal(t, len(statefulSetContainers), 2)
	require.Equal(t, actualHugePages, expectedHugePages)
}
