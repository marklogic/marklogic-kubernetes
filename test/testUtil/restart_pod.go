// Package testUtil contains utility functions for all the tests in this repo
package testUtil

import (
	"crypto/tls"
	"testing"
	"time"

	"github.com/gruntwork-io/terratest/modules/k8s"
)

// RestartPodAndVerify : testUtil function to restart pods and verify its ready and healthy for e2e tests
func RestartPodAndVerify(t *testing.T, delAtOnce bool, podList []string, namespaceName string, kubectlOpt *k8s.KubectlOptions, tlsConfig *tls.Config) {

	if delAtOnce {
		// delete all pods at once to allow restart
		t.Logf("Deleting below pods at once in the %s namespace\n", namespaceName)
		output, err := k8s.RunKubectlAndGetOutputE(t, kubectlOpt, "get", "pods", "--namespace", namespaceName)
		t.Log(output)
		if err != nil {
			t.Logf(err.Error())
		}
		k8s.RunKubectl(t, kubectlOpt, "delete", "--all", "pod", "--namespace", namespaceName)
	} else {
		// delete one pod at a time to allow restart
		for _, pod := range podList {
			t.Logf("Deleting %s pod\n", pod)
			k8s.RunKubectl(t, kubectlOpt, "delete", "pod", pod)
		}
	}

	// wait until the pod is in Ready status and MarkLogic server is ready
	for _, pod := range podList {
		k8s.WaitUntilPodAvailable(t, kubectlOpt, pod, 15, 15*time.Second)
		_, err := MLReadyCheck(t, kubectlOpt, pod, tlsConfig)
		if err != nil {
			t.Fatal("MarkLogic failed to start")
		}
	}
}
