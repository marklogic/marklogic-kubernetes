dockerImage?=ml-docker-db-dev-tierpoint.bed-artifactory.bedford.progress.com/marklogic/marklogic-server-centos:11.1.20230522-centos-1.0.2
prevDockerImage?=ml-docker-db-dev-tierpoint.bed-artifactory.bedford.progress.com/marklogic/marklogic-server-centos:10.0-20230522-centos-1.0.2
kubernetesVersion?=v1.25.8
minikubeMemory?=10gb
testSelection?=...
## System requirement:
## - Go 
## 		- gotestsum (if you want to enable output saving for testing commands)
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
	helm lint --set allowLongHostnames=true --with-subcharts charts/ $(if $(saveOutput),> helm-lint-output.txt,)

	@echo "> Linting all tests....."
	golangci-lint run --timeout=5m $(if $(saveOutput),> test-lint-output.txt,)

## ---------- Testing Tasks ----------

#***************************************************************************
# e2e-test
#***************************************************************************
## Run all end to end tests
## Options:
## * [dockerImage] optional. default is progressofficial/marklogic-db:latest. Example: dockerImage=marklogic-centos/marklogic-server-centos:10-internal
## * [prevDockerImage] optional. used for marklogic upgrade tests
## * [kubernetesVersion] optional. Default is v1.25.8. Used for testing kubernetes version compatibility
## * [saveOutput] optional. Save the output to a xml file. Example: saveOutput=true
.PHONY: e2e-test
e2e-test: prepare
	@echo "=====Delete if there are existing minikube cluster"
	minikube delete --all --purge

	@echo "=====Installing minikube cluster"
	minikube start --driver=docker --kubernetes-version=$(kubernetesVersion) -n=1 --memory=$(minikubeMemory) --cpus=2

	@echo "=====Pull $(dockerImage) image for upgrade test"
	## This is only needed while we use minikube since the image is not accessible to go at runtime
	docker pull $(dockerImage)

	# Get env details for debugging
	kubectl get nodes
	kubectl -n kube-system get pods
	minikube version
	kubectl version
	go version
	docker version

	# Update security context in values for ubi image
ifneq ($(findstring rootless,$(dockerImage)),rootless)
	echo "=Updating security context in values for root image."
	sed -i 's/allowPrivilegeEscalation: false/allowPrivilegeEscalation: true/' charts/values.yaml
else
	echo "=Security context is not changed for rootless image."
endif

	@echo "=====Setting hugepages values to 0 for e2e tests"
	sudo sysctl -w vm.nr_hugepages=0

	@echo "=====Running e2e tests"
	$(if $(saveOutput),gotestsum --junitfile test/test_results/e2e-tests.xml ./test/e2e/$(testSelection) -count=1 -timeout 180m, go test -v -count=1 -timeout 180m ./test/e2e/$(testSelection))

	@echo "=====Setting hugepages value to 1280 for hugepages-e2e test"
	sudo sysctl -w vm.nr_hugepages=1280

	@echo "=====Restart minikube cluster"
	minikube stop
	minikube start

	@echo "=====Running hugepages e2e test"
	$(if $(saveOutput),gotestsum --junitfile test/test_results/hugePages-tests.xml ./test/hugePages/... -count=1 -timeout 70m, go test -v -count=1 -timeout 70m ./test/hugePages/...)

	@echo "=====Resetting hugepages value to 0"
	sudo sysctl -w vm.nr_hugepages=0

	@echo "=====Delete minikube cluster"
	minikube delete

#***************************************************************************
# hc-test
#***************************************************************************
## Run all HC tests
.PHONY: hc-test
hc-test: 
 	
	@echo "=====Delete if there are existing minikube cluster"
	minikube delete --all --purge

	@echo "=====Installing minikube cluster"
	minikube start --driver=docker --kubernetes-version=$(kubernetesVersion) -n=1 --memory=$(minikubeMemory) --cpus=2

	@echo "=====Deploy helm with a single MarkLogic node"
	helm install hc charts --set auth.adminUsername=admin --set auth.adminPassword=admin --set persistence.enabled=false --wait
	kubectl wait -l statefulset.kubernetes.io/pod-name=hc-0 --for=condition=ready pod --timeout=30m

	# Get env details for debugging
	kubectl get nodes
	kubectl -n kube-system get pods
	minikube version
	kubectl version
	go version
	docker version

	# Update security context in values for rootless image
ifeq ($(findstring rootless,$(dockerImage)),rootless)
	sed -i 's/allowPrivilegeEscalation: true/allowPrivilegeEscalation: false/' charts/values.yaml
endif

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
## * [dockerImage] optional. default is progressofficial/marklogic-db:latest. Example: dockerImage=marklogic-centos/marklogic-server-centos:10-internal
## * [kubernetesVersion] optional. Default is v1.25.8. Used for testing kubernetes version compatibility
## * [saveOutput] optional. Save the output to a xml file. Example: saveOutput=true
.PHONY: test
test: template-test e2e-test

#***************************************************************************
# test
#***************************************************************************
## Run upgrade in e2e tests
## Set following environment variables 
## [upgradeTest] to true. Use `export upgradeTest=true`
## [initialChartVersion] to a valid MarkLogic helm chart version for ex.: 1.1.2 to run upgrade tests. Use `export initialChartVersion=1.1.2`
.PHONY: upgrade-test
upgrade-test: prepare
	@echo "=====upgradeTest env var for upgrade tests"
	echo $(upgradeTest)

	@echo "=====initialChartVersion env var for upgrade tests"
	echo ${initialChartVersion}
	
	@echo "=====Running upgrades in e2e tests"
	make e2e-test
	
#***************************************************************************
# image-scan
#***************************************************************************
## Find and scan dependent Docker images for security vulnerabilities
## Options:
## * [saveOutput] optional. Save the output to a xml file. Example: saveOutput=true
.PHONY: image-scan
image-scan:

	@echo "=====Scan dependent Docker images in charts/values.yaml" $(if $(saveOutput), | tee -a dep-image-scan.txt,)
	@for depImage in $(shell grep -E "^\s*\bimage:\s+(.*)" charts/values.yaml | sed 's/image: //g' | sed 's/"//g'); do\
		echo "= $${depImage}:" $(if $(saveOutput), | tee -a dep-image-scan.txt,) ; \
		docker run --rm -v /var/run/docker.sock:/var/run/docker.sock anchore/grype:latest --output json $${depImage} | jq -r '[(.matches[] | [.artifact.name, .artifact.version, .vulnerability.id, .vulnerability.severity])] | .[] | @tsv' | sort -k4 | column -t $(if $(saveOutput), | tee -a dep-image-scan.txt,);\
		echo $(if $(saveOutput), | tee -a dep-image-scan.txt,) ;\
	done
