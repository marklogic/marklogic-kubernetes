#!/bin/bash

echo "=====installing kind cluster"
# kind create cluster --name="kind-test"

echo "=====loading marklogc images to clusters"
# kind load docker-image store/marklogicdb/marklogic-server:10.0-8.3-centos-1.0.0-ea3

echo "=====Deploying Marklogic to test"
# helm install marklogic-test ../charts

echo "=====Running tests"
go test -v ./test/e2e/...

echo "=====Delete kind cluster"
# kind delete cluster kind