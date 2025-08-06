/*
Copyright (c) Advanced Micro Devices, Inc. All rights reserved.

Licensed under the Apache License, Version 2.0 (the \"License\");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an \"AS IS\" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package globals

const (
	// config map json path inside k8
	JsonFilePath            = "/etc/config-manager/config.json"
	DefaultComputePartition = "SPX"
	DefaultMemoryPartition  = "NPS1"
	DefaultProfileName      = "default"
	LabelKey                = "dcm.amd.com/gpu-config-profile"
	TriggerLabelKey         = "dcm.amd.com/apply-gpu-config-profile"

	EventSourceComponentName       = "amd-device-config-manager"
	K8EventInvalidComputeType      = "InvalidComputeType"
	K8EventInvalidMemoryType       = "InvalidMemoryType"
	K8EventNoPartition             = "NodeNotTaintedBeforeParition"
	K8EventPartitionFailed         = "PartitionFailure"
	K8EventInvalidProfile          = "InvalidProfileInfo"
	K8EventNonExistentProfile      = "NonExistentProfile"
	K8EventSuccessfullyPartitioned = "SuccessfullyPartitioned"
	K8EventPartitionNotNeeded      = "RequestedPartitionConfigAlreadyExists"
	K8EventPartitionRetrying       = "PartitionRetrying"
	K8EventConfigMapNotPresent     = "ConfigMapNotPresent"
	K8EventInvalidJSONInConfigMap  = "InvalidJSONInConfigMap"
	K8EventAMDSMIAPIFailure        = "AMDSMIAPIFailure"
)

var ValidComputePartitions = []string{"SPX", "CPX", "DPX", "QPX"}
var ValidMemoryPartitions = []string{"NPS1", "NPS2", "NPS4"}

// Map of AMD SMI status codes to their descriptions based on
// https://rocm.docs.amd.com/projects/amdsmi/en/docs-6.3.0/doxygen/docBin/html/amdsmi_8h.html#ab05c37a8d1e512898eef2d25fb9fe06b
var AmdsmiStatusStrings = map[int]string{
	0:          "Call succeeded.",
	1:          "Invalid parameters.",
	2:          "Command not supported.",
	3:          "Not implemented yet.",
	4:          "Fail to load lib.",
	5:          "Fail to load symbol.",
	6:          "Error when calling libdrm.",
	7:          "API call failed.",
	8:          "Timeout in API call.",
	9:          "Retry operation.",
	10:         "Permission Denied.",
	11:         "An interrupt occurred during execution of function.",
	12:         "I/O Error.",
	13:         "Bad address.",
	14:         "Problem accessing a file.",
	15:         "Not enough memory.",
	16:         "An internal exception was caught.",
	17:         "The provided input is out of allowable or safe range.",
	18:         "An error occurred when initializing internal data structures.",
	19:         "An internal reference counter exceeded INT32_MAX.",
	30:         "Device busy.",
	31:         "Device not found.",
	32:         "Device not initialized.",
	33:         "No more free slots.",
	34:         "Processor driver not loaded.",
	40:         "No data was found for a given input.",
	41:         "Not enough resources were available for the operation.",
	42:         "An unexpected amount of data was read.",
	43:         "The data read or provided to function is not what was expected.",
	44:         "System has a different CPU than AMD.",
	45:         "Energy driver not found.",
	46:         "MSR driver not found.",
	47:         "HSMP driver not found.",
	48:         "HSMP not supported.",
	49:         "HSMP message/feature not supported.",
	50:         "HSMP message timeout.",
	51:         "No Energy and HSMP driver present.",
	52:         "File or directory not found.",
	53:         "Parsed argument is invalid.",
	54:         "AMDGPU restart failed.",
	55:         "Setting is not available.",
	0xFFFFFFFE: "The internal library error did not map to a status code.",
	0xFFFFFFFF: "An unknown error occurred.",
}
