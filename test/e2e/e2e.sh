#!/bin/bash

echo "=====Delete minikube cluster"
minikube delete

echo "=====Installing minikube cluster"
minikube start --driver=docker -n=1

echo "=====Loading marklogc images to minikube cluster"
minikube image load marklogicdb/marklogic-db:10.0-9.1-centos-1.0.0-ea4

echo "=====Running tests"
go test -v -count=1 ./test/e2e/...
