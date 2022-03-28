# Local Development Tutorial: Getting Started with Kubernetes in MarkLogic 

# Table of contents
* [Introduction](#Introduction)
* [Prerequisites](##Prerequisites)
* [Procedure](#Procedure)
* [Setting Up minikube](##Setting-Up-minikube)
* [Installing MarkLogic to minikube](##Installing-a-Single-MarkLogic-Host-to-Minikube)
* [Verifying the Installation](##Verifying-the-Installation)
* [Debugging](#Debugging)
* [Cleanup](#Cleanup)

# Introduction
This tutorial serves as a guide for local development setup with Minikube and MarkLogic. These tasks are covered in this guild:
- Setting up the necessary prerequisites for local MarkLogic in Kubernetes development 
- How to run Minikube and load MarkLogic into the cluster 
- How to Access the MarkLogic cluster
- How to Debug the environment 
- How to Clean up


## Prerequisites
The following assumes you are running this tutorial from a desktop environment, mobile environments will likely experience problems and may not work
- [Docker](https://docs.docker.com/engine/install/)
  -- Subscribe and pulldown the latest image from: https://hub.docker.com/_/marklogic
  ```sh
  # Something similar to this, with the latest version tag, which can be found on the dockerhub link above
  docker pull store/marklogicdb/marklogic-server:10.0-9-centos-1.0.0-ea4 
  ```
- [KubeCTL](https://kubernetes.io/docs/tasks/tools/)
- [HELM](https://helm.sh/docs/intro/install/)
  -- Clone or download the chart repository: https://github.com/marklogic/marklogic-kubernetes
- [Minikube](https://k8s-docs.netlify.app/en/docs/tasks/tools/install-minikube/)
- If running on Mac OSX: [Virtual Box](https://www.virtualbox.org/)
- A supported [Web Browser](https://developer.marklogic.com/products/support-matrix/) 


# Procedure 
Below is the procedure for setting up Minikube, installing MarkLogic, and verifying the installation. 

## Setting Up minikube
- Start minikube: `minikube start --driver=virtualbox`
  -- If running in Linux: `minikube start --driver=docker`
To verify the minikube started correctly run the following command: 
```sh
kubectl get nodes
```
```
Expected Output: 
NAME       STATUS   ROLES                  AGE   VERSION
minikube   Ready    control-plane,master   1d    v1.23.3
```

- Enable Addons: `minikube addons enable ingress` for ingress
##  Installing a Single MarkLogic Host to minikube
- Push the image used for ML to the VM: `minikube image load store/marklogicdb/marklogic-server:10.0-9-centos-1.0.0-ea4`
  -- The above Image ID: `store/marklogicdb/marklogic-server:10.0-9-centos-1.0.0-ea4` is whatever the latest image is, to find the latest id go to https://hub.docker.com/_/marklogic
- Navigate to where you downloaded or cloned the MarkLogic helm repository 
  -- Verify the image loaded to minikube above matches the `repository` and `tag` in the `values.yaml` 
  ```YAML
  image:
    repository: store/marklogicdb/marklogic-server
    tag: 10.0-9-centos-1.0.0-ea4
  ```
  -- Navigate to the `/charts` folder
  -- Run `helm install RELEASE_NAME .` Where the `RELEASE_NAME` can be anything you want to identify this deployment EX: `helm install marklogic-local-dev-env .`
## Installing Multiple MarkLogic Hosts to Minikube
TODO

## Verifying the Installation
- After the installation, verify the status of the deployment to the cluster with:
```sh
kubectl get pods --watch 
```
and wait for the following output: 
```
marklogic-0   1/1     Running   0          55s
```
It may take a minute or two.

- Port forward requests to the cluster in order to access it from `localhost` with the following command:
```sh
kubectl port-forward marklogic-0 8001 8000 7997
```
 --  If you want to forward other ports just append them to the command separated by a space

- Finally access your browser and navigate to `localhost:8001` and you should see the MarkLogic Server Admin Interface
  -- If you're unable to see the MarkLogic Server Admin Interface interface proceed to the debugging section to gather more information on the cluster and potential errors. 

- Proceed to the [Cleanup](##Cleanup) section in order to teardown the cluster when finished 

# Debugging
The Debugging section contains useful commands to help debug a Kubernetes cluster running MarkLogic Server. Additional information and commands can be found here: https://kubernetes.io/docs/tasks/debug-application-cluster/debug-running-pod/

The below command gives additional information on the state of each pod in a cluster

```sh
kubectl describe pods
```
The output of this command is large and contains a lot of information pay attention to the bottom events section to see the state of events and what occurred during the startup of the pod. 
EX: 

```
...
Events:
  Type     Reason     Age   From               Message
  ----     ------     ----  ----               -------
  Normal   Scheduled  13m   default-scheduler  Successfully assigned default/marklogic-0 to minikube
  Normal   Pulled     13m   kubelet            Container image "store/marklogicdb/marklogic-server:10.0-8.3-centos-1.0.0-ea3" already present on machine
  Normal   Created    13m   kubelet            Created container marklogic
  Normal   Started    13m   kubelet            Started container marklogic
  Warning  Unhealthy  13m   kubelet            Startup probe failed: ls: cannot access /var/opt/MarkLogic/ready: No such file or directory
```
Note here it's ok that the startup probe failed once, the probe will poll a few times. 

-----

To get the logs on a pod run the following:
```sh
kubectl logs {POD_NAME} 
```
The `{POD_NAME}` can be found with the `kubectl get pods` command. 

Example Output: 

```
2022-03-28 16:53:00.127 Info: Memory 4% phys=5812 size=536(9%) rss=275(4%) anon=205(3%) file=9(0%) forest=853(14%) cache=1920(33%) registry=1(0%)
2022-03-28 16:54:01.630 Info: Merging 1 MB from /var/opt/MarkLogic/Forests/Security/00000002 to /var/opt/MarkLogic/Forests/Security/00000003, timestamp=16484858400268930
2022-03-28 16:54:01.657 Info: Merged 1 MB at 37 MB/sec to /var/opt/MarkLogic/Forests/Security/00000003
2022-03-28 16:54:04.155 Info: Deleted 2 MB at 1314 MB/sec /var/opt/MarkLogic/Forests/Security/00000002
```

-----

To get local access to a pod you can run the following: 

```sh
kubectl exec --stdin --tty {POD_NAME} -- /bin/bash
```
The `{POD_NAME}` can be found with the `kubectl get pods` command

EX:
```sh
> kubectl exec --stdin --tty marklogic-0 -- /bin/bash
[marklogic_user@marklogic-0 /]$ 
```

# Cleanup 
To cleanup a running cluster run the following commands
- `helm uninstall <deployment_name>` EX: `helm uninstall marklogic-local-dev-env`
- `minikube delete`


