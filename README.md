# device-config-manager
Device config manager(DCM) is a component of the GPU Operator which is used to handle AMD Devices' configuration. To begin with, we will be handling the GPU partitioning configurations, but it will be flexible to support any kind of GPU configurations (or AINIC configurations) in the future.
Users will provide the GPU configurations using a K8s config-map. The config-map will be associated with the DCM daemonset.

## Quick Start

```bash
make default
```

The default target creates a docker build container that packages the developer tools required to build all other targets in the Makefile and builds the `amdsmi-build-rhel` and `all` targets in this build container.
The target generates a container image `docker.io/rocm/device-config-manager:rocm_dcm_v131` which can be used to deploy the DCM pod in k8s environment.
For more details, please refer to [_docs/developerguide.md_](https://github.com/ROCm/device-config-manager/blob/v1.3.1/docs/developerguide.md#L1)

## Supported Platforms
  - Ubuntu 22.04, Ubuntu 24.04

## RDC version
  - ROCM 6.3, ROCM 6.4

## Documentation

For detailed documentation including installation guides, configuration options, and partition descriptions, see the [documentation](https://instinct.docs.amd.com/projects/gpu-operator/en/latest/dcm/device-config-manager.html).

## License

This project is licensed under the Apache 2.0 License - see the [LICENSE](LICENSE) file for details.