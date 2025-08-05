# Kubernetes (Helm) installation

This page explains how to install AMD Device Config Manager using Kubernetes.

## System requirements

- ROCm 6.2.0
- Ubuntu 22.04 or later
- Kubernetes cluster v1.29.0 or later
- Helm v3.2.0 or later
- `kubectl` command-line tool configured with access to the cluster

## Installation

For Kubernetes environments, a Helm chart is provided for easy deployment.

- Prepare a `values.yaml` file:

```yaml
platform: k8s

#optional parameter
nodeSelector: {}

image:
  repository: rocm/device-config-manager
  tag: v1.3.0
  pullPolicy: Always

# specify configmap name (mandatory)
configMap: "config-manager-config"
```

- Install using Helm:

```bash
make helm-build
cd ./helm-charts
helm install amd-gpu-operator \
./device-config-manager-charts-v1.3.0.tgz -n kube-amd-gpu \ 
--create-namespace -f values.yaml
```
