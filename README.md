# AMD Device Config Manager

AMD Device Config Manager Helm Chart Repository.

## Quick Start
```bash
# Install Helm
curl -fsSL -o get_helm.sh https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3
chmod 700 get_helm.sh
./get_helm.sh

# Install Helm Charts
helm repo add dcm https://rocm.github.io/device-config-manager
helm repo update
helm install dcm dcm/device-config-manager-charts --namespace kube-amd-gpu --create-namespace
```