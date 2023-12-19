package template_test

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	//appsv1 "k8s.io/api/apps/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/gruntwork-io/terratest/modules/k8s"
)

func TestTemplateTestHAproxyDisabled(t *testing.T) {
	helmChartPath, err := filepath.Abs("../../charts")
	releaseName := "haproxy"
	require.NoError(t, err)

	options := &helm.Options{
		SetValues: map[string]string{
			"haproxy.enabled": "false",
		},
		KubectlOptions: k8s.NewKubectlOptions("", "", ""),
	}

	_, err = helm.RenderTemplateE(t, options, helmChartPath, releaseName, []string{"charts/haproxy/templates/deployment.yaml"})
	require.NotNil(t, err)
	require.Contains(t, err.Error(), "could not find template charts/haproxy/templates/deployment.yaml")

	_, err = helm.RenderTemplateE(t, options, helmChartPath, releaseName, []string{"templates/configmap-haproxy.yaml"})
	require.NotNil(t, err)
	t.Log(err)
	require.Contains(t, err.Error(), "could not find template templates/configmap-haproxy.yaml")

}

func TestTemplateTestHAproxyDeployment(t *testing.T) {

	// Path to the helm chart we will test
	helmChartPath, err := filepath.Abs("../../charts")
	releaseName := "haproxy"
	require.NoError(t, err)

	{
		options := &helm.Options{
			SetValues: map[string]string{
				"haproxy.enabled": "true",
			},
			KubectlOptions: k8s.NewKubectlOptions("", "", ""),
		}

		var deployment appsv1.Deployment

		output, err := helm.RenderTemplateE(t, options, helmChartPath, releaseName, []string{"charts/haproxy/templates/deployment.yaml"})
		if err != nil {
			t.Log("Error when render HAProxy Deployment")
			t.Fatal(err)
		}
		require.Nil(t, err)
		helm.UnmarshalK8SYaml(t, output, &deployment)
		// test default setting for rollme annotation
		require.Contains(t, deployment.Spec.Template.Annotations, "rollme")
		require.EqualValues(t, *deployment.Spec.Replicas, 2)
	}

	{
		options := &helm.Options{
			SetValues: map[string]string{
				"haproxy.enabled":                    "true",
				"haproxy.restartWhenUpgrade.enabled": "false",
				"haproxy.replicaCount":               "1",
			},
			KubectlOptions: k8s.NewKubectlOptions("", "", ""),
		}

		var deployment appsv1.Deployment

		output, err := helm.RenderTemplateE(t, options, helmChartPath, releaseName, []string{"charts/haproxy/templates/deployment.yaml"})
		if err != nil {
			t.Log("Error when render HAProxy Deployment")
			t.Fatal(err)
		}
		require.Nil(t, err)
		helm.UnmarshalK8SYaml(t, output, &deployment)

		require.NotContains(t, deployment.Spec.Template.Annotations, "rollme")
		require.EqualValues(t, *deployment.Spec.Replicas, 1)
	}

}

func TestTemplateTestHAproxyService(t *testing.T) {

	helmChartPath, err := filepath.Abs("../../charts")
	releaseName := "haproxy"
	require.NoError(t, err)

	{
		options := &helm.Options{
			SetValues: map[string]string{
				"haproxy.enabled": "true",
			},
			KubectlOptions: k8s.NewKubectlOptions("", "", ""),
		}

		var service corev1.Service

		// render the service templete
		output, err := helm.RenderTemplateE(t, options, helmChartPath, releaseName, []string{"charts/haproxy/templates/service.yaml"})
		if err != nil {
			t.Log("Error when render HAProxy Service")
			t.Fatal(err)
		}
		require.Nil(t, err)
		helm.UnmarshalK8SYaml(t, output, &service)
		require.EqualValues(t, service.Spec.Type, "ClusterIP")
	}

	{
		options := &helm.Options{
			SetValues: map[string]string{
				"haproxy.enabled":                       "true",
				"haproxy.service.type":                  "LoadBalancer",
				"haproxy.service.externalTrafficPolicy": "Cluster",
			},
			KubectlOptions: k8s.NewKubectlOptions("", "", ""),
		}

		var service corev1.Service

		output, err := helm.RenderTemplateE(t, options, helmChartPath, releaseName, []string{"charts/haproxy/templates/service.yaml"})
		if err != nil {
			t.Log("Error when render HAProxy Service")
			t.Fatal(err)
		}
		require.Nil(t, err)
		helm.UnmarshalK8SYaml(t, output, &service)
		require.EqualValues(t, service.Spec.Type, "LoadBalancer")
		require.EqualValues(t, service.Spec.ExternalTrafficPolicy, "Cluster")
	}
}

func TestTemplateTestHAproxyConfigmap(t *testing.T) {

	helmChartPath, err := filepath.Abs("../../charts")
	releaseName := "haproxy"
	require.NoError(t, err)

	{
		options := &helm.Options{
			SetValues: map[string]string{
				"haproxy.enabled": "false",
			},
			KubectlOptions: k8s.NewKubectlOptions("", "", ""),
		}

		// render the service templete
		_, err := helm.RenderTemplateE(t, options, helmChartPath, releaseName, []string{"templates/configmap-haproxy.yaml"})
		require.NotNil(t, err)
		require.Contains(t, err.Error(), "could not find template templates/configmap-haproxy.yaml")

	}

	{
		options := &helm.Options{
			SetValues: map[string]string{
				"haproxy.enabled": "true",
			},
			KubectlOptions: k8s.NewKubectlOptions("", "", ""),
		}

		var configmap corev1.Service

		// render the service templete
		output, err := helm.RenderTemplateE(t, options, helmChartPath, releaseName, []string{"templates/configmap-haproxy.yaml"})
		if err != nil {
			t.Log("Error when render HAProxy Configmap")
			t.Fatal(err)
		}
		require.Nil(t, err)
		helm.UnmarshalK8SYaml(t, output, &configmap)

	}
}
