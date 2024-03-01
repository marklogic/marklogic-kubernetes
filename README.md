# MarkLogic Kubernetes Helm Chart

This repository contains a Helm Chart that can be used to deploy MarkLogic on a Kubernetes cluster. Below is a brief description of how to easily create a MarkLogic StatefulSet for development and testing. See [MarkLogic Server on Kubernetes](https://docs.marklogic.com/11.0/guide/kubernetes-guide/?lang=en) for detailed documentation about running this.

## Getting Started

### Prerequisites

[Helm](https://helm.sh/docs/intro/install/) and [Kubectl](https://kubernetes.io/docs/tasks/tools/)  must be installed locally in order to use this chart.

For production environments, it is recommend to use a managed Kubenetes service such as AWS EKS. The [eksctl](https://docs.aws.amazon.com/eks/latest/userguide/eksctl.html) command line tool can be used to bring up a Kubernetes cluster on EKS. Please refer to [Using eksctl to Provision a Kubernetes Cluster on EKS](https://docs.marklogic.com/11.0/guide/kubernetes-guide/en/setting-up-the-required-tools/tools-for-setting-up-the-kubernetes-cluster.html#UUID-44d2e035-b8d5-5c08-4b52-7a8b002d34aa_section-idm4533330969176033593431540071) for detailed steps.

For non-production deployments, please see [MiniKube Setup Guide](https://docs.marklogic.com/11.0/guide/kubernetes-guide/en/setting-up-the-required-tools/tools-for-setting-up-the-kubernetes-cluster.html#UUID-44d2e035-b8d5-5c08-4b52-7a8b002d34aa_section-idm4480543593867233593415017144) to create the Kubernetes cluster locally.

### Kubernetes Version

This Helm-chart currently support Kubernetes 1.23 or later.
 
### Installing MarkLogic Helm Chart

This below example Helm Chart installation will create a single-node MarkLogic cluster with a "Default" group. A 20GB persistent volume, 2 vCPUs, and 4GB of RAM will be allocated for the pod.

1. Add MarkLogic Repo to Helm:
```
helm repo add marklogic https://marklogic.github.io/marklogic-kubernetes/
```
2. Create a Kubernetes namespace:
```
kubectl create namespace marklogic
```
3. When installing the Helm Chart, if a secret is not provided, the MarkLogic admin credentials will be generated automatically. To create a secret to specify custom admin credentials including the username, password and wallet-password, use the following command (substituting the desired values):
```
kubectl create secret generic ml-admin-secrets \
    --from-literal=username='' \
    --from-literal=password='' \
    --from-literal=wallet-password='' \
    --namespace=marklogic
```
Refer to the official Kubernetes documentation for detailed steps on how to [create a secret](https://kubernetes.io/docs/tasks/configmap-secret/managing-secret-using-kubectl/#create-a-secret).

4. Create a `values.yaml` file to customize the settings. Specify the number of pods (one MarkLogic host in this case), add the secret name for the admin credentials (if not using the automatically generated one), and specify the resources that should be allocated to each MarkLiogic pod.

Note: Please ensure to use the latest MarkLogic Docker image for the new implementation as specified in the values.yaml file below. Refer to [https://hub.docker.com/r/marklogicdb/marklogic-db/tags](https://hub.docker.com/r/marklogicdb/marklogic-db/tags) for the latest image available.
```
# Create a single MarkLogic pod
replicaCount: 1

# Marklogic image parameters
# using the latest image 11.0.3-centos-1.0.2 
image:
  repository: marklogicdb/marklogic-db;
  tag: 11.0.3-centos-1.0.2 
  pullPolicy: IfNotPresent

# Set the admin credentials secret. Leave this out or set to blank "" to use the automatically generated secret.
auth:
  secretName: "ml-admin-secrets" 

# Configure compute resources
resources:
  requests:      
    cpu: 2000m      
    memory: 4000Mi
  limits:
    cpu: 2000m
    memory: 4000Mi

# Configure the persistent volume
persistence:
  enabled: true
  size: 20Gi
```
5. Install the MarkLogic Helm Chart with the above custom settings. The rest of the settings will default to the values as listed below in the [Parameters](#parameters) section.
```
helm install my-release marklogic/marklogic --values values.yaml --namespace=marklogic
```
Once the installation is complete and the pod is in a running state, the MarkLogic admin UI can be accessed using the port-forwarding command as below:
```
kubectl port-forward my-release-marklogic-0 8000:8000 8001:8001
```
Please refer [Official Documentation](https://docs.marklogic.com/11.0/guide/kubernetes-guide/en/accessing-marklogic-server-in-a-kubernetes-cluster.html) for more options on accessing MarkLogic server in a Kubernetes cluster.

If using the automatically generated admin credentials, use the following steps to extract the admin username, password and wallet-password from a secret:

1. Run the below command to fetch all of the secret names:
``` 
kubectl get secrets 
```
The MarkLogic admin secret name will be in the format  `RELEASE_NAME-marklogic-admin` (`my-release-marklogic-admin` for the example above).

2. Using the secret name from step 1 to get MarkLogic admin credentials, retrieve the values using the following commands:
``` 
kubectl get secret my-release-marklogic-admin -o jsonpath='{.data.username}' | base64 --decode 
kubectl get secret my-release-marklogic-admin -o jsonpath='{.data.password}' | base64 --decode 
kubectl get secret my-release-marklogic-admin -o jsonpath='{.data.wallet-password}' | base64 --decode 
``` 

To configure other settings, add them to the `values.yaml` file. See [Parameters](#parameters) section for more information about these settings.

## Parameters

Following table lists all the parameters supported by the latest MarkLogic Helm chart:

| Name                                                | Description                                                                                                                                                                            | Default Value              |
| --------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------- |
| `replicaCount`                                      | Number of MarkLogic Nodes                                                                                                                                                              | `1`                        |
| `updateStrategy.type`                               | Update strategy for MarkLogic pods                                                                                                                                                     | `OnDelete`                 |
| `terminationGracePeriod`                            | Seconds the MarkLogic Pod terminate gracefully                                                                                                                                         | `120`                      |
| `clusterDomain`                                     | Domain for the Kubernetes cluster                                                                                                                                                      | `cluster.local`            |
| `group.name`                                        | Group name for joining MarkLogic cluster                                                                                                                                               | `Default`                  |
| `group.enableXdqpSsl`                               | SSL encryption for XDQP                                                                                                                                                                | `true`                     |
| `bootstrapHostName`                                 | Host name of MarkLogic bootstrap host (to join a cluster)                                                                                                                              | `""`                       |
| `image.repository`                                  | Repository for MarkLogic image                                                                                                                                                         | `marklogicdb/marklogic-db` |
| `image.tag`                                         | Image tag for MarkLogic image                                                                                                                                                          | `11.1.0-centos-1.1.2`      |
| `image.pullPolicy`                                  | Image pull policy for MarkLogic image                                                                                                                                                  | `IfNotPresent`             |
| `initContainers.configureGroup.image`               | Image for configureGroup InitContainer                                                                                                                                                 | `curlimages/curl:8.6.0`    |
| `initContainers.configureGroup.pullPolicy`          | Pull policy for configureGroup InitContainer                                                                                                                                           | `IfNotPresent`             |
| `initContainers.copyCerts.image`                    | Image for copyCerts InitContainer                                                                                                                                                      | `redhat/ubi9:9.3`          |
| `initContainers.copyCerts.pullPolicy`               | Pull policy for copyCerts InitContainer                                                                                                                                                | `IfNotPresent`             |
| `imagePullSecrets`                                  | Registry secret names as an array                                                                                                                                                      | `[]`                       |
| `hugepages.enabled`                                 | Parameter to enable Hugepage on MarkLogic                                                                                                                                              | `false`                    |
| `hugepages.mountPath`                               | Mountpath for Hugepages                                                                                                                                                                | `/dev/hugepages`           |
| `resources`                                         | The resource requests and limits for MarkLogic container                                                                                                                               | `{}`                       |
| `nameOverride`                                      | String to override the app name                                                                                                                                                        | `""`                       |
| `fullnameOverride`                                  | String to completely replace the generated name                                                                                                                                        | `""`                       |
| `auth.secretName`                                   | Kubernetes Secret name for MarkLogic Admin credentials                                                                                                                                 | `""`                       |
| `auth.adminUsername`                                | Username for default MarkLogic Administrator                                                                                                                                           | `""`                       |
| `auth.adminPassword`                                | Password for default MarkLogic Administrator                                                                                                                                           | `""`                       |
| `auth.walletPassword`                               | Password for wallet                                                                                                                                                                    | `""`                       |
| `tls.enableOnDefaultAppServers`                     | Parameter to enalbe TLS on Default App Servers (8000, 8001, 8002)                                                                                                                      | `false`                    |
| `tls.certSecretNames`                               | Names of the secrets that contain the named certificate                                                                                                                                | `[]`                       |
| `tls.caSecretName`                                  | Name of the secret that contain the CA certificate                                                                                                                                     | `""`                       |
| `enableConverters`                                  | Parameter to Install converters for the client if they are not already installed.                                                                                                      | `false`                    |
| `license.key`                                       | Set MarkLogic license key installed                                                                                                                                                    | `""`                       |
| `license.licensee`                                  | Set MarkLogic licensee information                                                                                                                                                     | `""`                       |
| `affinity`                                          | Affinity for MarkLogic pods assignment                                                                                                                                                 | `{}`                       |
| `topologySpreadConstraints`                         | POD Topology Spread Constraints to spread Pods across cluster                                                                                                                          | `[]`                       |
| `nodeSelector`                                      | Node labels for MarkLogic pods assignment                                                                                                                                              | `{}`                       |
| `persistence.enabled`                               | Parameter to enable MarkLogic data persistence using Persistence Volume Claim (PVC). If set to false, EmptyDir will be used.                                                           | `true`                     |
| `persistence.storageClass`                          | Storage class for MarkLogic data volume, leave empty to use the default storage class                                                                                                  | `""`                       |
| `persistence.size`                                  | Size of storage request for MarkLogic data volume                                                                                                                                      | `10Gi`                     |
| `persistence.annotations`                           | Annotations for Persistence Volume Claim (PVC)                                                                                                                                         | `{}`                       |
| `persistence.accessModes`                           | Access mode for persistence volume                                                                                                                                                     | `["ReadWriteOnce"]`        |
| `additionalVolumeClaimTemplates`                    | List of additional volumeClaimTemplates to each MarkLogic container                                                                                                                    | `[]`                       |
| `additionalVolumes`                                 | List of additional volumes to add to the MarkLogic containers                                                                                                                          | `[]`                       |
| `additionalVolumeMounts`                            | List of mount points for the additional volumes to add to the MarkLogic containers                                                                                                     | `[]`                       |
| `additionalContainerPorts`                          | List of ports in addition to the defaults exposed at the container level (Note: This does not typically need to be updated. Use `service.additionalPorts` to expose app server ports.) | `[]`                       |
| `service.annotations`                               | Annotations for MarkLogic service                                                                                                                                                      | `{}`                       |
| `service.type`                                      | Default service type                                                                                                                                                                   | `ClusterIP`                |
| `service.additionalPorts`                           | List of ports in addition to the defaults exposed at the service level                                                                                                                 | `[]`                       |
| `serviceAccount.create`                             | Parameter to enable creating a service account for a MarkLogic Pod                                                                                                                     | `true`                     |
| `serviceAccount.annotations`                        | Annotations for MarkLogic service account                                                                                                                                              | `{}`                       |
| `serviceAccount.name`                               | Name of the serviceAccount                                                                                                                                                             | `""`                       |
| `priorityClassName`                                 | Name of a PriortyClass defined to set pod priority                                                                                                                                     | `""`                       |
| `networkPolicy.enabled`                             | Parameter to enable network policy                                                                                                                                                     | `false`                    |
| `networkPolicy.customRules`                         | Placeholder to specify selectors                                                                                                                                                       | `{}`                       |
| `networkPolicy.ports`                               | Parameter to specify the ports where traffic is allowed                                                                                                             | `[{port:8000, endPort: 8020, protocol: TCP}]` |
| `podSecurityContext.enabled`                        | Parameter to enable security context for pod running MarkLogic containers                                                                                                              | `true`                     |
| `podSecurityContext.fsGroup`                        | Parameter to specify the group id for mounted data volume                                                                                                                              | `2`                        |
| `podSecurityContext.fsGroupChangePolicy`            | Parameter to specify how the volume ownership should be changed when a pod's volumes needs to be updated with an fsGroup                                                               | `OnRootMismatch`           |
| `containerSecurityContext.enabled`                  | Parameter to enable security context for MarkLogic containers                                                                                                                          | `true`                     |
| `containerSecurityContext.runAsUser`                | User ID to run the entrypoint of the container process                                                                                                                                 | `1000`                     |
| `containerSecurityContext.runAsNonRoot`             | Indicates that the container must run as a non-root user                                                                                                                               | `true`                     |
| `containerSecurityContext.allowPrivilegeEscalation` | Controls whether a process can gain more privileges than its parent process                                                                                                            | `true`                     |
| `livenessProbe.enabled`                             | Parameter to enable the liveness probe                                                                                                                                                 | `true`                     |
| `livenessProbe.initialDelaySeconds`                 | Initial delay seconds for liveness probe                                                                                                                                               | `300`                      |
| `livenessProbe.periodSeconds`                       | Period seconds for liveness probe                                                                                                                                                      | `10`                       |
| `livenessProbe.timeoutSeconds`                      | Timeout seconds for liveness probe                                                                                                                                                     | `5`                        |
| `livenessProbe.failureThreshold`                    | Failure threshold for liveness probe                                                                                                                                                   | `15`                       |
| `livenessProbe.successThreshold`                    | Success threshold for liveness probe                                                                                                                                                   | `1`                        |
| `readinessProbe.enabled`                            | Parameter to enable the readiness probe                                                                                                                                                | `false`                    |
| `readinessProbe.initialDelaySeconds`                | Initial delay seconds for readiness probe                                                                                                                                              | `10`                       |
| `readinessProbe.periodSeconds`                      | Period seconds for readiness probe                                                                                                                                                     | `10`                       |
| `readinessProbe.timeoutSeconds`                     | Timeout seconds for readiness probe                                                                                                                                                    | `5`                        |
| `readinessProbe.failureThreshold`                   | Failure threshold for readiness probe                                                                                                                                                  | `3`                        |
| `readinessProbe.successThreshold`                   | Success threshold for readiness probe                                                                                                                                                  | `1`                        |
| `logCollection.enabled`                             | Parameter to enable cluster wide log collection of Marklogic server logs                                                                                                               | `false`                    |
| `logCollection.image`                               | Image repository and tag for fluent-bit container                                                                                                                                      | `fluent/fluent-bit:2.2.2`  |
| `logCollection.resources.requests.cpu`              | The requested cpu resource for the fluent-bit container                                                                                                                                | `100m`                     |
| `logCollection.resources.requests.memory`           | The requested memory resource for the fluent-bit container                                                                                                                             | `128Mi`                    |
| `logCollection.resources.limits.cpu`                | The cpu resource limit for the fluent-bit container                                                                                                                                    | `100m`                     |
| `logCollection.resources.limits.memory`             | The memory resource limit for the fluent-bit container                                                                                                                                 | `128Mi`                    |
| `logCollection.files.errorLogs`                     | Parameter to enable collection of MarkLogics error logs when log collection is enabled                                                                                                 | `true`                     |
| `logCollection.files.accessLogs`                    | Parameter to enable collection of MarkLogics access logs when log collection is enabled                                                                                                | `true`                     |
| `logCollection.files.requestLogs`                   | Parameter to enable collection of MarkLogics request logs when log collection is enabled                                                                                               | `true`                     |
| `logCollection.files.crashLogs`                     | Parameter to enable collection of MarkLogics crash logs when log collection is enabled                                                                                                 | `true`                     |
| `logCollection.files.auditLogs`                     | Parameter to enable collection of MarkLogics audit logs when log collection is enabled                                                                                                 | `true`                     |
| `logCollection.outputs`                             | Configure desired output for fluent-bit                                                                                                                                                | `""`                       |
| `haproxy.enabled`                                   | Parameter to enable the HAProxy Load Balancer for MarkLogic Server                                                                                                                     | `false`                    |
| `haproxy.existingConfigmap`                         | Name of an existing configmap with configuration for HAProxy                                                                                                                           | `marklogic-haproxy`        |
| `haproxy.replicaCount`                              | Number of HAProxy Deployment                                                                                                                                                           | `2`                        |
| `haproxy.restartWhenUpgrade.enabled`                | Automatically roll Deployments for every helm upgrade                                                                                                                                  | `true`                     |
| `haproxy.stats.enabled`                             | Parameter to enable the stats page for HAProxy                                                                                                                                         | `false`                    |
| `haproxy.stats.port`                                | Port for stats page                                                                                                                                                                    | `1024`                     |
| `haproxy.stats.auth.enabled`                        | Parameter to enable the basic auth for stats page                                                                                                                                      | `false`                    |
| `haproxy.stats.auth.username`                       | Username for stats page                                                                                                                                                                | `""`                       |
| `haproxy.stats.auth.password`                       | Password for stats page                                                                                                                                                                | `""`                       |
| `haproxy.service.type`                              | The service type of the HAproxy                                                                                                                                                        | `ClusterIP`                |
| `haproxy.ports`                                     | Ports and load balancing type configuration for HAproxy                                                                                                                                | `[]`                       |
| `haproxy.tls.enabled`                               | Parameter to enable TLS for HAProxy                                                                                                                                                    | `false`                    |
| `haproxy.tls.secretName`                            | Name of the secret that stores the certificate                                                                                                                                         | `""`                       |
| `haproxy.tls.certFileName`                          | The name of the certificate file in the secret                                                                                                                                         | `""`                       |
| `haproxy.nodeSelector`                              | Node labels for HAProxy pods assignment                                                                                                                                                | `{}`                       |
| `haproxy.affinity`                                  | Affinity for HAProxy pods assignment                                                                                                                                                   | `{}`                       |
| `haproxy.resources.requests.cpu`                    | The requested cpu resource for the HAProxy container                                                                                                                                   | `250m`                     |
| `haproxy.resources.requests.memory`                 | The requested memory resource for the HAProxy container                                                                                                                                | `128Mi`                    |
| `haproxy.resources.limits.cpu`                      | The cpu resource limit for the HAProxy container                                                                                                                                       | `250m`                     |
| `haproxy.resources.limits.memory`                   | The memory resource limit for the HAProxy container                                                                                                                                    | `128Mi`                    |

## Known Issues and Limitations

1. If the hostname is greater than 64 characters there will be issues with certificates. It is highly recommended to use hostname shorter than 64 characters or use SANs for hostnames in the certificates.
2. The MarkLogic Docker image must be run in privileged mode. At the moment if the image isn't run as privileged many calls that use sudo during the startup script will fail due to lack of required permissions as the image will not be able to create a user with the required permissions.
3. The latest released version of CentOS 7 has known security vulnerabilities with respect to glib2 CVE-2016-3191, CVE-2015-8385, CVE-2015-8387, CVE-2015-8390, CVE-2015-8394, CVE-2016-3191, glibc CVE-2019-1010022, pcre CVE-2015-8380, CVE-2015-8387, CVE-2015-8390, CVE-2015-8393, CVE-2015-8394, SQLite CVE-2019-5827. These libraries are included in the CentOS base image but, to-date, no fixes have been made available. Even though these libraries may be present in the base image that is used by MarkLogic Server, they are not used by MarkLogic Server itself, hence there is no impact or mitigation required.
4. TLS cannot be turned on at the MarkLogic level for the Admin (port 8001) and Manage (port 8002) app servers. TLS can be configured for any/all other ports at the MarkLogic level and if the Admin and Manage ports need to be exposed outside of the Kubernetes network, TLS can be terminated at the load balancer. Alternatively, additional custom app servers can be configured to serve the Admin UI and Management REST API on custom ports with TLS configured.
5. With respect to security context “allowPrivilegeEscalation” is set to TRUE by default in values.yaml file to run MarkLogic container. Work is in progress to run MarkLogic container as rootless user.