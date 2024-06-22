// Package testUtil contains utility functions for all the tests in this repo
package testUtil

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/gruntwork-io/terratest/modules/k8s"
)

// HelmUpgrade : testUtil function to upgrade helm chart for e2e tests
func HelmUpgrade(t *testing.T, helmUpgradeOptions *helm.Options, releaseName string, kubectlOpt *k8s.KubectlOptions, podList []string, oldChartVersion string) {

	// Path to the current helm chart(to be released) we will upgrade to
	helmChartPath, e := filepath.Abs("../../charts")
	if e != nil {
		t.Fatalf(e.Error())
	}

	t.Logf("Initial Helm Chart Version: %s", oldChartVersion)
	stsName := releaseName
	if strings.HasPrefix(oldChartVersion, "1.0") {
		stsName = releaseName + "-marklogic"
	}

	//delete statefulset to upgrade if initial helm chart version is 1.x.x
	if strings.HasPrefix(oldChartVersion, "1.") {
		t.Logf("====Deleting Statefulset: %s", stsName)
		k8s.RunKubectl(t, kubectlOpt, "delete", "statefulset", stsName)
	}

	//upgrade the helm chart
	helm.Upgrade(t, helmUpgradeOptions, helmChartPath, releaseName)

	if !strings.HasPrefix(oldChartVersion, "1.") {
		// delete one pod at a time to allow restart
		for _, pod := range podList {
			t.Logf("====Deleting %s pod\n", pod)
			k8s.RunKubectl(t, kubectlOpt, "delete", "pod", pod)
		}
	}
	// wait until all pods are in Ready status
	for _, pod := range podList {
		k8s.WaitUntilPodAvailable(t, kubectlOpt, pod, 10, 15*time.Second)
	}
}
