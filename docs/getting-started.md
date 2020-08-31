# Getting started

This document explains how to create a simple Kubernetes cluster and use it to do a basic test deployment of TiKV Cluster using TiKV Operator.

<!-- toc -->
- [Step 1: Create a Kubernetes cluster](#step-1-create-a-kubernetes-cluster)
- [Step 2: Deploy TiKV Operator](#step-2-deploy-tikv-operator)
- [Step 3: Deploy TiKV Cluster](#step-3-deploy-tikv-cluster)
- [Step 4: Access the PD endpoint](#step-4-access-the-pd-endpoint)
<!-- /toc -->

## Step 1: Create a Kubernetes cluster

If you have already created a Kubernetes cluster, skip to [Step 2: Deploy TiKV Operator](#step-2-deploy-tikv-operator).

This section covers 2 different ways to create a simple Kubernetes cluster that
can be used to test TiKV Cluster locally. Choose whichever best matches your
environment or experience level.

- Using [kind](https://kind.sigs.k8s.io/docs/user/quick-start/) (Kubernetes in Docker)
- Using [minikube](https://kubernetes.io/docs/tasks/tools/install-minikube/) (Kubernetes running locally in a VM)

You can refer to their official documents to prepare a Kubernetes cluster.

The following shows a simple way to create a Kubernetes cluster using kind. Make sure Docker is up and running before proceeding.

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

## Step 2: Deploy TiKV Operator

Before deployment, make sure the following requirements are satisfied:

- A running Kubernetes Cluster that kubectl can connect to
- Helm 3

1. Install helm

    ```shell
    curl https://raw.githubusercontent.com/helm/helm/master/scripts/get-helm-3 | bash
    ```

    Refer to [Helm Documentation](https://helm.sh/docs/intro/install/) for more installation alternatives.

2. Install CRD

    ```shell
    kubectl apply -f https://raw.githubusercontent.com/tikv/tikv-operator/master/manifests/crd.v1beta1.yaml
    ```

3. Install tikv-operator

    1. Add the PingCAP Repository:

        ```shell
        helm repo add pingcap https://charts.pingcap.org/
        ```

    2. Create a namespace for TiKV Operator:

        ```shell
        kubectl create ns tikv-operator
        helm install --namespace tikv-operator tikv-operator pingcap/tikv-operator --version v0.1.0
        ```

    3. Install TiKV Operator:

        ```shell
        helm install --namespace tikv-operator tikv-operator pingcap/tikv-operator --version v0.1.0
        ```

    4. Confirm that the TiKV Operator components are running:

        ```shell
        kubectl --namespace tikv-operator get pods
        ```

## Step 3: Deploy TiKV Cluster

1. Deploy the TiKV Cluster:

    ```shell
    curl -LO https://raw.githubusercontent.com/tikv/tikv-operator/master/examples/basic/tikv-cluster.yaml
    kubectl apply -f tikv-cluster.yaml
    ```

    Expected output:

    ```
    tikvcluster.tikv.org/basic created
    ```

2. Wait for it to be ready:

    ```shell
    kubectl wait --for=condition=Ready --timeout 10m tikvcluster/basic
    ```

    It may takes several minutes as it needs to pull images from Docker Hub.

3. Check the progress with the following command:

    ```shell
    kubect get pods -o wide
    ```

If the network connection to the Docker Hub is slow, you can try this example which uses images hosted in Alibaba Cloud:

```shell
curl -LO https://raw.githubusercontent.com/tikv/tikv-operator/master/examples/basic-cn/tikv-cluster.yaml
kubectl apply -f tikv-cluster.yaml
```

## Step 4: Access the PD endpoint

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
