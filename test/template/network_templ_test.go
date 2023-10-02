package template_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	netv1 "k8s.io/api/networking/v1"

	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/gruntwork-io/terratest/modules/random"
)

func TestChartTemplateNetworkPolicyEnabled(t *testing.T) {

	// Path to the helm chart we will test
	helmChartPath, err := filepath.Abs("../../charts")
	releaseName := "marklogic-network-test"
	t.Log(helmChartPath, releaseName)
	require.NoError(t, err)

	// Set up the namespace; confirm that the template renders the expected value for the namespace.
	namespaceName := "ml-" + strings.ToLower(random.UniqueId()) + "-network-policy"
	t.Logf("Namespace: %s\n", namespaceName)

	// Setup the args for helm install
	options := &helm.Options{
		SetValues: map[string]string{
			"image.repository":      "marklogicdb/marklogic-db",
			"image.tag":             "latest",
			"persistence.enabled":   "false",
			"networkPolicy.enabled": "true",
		},
		KubectlOptions: k8s.NewKubectlOptions("", "", namespaceName),
	}
	output := helm.RenderTemplate(t, options, helmChartPath, releaseName, []string{"templates/networkPolicy.yaml"})

	var networkpolicy netv1.NetworkPolicy
	helm.UnmarshalK8SYaml(t, output, &networkpolicy)

	// Verify the name and namespace matches
	require.Equal(t, namespaceName, networkpolicy.Namespace)

	// Verify the network policy type matches
	networkPolicies := networkpolicy.Spec
	expectedPolicyTypes := "Ingress"
	require.Equal(t, string(networkPolicies.PolicyTypes[0]), expectedPolicyTypes)
}
