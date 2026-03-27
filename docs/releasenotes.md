# Release Notes

## v1.5.0

- **Bug Fixes and Stability Improvements**

## v1.4.1

### Release Highlights

- **Bug Fixes and Stability Improvements**
  - Enhanced robustness and error handling for GPU partition configuration in Kubernetes clusters
  - Enhanced driver reload support for ROCm 7.0.x with KMM (Kernel Module Management)

### Platform Support
ROCm 7.1.x, 7.2.x

## v1.4.0

### Release Highlights

 - **MI35X Support**
    - Add support for MI35X series GPUs to enable the configuration of GPU partitions.

### Platform Support
ROCM 7.0.x

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