# Getting started

## Creating a Kubernetes cluster

If you have already created a Kubernetes cluster, you can skip to step 2,
Deploy TiKV Operator.

This section covers 2 different ways to create a simple Kubernetes cluster that
can be used to test TiKV Cluster locally. Choose whichever best matches your
environment or experience level.

- Using [kind](https://kind.sigs.k8s.io/docs/user/quick-start/) (Kubernetes in Docker)
- Using [minikube](https://kubernetes.io/docs/tasks/tools/install-minikube/) (Kubernetes running locally in a VM)

Please refer to their official document to prepare a Kubernetes cluster. If you
have docker installed, here is the quick way to start a Kubernetes cluster with
`kind`:

On macOS / Linux:

```shell
curl -Lo ./kind https://kind.sigs.k8s.io/dl/v0.8.1/kind-$(uname)-amd64
chmod +x ./kind
./kind create cluster
```

On Windows:

```shell
curl.exe -Lo kind-windows-amd64.exe https://kind.sigs.k8s.io/dl/v0.8.1/kind-windows-amd64
.\kind-windows-amd64.exe create cluster
```

## Deploy TiKV Operator

Before proceeding, make sure the following requirements are satisfied:

- A running Kubernetes Cluster that kubectl can connect to
- Helm 3

### Install helm

Here is a quick way to install it in CLI:

```shell
curl https://raw.githubusercontent.com/helm/helm/master/scripts/get-helm-3 | bash
```

Please refer to [the official installation guide](https://helm.sh/docs/intro/install/) for more alternatives.

### Install CRD

```shell
kubectl apply -f https://raw.githubusercontent.com/tikv/tikv-operator/master/manifests/crd.v1beta1.yaml
```

### Install tikv-operator

First, add the PingCAP Repository:

```shell
helm repo add pingcap https://charts.pingcap.org/
```

Then, install with the following command:

```shell
kubectl create ns tikv-operator
helm install --namespace tikv-operator tikv-operator pingcap/tikv-operator --version v0.1.0
```

Finally, confirm that the TiKV Operator components are running with this command:

```shell
kubectl --namespace tikv-operator get pods
```

## Deploy TiKV Cluster

Deploy the TiKV Cluster:

```shell
curl -LO https://raw.githubusercontent.com/tikv/tikv-operator/master/examples/basic/tikv-cluster.yaml
kubectl apply -f tikv-cluster.yaml
```

Expected output:

```
tikvcluster.tikv.org/basic created
```

Wait for it to be ready:

```shell
kubectl wait --for=condition=Ready --timeout 10m tikvcluster/basic
```

It may takes several minutes as it needs to pull images from Docker Hub. You can check the progress with the following command:

```shell
kubect get pods -o wide
```

If the network connection to the Docker Hub is slow, you can try this example which uses images hosted in Alibaba Cloud:

```shell
curl -LO https://raw.githubusercontent.com/tikv/tikv-operator/master/examples/basic-cn/tikv-cluster.yaml
kubectl apply -f tikv-cluster.yaml
```

## Accessing the PD endpoint

Open a new terminal tab and run this command:

```shell
kubectl port-forward svc/basic-pd 2379:2379
```

This will forward local port `2379` to PD service `basic-pd`.

Now, you can access the PD endpoint with `pd-ctl` or any other PD client:

```shell
$ pd-ctl cluster
{
  "id": 6841476120821315702,
  "max_peer_count": 3
}
```
