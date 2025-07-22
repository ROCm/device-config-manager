# Device Config Manager

Device config manager(DCM) is a component of the GPU Operator which is used to handle AMD Devices' configuration. To begin with, we will be handling the GPU partitioning configurations, but it will be flexible to support any kind of GPU configurations (or AINIC configurations) in the future. Users will provide the GPU configurations using a K8s config-map. The config-map will be associated with the DCM daemonset.

## Configure device config manager

To start the Device Config Manager along with the GPU Operator configure fields under the ``` spec/configManager ``` field in deviceconfig Custom Resource(CR)

```yaml
  configManager:
    # To enable/disable the config manager, enable to partition
    enable: True

    # image for the device-config-manager container
    image: "rocm/device-config-manager:v1.3.0"

    # image pull policy for config manager set to always to pull image of latest version
    imagePullPolicy: Always

    # specify configmap name which stores profile config info
    config: 
      name: "config-manager-config"

    # DCM pod deployed either as a standalone pod or through the GPU operator will have 
    # a toleration attached to it. User can specify additional tolerations if required
    # key: amd-dcm , value: up , Operator: Equal, effect: NoExecute 

    # OPTIONAL
    # toleration field for dcm pod to bypass nodes with specific taints
    configManagerTolerations:
      - key: "key1"
        operator: "Equal" 
        value: "value1"
        effect: "NoExecute"

```

The **device-config-manager** pod start after updating the **DeviceConfig** CR