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
## * [saveOutput] optional. Save the output to a text file. Example: saveOutput=true
.PHONY: image-scan
image-scan:
	@rm -f helm_image.list dep-image-scan.txt
	@$(if $(saveOutput), > dep-image-scan.txt)
	@echo "=====Scan dependent Docker images in charts/values.yaml" $(if $(saveOutput), | tee -a dep-image-scan.txt,)
	set -e; \
	scanned_images_tracker_file="$$(mktemp)"; \
	scan_image() { \
	  img="$$1"; \
	  src_file="$$2"; \
	  if [ -z "$$img" ]; then \
	    echo "Warning: Empty image name provided from $$src_file. Skipping." $(if $(saveOutput), | tee -a dep-image-scan.txt,); \
	    return; \
	  fi; \
	  if grep -Fxq "$$img" "$$scanned_images_tracker_file"; then \
	    echo "= $$img (from $$src_file) - Already Processed" $(if $(saveOutput), | tee -a dep-image-scan.txt,); \
	    return; \
	  fi; \
	  echo "= Scanning $$img (from $$src_file)" $(if $(saveOutput), | tee -a dep-image-scan.txt,); \
	  if ! docker pull "$$img"; then \
	    echo "Error: Failed to pull Docker image $$img. Skipping scan." $(if $(saveOutput), | tee -a dep-image-scan.txt,); \
	    echo "$$img" >> "$$scanned_images_tracker_file"; \
	    return; \
	  fi; \
	  echo "$$img" >> "$$scanned_images_tracker_file"; \
	  printf "%s," "$${img}" >> helm_image.list ; \
	  grype_json_output=$$(docker run --rm -v /var/run/docker.sock:/var/run/docker.sock anchore/grype:latest --output json "$$img" 2>/dev/null); \
	  if [ -z "$$grype_json_output" ]; then \
	    echo "Warning: Grype produced no output for $$img. Command might have failed or image not found/supported by grype." $(if $(saveOutput), | tee -a dep-image-scan.txt,); \
	    echo $(if $(saveOutput), | tee -a dep-image-scan.txt,); \
	    return; \
	  fi; \
	  if ! echo "$$grype_json_output" | jq -e '.descriptor.name' > /dev/null; then \
	    echo "Warning: Grype output for $$img is not valid JSON or image metadata is missing. Output was:" $(if $(saveOutput), | tee -a dep-image-scan.txt,); \
	    echo "$$grype_json_output" $(if $(saveOutput), | tee -a dep-image-scan.txt,); \
	    echo $(if $(saveOutput), | tee -a dep-image-scan.txt,); \
	    return; \
	  fi; \
	  summary=$$(echo "$$grype_json_output" | jq -r '([.matches[]?.vulnerability.severity] // []) as $$all_severities | reduce ["Critical","High","Medium","Low","Negligible","Unknown"][] as $$sev ( {Critical:0,High:0,Medium:0,Low:0,Negligible:0,Unknown:0} ; .[$$sev] = ([$$all_severities[] | select(. == $$sev)] | length) ) | "Critical=\(.Critical) High=\(.High) Medium=\(.Medium) Low=\(.Low) Negligible=\(.Negligible) Unknown=\(.Unknown)"'); \
	  echo "Summary: $$summary" $(if $(saveOutput), | tee -a dep-image-scan.txt,); \
	  if echo "$$grype_json_output" | jq -e '.matches == null or (.matches | length == 0)' > /dev/null; then \
	    echo "No vulnerabilities found to tabulate for $$img." $(if $(saveOutput), | tee -a dep-image-scan.txt,); \
	  else \
	    scan_out_body=$$(echo "$$grype_json_output" | jq -r 'def sevorder: {Critical:0, High:1, Medium:2, Low:3, Negligible:4, Unknown:5}; [.matches[]? | {pkg: .artifact.name, ver: .artifact.version, cve: .vulnerability.id, sev: .vulnerability.severity}] | map(. + {sort_key: sevorder[.sev // "Unknown"]}) | sort_by(.sort_key) | .[] | [.pkg // "N/A", .ver // "N/A", .cve // "N/A", .sev // "N/A"] | @tsv'); \
	    if [ -n "$$scan_out_body" ]; then \
	      (echo "Package\tVersion\tCVE\tSeverity"; echo "$$scan_out_body") | column -t -s $$'\t' $(if $(saveOutput), | tee -a dep-image-scan.txt,); \
	    else \
	      echo "No vulnerability details to display for $$img (though summary reported counts)." $(if $(saveOutput), | tee -a dep-image-scan.txt,); \
	    fi; \
	  fi; \
	  echo $(if $(saveOutput), | tee -a dep-image-scan.txt,); \
	}; \
	util_image=$$(grep -A2 'utilContainer:' charts/values.yaml | grep 'image:' | sed 's/.*image:[[:space:]]*//g' | sed 's/"//g' | xargs); \
	scan_image "$$util_image" "charts/values.yaml"; \
	haproxy_image=$$(grep -A 3 '^haproxy:' charts/values.yaml | grep -A 1 '^\s*image:' | grep '^\s*repository:' | sed 's/.*repository:[[:space:]]*//g' | sed 's/"//g' | sed 's/#.*//g' | xargs); \
	haproxy_tag=$$(grep -A 4 '^haproxy:' charts/values.yaml | grep -A 2 '^\s*image:' | grep '^\s*tag:' | sed 's/.*tag:[[:space:]]*//g' | sed 's/"//g' | sed 's/{{.*}}/latest/' | sed 's/#.*//g' | xargs); \
	scan_image "$$haproxy_image:$$haproxy_tag" "charts/values.yaml";
	@# Remove trailing comma from helm_image.list if present
	@if [ -f helm_image.list ]; then \
		sed -i '' -e 's/,\s*$$//' helm_image.list; \
	fi
