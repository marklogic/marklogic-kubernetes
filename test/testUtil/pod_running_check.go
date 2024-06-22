// Package testUtil contains utility functions for all the tests in this repo
package testUtil

import (
	"strings"
	"testing"
	"time"

	"github.com/gruntwork-io/terratest/modules/k8s"
)

// WaitUntilPodRunning : testUtil function to check if pod is running
func WaitUntilPodRunning(t *testing.T, kubectlOpt *k8s.KubectlOptions, podName string, retries int, interval time.Duration) (string, error) {
	var err error
	var podOutput string
	// podRunning := false
	i := 1
	for i <= retries {
		podOutput, err = k8s.RunKubectlAndGetOutputE(t, kubectlOpt, "get", "pod", podName)
		if strings.Contains(podOutput, "Running") {
			t.Log(podName, " is Running")
			return "Running", err
		}
		time.Sleep(interval)
		i = i + 1
	}
	if err != nil {
		return "", err
	}
	return "Timedout waiting for Pod to be running", err
}
