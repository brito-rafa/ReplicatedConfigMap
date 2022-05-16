# Replicated Config Map

This is a "toy" K8s controller to demonstrate how to build a cluster-level CRD and a controller using Kubebuilder 3.x.
This controller will take a Replicated Config Map (aka rcm) cluster-level object and create on many namespaces.

## Description

This controller synchronizes ConfigMap resources to multiple namespaces from a cluster scoped CRD named ReplicatedConfigMap.
The ReplicatedConfigMap CRs contain a "data" map which is synchronized to ConfigMapis in each namespace that has the label "rcm-sync: true".
ReplicatedConfigMap CR name is used for the Config Maps name on each namespace.
Updates and deletes to the ReplicatedConfigMap are also be propagated out to the ConfigMap in each namespace.

## TL;DR Deploying the Controller

You’ll need a Kubernetes cluster to run against. You can use [KIND](https://sigs.k8s.io/kind) to get a local cluster for testing, or run against a remote cluster.
**Note:** Your controller will automatically use the current context in your kubeconfig file (i.e. whatever cluster `kubectl cluster-info` shows).

Run:

```sh
export IMG=quay.io/brito_rafa0/rcm-controller:latest
make deploy 

# check the controller
kubectl get pods -n rcm-system

# install Sample CR
kubectl create -f config/samples/sample-data.yaml 
```


### Running on the cluster
1. Install Instances of Custom Resources:

```sh
kubectl apply -f config/samples/
```

2. Build and push your image to the location specified by `IMG`:
	
```sh
make docker-build docker-push IMG=<some-registry>/rcm:tag
```
	
3. Deploy the controller to the cluster with the image specified by `IMG`:

```sh
make deploy IMG=<some-registry>/rcm:tag
```

### Uninstall CRDs
To delete the CRDs from the cluster:

```sh
make uninstall
```

### Undeploy controller
UnDeploy the controller to the cluster:

```sh
make undeploy
```

## Contributing
// TODO(user): Add detailed information on how you would like others to contribute to this project

### How it works
This project aims to follow the Kubernetes [Operator pattern](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/)

It uses [Controllers](https://kubernetes.io/docs/concepts/architecture/controller/) 
which provides a reconcile function responsible for synchronizing resources untile the desired state is reached on the cluster 

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

Copyright 2022 Rafael Brito.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

