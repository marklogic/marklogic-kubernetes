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


## Install Nginx Ingress Controller

### Create Internal Ingress

```
helm upgrade --install ingress-nginx-internal ingress-nginx \
  --repo https://kubernetes.github.io/ingress-nginx \
  --namespace ingress-nginx-internal --create-namespace \
  --set controller.ingressClassResource.name=ingress-internal \
  --set controller.service.external.enabled=false \
  --set controller.service.internal.enable=true \
  --set controller.service.internal.annotations."service\.beta\.kubernetes\.io/aws-load-balancer-internal"=true
```

### Create External Ingress
```
helm upgrade --install ingress-nginx-external ingress-nginx \
  --repo https://kubernetes.github.io/ingress-nginx \
  --namespace ingress-nginx-external --create-namespace \
  --set controller.ingressClassResource.name=ingress-external \
  --set controller.service.external.enabled=true \
  --set controller.service.internal.enable=false
```