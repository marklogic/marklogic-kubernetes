# MarkLogic Kubernetes Helm Chart

- [MarkLogic Kubernetes Helm Chart](#marklogic-kubernetes-helm-chart)
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
- [Uninstalling the Chart](#uninstalling-thechart)
- [Parameters](#parameters)

# Introduction

MarkLogic Server is a multi-model database that has both NoSQL and trusted enterprise data management capabilities. It is the most secure multi-model database.

This custom Helm Chart deploys MarkLogic Server on Kubernetes using Helm.

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
The MarkLogic Helm chart creates one MarkLogic "host" per Kubernetes pod in a StatefulSet. To add a new MarkLogic host to an existing cluster, simply increase the number of pods in your StatefulSet. For example, if we want to change the host count of an existing MarkLogic cluster from 2 to 3, run the following Helm command
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

## Cluster Scaling

### Adding Hosts to a Cluster

The MarkLogic Helm chart creates one MarkLogic "host" per Kubernetes pod in a StatefulSet.
To add a new MarkLogic host to an existing cluster, simply increase the number of pods in your StatefulSet.
For example, if we want to change the host count of an existing MarkLogic cluster from 2 to 3, run the following Helm command

```
helm upgrade release-name [chart-path] --namespace name-space --set replicaCount=3
```

When created, the MarkLogic host will join the existing cluster once the deployment is completed.
Status can be tracked using the “**kubectl get pods**”. Note that no forests will not be created on the new host.
If the host will be managing forests for a database, they will need to be created via MarkLogic's administrative UI or APIs once the Pod is up and running.

### Removing Hosts from a Cluster

When scaling a StatefulSet down, Kubernetes will attempt to stop one or more pods in the set to achieve the desired number of pods.
When doing so, Kubernetes will stop the pod(s) but the storage attached to the pod will remain until the Persistent Volume Claim(s) have been deleted.
Shutting down a pod from the Kubernetes side does not modify the MarkLogic cluster configuration.
It only stops the pod which causes the MarkLogic host to go offline. If there are forests assigned to the stopped host(s), those forests will go offline.

The procedure to scale down the number of MarkLogic hosts in a cluster depends on whether or not forests are assigned to
the host(s) to be removed and if the goal is to permanently remove the host(s) from the MarkLogic cluster.
If there are forests assigned to the host(s) and we we want to remove the host(s) from the cluster,
follow MarkLogic administrative procedures to migrate the data from the forests assigned to the host(s) forests assigned
to the remaining hosts in the cluster (see https://docs.marklogic.com/guide/admin/database-rebalancing#id_23094 and
https://help.marklogic.com/knowledgebase/article/View/507/0/using-the-rebalancer-to-move-the-content-in-one-forest-to-another-location for details).
Once the data are safely migrated from the forests on the host(s) to be removed, the host can be removed from the MarkLogic cluster.
If there are forests assigned to the host(s) but we just want to temporarily shut down the MarkLogic host/pod,
the data do not need to be migrated but the forests will go offline while the host is shutdown.

For example, once we have migrated any forest data from the 3rd MarkLogic host, we can change the host count on an
existing MarkLogic cluster from 3 to 2 by running the following Helm command

```
helm upgrade release-name [chart-path] --namespace name-space --set replicaCount=2
```

In order to track the host shutdown progress, run the following command
```
kubectl logs pod/terminated-host-pod-name
```

If permanently removing the host from the MarkLogic cluster, once the pod is terminated, follow standard MarkLogic 
administrative procedures using the administrative UI or APIs to remove the MarkLogic host from the cluster. 
Also, because Kubernetes will keep the persistent volume claims and persistent volumes around until they are explicitly deleted, 
they must be manually deleted using the Kubernetes APIs before attempting to scale the hosts in the StatefulSet back up again.

# Uninstalling the Chart

Use this Helm command to uninstall the chart:

```
helm delete my-release
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
| `nameOverride`                       | String to override the app name                                                                                | `""`                                 |
| `auth.adminUsername`                 | Username for default MarkLogic Administrator                                                                   | `admin`                              |
| `auth.adminPassword`                 | Password for default MarkLogic Administrator                                                                   | `admin`                              |
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
| `persistence.enabled`                | Enable MarkLogic data persistence using Persistence Volume Claim (PVC). If set to false, EmptyDir will be used | `true`                               |
| `persistence.storageClass`           | Storage class for MarkLogic data volume, leave empty to use the default storage class                          | `""`                                 |
| `persistence.size`                   | Size of storage request for MarkLogic data volume                                                              | `10Gi`                               |
| `persistence.annotations`            | Annotations for Persistence Volume Claim (PVC)                                                                 | `{}`                                 |
| `persistence.accessModes`            | Access mode for persistence volume                                                                             | `["ReadWriteOnce"]`                  |
| `persistence.mountPath`              | The path for the mounted persistence data volume                                                               | `/var/opt/MarkLogic`                 |
