# Developer Guide

This document provides build instructions and guidance for developers working on the AMD Device Config Manager repository.

## Git submodule setup

Make sure to update the submodules on every pull from the repository.
```bash
git submodule update --init --recursive
```

## Environment Setup

The project Makefile provides a easy way to create a docker build container that packages the Docker and Go versions needed to build this repository. The following environment variables can be set, either directly or via a `dev.env` file:

- `DOCKER_REGISTRY`: Docker registry (default: `docker.io/rocm`).
- `DOCKER_BUILDER_TAG`: Docker build container tag (default: `v1.0`).
- `BUILD_BASE_IMAGE`: Base image for Docker build container (default: `ubuntu:22.04`).
- `UBUNTU_VERSION`: Ubuntu version for builds (`jammy` for 22.04, `noble` for 24.04).

## Build Prerequisites

Before starting, ensure you have Docker installed and running with the user permissions set appropriately.

## Quick Start

To quickly build everything using Docker:
```bash
make default
```

The default target creates a docker build container that packages the developer tools required to build all other targets in the Makefile and builds the `amdsmi-build-ub22` and `all` targets in this build container.
The target generates a container image `docker.io/rocm/device-config-manager:rocm_dcm_v130` which can be used to deploy the DCM pod in k8s environment.

## Building Components

### Build and Launch Docker Build Container Shell

Run the following command to start a Docker-based build container shell:

```bash
make docker-shell
```

This gives you an interactive Docker environment with necessary tools pre-installed. It is recommended to run all other Makefile targets in this build environment.

### Compiling the AMD Device Config Manager

To compile from within the build environment, run:

```bash
make all
```

This command builds:
- AMD Device Config Manager
- Proto-generated code
- AMD Device Config Manager docker

### Build Docker images

Build standard dcm image:

```bash
make dcm-docker
```

### Helm Chart Packaging

To package Helm charts:

```bash
make helm-install

cd /home/amd/user/device-config-manager/helm-charts; helm lint
==> Linting .
[INFO] Chart.yaml: icon is recommended

1 chart(s) linted, 0 chart(s) failed
helm package helm-charts/ --destination ./helm-charts
Successfully packaged chart and saved it to: helm-charts/device-config-manager-charts-v1.0.0.tgz
cd /home/amd/user/device-config-manager/helm-charts; helm install amd-gpu-operator ./device-config-manager-charts-v1.0.0.tgz -n kube-amd-gpu --create-namespace -f values.yaml
NAME: amd-gpu-operator
LAST DEPLOYED: Thu Apr 3 04:57:29 2025
NAMESPACE: kube-amd-gpu
STATUS: deployed
REVISION: 1
TEST SUITE: None
```
- This internally builds the helm-charts of DCM and then installs the charts in `kube-amd-gpu` namespace.
- DCM daemonset pod is now up and users can perform the partitioning using the labels approach as mentioned above.
- Users can also try the `make helm-build` command to build the helm-charts.

### E2E Testing

Run tests:

```bash
make e2e
```

- Tests are triggered from [_test/k8s-e2e/suite_test.go_](https://github.com/ROCm/device-config-manager/blob/main/test/k8s-e2e/suite_test.go#L1)
    - We have variables defined to specify image name, helm-chart, image tag and so on. Example: [_test/k8s-e2e/suite_test.go#21_](https://github.com/ROCm/device-config-manager/blob/main/test/k8s-e2e/suite_test.go#L21)
    - User can change these parameters to their requirements and run the tests.

- Testcases can be viewed/added in [_test/k8s-e2e/dcm_test.go_](https://github.com/ROCm/device-config-manager/blob/main/test/k8s-e2e/dcm_test.go#L1)

- We use the following command

```bash
go test -failfast -helmchart $(TOP_DIR)/helm-charts/ -test.timeout=30m -v
# To run a specific test case
go test -check.f Test001FirstDeplymentDefaults -helmchart $(TOP_DIR)/helm-charts/ -test.timeout=30m -v
# To run testcases matching a particular string, we can use wildcards
go test -check.f Test00* -helmchart $(TOP_DIR)/helm-charts/ -test.timeout=30m -v
```

## Build AMD SMI
This is a built out of [AMD SMI Lib](git@github.com:ROCm/amdsmi.git), to
access AMD GPU hardware driver

#### Build Container (one time)
```bash
make amdsmi-build
```

#### Compile AMDSMI
```bash
make amdsmi-compile
```

## Deploying Standalone DCM on a cluster
- Create a cluster and setup a worker node to deploy DCM.
- DCM pod can be deployed using it's independent helm-charts as a standalone daemonset without the need of a GPU Operator.
- Steps to deploy:
    - Populate values.yaml to specify image name, tag , nodeSelector, etc.
        - Please find an example values.yaml file in [_helm-charts/values.yaml_](https://github.com/ROCm/device-config-manager/blob/main/helm-charts/values.yaml#L1)
    - Run the below command to build the helm-chart using the values.yaml.

### Partitioning GPUs using DCM
-  GPU on the node cannot be partitioned on the go, we need to bring down all daemonsets using the GPU resource before partitioning. Hence we need to taint the node and the partition.
- DCM pod comes with a toleration
    - `key: amd-dcm , value: up , Operator: Equal, effect: NoExecute `
    - User can specify additional tolerations if required

### Steps for deploying DCM pod
- Add tolerations to the required pods
- Taint the node
- Deploy the DCM pod using a custom resource file
- Once partition is done, untaint the node

#### Taint
-  To TAINT a specific node for partitioning the GPU:
```bash
kubectl taint nodes asrock-126-b3-3b amd-dcm=up:NoExecute
```
- To TAINT a node for partitioning in a `single node cluster`, we can use the `NoSchedule` effect rather than a `NoExecute` effect to prevent eviction of existing control-plane pods.
```bash
kubectl taint nodes asrock-126-b3-3b amd-dcm=up:NoSchedule
```
- Since DCM comes up with a toleration for `NoExecute` by default, user has to add an extra toleration to support the `NoSchedule` taint.

#### Add toleration for the taint
-  Since tainting a node will bring down all pods/daemonsets, we need to add toleration to the pods to prevent it from getting evicted.
-  Add toleration to system level pods as well like flannel, proxy etc before tainting the node.
```bash
Example:
kubectl get ds -n kube-flannel kube-flannel-ds -o yaml > fnl.yaml

amd@asrock-126-b3-3b:~$ vi fnl.yaml

#Add this under the spec.template.spec.tolerations object
tolerations:
      - key: "amd-dcm"
        operator: "Equal"
        value: "up"
        effect: "NoExecute" # Replace with NoSchedule for single node cluster
amd@asrock-126-b3-3b:~$ kubectl apply -f nfd.yaml
```
#### Deploy DCM using a custom resource file
-  Create a CR to bring up the DCM daemonset.
-  Sample CR can be found in [_example/deviceConfigs_example.yaml_](https://github.com/ROCm/device-config-manager/blob/main/example/deviceConfigs_example.yaml#L1)

#### Untaint
```bash
kubectl taint nodes asrock-126-b3-3b amd-dcm:NoExecute-
```
- For single node cluster
```bash
kubectl taint nodes asrock-126-b3-3b amd-dcm:NoSchedule-
```