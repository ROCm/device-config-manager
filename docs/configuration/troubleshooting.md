# Troubleshooting Device Config Manager

This topic provides an overview of troubleshooting options for Device Config Manager.

## Logs
You can view the container logs by executing the following command:

### K8s deployment
```bash
kubectl logs -n <namespace> <configmanager-container-on-node>
```

## Common Issues

This section describes common issues with AMD Device Config Manager

1. Device access:
   - Ensure proper permissions on `/dev/dri` and `/dev/kfd`
   - Verify ROCm is properly installed

2. Driver/ROCM issues:
   - Check GPU driver status
   - Verify ROCm version compatibility
