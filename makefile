dockerImage?=marklogic-centos/marklogic-server-centos:10-internal

## System requirement:
## - Go 
## - Helm 
## - Minikube
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
## * [Jenkins] optional. If we are running in Jenkins enviroment. Example: Jenkins=true
.PHONY: lint
lint:
	@echo "> Linting helm charts....."
	helm lint --with-subcharts charts/ $(if $(Jenkins), > helm-lint-output.txt,)

	@echo "> Linting tests....."
	docker run --rm -v $(pwd):/app -w /app golangci/golangci-lint:v1.49.0 golangci-lint run $(if $(Jenkins), > test-lint-output.txt,)

## ---------- Testing Tasks ----------

#***************************************************************************
# e2e-test
#***************************************************************************
## Run all end to end tests
## Options:
## * [dockerImage] optional. default is marklogicdb/marklogic-db:latest. Example: dockerImage=marklogic-centos/marklogic-server-centos:10-internal
.PHONY: e2e-test
e2e-test: prepare
	@echo "=====Installing minikube cluster"
	minikube start --driver=docker -n=1

	@echo "=====Loading marklogc image $(dockerImage) to minikube cluster"
	minikube image load $(dockerImage)

	@echo "=====Running tests"
	go test -v -count=1 ./test/e2e/...

	@echo "=====Delete minikube cluster"
	minikube delete

#***************************************************************************
# template-test
#***************************************************************************
## Run all template tests
.PHONY: template-test
template-test: prepare
	go test -v ./test/template/...

#***************************************************************************
# test
#***************************************************************************
## Run all tests
## Options:
## * [dockerImage] optional. default is marklogicdb/marklogic-db:latest. Example: dockerImage=marklogic-centos/marklogic-server-centos:10-internal
.PHONY: test
test: template-test e2e-test