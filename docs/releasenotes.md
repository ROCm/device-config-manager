# Release Notes

## v1.3.0

### Release Highlights

- **GPU Partition using Device Config Manager**
  - Device config manager can be deployed as a daemonset inside your k8s cluster to partition GPUs
  - DCM can be deployed using AMD GPU operator or as a standalone daemonset
  - Supported compute type partitions are:
    - Compute Partitions: SPX, CPX (also DPX and QPX in beta stage)
    - Memory Partitions: NPS1, NPS2, NPS4
  - Partition Status can be seen through k8s labels
    - `dcm.amd.com/gpu-config-profile-state`
    - Users can also check the k8s events raised from DCM pod to check the partition status

### Platform Support
ROCm 6.4.x