dockerImage?=marklogic-centos/marklogic-server-centos:10-internal

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
	$(if $(path),$(path),)golangci-lint run $(if $(saveOutput),> test-lint-output.txt,)

## ---------- Testing Tasks ----------

#***************************************************************************
# e2e-test
#***************************************************************************
## Run all end to end tests
## Options:
## * [dockerImage] optional. default is marklogicdb/marklogic-db:latest. Example: dockerImage=marklogic-centos/marklogic-server-centos:10-internal
## * [saveOutput] optional. Save the output to a xml file. Example: saveOutput=true
## * [jenkins] optional. Jenkins specific enviroment setting, Example: jenkins=true

.PHONY: e2e-test
e2e-test: prepare
	@echo "=====Installing minikube cluster"
	minikube start --driver=docker -n=1

	@echo "=====Loading marklogc image $(dockerImage) to minikube cluster"
	minikube image load $(dockerImage)

	@echo "=====Running e2e tests"
	cd test; $(if $(saveOutput),gotestsum --junitfile test_results/e2e-tests.xml ./e2e/... -count=1 -timeout 30m, go test -v -count=1 ./e2e/...) 

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
	cd test; $(if $(saveOutput),gotestsum --junitfile test_results/testplate-tests.xml ./template/... -count=1, go test -v -count=1 ./test/template/...) 

#***************************************************************************
# test
#***************************************************************************
## Run all tests
## Options:
## * [dockerImage] optional. default is marklogicdb/marklogic-db:latest. Example: dockerImage=marklogic-centos/marklogic-server-centos:10-internal
## * [saveOutput] optional. Save the output to a xml file. Example: saveOutput=true
.PHONY: test
test: template-test e2e-test