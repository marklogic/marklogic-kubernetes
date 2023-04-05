# MarkLogic Kubernetes Helm Chart

This repository contains a Helm Chart that allows you to deploy MarkLogic on a Kubernetes cluster. Below is a brief description of how to easily create a MarkLogic StatefulSet for development and testing. See [MarkLogic Server on Kubernetes](http://cms-ml-docs-stage.marklogic.com/11.0/guide/kubernetes-guide/en/marklogic-server-on-kubernetes.html) for detailed documentation about running this.

## Getting Started

### Prerequisites

To install this chart, you need to install [Helm](https://helm.sh/docs/intro/install/) and [Kubectl](https://kubernetes.io/docs/tasks/tools/).

To set up a Kubernetes Cluster for Production Workload, we recommend using EKS platform on AWS. To bring up a Kubernetes cluster on EKS, you can install [eksctl](https://docs.aws.amazon.com/eks/latest/userguide/eksctl.html) tool. Please refer to [Using eksctl to Provision MarkLogic Kubernetes Cluster on EKS](http://cms-ml-docs-stage.marklogic.com/11.0/guide/kubernetes-guide/en/setting-up-the-required-tools/tools-for-setting-up-the-kubernetes-cluster/installing-amazon-web-services-elastic-kubernetes-service--for-production-.html) for detailed steps.

For non-production deployments, please see [MiniKube Setup Guide](docs/Local_Development_Tutorial.md) to create the Kubernetes cluster locally.
 
### Installing MarkLogic Helm Chart

This helm chart installation will create a single-node MarkLogic cluster with a Default group. The cluster will be allocated with a persistent volume of 10GB, 2 vCPUs, and 4 GB RAM.

1. Add MarkLogic Repo to Helm using this command:
```
helm repo add marklogic https://marklogic.github.io/marklogic-kubernetes/
```
2. Create a Kubernetes namespace using below command:
```
kubectl create namespace marklogic
```
3. Create a secret to store MarkLogic admin credentials that includes username, password and wallet-password:
Substitute your choice of values for admin credentials in the below command:
```
kubectl create secret generic ml-admin-secrets \
    --from-literal=username='' \
    --from-literal=password='' \
    --from-literal=wallet-password='' \
    --namespace=marklogic
```
Please refer Kubernetes official documentation for detailed steps on how to [create a secret](https://kubernetes.io/docs/tasks/configmap-secret/managing-secret-using-kubectl/#create-a-secret)

4. Set below parameters in the values.yaml to allocate compute resources and the secret created for admin credentiials
```
# If no secret is specified and the admin credentials are not provided, a secret will be automatically
# generated with random admin and wallet passwords.
auth:
  secretName: "ml-admin-secrets"
# Compute Resources
resources:
  requests:      
    cpu: 2000m      
    memory: 4000Mi
```
5. Install MarkLogic Helm Chart with above custom settings and rest set to default in the values.yaml:
```
helm install my-release marklogic/marklogic --version=1.0.0 --values values.yaml --namespace=marklogic
```
Once the installation is complete and the pod is in a running state, MarkLogic admin UI can be accessed using the port-forwarding command as below:
```
kubectl port-forward my-release-marklogic-0 8000:8000 8001:8001
```
Please refer [Documentation](http://cms-ml-docs-stage.marklogic.com/11.0/guide/kubernetes-guide/en/creating-a-marklogic-cluster/accessing-marklogic-server-in-a-kubernetes-cluster.html) for more options on accessing MarkLogic server in a Kubernetes cluster.

By default, MarkLogic admin credentials will be set to random alphanumeric value. Use the following steps to extract the admin username, password and wallet-password from a secret:

1. Run the below command to fetch the secret name for MarkLogic deployment
``` 
kubectl get secrets 
```
2. Use the secret name from step 1 to get MarkLogic admin credentials. For randomly generated credentials, secret name will be in the format  `RELEASE_NAME-marklogic-admin`. Otherwise please use the secret created in step 2 of [Installing MarkLogic Helm Chart](#installing-markLogic-helm-chart) in this example `ml-admin-secrets`
``` 
kubectl get secret ml-admin-secrets -o jsonpath='{.data.password}' | base64 --decode 
``` 
Replace `password` with `username` and `wallet-password` in the above command to extract admin username and wallet password. 

To configure other settings, use `values.yaml` file with `-f` or `--values` option. See [Parameters](#parameters) section for more information about these settings.

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
| `auth.secretName`                    | Kubernetes Secret name for MarkLogic Admin credentials                                                                  | ``  
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

1. If the hostname is greater than 64 characters there will be issues with certificates. It is highly recommended to use hostname shorter than 64 characters or use SANs for hostnames in the certificates.
2. The MarkLogic Docker image must be run in privileged mode. At the moment if the image isn't run as privileged many calls that use sudo during the startup script will fail due to lack of required permissions as the image will not be able to create a user with the required permissions.
3. The latest released version of CentOS 7 has known security vulnerabilities with respect to glib2 CVE-2016-3191, CVE-2015-8385, CVE-2015-8387, CVE-2015-8390, CVE-2015-8394, CVE-2016-3191, glibc CVE-2019-1010022, pcre CVE-2015-8380, CVE-2015-8387, CVE-2015-8390, CVE-2015-8393, CVE-2015-8394, SQLite CVE-2019-5827. These libraries are included in the CentOS base image but, to-date, no fixes have been made available. Even though these libraries may be present in the base image that is used by MarkLogic Server, they are not used by MarkLogic Server itself, hence there is no impact or mitigation required.
4. TLS cannot be turned on at the MarkLogic level for the Admin (port 8001) and Manage (port 8002) app servers. TLS can be configured for any/all other ports at the MarkLogic level and if the Admin and Manage ports need to be exposed outside of the Kubernetes network, TLS can be terminated at the load balancer. Alternatively, additional custom app servers can be configured to serve the Admin UI and Management REST API on custom ports with TLS configured.
5. With respect to security context “allowPrivilegeEscalation” is set to TRUE by default in values.yaml file to run MarkLogic container. Work is in progress to run MarkLogic container as rootless user.
