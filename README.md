# invoker-operator
// TODO(user): Add simple overview of use/purpose

## Description
## Quick Setup

### Setting up Kube Prometheus Stack
Using Kube prometheus stack helm charts to setup basic prometheus:
https://github.com/prometheus-community/helm-charts/tree/main/charts/kube-prometheus-stack

```shell
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo update

helm install kube-prometheus-stack prometheus-community/kube-prometheus-stack
```

Expose Prometheus endpoint via Port forwarding
```shell
kubectl port-forward svc/prometheus-operated 9090:9090
```

Access Prometheus dashboard via https://localhost:9090

### Setting up operator
This setup assumes you have a Kubernetes cluster set up with correct permissions and network that can reach public internet.

1. Depending on your kubernetes CLI command, change in Makefile line 187 to `KUBECTL ?= {kubectl command}`.
Eg. For Microk8s you may change it to `KUBECTL ?= microk8s kubectl`. 

2. Run and ensure the operator deployment is running on the kubernetes cluster.
```shell
make deploy

# wait for operator pods to be ready
kubectl wait --for=condition=available deployment/invoker-operator-controller-manager
```

3. After the operator pods are running. Apply the custom resource yamls. The example is located in `config/samples`. 
```
kubectl apply -f config/samples/invokeroperator_v1alpha1_invokerdeployment.yaml
``` 

## Getting Started
You’ll need a Kubernetes cluster to run against. You can use [KIND](https://sigs.k8s.io/kind) to get a local cluster for testing, or run against a remote cluster.
**Note:** Your controller will automatically use the current context in your kubeconfig file (i.e. whatever cluster `kubectl cluster-info` shows).

### Running on Kind
1. Install Instances of Custom Resources:

```sh
kubectl apply -f config/samples/
```

2. Build and push your image to the location specified by `IMG`:

```sh
make docker-build docker-push IMG=<some-registry>/invoker-operator:tag
```

3. Deploy the controller to the cluster with the image specified by `IMG`:

```sh
make deploy IMG=<some-registry>/invoker-operator:tag
```

### Uninstall CRDs
To delete the CRDs from the cluster:

```sh
make uninstall
```

### Undeploy controller
UnDeploy the controller from the cluster:

```sh
make undeploy
```

## Contributing
// TODO(user): Add detailed information on how you would like others to contribute to this project

### How it works
This project aims to follow the Kubernetes [Operator pattern](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/).

It uses [Controllers](https://kubernetes.io/docs/concepts/architecture/controller/),
which provide a reconcile function responsible for synchronizing resources until the desired state is reached on the cluster.

### Test It Out
1. Install the CRDs into the cluster:

```sh
make install
```

2. Run your controller (this will run in the foreground, so switch to a new terminal if you want to leave it running):

```sh
make run
```

**NOTE:** You can also run this in one step by running: `make install run`

### Modifying the API definitions
If you are editing the API definitions, generate the manifests such as CRs or CRDs using:

```sh
make manifests
```

**NOTE:** Run `make --help` for more information on all potential `make` targets

More information can be found via the [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html)

## License

Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

