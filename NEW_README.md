# MarkLogic Kubernetes Helm Chart

## Prerequisites

To install this chart, you need to install [Helm](https://helm.sh/docs/intro/install/) and [Kubectl](https://kubernetes.io/docs/tasks/tools/).

To set up a Kubernetes Cluster for Production Workload, we recomend using EKS platform on AWS. To bring up a kubernetees cluster on EKS, you can install [eksctl](https://docs.aws.amazon.com/eks/latest/userguide/eksctl.html) tool. Please refer to [Using eksctl to Provision MarkLogic Kubernetes Cluster on EKS](README.md/) for detailed steps.


For non-production deployments, please visit [MiniKube Setup Guide](docs/Local_Development_Tutorial.md) to create kubernetes cluster locally.

## Installing MarkLogic Helm Chart

### Add MarkLogic Repo

If you haven’t already, add the MarkLogic official repo to Helm using this command:

```
helm repo add marklogic https://marklogic.github.io/marklogic-kubernetes/
```
### Install the Chart

Use this command to install MarkLogic Chart to the current namespace with default settings:

```
helm install my-release marklogic/marklogic --version=1.0.0-ea1
```
To configure other settings, values.yaml can be used with ```-f``` flag. A reference to these settings is available in the [Documentation](README.md).

## Contibute

File issues before creating PRs to this repo. Please follow guidelines from CONTRIBUTE file.

## Known Issues and Limitations

If the hostname is greater than 64 characters there may be issues with certificates. The certificates may shorten the name or use SANs for hostnames in the certificates.

