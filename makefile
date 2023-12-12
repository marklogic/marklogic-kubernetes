dockerImage?=ml-docker-db-dev-tierpoint.bed-artifactory.bedford.progress.com/marklogic/marklogic-server-centos:11.1.20230522-centos-1.0.2
prevDockerImage?=ml-docker-db-dev-tierpoint.bed-artifactory.bedford.progress.com/marklogic/marklogic-server-centos:10.0-20230522-centos-1.0.2
kubernetesVersion?=v1.25.8
minikubeMemory?=10gb
## System requirement:
## - Go 
## 		- gotestsum (if you want to enable saveOutput for testing commands)
## 		- golangci-lint
## - Helm 
## - Minikube
## - Docker
## - GNU Make >= 3.8.2 (preferrably >=4.2.1)

#***************************************************************************
# help
#***************************************************************************
## Get this help text
.PHONY: help
help:
	@printf "Usage\n";

	@awk '{ \
			if ($$0 ~ /^.PHONY: [a-zA-Z\-\_0-9]+$$/) { \
				helpCommand = substr($$0, index($$0, ":") + 2); \
				if (helpMessage) { \
					printf "\033[36m%-20s\033[0m %s\n", \
						helpCommand, helpMessage; \
					helpMessage = ""; \
				} \
			} else if ($$0 ~ /^[a-zA-Z\-\_0-9.]+:/) { \
				helpCommand = substr($$0, 0, index($$0, ":")); \
				if (helpMessage) { \
					printf "\033[36m%-20s\033[0m %s\n", \
						helpCommand, helpMessage; \
					helpMessage = ""; \
				} \
			} else if ($$0 ~ /^##/) { \
				if (helpMessage) { \
					helpMessage = helpMessage"\n                     "substr($$0, 3); \
				} else { \
					helpMessage = substr($$0, 3); \
				} \
			} else { \
				if (helpMessage) { \
					print "\n                     "helpMessage"\n" \
				} \
				helpMessage = ""; \
			} \
		}' \
		$(MAKEFILE_LIST)

## ---------- Development Tasks ----------

#***************************************************************************
# prepare
#***************************************************************************
## Install dependencies 
.PHONY: prepare
prepare:
	go mod tidy

#***************************************************************************
# lint
#***************************************************************************
## Lint the code
## Options:
## * [saveOutput] optional. Save the output to a text file. Example: saveOutput=true
## * [path] optional. path to golangci-lint executable. Example: path=/home/builder/go/bin/
.PHONY: lint
lint:
	@echo "> Linting helm charts....."
	helm lint --with-subcharts charts/ $(if $(saveOutput),> helm-lint-output.txt,)

	@echo "> Linting all tests....."
	golangci-lint run --timeout=5m $(if $(saveOutput),> test-lint-output.txt,)

## ---------- Testing Tasks ----------

#***************************************************************************
# e2e-test
#***************************************************************************
## Run all end to end tests
## Options:
## * [dockerImage] optional. default is marklogicdb/marklogic-db:latest. Example: dockerImage=marklogic-centos/marklogic-server-centos:10-internal
## * [prevDockerImage] optional. used for marklogic upgrade tests
## * [kubernetesVersion] optional. Default is v1.25.8. Used for testing kubernetes version compatibility
## * [saveOutput] optional. Save the output to a xml file. Example: saveOutput=true
.PHONY: e2e-test
e2e-test: prepare
	@echo "=====Delete if there are existing minikube cluster"
	minikube delete

	@echo "=====Installing minikube cluster"
	minikube start --driver=docker --kubernetes-version=$(kubernetesVersion) -n=1 --memory=$(minikubeMemory) --cpus=2


	@echo "=====Loading marklogc image $(dockerImage) to minikube cluster"
	minikube image load $(dockerImage)

	@echo "=====Loading marklogc image $(prevDockerImage) to minikube cluster"
	minikube image load $(prevDockerImage)

	@echo "=====Running e2e tests"
	$(if $(saveOutput),gotestsum --junitfile test/test_results/e2e-tests.xml ./test/e2e/... -count=1 -timeout 45m, go test -v -count=1 -timeout 45m ./test/e2e/...) 

	@echo "=====Delete minikube cluster"
	minikube delete

#***************************************************************************
# hc-test
#***************************************************************************
## Run all HC tests
.PHONY: hc-test
hc-test: 
 	
	@echo "=====Delete if there are existing minikube cluster"
	minikube delete

	@echo "=====Installing minikube cluster"
	minikube start --driver=docker --kubernetes-version=$(kubernetesVersion) -n=1 --memory=$(minikubeMemory) --cpus=2


	@echo "=====Loading marklogc image $(dockerImage) to minikube cluster"
	minikube image load $(dockerImage)

	@echo "=====Deploy helm with a single MarkLogic node"
	helm install hc charts --set auth.adminUsername=admin --set auth.adminPassword=admin --set persistence.enabled=false --wait
	kubectl wait -l statefulset.kubernetes.io/pod-name=hc-marklogic-0 --for=condition=ready pod --timeout=30m

	@echo "=====Clone Data Hub repository"
	rm -rf marklogic-data-hub; git clone https://github.com/marklogic/marklogic-data-hub

	@echo "=====Run HC tests with a shell script (~3 hours)"
	./test/hc_e2e.sh

	@echo "=====Finalize test report"
	mkdir -p ./test/test_results
	cp ./marklogic-data-hub/marklogic-data-hub-central/ui/e2e/results/* ./test/test_results/
	rm -rf marklogic-data-hub/*
	rm -rf marklogic-data-hub || ( ls -a marklogic-data-hub && exit 1 )

	@echo "=====Uninstall helm"
	helm uninstall hc

	@echo "=====Delete minikube cluster"
	minikube delete

#***************************************************************************
# template-test
#***************************************************************************
## Run all template tests
## * [saveOutput] optional. Save the output to a xml file. Example: saveOutput=true
.PHONY: template-test
template-test: prepare
	@echo "=====Running template tests"
	$(if $(saveOutput),gotestsum --junitfile test/test_results/testplate-tests.xml ./test/template/... -count=1, go test -v -count=1 ./test/template/...) 

#***************************************************************************
# test
#***************************************************************************
## Run all tests
## Options:
## * [dockerImage] optional. default is marklogicdb/marklogic-db:latest. Example: dockerImage=marklogic-centos/marklogic-server-centos:10-internal
## * [kubernetesVersion] optional. Default is v1.25.8. Used for testing kubernetes version compatibility
## * [saveOutput] optional. Save the output to a xml file. Example: saveOutput=true
.PHONY: test
test: template-test e2e-test
