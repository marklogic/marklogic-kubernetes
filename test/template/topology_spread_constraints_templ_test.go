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

func TestChartTemplateTopologySpreadConstraintClass(t *testing.T) {

	// Path to the helm chart we will test
	helmChartPath, err := filepath.Abs("../../charts")
	releaseName := "topology"
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

	// Verify the topologySpreadConstraint rule is set
	expectedLabelSelector := map[string]string{
		"app.kubernetes.io/name": "marklogic",
	}
	expectedMaxSkewValue := 1
	expectedHostTopologyKey := "kubernetes.io/hostname"
	expectedHostScheduleCondition := "DoNotSchedule"
	expectedZoneTopologyKey := "topology.kubernetes.io/zone"
	expectedZoneScheduleCondition := "ScheduleAnyway"

	topologySpreadConstraintsRule := statefulset.Spec.Template.Spec.TopologySpreadConstraints

	hostMaxSkew := int(topologySpreadConstraintsRule[0].MaxSkew)
	hostTopologyKey := topologySpreadConstraintsRule[0].TopologyKey
	hostScheduleCondition := topologySpreadConstraintsRule[0].WhenUnsatisfiable
	hostLabelSelector := topologySpreadConstraintsRule[0].LabelSelector.MatchLabels

	zoneMaxSkew := int(topologySpreadConstraintsRule[1].MaxSkew)
	zoneTopologyKey := topologySpreadConstraintsRule[1].TopologyKey
	zoneScheduleCondition := topologySpreadConstraintsRule[1].WhenUnsatisfiable
	zoneLabelSelector := topologySpreadConstraintsRule[1].LabelSelector.MatchLabels

	require.Equal(t, hostMaxSkew, expectedMaxSkewValue)
	require.Equal(t, hostTopologyKey, expectedHostTopologyKey)
	require.Equal(t, string(hostScheduleCondition), expectedHostScheduleCondition)
	require.Equal(t, hostLabelSelector, expectedLabelSelector)

	require.Equal(t, zoneMaxSkew, expectedMaxSkewValue)
	require.Equal(t, zoneTopologyKey, expectedZoneTopologyKey)
	require.Equal(t, string(zoneScheduleCondition), expectedZoneScheduleCondition)
	require.Equal(t, zoneLabelSelector, expectedLabelSelector)
}
