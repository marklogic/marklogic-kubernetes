# MarkLogic Kubernetes Helm Chart

This repository contains a Helm Chart that allows you to deploy MarkLogic on a Kubernetes cluster. Below is a brief description of how to easily create a MarkLogic StatefulSet for development and testing. See [MarkLogic Server on Kubernetes](http://cms-ml-docs-stage.marklogic.com/11.0/guide/kubernetes-guide/en/marklogic-server-on-kubernetes.html) for detailed documentation about running this.

## Getting Started

### Prerequisites

To install this chart, you need to install [Helm](https://helm.sh/docs/intro/install/) and [Kubectl](https://kubernetes.io/docs/tasks/tools/).

To set up a Kubernetes Cluster for Production Workload, we recommend using EKS platform on AWS. To bring up a Kubernetes cluster on EKS, you can install [eksctl](https://docs.aws.amazon.com/eks/latest/userguide/eksctl.html) tool. Please refer to [Using eksctl to Provision MarkLogic Kubernetes Cluster on EKS](http://cms-ml-docs-stage.marklogic.com/11.0/guide/kubernetes-guide/en/setting-up-the-required-tools/tools-for-setting-up-the-kubernetes-cluster/installing-amazon-web-services-elastic-kubernetes-service--for-production-.html) for detailed steps.


For non-production deployments, please see [MiniKube Setup Guide](docs/Local_Development_Tutorial.md) to create the Kubernetes cluster locally.
 
### Installing MarkLogic Helm Chart

1. Add MarkLogic Repo to Helm using this command:

```
helm repo add marklogic https://marklogic.github.io/marklogic-kubernetes/
```
2. Install MarkLogic Helm Chart to the current namespace with default settings:

```
helm install my-release marklogic/marklogic --version=1.0.0
```

This helm chart installation will create a single node MarkLogic cluster with a Default group and a persistent volume of 10Gi attached to the pod created.

To configure other settings, use `values.yaml` file with `-f` option. See [Parameters](#parameters) section for more information about these settings.

## Parameters

Following table lists all the parameters supported by the latest MarkLogic Helm chart:

| Name                                 | Description                                                                                                    | Default Value                        |
| ------------------------------------ | -------------------------------------------------------------------------------------------------------------- | ------------------------------------ |
| `replicaCount`                       | Number of MarkLogic Nodes                                                                                      | `1`                                  |
| `image.repository`                   | repository for MarkLogic image                                                                                 | `marklogicdb/marklogic-db`           |
| `image.tag`                          | Image tag for MarkLogic                                                                                        | `latest`                             |
| `image.pullPolicy`                   | Image pull policy                                                                                              | `IfNotPresent`                       |
| `imagePullSecret.registry`           | Registry of the imagePullSecret                                                                                | `""`                                 |
| `imagePullSecret.username`           | Username of the imagePullSecret                                                                                | `""`                                 |
| `imagePullSecret.password`           | Password of the imagePullSecret                                                                                | `""`                                 |
| `resources.limits`                   | The resource limits for MarkLogic container                                                                    | `{}`                                 |
| `resources.requests`                 | The resource requests for MarkLogic container                                                                  | `{}`                                 |
| `nameOverride`                       | String to override the app name                                                                                | `""`                                 |
| `fullnameOverride`                   | String to completely replace the generated name                                                                | `""`                                 |
| `auth.adminUsername`                 | Username for default MarkLogic Administrator                                                                   | `admin`                              |
| `auth.adminPassword`                 | Password for default MarkLogic Administrator                                                                   | ``    
| `auth.walletPassword`                 | Password for wallet                                                                    | `` 
| `bootstrapHostName`                 | Host name of MarkLogic bootstrap host                                                                | `""`   
| `group.name`               | group name for joining MarkLogic cluster                                                                    | `Default`                              |
| `group.enableXdqpSsl`                 | SSL encryption for XDQP                                                                   | `true`                         |
| `license.key`                       | 	set MarkLogic license key installed                       | `""` |
| `license.licensee`                 | set MarkLogic licensee information                       | `""` |
| `enableConverters`                 | Installs converters for the client if they are not already installed                       | `false` |
| `affinity`                           | Affinity property for pod assignment                                                                           | `{}`                                 |
| `nodeSelector`                       | nodeSelector property for pod assignment                                                                       | `{}`                                 |
| `persistence.enabled`                | Enable MarkLogic data persistence using Persistence Volume Claim (PVC). If set to false, EmptyDir will be used | `true`                               |
| `persistence.storageClass`           | Storage class for MarkLogic data volume, leave empty to use the default storage class                          | `""`                                 |
| `persistence.size`                   | Size of storage request for MarkLogic data volume                                                              | `10Gi`                               |
| `persistence.annotations`            | Annotations for Persistence Volume Claim (PVC)                                                                 | `{}`                                 |
| `persistence.accessModes`            | Access mode for persistence volume                                                                             | `["ReadWriteOnce"]`                  |
| `additionalContainerPorts`                | List of ports in addition to the defaults exposed at the container level (Note: This does not typically need to be updated. Use `service.additionalPorts` to expose app server ports.)                                                | `[]`                                 |
| `additionalVolumes`                  | List of additional volumes to add to the MarkLogic containers                                                  | `[]`                                 |
| `additionalVolumeMounts`             | List of mount points for the additional volumes to add to the MarkLogic containers                             | `[]`                                 |
| `service.type`                       | type of the default service                                                                                    | `ClusterIP`                          |
| `service.additionalPorts`                      | List of ports in addition to the defaults exposed at the service level.                                                                                    | `[]`                       |
| `serviceAccount.create`              | Enable this parameter to create a service account for a MarkLogic Pod                                          | `true`                               |
| `serviceAccount.annotations`         | Annotations for MarkLogic service account                                                                      | `{}`                                 |
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
| `updateStrategy`      | Update strategy for helm chart and app version updates        | `OnDelete`                               |

## Known Issues and Limitations

If the hostname is greater than 64 characters there may be issues with certificates. The certificates may shorten the name or use SANs for hostnames in the certificates.

