# vin-order
//it's controller for order custom object

## Description
// TODO(user): An in-depth paragraph about your project and overview of use

## Getting Started
You’ll need a Kubernetes cluster to run against. You can use [KIND](https://sigs.k8s.io/kind) to get a local cluster for testing, or run against a remote cluster.
**Note:** Your controller will automatically use the current context in your kubeconfig file (i.e. whatever cluster `kubectl cluster-info` shows).
it should have cmdb configmap in namespace which you want to run this order operator.
you can use below way to create cmdb config map. 

1.
```shell
  kubectl create configmap cmdb --from-literal=db.host=localhost --from-literal=db.port=3306 \
  --from-literal=db.username=vin 
  --from-literal=db.password=vin 
  --from-literal=db.schema=cdr 
```
2.
```yaml
zouwei-macbook:samples zouw$ kubectl get cm cmdb -o yaml --context=docker-desktop
apiVersion: v1
data:
  db.host: localhost
  db.port: "3306"
  db.username: vin
  db.password: vin
  db.schema: cdr
kind: ConfigMap
metadata:
  name: cmdb
```

### Running on the cluster
1. Install Instances of Custom Resources:



```sh
kubectl apply -f config/samples/
```

2. Build and push your image to the location specified by `IMG`:
	
```sh
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o manager
make docker-build docker-push IMG=blueskyxi3/vin-order:0.0.9
make deploy IMG=blueskyxi3/vin-order:0.0.9
date
make docker-build docker-push IMG=<some-registry>/vin-order:tag
```
	
3. Deploy the controller to the cluster with the image specified by `IMG`:

```sh
make deploy IMG=<some-registry>/vin-order:tag
make deploy IMG=blueskyxi3/vin-order:0.0.8

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

Copyright 2022 vicentzou.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

