minikube delete                                                         
minikube start --driver=docker -n=1

kubectl apply -f logging/es-config.yaml
kubectl apply -f logging/kibana-config.yaml
kubectl port-forward kibana-7f4cc47bkw4d-db7dn 5601:5601

echo "=====Loading marklogc images to minikube cluster"
minikube image load marklogic-centos/marklogic-server-centos:10-internal

helm install ml ./charts/ -f values.yaml # Use local charts for helm install
