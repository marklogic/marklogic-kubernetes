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

func TestChartTemplatePodAntiAffinityClass(t *testing.T) {

	// Path to the helm chart we will test
	helmChartPath, err := filepath.Abs("../../charts")
	releaseName := "marklogic-pod-antiaffinity-test"
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
		},
		KubectlOptions: k8s.NewKubectlOptions("", "", namespaceName),
	}

	// render the tempate
	output := helm.RenderTemplate(t, options, helmChartPath, releaseName, []string{"templates/statefulset.yaml"})

	var statefulset appsv1.StatefulSet
	helm.UnmarshalK8SYaml(t, output, &statefulset)

	// Verify the name and namespace matches
	require.Equal(t, namespaceName, statefulset.Namespace)

	// Verify the pod anti affinity rule is set
	expectedLabelSelectorKey := "app.kubernetes.io/name"
	expectedLabelSelectorValue := "marklogic"
	expectedTopologyKey := "kubernetes.io/hostname"
	statefulSetAffinityRule := statefulset.Spec.Template.Spec.Affinity.PodAntiAffinity
	affinityPreferredRule := statefulSetAffinityRule.PreferredDuringSchedulingIgnoredDuringExecution
	actualLabelSelectorKey := affinityPreferredRule[0].PodAffinityTerm.LabelSelector.MatchExpressions[0].Key
	actualLabelSelectorValue := affinityPreferredRule[0].PodAffinityTerm.LabelSelector.MatchExpressions[0].Values[0]
	actualTopologyKey := affinityPreferredRule[0].PodAffinityTerm.TopologyKey
	require.Equal(t, actualLabelSelectorKey, expectedLabelSelectorKey)
	require.Equal(t, actualLabelSelectorValue, expectedLabelSelectorValue)
	require.Equal(t, actualTopologyKey, expectedTopologyKey)
}
