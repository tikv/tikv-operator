# Development

## Go

TiKV Operator is written in [Go](https://golang.org). If you don't have a Go development environment, [set one up](https://golang.org/doc/code.html).

The version of Go should be 1.13 or later.

## Verify

Run following commands to verify your code change.

```shell
$ make verify
```

This will show errors if your code change does not pass checks (e.g. fmt,
lint). Please fix them before submitting the PR.

## Unit tests

Before running your code in a real Kubernetes cluster, make sure it passes all unit tests.

```shell
$ make test
```

## Run tikv-operator locally

We uses [kind](https://kind.sigs.k8s.io/docs/user/quick-start/#installation) to
start a Kubernetes cluster locally and
[kubectl](https://kubernetes.io/docs/reference/kubectl/overview/) must be
installed to access Kubernetes cluster.

You can refer to their official references to install them on your machine, or
run the following command to install them into our local binary directory:
`output/bin`.

```shell
$ hack/local-up-operator.sh -i
$ export PATH=$(pwd)/output/bin:$PATH
```

Make sure they are installed correctly:

```
$ kind --version
...
$ kubectl version --client
...
```

Create a Kubernetes cluster with `kind`:

```shell
$ kind create cluster
```

Build and run tidb-operator:

```shell
$ ./hack/local-up-operator.sh
```

Start a basic TiKV cluster:

```shell
$ kubectl apply -f examples/basic/tikv-cluster.yaml
```
