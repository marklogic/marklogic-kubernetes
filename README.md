# Marklogic Kubernetes

## Guide for Chart developer
### Install Helm

Following [helm installation guide](https://helm.sh/docs/intro/install/) to install Helm.

Use `helm version` to verify the installation is succeeded.

### Install Marklogic Chart

`cd charts` change you directory to the charts folder

```
helm install RELEASE_NAME .
```

### List Installed Helm Chart
```
helm list
```

### Uninstall Marklogic Chart

```
helm uninstall RELEASE_NAME
```




