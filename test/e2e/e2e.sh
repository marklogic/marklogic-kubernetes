#!/bin/bash

echo "=====Delete minikube cluster"
minikube delete

echo "=====Installing minikube cluster"
minikube start --driver=docker -n=1

echo "=====Loading marklogc images to minikube cluster"
minikube image load store/marklogicdb/marklogic-server:10.0-9-centos-1.0.0-ea4

echo "=====Running tests"
go test -v ./test/e2e/...
