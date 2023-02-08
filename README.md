# MarkLogic Kubernetes Helm Chart

- [MarkLogic Kubernetes Helm Chart](#marklogic-kubernetes-helm-chart)
- [Introduction](#introduction)
- [Prerequisites](#prerequisites)
  - [Set Up the Required Tools](#set-up-the-required-tools)
    - [Helm](#helm)
    - [Kubectl](#kubectl)
  - [Set Up the Kubernetes Cluster](#set-up-the-kubernetes-cluster)
    - [Local Development MiniKube](#local-development-minikube)
    - [Production Workload: AWS EKS](#production-workload-aws-eks)
      - [Install eksctl](#install-eksctl)
      - [Using eksctl to Provision Kubernetes Cluster on EKS](#using-eksctl-to-provision-kubernetes-cluster-on-eks)
      - [Suggestions for Naming](#suggestions-for-naming)
- [Install MarkLogic Helm Chart](#install-marklogic-helm-chart)
  - [Add MarkLogic Repo](#add-marklogic-repo)
  - [Installing the Chart](#installing-the-chart)
  - [Configuration Options](#configuration-options)
    - [--values](#--values)
    - [--set](#--set)
    - [Setting MarkLogic admin password](#setting-marklogic-admin-password)
    - [Log Collection](#log-collection)
  - [Adding and Removing Hosts from Clusters](#adding-and-removing-hosts-from-clusters)
    - [Adding Hosts](#adding-hosts)
    - [Removing Hosts](#removing-hosts)
    - [Enabling SSL over XDQP](#enabling-ssl-over-xdqp)
- [Deploying a MarkLogic Cluster with Multiple Groups](#deploying-a-marklogic-cluster-with-multiple-groups)
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
- [Known Issues and Limitations](#Known-Issues-and-Limitations)


# Introduction

This tutorial describes how to set up Kubernetes development environment with AWS EKS and MarkLogic Server. It covers these tasks:
- Set up the prerequisites necessary for setting up MarkLogic Server in Kubernetes
- How to set up Kubernetes cluster and install MarkLogic Server on Minikube
- How to set up Kubernetes cluster and install MarkLogic Server on AWS EKS using eksctl
- Access the MarkLogic Server cluster
- How to clean up your environment
- List of parameters used for configuration

# Prerequisites

## Set Up the Required Tools

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

## Set Up the Kubernetes Cluster

### Local Development MiniKube

For local development, you will want to set up MiniKube. See the set up instructions here: [MiniKube Setup Guide](docs/Local_Development_Tutorial.md)

### Production Workload: AWS EKS

For production workload development, you will want to use a cloud platform.

EKS is a managed Kubernetes platform provided by AWS. The eksctl tool is a simple way to bring up a Kubernetes cluster on EKS.

#### Install eksctl

To install eksctl, follow the steps described here: https://docs.aws.amazon.com/eks/latest/userguide/eksctl.html

#### Using eksctl to Provision Kubernetes Cluster on EKS

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
* NUMBER_OF_NODES: Total number of nodes running not only MarkLogic database, but also nodes running other applications.

# Install MarkLogic Helm Chart

## Add MarkLogic Repo

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

### Setting MarkLogic admin password

If the password does not provided when installing the MarkLogic Chart, a randomly generated aphanumeric value will be set for MarkLogic admin password. This value is stored in Kuberenetes secrets. 
User can also set a custom password by setting auth.adminPassword value during installation.
To retrieve the randomly generated admin password, use the following commands:

1. List the secrets for MarkLogic deployment:
```
kubectl get secrets
```
Identify the name of the secret.

2. Save the secret name from step 1 and get the admin password using the following script:
```
kubectl get secret SECRET_NAME -o jsonpath='{.data.marklogic-password}' | base64 --decode
```
### Log Collection

To enable log collection for all Marklogic logs set logCollection.enabled to true. Set each option in logCollection.files to true of false depending on if you want to track each type of Marklogic log file.

In order to use the logs that are colleceted you must define an output in the outputs section of the values file. Fluent Bit will parse and output all the log files from each pod to the output(s) you set.

For documentation on how to configure the Fluent Bit output with your logging backend see Fluent Bit's output documentation here: <https://docs.fluentbit.io/manual/pipeline/outputs>


## Adding and Removing Hosts from Clusters

### Adding Hosts

The MarkLogic Helm chart creates one MarkLogic "host" per Kubernetes pod in a StatefulSet.
To add a new MarkLogic host to an existing cluster, simply increase the number of pods in your StatefulSet.
For example, if you want to change the host count of an existing MarkLogic cluster from 2 to 3, run the following Helm command:

```
helm upgrade release-name [chart-path] --namespace name-space --set replicaCount=3
```

Once this deployment is completed, the new MarkLogic host joins the existing cluster.
To track deployment status, use “**kubectl get pods**” command. This procedure does not automatically create forests on the new host.
If the host will be managing forests for a database, create them via MarkLogic's administrative UI or APIs once the pod is up and running.

### Removing Hosts

When scaling a StatefulSet down, Kubernetes will attempt to stop one or more pods in the set to achieve the desired number of pods.
When doing so, Kubernetes will stop the pod(s), but the storage attached to the pod will remain until you delete the Persistent Volume Claim(s).
Shutting down a pod from the Kubernetes side does not modify the MarkLogic cluster configuration.
It only stops the pod, which causes the MarkLogic host to go offline. If there are forests assigned to the stopped host(s), those forests will go offline.

The procedure to scale down the number of MarkLogic hosts in a cluster depends on whether or not forests are assigned to the host(s) to be removed and if the goal is to permanently remove the host(s) from the MarkLogic cluster. If there are forests assigned to the host(s) and you want to remove the host(s) from the cluster, follow MarkLogic administrative procedures to migrate the data from the forests assigned to the host(s) you want to shut down to the forests assigned to the remaining hosts in the cluster (see https://docs.marklogic.com/guide/admin/database-rebalancing#id_23094 and
https://help.marklogic.com/knowledgebase/article/View/507/0/using-the-rebalancer-to-move-the-content-in-one-forest-to-another-location for details).
Once the data are safely migrated from the forests on the host(s) to be removed, the host can be removed from the MarkLogic cluster. If there are forests assigned to the host(s) but you just want to temporarily shut down the MarkLogic host/pod, the data do not need to be migrated, but the forests will go offline while the host is shut down.

For example, once you have migrated any forest data from the third MarkLogic host, you can change the host count on an
existing MarkLogic cluster from 3 to 2 by running the following Helm command:

```
helm upgrade release-name [chart-path] --namespace name-space --set replicaCount=2
```

Before Kubernetes stops the pod, it makes a call to the MarkLogic host to tell it to shut down with the "fastFailOver" flag set to TRUE. This tells the remaining hosts in the cluster that this host is shutting down and to trigger failover for any replica forests that may be available for forests on this host. There is a two-minute grace period to allow MarkLogic to shut down cleanly before Kubernetes kills the pod.

In order to track the host shutdown progress, run the following command:
```
kubectl logs pod/terminated-host-pod-name
```

If you are permanently removing the host from the MarkLogic cluster, once the pod is terminated, follow standard MarkLogic administrative procedures using the administrative UI or APIs to remove the MarkLogic host from the cluster. Also, because Kubernetes keeps the Persistent Volume Claims and Persistent Volumes around until they are explicitly deleted, you must manually delete them using the Kubernetes APIs before attempting to scale the hosts in the StatefulSet back up again.
### Enabling SSL over XDQP

To enable SSL over XDQP, set the `enableXdqpSsl` to true either in the values.yaml file or using the `--set` flag. All communications to and from hosts in the cluster will be secured. When this setting is on, default SSL certificates will be used for XDQP encryption.

Note: To enable other XDQP/SSL settings like `xdqp ssl allow sslv3`, `xdqp ssl allow tls`, `xdqp ssl ciphers`, use MarkLogic REST Management API. See the MarkLogic documentation [here](https://docs.marklogic.com/REST/management).

# Deploying a MarkLogic Cluster with Multiple Groups

To deploy a MarkLogic cluster with multiple groups (separate E and D nodes for example) the `bootstrapHostName` and `group.name` must be configured in values.yaml or set the values provided for these configurations using the `--set` flag while installing helm charts.
For example, if you want to create a MarkLogic cluster with three nodes in a "dnode" group and two nodes in an "enode" group, start with the following helm command:

```
helm install dnode-group ./charts/ --set group.name=dnode --set replicaCount=3
```
Once this deployment is complete, a MarkLogic cluster with three hosts should be running.
To add the "enode" group and nodes to the cluster, the `bootstrapHostName` must be set to join the existing MarkLogic cluster. The first host in the other group can be used. For this example, set `bootstrapHostName` to `dnode-group-marklogic-0.dnode-group-marklogic-headless.default.svc.cluster.local` with the following command:

```
helm install enode-group ./charts/ --set group.name=enode --set replicaCount=2 --set bootstrapHostName=dnode-group-marklogic-0.dnode-group-marklogic-headless.default.svc.cluster.local
```
Once this deployment is complete, there will be a new "enode" group with two hosts in the MarkLogic cluster.

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

## Security Context

Security context defines privilege and access control settings for a Pod or Container. By default security context for containers is enabled with runAsUser, runAsNonRoot, allowPrivilegeEscalation settings. To configure these values for containers, set the containerSecurityContext in the values.yaml file or using the `--set` flag. Additional security context settings can be added to containerSecurityContext configuration. Please refer [https://kubernetes.io/docs/tasks/configure-pod-container/security-context/](https://kubernetes.io/docs/tasks/configure-pod-container/security-context/).

## Network Policy

Note: To use network policies, you must be using a networking solution that supports NetworkPolicy. Creating a NetworkPolicy resource without a controller that implements it will have no effect. Please refer [https://kubernetes.io/docs/concepts/services-networking/network-policies/#prerequisites](https://kubernetes.io/docs/concepts/services-networking/network-policies/#prerequisites).

Use NetworkPolicy to control network traffic flow for your applications, it allows you to specify how pods should communicate over the network. By default network policy is disabled in the values.yaml file. Set the networkPolicy.enabled to true to enable the use of network policy resource, default ports are provided in the settings, you can define custom rules for the sources of the traffic to the desired ports.

## Pod Priorty

Pods can be assigned priority that reflects the signifance of a pod compared to other pods. For detailed explanation, please refer [https://kubernetes.io/docs/concepts/scheduling-eviction/pod-priority-preemption/](https://kubernetes.io/docs/concepts/scheduling-eviction/pod-priority-preemption/)

To add priority for pods, use the following the steps:

1. Add a PriorityClass. Following is an example of PriorityClass:
```
apiVersion: scheduling.k8s.io/v1
kind: PriorityClass
metadata:
  name: high-priority
value: 1000000
globalDefault: false
description: "This high priority class should be used for MarkLogic service pods only."
```
  For more details and examples, refer [https://kubernetes.io/docs/concepts/scheduling-eviction/pod-priority-preemption/#priorityclass](https://kubernetes.io/docs/concepts/scheduling-eviction/pod-priority-preemption/#priorityclass)

2. Set priorityClassName in the values.yaml file or using --set flag while installing the chart. Value of priorityClassName should be set to one of the added PriorityClassName.


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
| `image.repository`                   | repository for MarkLogic image                                                                                 | `marklogicdb/marklogic-db`           |
| `image.tag`                          | Image tag for MarkLogic                                                                                        | `latest`                             |
| `image.pullPolicy`                   | Image pull policy                                                                                              | `IfNotPresent`                       |
| `imagePullSecret.registry`           | Registry of the imagePullSecret                                                                                | `""`                                 |
| `imagePullSecret.username`           | Username of the imagePullSecret                                                                                | `""`                                 |
| `imagePullSecret.password`           | Password of the imagePullSecret                                                                                | `""`                                 |
| `resources.limits`                   | The resource limits for MarkLogic container                                                                    | `{}`                                 |
| `resources.requests`                 | The resource requests for MarkLogic container                                                                  | `{}`                                 |
| `nameOverride`                       | String to override the app name                                                                                | `""`                                 |
| `fullnameOverride`                   | String to completely replace the generated name                                                                | `""`                                 |
| `auth.adminUsername`                 | Username for default MarkLogic Administrator                                                                   | `admin`                              |
| `auth.adminPassword`                 | Password for default MarkLogic Administrator                                                                   | `admin`     
| `bootstrapHostName`                 | Host name of MarkLogic bootstrap host                                                                | `""`   
| `group.name`               | group name for joining MarkLogic cluster                                                                    | `Default`                              |
| `group.enableXdqpSsl`                 | SSL encryption for XDQP                                                                   | `true`                         |
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
| `logCollection.enabled`              | Enable this parameter to enable cluster wide log collection of Marklogic server logs                           | `false`                              |
| `logCollection.files.errorLogs`      | Enable this parameter to enable collection of Marklogics error logs when clog collection is enabled            | `true`                               |
| `logCollection.files.accessLogs`     | Enable this parameter to enable collection of Marklogics access logs when log collection is enabled            | `true`                               |
| `logCollection.files.requestLogs`    | Enable this parameter to enable collection of Marklogics request logs when log collection is enabled           | `true`                               |
| `logCollection.files.crashLogs`      | Enable this parameter to enable collection of Marklogics crash logs when log collection is enabled             | `true`                               |
| `logCollection.files.auditLogs`      | Enable this parameter to enable collection of Marklogics audit logs when log collection is enabled             | `true`                               |
| `containerSecurityContext.enabled`      | Enable this parameter to enable security context for containers             | `true`                               |
| `containerSecurityContext.runAsUser`      | User ID to run the entrypoint of the container process             | `1000`                               |
| `containerSecurityContext.runAsNonRoot`      | Indicates that the container must run as a non-root user             | `true`                               |
| `containerSecurityContext.allowPrivilegeEscalation`      | Controls whether a process can gain more privileges than its parent process             | `true`                               |
| `networkPolicy.enabled`      | Enable this parameter to enable network policy             | `false`                               |
| `networkPolicy.customRules`      | Placeholder to specify selectors              | `{}`                               |
| `networkPolicy.ports`      | Ports to which traffic is allowed              | `[8000, 8001, 8002]`                               |
| `priorityClassName`      | Name of a PriortyClass defined to set pod priority        | `""`                               |

# Known Issues and Limitations

1. If the hostname is greater than 64 characters there may be issues with certificates. The certificates may shorten the name or use SANs for hostnames in the certificates.
