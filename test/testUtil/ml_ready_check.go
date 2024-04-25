// Package testUtil contains utility functions for all the tests in this repo
package testUtil

import (
	"crypto/tls"
	"fmt"
	"testing"
	"time"

	http_helper "github.com/gruntwork-io/terratest/modules/http-helper"
	"github.com/gruntwork-io/terratest/modules/k8s"
)

// MLReadyCheck : testUtil function to check if MarkLogic is ready for e2e tests
func MLReadyCheck(t *testing.T, kubectlOpt *k8s.KubectlOptions, podName string, tlsConfig *tls.Config) (bool, error) {

	tunnel7997 := k8s.NewTunnel(kubectlOpt, k8s.ResourceTypePod, podName, 7997, 7997)
	defer tunnel7997.Close()
	tunnel7997.ForwardPort(t)
	endpoint7997 := fmt.Sprintf("http://%s/LATEST/healthcheck", tunnel7997.Endpoint())

	// verify if 7997 health check endpoint returns 200
	err := http_helper.HttpGetWithRetryWithCustomValidationE(
		t,
		endpoint7997,
		tlsConfig,
		10,
		15*time.Second,
		func(statusCode int, body string) bool {
			return statusCode == 200
		},
	)

	if err != nil {
		return false, err
	}
	return true, nil
}
