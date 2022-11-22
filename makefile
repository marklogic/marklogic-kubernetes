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

#***************************************************************************
# EKS Deploy
#***************************************************************************
## Deploy Cluster to EKS
## EKS Cluster configuration can be changed in eks-cluster.yaml file in eks folder
## prerequisites
## AWS CLI must be configured with admin credentials
## To deploy marklogic via helm charts to the EKS cluster you must
## Configure kubectl to work with EKS. See documentation to make sure it is setup correctly
## Example command: aws eks update-kubeconfig --region $(region) --name $(clusterName)
## https://docs.aws.amazon.com/eks/latest/userguide/create-kubeconfig.html
## Once this is complete helm charts can be deployed via the helm install command
## Options:
## * [region] required. region to deploy EKS cluster to.
## * [clusterName] required. Must be the same name as cluster name in eks-cluster.yaml.
## * [accountNumber] required. Account number to use service account role from.
.PHONY: eks-deploy
eks-deploy:
	eksctl create cluster -f eks/eks-cluster.yaml
	eksctl utils associate-iam-oidc-provider --region=$(region) --cluster $(clusterName) --approve
	eksctl create iamserviceaccount --name ebs-csi-controller-sa \
		--namespace kube-system --cluster $(clusterName) \
		--attach-policy-arn arn:aws:iam::aws:policy/service-role/AmazonEBSCSIDriverPolicy \
		--approve --role-only --role-name $(clusterName)_sa \
		--override-existing-serviceaccounts
	eksctl create addon --name aws-ebs-csi-driver --cluster $(clusterName) --service-account-role-arn arn:aws:iam::$(accountNumber):role/$(clusterName)_sa --force

#***************************************************************************
# EKS delete
#***************************************************************************
## Tear down an EKS Cluster deployed via eks-deploy command. 
## Options (minus account number) and eks-cluster.yaml file must be exactly the same as those used to deploy EKS cluster initially 
## AWS CLI must be configured with admin credentials
## Options:
## * [region] required. region to remove EKS cluster from.
## * [clusterName] required. Name of cluster to remove from EKS.
.PHONY: eks-delete
eks-delete:
	eksctl delete addon --name aws-ebs-csi-driver --region=$(region) --cluster $(clusterName)
	eksctl delete iamserviceaccount --name ebs-csi-controller-sa \
		--namespace kube-system  --region=$(region) --cluster $(clusterName)
	eksctl delete cluster  --region=$(region) --name $(clusterName)

## ---------- Testing Tasks ----------

#***************************************************************************
# e2e-test
#***************************************************************************
## Run all end to end tests
## Options:
## * [dockerImage] optional. default is marklogicdb/marklogic-db:latest. Example: dockerImage=marklogic-centos/marklogic-server-centos:10-internal
## * [saveOutput] optional. Save the output to a xml file. Example: saveOutput=true
.PHONY: e2e-test
e2e-test: prepare
	@echo "=====Installing minikube cluster"
	minikube start --driver=docker -n=1

	@echo "=====Loading marklogc image $(dockerImage) to minikube cluster"
	minikube image load $(dockerImage)

	@echo "=====Running e2e tests"
	$(if $(saveOutput),gotestsum --junitfile test/test_results/e2e-tests.xml ./test/e2e/... -count=1 -timeout 30m, go test -v -count=1 ./test/e2e/...) 

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
## * [saveOutput] optional. Save the output to a xml file. Example: saveOutput=true
.PHONY: test
test: template-test e2e-test