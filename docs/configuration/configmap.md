# Kubernetes configuration

When deploying AMD Device Config Manager on Kubernetes, a `ConfigMap` is deployed in the configmanager namespace.

## Configuration parameters

- Please find an example config map in [_example/configmap.yaml_](https://github.com/ROCm/device-config-manager/blob/main/example/configmap.yaml#L1)
- Make sure to apply the config map in the configmanager namespace before deploying the DCM pod.
- Example config map and it's meaning

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: config-manager-config
  namespace: kube-amd-gpu
data:
  config.json: |
    {
      "gpu-config-profiles":
      {
          "default":
          {
              "skippedGPUs": {
                  "ids": []
              },
              "profiles": [
                  {
                      "computePartition": "CPX", 
                      "memoryPartition": "NPS1",
                      "numGPUsAssigned": 6
                  },
                  {
                      "computePartition": "SPX", 
                      "memoryPartition": "NPS1",
                      "numGPUsAssigned": 2
                  }
              ]
          },
          "profile-1":
          { 
              "skippedGPUs": {
                  "ids": [0, 1, 2]
              },
              "profiles": [
                  {
                      "computePartition": "CPX",
                      "memoryPartition": "NPS1",
                      "numGPUsAssigned": 5
                  }          
              ]
          }
      }
    }

```

- `gpu-config-profiles` defines a set of partitioning config profiles from which the user can choose the profile he wants to apply.
- `default` and `profile-1` are example profile names.
- `skippedGPUs` (Optional) list of GPU IDs to skip partitioning
- `computePartition` compute partition type
- `memoryPartition` memory partition type
- `numGPUsAssigned` number of GPUs to be partitioned on the node
- NOTE: User can also create a heterogenous partitioning config profile by mentioning different sets, each set having info about compute/memory types and the number of GPUs to have that partition (refer `default` profile example)
   
## Configmap Profile Checks

- Let's assume a node with 8 GPUs in it.
### List of profiles checks
- Total number of all `numGPUsAssigned` values of a single profile must be equal to the total number of GPUs on the node.
    - In `default` profile, you can observe that, we are requesting 6 GPUs of type CPX-NPS1 and 2 GPUs of SPX-NPS1 which is valid since it comes to a total of 8 GPUs
    - If `skippedGPUs` field is present, we need to account for those IDs as well.
    - Hence, `Sum of numGPUsAssigned + len(skippedGPUs) = TotalGPUCount`
- `skippedGPUs` field
    - GPU IDs in the list can range from `0` to `total number of GPUs - 1`
    - Length of list must be equal to `total number of GPUs` - `sum of numGPUsAssigned` in that profile
        - Example, in `profile-1`, we have 5 GPUs set to CPX-NPS1 and exactly 3 more GPU IDs mentioned in the skip list
- Compute types supported are SPX and CPX.
    - Beta stage: DPX, QPX
- Memory types supported are NPS1, NPS2 and NPS4
    - NPS4 is supported only for CPX compute type
    - Combination of any two memory types cannot be used in a single profile
    - NPS2 is supported only for DPX compute type