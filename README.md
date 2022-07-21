# MarkLogic Kubernetes Helm Chart

- [Introduction](#introduction)
- [Prerequisites](#prerequisites)
  - [Set up the required tools](#set-up-the-required-tools)
    - [Helm](#helm)
    - [Kubectl](#kubectl)
  - [Set up the Kubernetes Cluster](#set-up-the-kubernetes-cluster)
    - [Local Development MiniKube](#local-development-minikube)
    - [Production Workload: AWS EKS](#production-workload-aws-eks)
      - [Install eksctl](#install-eksctl)
      - [Using eksctl to provision Kubernetes cluster on EKS](#using-eksctl-to-provision-kubernetes-cluster-on-eks)
      - [Suggestions for Naming](#suggestions-for-naming)
- [Install Marklogic Helm Chart](#install-marklogic-helm-chart)
  - [Add Marklogic Repo](#add-marklogic-repo)
  - [Installing the Chart](#installing-the-chart)
  - [Configuration Options](#configuration-options)
    - [--values](#--values)
    - [--set](#--set)
- [Access the MarkLogic Server](#access-the-marklogic-server)
  - [Service](#service)
    - [Get the ClusterIP Service Name](#get-the-clusterip-service-name)
    - [Using the Service DNS Record to Access MarkLogic](#using-the-service-dns-record-to-access-marklogic)
  - [Port Forward](#port-forward)
    - [Forward to Pod](#forward-to-pod)
    - [Forward to Service](#forward-to-service)
  - [Notice](#notice)
- [Uninstalling the Chart](#uninstalling-thechart)
- [Parameters](#parameters)

# Introduction

This tutorial describes how to set up Kubernetes development environment with AWS EKS and MarkLogic Server. It covers these tasks:
- Set up the prerequisites necessary for setting up MarkLogic Server in Kubernetes
- How to setup Kubernetes cluster and install MarkLogic Server on Minikube
- How to setup Kubernetes cluster and install MarkLogic Server on AWS EKS using eksctl
- Access the MarkLogic Server cluster
- How to clean up your environment
- List of parameters using for configuration

# Prerequisites

## Set up the required tools

### Helm

Helm is a Kubernetes package manager that makes it easy to install MarkLogic on Kubernetes.

To install Helm, follow the steps described in: https://helm.sh/docs/intro/install/

Verify the installation with this command:

```
helm -h 
```

If Helm is installed correctly, you will see the Helm user manual.

If Helm is not installed correctly, you will see the error: `command not found: helm`

### Kubectl

Kubectl is a command line tool that serves as a client, to connect to a Kubernetes cluster.

To install Kubectl, follow the steps at: https://kubernetes.io/docs/tasks/tools/

To verify the Kubectl installation, use this command: 

```
kubectl -h 
```
If Kubectl is installed correctly, you will see the the Kubectl user manual.

If kubectl is not installed correctly, you will see the error: `command not found: kubectl`

## Set up the Kubernetes Cluster

### Local Development MiniKube

For local development, you will want to set up MiniKube. See the set up instructions here: [MiniKube Setup Guide](docs/Local_Development_Tutorial.md)

### Production Workload: AWS EKS

For production workload development, you will want to use a cloud platform. 

EKS is a managed Kubernetes platform provided by AWS. The eksctl tool is a simple way to bring up a Kubernetes cluster on EKS.

#### Install eksctl

To install eksctl, follow the steps described here: https://docs.aws.amazon.com/eks/latest/userguide/eksctl.html

#### Using eksctl to provision Kubernetes cluster on EKS

The following eksctl code can be used to create a Kubernetes cluster in EKS. You will need to replace CLUSTER_NAME, KUBERNETES_VERSION, REGION, NODEGROUP_NAME, NODE_TYPE and NUMBER_OF_NODES based on your configuration.

```
eksctl create cluster \
  --name CLUSTER_NANE \
  --version KUBERNETES_VERSION \
  --region REGION \
  --nodegroup-name NODEGROUP_NAME \
  --node-type NODE_TYPE \
  --nodes NUMBER_OF_NODES
```

#### Suggestions for Naming

* CLUSTER_NAME: Choose a distinctive cluster name.
* KUBERNETES_VERSION: For now, we only support the latest version of Kubernetes in EKS, which is 1.21.
* NODEGROUP_NAME: Choose a distinctive node group name.
* NODE_TYPE: The recommendation from our performance team is to use the r5.large node type for development purposes.
* NUMBER_OF_NODES: Total number of nodes running not only Marklogic database, but also nodes running other applications.

# Install Marklogic Helm Chart

## Add Marklogic Repo

If you haven’t already, add the MarkLogic official repo to Helm using this command:

```
helm repo add marklogic https://marklogic.github.io/marklogic-kubernetes/
```

The output will look like this:

```
"marklogic" has been added to your repositories
```

Use this command to verify that the repo has been added to Helm:

```
helm repo list
```

You should see an entry like this:

`marklogic           https://marklogic.github.io/marklogic-kubernetes/`

Use this command to ensure the Helm repo is up to date:

```
helm repo update
```

## Installing the Chart

Use this command to install MarkLogic Chart to the current namespace with default settings:

```
helm install my-release marklogic/marklogic --version=1.0.0-ea1
```

After you install MarkLogic Chart, the output will look like this:

```
NAME: my-release
LAST DEPLOYED: 
NAMESPACE: default
STATUS: deployed
REVISION: 1
```

**Note:** --version=1.0.0-ea1 must be provided as part of the name. You can choose a distinctive release name to replace "my-release".

We strongly recommend that you deploy MarkLogic Chart in an exclusive namespace. Use the `--create-namespace` flag if the namespace has not already been created:

```
helm install my-release marklogic/marklogic --version=1.0.0-ea1 --namespace=marklogic --create-namespace
```

Use this command to verify the deployment:

```
helm list --all-namespaces
```

You should see an entry named "my-release" (or the release name you chose) with a status of "deployed".

## Configuration Options

This section describes the configuration options you can use with Helm. 

### --values

The `--values` flag points to a YAML file. The values in the file will override the default Helm values.

Use this command to view the default configurable values:

```
helm show values marklogic/marklogic --version=1.0.0-ea1
```

To configure a different value for your installation, create a `values.yaml` file.

For example, if you want to set the credential for Docker Hub, configure the `values.yaml` file like this:

```
imagePullSecret: 
  registry: "https://index.docker.io/v1/"
  username: YOUR_USERNAME
  password: YOUR_PASSWORD
```

Use the following command to install MarkLogic with the `values.yaml` file you just created.

```
helm install my-release marklogic/marklogic --version=1.0.0-ea1 --values values.yaml
```

### --set

Use the `--set` flag to make one or more configuration changes directly:

```
helm install my-release marklogic/marklogic --version=1.0.0-ea1 \
--set imagePullSecret.registry="https://index.docker.io/v1/" \
--set imagePullSecret.username=YOUR_USERNAME \
--set imagePullSecret.password=YOUR_PASSWORD
```

We recommend that you use the `values.yaml` file for configuring your installation.

# Access the MarkLogic Server

## Service

You can use the ClusterIP service to access MarkLogic within the Kubernetes cluster.

### Get the ClusterIP Service Name

Use the following command to get a list of Kubernetes services:

```
kubectl get services
```

The output will look like this: (the actual names may be different)

```
NAME                 TYPE        CLUSTER-IP       EXTERNAL-IP   PORT(S)                                                 AGE
kubernetes           ClusterIP   10.96.0.1        <none>        443/TCP                                                 1d
marklogic            ClusterIP   10.109.182.205   <none>        8000/TCP,8002/TCP                                       1d
marklogic-headless   ClusterIP   None             <none>        7997/TCP,7998/TCP,7999/TCP,8000/TCP,8001/TCP,8002/TCP   1d
```

The service you are looking for is the one ending with "marklogic" and where the CLUSTER-IP is not None. In the example above, "marklogic" is the service name for the ClusterIP service.

### Using the Service DNS Record to Access MarkLogic

For each Kubernetes service, a DNS with the following format is created:

```
<service-name>.<namespace-name>.svc.cluster.local
```

For example, if the service-name is "marklogic" and namespace-name is "default", the DNS URL to access the MarkLogic cluster is "marklogic.default.svc.cluster.local".

## Port Forward

The `kubectl port-forward` command can help you access MarkLogic outside of the Kubernetes cluster. Use the service to access a specific pod, or the whole cluster.
### Forward to Pod

To access each pod directly, use the `kubectl port-forward` command using the following format:

```
kubectl port-forward <POD-NAME> <LOCAL-PORT>:<CONTAINER-PORT>
```

For example, run this command to forward ports 8000 and 8001 from marklogic-0 pod to localhost:

```
kubectl port-forward marklogic-0 8000:8000 8001:8001
```

This pod can now be accessed via http://localhost:8001.

### Forward to Service

To access the whole cluster, use the `kubectl port-forward` command with the following format:

```
kubectl port-forward svc/<SERVICE-NAME> <LOCAL-PORT>:<CONTAINER-PORT>
```

For example, run this command to forward ports 8000 from marklogic service to localhost:

```
kubectl port-forward svc/marklogic 8000:8000
```

This pod can now be accessed via http://localhost:8001.

## Notice

To use transactional functionality with MarkLogic, you have to set up Ingress and configure cookie-based session affinity. This function will be supported in a future release.

# Uninstalling the Chart

Use this Helm command to uninstall the chart:

```
helm uninstall my-release
```

The output will look like this:

```
release "my-release" uninstalled
```

Use this command to verify that the uninstall was successful:

```
helm list --all-namespaces
```
You should not see an entry named "my-release" (or the release name you chose).

# Parameters

This table describes the list of available parameters for Helm Chart.

| Name                                 | Description                                                                                                    | Default Value                        |
| ------------------------------------ | -------------------------------------------------------------------------------------------------------------- | ------------------------------------ |
| `replicaCount`                       | Number of MarkLogic Nodes                                                                                      | `1`                                  |
| `image.repository`                   | repository for MarkLogic image                                                                                 | `store/marklogicdb/marklogic-server` |
| `image.tag`                          | Image tag for MarkLogic                                                                                        | `10.0-9-centos-1.0.0-ea4`            |
| `image.pullPolicy`                   | Image pull policy                                                                                              | `IfNotPresent`                       |
| `imagePullSecret.registry`           | Registry of the imagePullSecret                                                                                | `""`                                 |
| `imagePullSecret.username`           | Username of the imagePullSecret                                                                                | `""`                                 |
| `imagePullSecret.password`           | Password of the imagePullSecret                                                                                | `""`                                 |
| `resources.limits`                   | The resource limits for MarkLogic container                                                                    | `{}`                                 |
| `resources.requests`                 | The resource requests for MarkLogic container                                                                  | `{}`                                 |
| `nameOverride`                       | String to override the app name                                                                                | `""`                                 |
| `fullnameOverride`                   | String to completely replace the generated name                                                                | `""`                                 |
| `auth.adminUsername`                 | Username for default MarkLogic Administrator                                                                   | `admin`                              |
| `auth.adminPassword`                 | Password for default MarkLogic Administrator                                                                   | `admin`                              |
| `affinity`                           | Affinity property for pod assignment                                                                           | `{}`                                 |
| `nodeSelector`                       | nodeSelector property for pod assignment                                                                       | `{}`                                 |
| `persistence.enabled`                | Enable MarkLogic data persistence using Persistence Volume Claim (PVC). If set to false, EmptyDir will be used | `true`                               |
| `persistence.storageClass`           | Storage class for MarkLogic data volume, leave empty to use the default storage class                          | `""`                                 |
| `persistence.size`                   | Size of storage request for MarkLogic data volume                                                              | `10Gi`                               |
| `persistence.annotations`            | Annotations for Persistence Volume Claim (PVC)                                                                 | `{}`                                 |
| `persistence.accessModes`            | Access mode for persistence volume                                                                             | `["ReadWriteOnce"]`                  |
| `persistence.mountPath`              | The path for the mounted persistence data volume                                                               | `/var/opt/MarkLogic`                 |
| `extraVolumes`                       | Extra list of additional volumes for MarkLogic statefulset                                                     | `[]`                                 |
| `extraVolumeMounts`                  | Extra list of additional volumeMounts for MarkLogic container                                                  | `[]`                                 |
| `extraContainerPorts`                | Extra list of additional containerPorts for MarkLogic container                                                | `[]`                                 |
| `service.type`                       | type of the default service                                                                                    | `ClusterIP`                          |
| `service.ports`                      | ports of the default service                                                                                   | `[8000, 8002]`                       |
| `serviceAccount.create`              | Enable this parameter to create a service account for a MarkLogic Pod                                          | `true`                               |
| `serviceAccount.annotations`         | Annotations for MarkLogic service account                                                                      | `{}`                                 |
| `serviceAccount.name`                | Name of the serviceAccount                                                                                     | `""`                                 |
| `livenessProbe.enabled`              | Enable this parameter to enable the liveness probe                                                             | `true`                               |
| `livenessProbe.initialDelaySeconds`  | Initial delay seconds for liveness probe                                                                       | `30`                                 |
| `livenessProbe.periodSeconds`        | Period seconds for liveness probe                                                                              | `60`                                 |
| `livenessProbe.timeoutSeconds`       | Timeout seconds for liveness probe                                                                             | `5`                                  |
| `livenessProbe.failureThreshold`     | Failure threshold for liveness probe                                                                           | `3`                                  |
| `livenessProbe.successThreshold`     | Success threshold for liveness probe                                                                           | `1`                                  |
| `readinessProbe.enabled`             | Use this parameter to enable the readiness probe                                                               | `true`                               |
| `readinessProbe.initialDelaySeconds` | Initial delay seconds for readiness probe                                                                      | `10`                                 |
| `readinessProbe.periodSeconds`       | Period seconds for readiness probe                                                                             | `60`                                 |
| `readinessProbe.timeoutSeconds`      | Timeout seconds for readiness probe                                                                            | `5`                                  |
| `readinessProbe.failureThreshold`    | Failure threshold for readiness probe                                                                          | `3`                                  |
| `readinessProbe.successThreshold`    | Success threshold for readiness probe                                                                          | `1`                                  |
| `startupProbe.enabled`               | Parameter to enable startup probe                                                                              | `true`                               |
| `startupProbe.initialDelaySeconds`   | Initial delay seconds for startup probe                                                                        | `10`                                 |
| `startupProbe.periodSeconds`         | Period seconds for startup probe                                                                               | `20`                                 |
| `startupProbe.timeoutSeconds`        | Timeout seconds for startup probe                                                                              | `1`                                  |
| `startupProbe.failureThreshold`      | Failure threshold for startup probe                                                                            | `30`                                 |
| `startupProbe.successThreshold`      | Success threshold for startup probe                                                                            | `1`                                  |
