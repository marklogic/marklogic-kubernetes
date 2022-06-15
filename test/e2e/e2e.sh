#!/bin/bash

echo "=====Installing minikube cluster"
minikube start --driver=docker -n=1

echo "=====Loading marklogc images to minikube cluster"
minikube image load marklogic-centos/marklogic-server-centos:10-internal

echo "=====Running tests"
go test -v -count=1 ./test/e2e/...

echo "=====Delete minikube cluster"
minikube delete