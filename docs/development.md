# Development Guide

This document walks you through the process for TiKV Operator development.

<!-- toc -->
- [Prerequisites](#prerequisites)
- [Verify code changes](#verify-code-changes)
- [Run unit tests](#run-unit-tests)
- [Run tikv-operator locally](#run-tikv-operator-locally)
<!-- /toc -->

## Prerequisites

* [golang](https://golang.org): version >= 1.13
* [Docker](https://docs.docker.com/get-started/): the latest version is recommended

## Verify code changes

Run the following commands to verify your code changes:

```shell
$ make verify
```

This will show errors if your code changes fail to pass checks (e.g. fmt, lint). If there is any error, fix them before submitting the PR.

## Run unit tests

Before running your code in a real Kubernetes cluster, make sure it passes all unit tests:

```shell
$ make test
```

## Run tikv-operator locally

The following steps use [kind](https://kind.sigs.k8s.io) to start a Kubernetes cluster locally.

1. Install kind and kubectl:

    You can refer to [the official installation steps](https://kind.sigs.k8s.io/docs/user/quick-start/#installation) to install them on your machine, or run the following command to install them into the local binary directory `output/bin`:

    ```shell
    $ hack/local-up-operator.sh -i
    $ export PATH=$(pwd)/output/bin:$PATH
    ```

2. Make sure they are installed correctly:

    ```
    $ kind --version
    ...
    $ kubectl version --client
    ...
    ```

3. Create a Kubernetes cluster:

    ```shell
    $ kind create cluster
    ```

4. Build and run tidb-operator:

    ```shell
    $ ./hack/local-up-operator.sh
    ```

5. Start a basic TiKV cluster:

    ```shell
    $ kubectl apply -f examples/basic/tikv-cluster.yaml
    ```
