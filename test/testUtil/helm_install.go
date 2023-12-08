// Package testUtil contains utility functions for all the tests in this repo
package testUtil

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/gruntwork-io/terratest/modules/k8s"
)

// HelmInstall : testUtil function to install helm chart for e2e tests
func HelmInstall(t *testing.T, options map[string]string, releaseName string, kubectlOpt *k8s.KubectlOptions) string {

	// Path to the helm chart we will test
	helmChartPath, e := filepath.Abs("../../charts")
	if e != nil {
		t.Fatalf(e.Error())
	}

	helmOptions := &helm.Options{
		KubectlOptions: kubectlOpt,
		SetValues:      options,
	}

	helm.Install(t, helmOptions, helmChartPath, releaseName)

	podName := releaseName + "-0"
	// wait until the pod is in Ready status
	k8s.WaitUntilPodAvailable(t, kubectlOpt, podName, 10, 15*time.Second)
	return podName
}
