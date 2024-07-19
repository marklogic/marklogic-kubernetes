// Package testUtil contains utility functions for all the tests in this repo
package testUtil

import (
	"strings"
	"testing"
	"time"

	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/gruntwork-io/terratest/modules/k8s"
)

// HelmInstall : testUtil function to install helm chart for e2e tests
func HelmInstall(t *testing.T, helmOptions *helm.Options, releaseName string, kubectlOpt *k8s.KubectlOptions, helmChartPath ...string) string {

	// Path to the helm chart we will test
	t.Logf("Helm chart path: %s", helmChartPath[0])

	//install the helm chart
	helm.Install(t, helmOptions, helmChartPath[0], releaseName)

	podName := releaseName + "-0"
	if strings.HasPrefix(helmOptions.Version, "1.0") {
		podName = releaseName + "-marklogic-0"
	}

	// wait until the pod is in Ready status
	k8s.WaitUntilPodAvailable(t, kubectlOpt, podName, 15, 15*time.Second)
	return podName
}
