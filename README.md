# device-config-manager
Device config manager(DCM) is a component of the GPU Operator which is used to handle AMD Devices' configuration. To begin with, we will be handling the GPU partitioning configurations, but it will be flexible to support any kind of GPU configurations (or AINIC configurations) in the future.
Users will provide the GPU configurations using a K8s config-map. The config-map will be associated with the DCM daemonset.

## Supported Platforms
  - Ubuntu 22.04

## RDC version
  - ROCM 6.3, ROCM 6.4

## Documentation

For detailed documentation including installation guides, configuration options, and partition descriptions, see the [documentation](https://instinct.docs.amd.com/projects/gpu-operator/en/latest/dcm/device-config-manager.html).

## License

This project is licensed under the Apache 2.0 License - see the [LICENSE](LICENSE) file for details.