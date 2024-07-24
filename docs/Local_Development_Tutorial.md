# Local Development Tutorial: Getting Started with MarkLogic in Kubernetes

* [Introduction](#Introduction)
* [Prerequisites](##Prerequisites)
* [Procedure](#Procedure)
* [Setting Up Minikube](##Setting-Up-Minikube)
* [Installing MarkLogic to Minikube](##Installing-a-Single-MarkLogic-Host-to-Minikube)
* [Installing Multiple MarkLogic Hosts to Minikube](##Installing-Multiple-MarkLogic-Hosts-to-Minikube)
* [Verifying the Installation](##Verifying-the-Installation)
* [Debugging](#Debugging)
* [Cleanup](#Cleanup)

# Introduction
This tutorial describes how to set up local Kubernetes development environment with Minikube and MarkLogic Server. It covers these tasks:
- Set up the prerequisites necessary for local development using MarkLogic Server in Kubernetes 
- How to run Minikube and load MarkLogic Server into the Kubernetes cluster 
- Access the MarkLogic Server cluster
- How to debug the Kubernetes environment 
- How to clean up your environment


## Prerequisites
The following steps assume you are running this tutorial from a desktop environment.
- [Docker](https://docs.docker.com/engine/install/): Pull the latest MarkLogic Server image from: https://hub.docker.com/r/progressofficial/marklogic-db
  ```sh
  docker pull progressofficial/marklogic-db:latest
  ```
- [Kubectl](https://kubernetes.io/docs/tasks/tools/):  Download and install this tool to assist with debugging in a Kubernetes environment.
- [Helm](https://helm.sh/docs/intro/install/):  Clone or download the chart repository: https://github.com/marklogic/marklogic-kubernetes
- [Minikube](https://k8s-docs.netlify.app/en/docs/tasks/tools/install-minikube/): Download the Minikube Kubernetes environment, which will host the MarkLogic Server applications.
- Browser: The latest version of a supported web browser. See the list here: [Web Browser](https://developer.marklogic.com/products/support-matrix/) 


# Procedure 
This section describes the procedure for setting up Minikube, installing MarkLogic Server, and verifying the installation. 


## Setting Up Minikube
First you will need to set up the Kubernetes control plane on your local machine. Minikube is a tool that makes it easy to set up a local Kubernetes enviornment.

- Start Minikube: `minikube start --driver=docker`

To verify the Minikube started correctly, use the Kubernetes command line tool, KubeCTL:  

```sh
kubectl get nodes
```
```
Expected Output: 
NAME       STATUS   ROLES                  AGE   VERSION
minikube   Ready    control-plane,master   1d    v1.23.3
```

##  Installing a Single MarkLogic Host to Minikube
- Push the image used for MarkLogic Server to the Minikube:
`minikube image load progressofficial/marklogic-db:latest`
- Add the Helm repository
  `helm repo add marklogic https://marklogic.github.io/marklogic-kubernetes/`  
  Additionally create a `values.yaml` file for your installation, like the one found in the repository under `/charts`: https://marklogic.github.io/marklogic-kubernetes/. The `values.yaml` file controls configuration for MarkLogic Server running in kubernetes. 
  Run `helm install RELEASE_NAME marklogic/marklogic --version=1.0.0-ea1 --values values.yaml` where the `RELEASE_NAME` can be any name you want to use to identify this deployment.
  For example: `helm install marklogic-local-dev-env marklogic/marklogic --version=1.0.0-ea1 --values values.yaml`
## Installing Multiple MarkLogic Hosts to Minikube
To create a MarkLogic cluster in Minikube, change the `replicaCount` in the `values.yaml` file to 3, or any other odd number to avoid the [split brain problem](https://help.marklogic.com/Knowledgebase/Article/View/119/0/start-up-quorum-and-forest-level-failover). Then follow the procedure outlined in the [Installing a Single MarkLogic Host to Minikube](##Installing-a-Single-MarkLogic-Host-to-Minikube) section. 

## Verifying the Installation
- After the installation is complete, verify the status of the deployment to the cluster with this command:
```sh
kubectl get pods --watch 
```
and wait for the following output: 
```
marklogic-0   1/1     Running   0          55s
```
This process may take a minute or two.

- You will need to port forward requests to the cluster in order to access MarkLogic Server from `localhost`. Port forwarding can be achieved with the following command:
```sh
kubectl port-forward marklogic-0 8001 8000 7997
```
 If you want to forward other ports, just append them to the command separated by a space.  
For example: 

```sh
kubectl port-forward marklogic-0 8001 8000 7997 7996 7888 1234 1337
```


- To complete this step, access your browser and navigate to `localhost:8001`. You should see the MarkLogic Server Admin interface.
If you are unable to see the MarkLogic Server Admin interface, see the [Debugging](#Debugging) section to gather more information about the cluster and potential errors. 

- See the [Cleanup](#Cleanup) section in order to teardown the cluster when you are finished. 

# Debugging
This Debugging section contains useful commands to help debug a Kubernetes cluster running MarkLogic Server. Additional information and commands can be found here: https://kubernetes.io/docs/tasks/debug-application-cluster/debug-running-pod/

This command provides additional information about the state of each pod in the cluster. 

```sh
kubectl describe pods
```
The command outputs a large amount of data and contains a lot of information. Pay attention to the events section near the bottom of the output to view the state of events, and what occurred during the startup of the pod.  
For Example:

```
...
Events:
  Type     Reason     Age   From               Message
  ----     ------     ----  ----               -------
  Normal   Scheduled  13m   default-scheduler  Successfully assigned default/marklogic-0 to minikube
  Normal   Pulled     13m   kubelet            Container image "progressofficial/marklogic-db:latest" already present on machine
  Normal   Created    13m   kubelet            Created container marklogic
  Normal   Started    13m   kubelet            Started container marklogic
  Warning  Unhealthy  13m   kubelet            Startup probe failed: ls: cannot access /var/opt/MarkLogic/ready: No such file or directory
```
**Note:** It is OK that the startup probe failed once. The probe will poll a few times. 

-----

Run the following command to see the logs for a specific pod:
```sh
kubectl logs {POD_NAME} 
```
The `{POD_NAME}` can be found with the `kubectl get pods` command. 

Example output: 

```
2022-03-28 16:53:00.127 Info: Memory 4% phys=5812 size=536(9%) rss=275(4%) anon=205(3%) file=9(0%) forest=853(14%) cache=1920(33%) registry=1(0%)
2022-03-28 16:54:01.630 Info: Merging 1 MB from /var/opt/MarkLogic/Forests/Security/00000002 to /var/opt/MarkLogic/Forests/Security/00000003, timestamp=16484858400268930
2022-03-28 16:54:01.657 Info: Merged 1 MB at 37 MB/sec to /var/opt/MarkLogic/Forests/Security/00000003
2022-03-28 16:54:04.155 Info: Deleted 2 MB at 1314 MB/sec /var/opt/MarkLogic/Forests/Security/00000002
```

-----

Run the following command to see the logs for a specific pod : 

```sh
kubectl exec -it {POD_NAME} -- /bin/bash
```
The `{POD_NAME}` can be found with the `kubectl get pods` command.

For example:
```sh
> kubectl exec -it marklogic-0 -- /bin/bash
[marklogic_user@marklogic-0 /]$ 
```

# Cleanup 
To cleanup a running kubernetes cluster, run the following commands:
- `helm uninstall <deployment_name>`   
  For Example:   
  `helm uninstall marklogic-local-dev-env`
- `minikube delete`
