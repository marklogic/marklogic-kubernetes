package template_test

import (
	"os"
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
	releaseName := "network"
	t.Log(helmChartPath, releaseName)
	require.NoError(t, err)

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

	// Set up the namespace; confirm that the template renders the expected value for the namespace.
	namespaceName := "ml-" + strings.ToLower(random.UniqueId()) + "-network-policy"
	t.Logf("Namespace: %s\n", namespaceName)

	// Setup the args for helm install using custom values.yaml file
	options := &helm.Options{
		ValuesFiles: []string{"../test_data/values/nwPolicy_templ_values.yaml"},
		SetValues: map[string]string{
			"image.repository": imageRepo,
			"image.tag":        imageTag,
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
	require.Equal(t, string(networkPolicies.PolicyTypes[0]), "Ingress")
	require.Equal(t, string(networkPolicies.PolicyTypes[1]), "Egress")
}
