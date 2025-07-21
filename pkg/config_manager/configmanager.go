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

package configmanager

/*
#cgo CFLAGS: -I/device-config-manager/build/assets/amd_smi
#cgo LDFLAGS: -L/device-config-manager/build/assets -lamd_smi -ldrm_amdgpu -ldrm
#include "/device-config-manager/build/assets/amdsmi.h"
*/
import "C"
import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"reflect"
	"sync"
	"time"
	"unsafe"

	partition_pb "github.com/ROCm/device-config-manager/gen/partition"
	"github.com/ROCm/device-config-manager/pkg/amdgpu/k8sclient"
	"github.com/ROCm/device-config-manager/pkg/config_manager/globals"
	types "github.com/ROCm/device-config-manager/pkg/config_manager/interface"
	"github.com/fsnotify/fsnotify"
	log_e "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
)

var kc *k8sclient.K8sClient = k8sclient.NewClient(context.Background())
var nodeName string = k8sclient.GetNodeName()

var sockets []C.amdsmi_socket_handle
var totalGPUCount int
var partition_failed bool = false
var partStatus types.PartitionStatus

var (
	mu         sync.Mutex
	cancelFunc context.CancelFunc
	retryCh    = make(chan string, 1) // Signaling channel for retry requests
	wg         sync.WaitGroup
)

const logDivider = "#####################################"
const gpuidDivider = "***************************************************************************************************"

func generateK8sEvent(err error, event_n string, partStatus types.PartitionStatus) {
	k8sPodNamespace := k8sclient.GetPodNameSpace()
	k8sPodName := k8sclient.GetPodName()
	currTime := time.Now().UTC()

	eventType := v1.EventTypeNormal
	reason := event_n
	var message string

	if err != nil {
		eventType = v1.EventTypeWarning
	}

	msgbytes, err := json.Marshal(partStatus)
	if err != nil {
		log_e.Errorf("failed to marshal partition status message %+v err %+v", partStatus, err)
		return
	}
	message = string(msgbytes)

	evtObj := createEventObject(event_n, k8sPodNamespace, k8sPodName, currTime, eventType, reason, message)
	kc.CreateEvent(evtObj)
}

func GetPartitionProfile() (string, error) {

	log.Println(logDivider)
	log.Printf("Partition profile info:\n")
	defer log.Println(logDivider)
	var selectedProfile string
	if nodeName == "" {
		err := errors.New("not a k8s deployment")
		return "", err
	}
	labels, err := kc.GetNodeLabel(nodeName)
	if err != nil {
		return "", err
	}

	if len(labels) != 0 {
		gpuConfigProfileNodeLabel := labels[globals.LabelKey]

		if gpuConfigProfileNodeLabel == "" {
			log.Printf("No profile selected, please select a profile from the configmap to begin partitioning\n")
			return "", nil
		} else {
			selectedProfile = gpuConfigProfileNodeLabel
		}

		log.Printf("Selected profile name: %+v\n", selectedProfile)
	} else {
		log.Printf("No labels present on node, unusual\n")
	}
	return selectedProfile, nil
}

func getAMDSMIStatusString(code int) string {
	if status, exists := globals.AmdsmiStatusStrings[code]; exists {
		return status
	}
	return "UNKNOWN_STATUS"
}

func StartFileWatcher(selectedProfile string) {
	log.Printf("Adding file watcher for %v", globals.JsonFilePath)
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Print(err)
		return
	}
	defer watcher.Close()

	if _, err := os.Stat(globals.JsonFilePath); os.IsNotExist(err) {
		<-make(chan struct{})
	}
	// Add the JSON file to the watcher
	err = watcher.Add(globals.JsonFilePath)
	if err != nil {
		log.Print(err)
		return
	}

	log.Printf("starting file watcher for %v", globals.JsonFilePath)
	// Watch for changes
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					log.Print("Event channel closed")
					return
				}
				if event.Has(fsnotify.Create | fsnotify.Write | fsnotify.Remove | fsnotify.Rename) {
					log.Print("Detected changes in config.json, re-reading the file.")
					selectedProfile, err := GetPartitionProfile()
					if err != nil {
						log_e.Errorf("err: %+v", err)
					}
					if selectedProfile != "" {
						TriggerRetryLoop(selectedProfile, "configmap watcher")
					}
				} else {
					log.Printf("Event %v", event)
				}
				watcher.Remove(globals.JsonFilePath)
				watcher.Add(globals.JsonFilePath)
			case err, ok := <-watcher.Errors:
				if !ok {
					log.Print("Event channel closed, error")
					return
				}
				log.Print("Error:", err)
			}
		}
	}()

	// Keep the program running
	<-make(chan struct{})
}

func convertComputePartitonType(partitionType string) C.amdsmi_compute_partition_type_t {
	switch partitionType {
	case "CPX":
		return C.AMDSMI_COMPUTE_PARTITION_CPX
	case "SPX":
		return C.AMDSMI_COMPUTE_PARTITION_SPX
	case "DPX":
		return C.AMDSMI_COMPUTE_PARTITION_DPX
	case "QPX":
		return C.AMDSMI_COMPUTE_PARTITION_QPX
	default:
		log_e.Errorf("Unknown compute partition type: %s, using default type SPX", partitionType)
		return C.AMDSMI_COMPUTE_PARTITION_SPX // default value
	}
}

func convertMemoryPartitionType(memoryPartition string) C.amdsmi_memory_partition_type_t {
	switch memoryPartition {
	case "NPS1":
		return C.AMDSMI_MEMORY_PARTITION_NPS1
	case "NPS2":
		return C.AMDSMI_MEMORY_PARTITION_NPS2
	case "NPS4":
		return C.AMDSMI_MEMORY_PARTITION_NPS4
	default:
		log_e.Errorf("Unknown memory partition type: %s, using default type NPS1", memoryPartition)
		return C.AMDSMI_MEMORY_PARTITION_NPS1 // default value
	}
}

func amdsmiGetSocketHandles() ([]C.amdsmi_socket_handle, int) {
	var socketCount C.uint32_t
	ret := C.amdsmi_get_socket_handles(&socketCount, nil)
	if ret != C.AMDSMI_STATUS_SUCCESS {
		log_e.Errorf("Failed to get socket count")
		return nil, 0
	}

	// allocating the memory for the sockets
	sockets := make([]C.amdsmi_socket_handle, socketCount)

	// get the actual socket handles
	ret = C.amdsmi_get_socket_handles(&socketCount, &sockets[0])
	if ret != C.AMDSMI_STATUS_SUCCESS {
		log_e.Errorf("Failed to get socket handles")
		return nil, 0
	}

	// return the socket handles and the count
	return sockets, int(socketCount)
}

func amdsmiGetProcessorHandles(socket C.amdsmi_socket_handle) ([]C.amdsmi_processor_handle, int) {
	var device_count C.uint32_t
	ret := C.amdsmi_get_processor_handles(socket, &device_count, nil)
	if ret != C.AMDSMI_STATUS_SUCCESS {
		log_e.Errorf("Failed to get device count")
		return nil, 0
	}

	// allocating the memory for the processor
	processors := make([]C.amdsmi_processor_handle, device_count)

	ret = C.amdsmi_get_processor_handles(socket, &device_count, &processors[0])
	if ret != C.AMDSMI_STATUS_SUCCESS {
		log_e.Errorf("Failed to get processor handles")
		return nil, 0
	}

	// return the socket handles and the device count
	return processors, int(device_count)
}

func amdsmiGetProcessorType(processor_handle C.amdsmi_processor_handle) (C.processor_type_t, error) {
	var processor_type C.processor_type_t
	ret := C.amdsmi_get_processor_type(processor_handle, &processor_type)
	if ret != 0 {
		log.Printf("Error: %d\n", ret)
		err := errors.New("AMD SMI get processor type failed")
		return processor_type, err
	}
	return processor_type, nil
}

func createGPUIDList(filter_ids []uint32, totalGPUCount int) []int {
	result := []int{}
outer:
	for i := range totalGPUCount {
		for _, fID := range filter_ids {
			if int(fID) == i {
				continue outer
			}
		}
		result = append(result, i)
	}
	return result
}

func validateProfile(profile *partition_pb.GPUConfigProfile, totalGPUCount int) error {
	devices_conf_count := len(profile.Profiles)
	profiles := profile.Profiles
	total_devices := 0
	devicefilter := profile.Filters
	if len(devicefilter.Id) > totalGPUCount {
		log.Printf("Device filter count %d exceeding existing GPU count %d in node", len(devicefilter.Id), totalGPUCount)
		err := errors.New("GPU ID list specified in the device filter is invalid, list length is exceeding the total number of GPUs available on this node")
		log.Printf("ERROR %v", err)
		return err
	}

	for _, id := range devicefilter.Id {
		if int(id)+1 > totalGPUCount {
			log.Printf("Invalid GPU ID specified in skippedGPUs list: %v Valid GPU indices : 0 - %v", id, totalGPUCount-1)
			err := errors.New("invalid gpu id")
			return err
		}
	}

	for i := range devices_conf_count {
		nod := profiles[i].NumGPUsAssigned
		total_devices = total_devices + int(nod)
	}
	if total_devices+len(devicefilter.Id) != totalGPUCount {
		err := errors.New("the total of all numGPUsAssigned values across profiles, combined with the count of IDs in the skippedGPUs list, does not equal the total number of GPUs available on this node")
		log.Printf("ERROR %v", err)
		return err
	}
	gpu_ids_list := createGPUIDList(devicefilter.Id, totalGPUCount)
	log.Printf("Usable GPU IDs for partitioning %v", gpu_ids_list)
	currentMemory := profiles[0].MemoryPartition
	for i := 0; i < devices_conf_count; i++ {
		currentCompute := profiles[i].ComputePartition
		err := checkInvalidPartitionType(currentCompute, profiles[i].MemoryPartition)
		if err != nil {
			log.Printf("Invalid partition types %v-%v", currentCompute, currentMemory)
			return err
		}
		if currentMemory != profiles[i].MemoryPartition {
			log.Printf("All profiles must have a common memory type NPS1, NPS2 or NPS4")
			err := errors.New("profile cannot have combination of NPS1, NPS2 and NPS4 memory types")
			return err
		}
		nod := profiles[i].NumGPUsAssigned
		log.Printf("Partitioning %v devices with compute partition type %v and memory type %v", nod, currentCompute, currentMemory)
	}
	log.Println("Profile validation successful")
	return nil
}

func getCurrentGPUComputePartition(processor_handle C.amdsmi_processor_handle) string {
	var len C.uint32_t = 4
	computePartition := make([]C.char, len)
	ret := C.amdsmi_get_gpu_compute_partition(processor_handle, &computePartition[0], len)
	if ret != C.AMDSMI_STATUS_SUCCESS {
		log_e.Errorf("Failed to get compute partition %v", ret)
		return ""
	}
	cStr := (*C.char)(unsafe.Pointer(&computePartition[0]))
	return C.GoString(cStr)
}

func getCurrentGPUMemoryPartition(processor_handle C.amdsmi_processor_handle) string {
	var len C.uint32_t = 5
	memoryPartition := make([]C.char, len)
	ret := C.amdsmi_get_gpu_memory_partition(processor_handle, &memoryPartition[0], len)
	if ret != C.AMDSMI_STATUS_SUCCESS {
		log_e.Errorf("Failed to get memory partition %v", ret)
		return ""
	}
	cStr := (*C.char)(unsafe.Pointer(&memoryPartition[0]))
	return C.GoString(cStr)
}

func populateGPUEventStatus(gpu_id int, partitionType string, status string, message string, idx int) {

	partStatus.GPUStatus[idx].GpuID = gpu_id
	partStatus.GPUStatus[idx].PartitionType = partitionType
	partStatus.GPUStatus[idx].Status = status
	partStatus.GPUStatus[idx].Message = message
}

func amdSMIHelper(selectedProfile string, profile *partition_pb.GPUConfigProfile) {

	log.Print("AMD SMI Initialized successfully.")
	sockets, totalGPUCount = amdsmiGetSocketHandles()
	podList := kc.GetPods(nodeName)
	if totalGPUCount == 0 {
		partStatus.Reason = "Partition failed with reason: 0 sockets found"
		generateK8sEvent(errors.New("no sockets found"), globals.K8EventAMDSMIAPIFailure, partStatus)
		err := kc.AddNodeLabel(nodeName, "dcm.amd.com/gpu-config-profile-state", "failure")
		if err != nil {
			log.Printf("Error adding status node label: %s\n", err.Error())
		}
	}
	var processor_handles []C.amdsmi_processor_handle
	var device_count int
	var gpu_id int
	var partition_err_reason string

	if profile.Filters == nil {
		profile.Filters = &partition_pb.SkippedGPUs{}
		profile.Filters.Id = []uint32{}
	}

	log.Print("Total number of GPUs in the node ", totalGPUCount)
	log.Printf("Skipped GPU IDs for partitioning %v", profile.Filters.Id)
	profiles := profile.Profiles
	idx := 0

	log.Println("\n------------------------------------\n")
	log.Println("\nValidating the selected profile.")
	log.Printf("Profile name: %+v\n", selectedProfile)
	log.Printf("Profile info: %+v\n", profile)
	err := validateProfile(profile, totalGPUCount)
	if err != nil {
		log.Println("Profile validation failed. Could not partition.")
	}
	log.Println("\n------------------------------------\n")
	if err != nil {
		partStatus.Reason = fmt.Sprintf("Partition failed with reason: %v", err)
		generateK8sEvent(err, globals.K8EventInvalidProfile, partStatus)
		err = kc.AddNodeLabel(nodeName, "dcm.amd.com/gpu-config-profile-state", "failure")
		if err != nil {
			log.Printf("Error adding status node label: %s\n", err.Error())
		}
		return
	}
	gpu_ids_list := createGPUIDList(profile.Filters.Id, totalGPUCount)
	// Allocating memory based on gpuCount
	partStatus.GPUStatus = make([]types.GPUPartitionStatus, len(gpu_ids_list))
	partition_needed := false
	partition_failed = false
	for i := 0; i < len(profile.Profiles); i++ {
		currentCompute := profiles[i].ComputePartition
		currentMemory := profiles[i].MemoryPartition
		partitionType := currentCompute + "-" + currentMemory
		nod := profiles[i].NumGPUsAssigned
		for j := 0; j < int(nod); j++ {
			log.Printf("\n%v\n\n", gpuidDivider)
			gpu_id = gpu_ids_list[idx]
			log.Printf("GPU ID %v\n", gpu_id)
			log.Printf("Requested compute partition %v", currentCompute)
			log.Printf("Requested memory partition %v", currentMemory)
			processor_handles, device_count = amdsmiGetProcessorHandles(sockets[gpu_id])
			log.Printf("Existing Device count : %d", device_count)
			processor_handle := processor_handles[0]
			processor_type, err := amdsmiGetProcessorType(processor_handle)
			if err != nil {
				log_e.Errorf("Error %v", err)
				partStatus.Reason = fmt.Sprintf("AMD-SMI API error : %v", err)
				generateK8sEvent(err, globals.K8EventAMDSMIAPIFailure, partStatus)
				err := kc.AddNodeLabel(nodeName, "dcm.amd.com/gpu-config-profile-state", "failure")
				if err != nil {
					log.Printf("Error adding status node label: %s\n", err.Error())
				}
				return
			}
			if processor_type != C.AMDSMI_PROCESSOR_TYPE_AMD_GPU {
				log.Print("Expected AMDSMI_PROCESSOR_TYPE_AMD_GPU device type!\n")
				continue
			}

			existingCompute := getCurrentGPUComputePartition(processor_handle)
			existingMemory := getCurrentGPUMemoryPartition(processor_handle)

			if (currentCompute == existingCompute) && (currentMemory == existingMemory) {
				log.Println("Existing compute and memory partition is same as the requested partition! Skipping partitioning for this GPU !")
				populateGPUEventStatus(gpu_id, partitionType, "Success", "Partition not required", idx)
				idx = idx + 1
				log.Printf("\n%v\n", gpuidDivider)
				continue
			}

			log.Println("Memory partition :")
			partition_needed = true
			if currentMemory != existingMemory {
				log.Println("Triggering memory partition !!")
				log.Printf("Existing memory partition: %s\n", existingMemory)

				memoryType := convertMemoryPartitionType(currentMemory)
				ret_n := C.amdsmi_set_gpu_memory_partition(processor_handle, memoryType)
				updatedMemory := getCurrentGPUMemoryPartition(processor_handle)
				if ret_n != C.AMDSMI_STATUS_SUCCESS || (updatedMemory == existingMemory) {
					partition_err_reason = getAMDSMIStatusString(int(ret_n))
					log_e.Errorf("Failed to memory partition %v \n", partition_err_reason)
					if ret_n == C.AMDSMI_STATUS_BUSY {
						log_e.Errorf("There might be existing pods/daemonsets on the cluster keeping the GPU resource busy, please remove them and retry. Pods list on this node: %v", podList)
					}
					err = kc.AddNodeLabel(nodeName, "dcm.amd.com/gpu-config-profile-state", "failure")
					if err != nil {
						log.Printf("Error adding status node label: %s\n", err.Error())
					}
					partition_failed = true
				} else {
					log.Println("Memory partition successful !!")
					log.Printf("Updated Memory Type %v\n", updatedMemory)
				}
			} else {
				log.Println("Existing and requested memory partition matching! Memory partition not required !!")
			}

			log.Println("Compute partition :")
			existingCompute = getCurrentGPUComputePartition(processor_handle)

			if currentCompute != existingCompute {
				log.Println("Triggering compute partition !!")
				log.Printf("Existing compute partition: %s\n", existingCompute)
				computeType := convertComputePartitonType(currentCompute)

				ret_n := C.amdsmi_set_gpu_compute_partition(processor_handle, computeType)
				// check for change in profile name or config map change

				updatedCompute := getCurrentGPUComputePartition(processor_handle)
				if ret_n != C.AMDSMI_STATUS_SUCCESS {
					partition_err_reason = getAMDSMIStatusString(int(ret_n))
					log_e.Errorf("Failed to compute partition %v \n", partition_err_reason)
					if ret_n == C.AMDSMI_STATUS_BUSY {
						log_e.Errorf("There might be existing pods/daemonsets on the cluster keeping the GPU resource busy, please remove them and retry. Pods list on this node: %v", podList)
					}
					err = kc.AddNodeLabel(nodeName, "dcm.amd.com/gpu-config-profile-state", "failure")
					if err != nil {
						log.Printf("Error adding status node label: %s\n", err.Error())
					}
					partition_failed = true
				} else {
					log.Println("Compute partition successful !!")
					log.Printf("Updated Compute Type %v", updatedCompute)
				}
			} else {
				log.Println("Existing and requested compute partition matching! Compute partition not required !!")
			}
			if partition_failed {
				populateGPUEventStatus(gpu_id, partitionType, "Failure", fmt.Sprintf("Partition failed with reason: %v", partition_err_reason), idx)
				partStatus.Reason = fmt.Sprintf("Partition failed with reason: %v", partition_err_reason)
			} else {
				populateGPUEventStatus(gpu_id, partitionType, "Success", "Successfully partitioned", idx)
			}
			idx = idx + 1
			log.Printf("\n%v\n", gpuidDivider)
		}
		if partition_needed && !partition_failed {
			log.Printf("Successfully Partitioned GPUs of profile %d", i+1)
		}
	}

	if partition_failed {
		log.Printf("Partition failed.")
	} else {
		partStatus.FinalStatus = "Success"
		if partition_needed {
			partStatus.Reason = "All GPUs were successfully partitioned"
			log.Printf("Partition completed successfully")
			generateK8sEvent(nil, globals.K8EventSuccessfullyPartitioned, partStatus)
		} else {
			log.Printf("Partition not required. Requested Partition Config Already Exists on node")
			partStatus.Reason = "Existing GPU's partition configuration same as profile's partition config"
			generateK8sEvent(errors.New("GPU's existing partition configuration same as profile's partition config"), globals.K8EventPartitionNotNeeded, partStatus)
		}

		err = kc.AddNodeLabel(nodeName, "dcm.amd.com/gpu-config-profile-state", "success")
		if err != nil {
			log.Printf("Error adding status node label: %s\n", err.Error())
			return
		}
	}

}

func shutDownAMDSMI() {
	ret := C.amdsmi_shut_down()
	if ret != C.AMDSMI_STATUS_SUCCESS {
		log_e.Errorf("Failed to shutdown AMD SMI!")
	} else {
		log.Printf("AMD SMI shutdown successfully\n")
	}
}

func createEventObject(event_n, k8sPodNamespace, k8sPodName string, currTime time.Time, eventType, reason, message string) *v1.Event {
	return &v1.Event{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: event_n,
			Namespace:    k8sPodNamespace,
		},
		FirstTimestamp: metav1.Time{
			Time: currTime,
		},
		LastTimestamp: metav1.Time{
			Time: currTime,
		},
		Count:   1,
		Type:    eventType,
		Reason:  reason,
		Message: message,
		InvolvedObject: v1.ObjectReference{
			Kind:      "Pod",
			Namespace: k8sPodNamespace,
			Name:      k8sPodName,
		},
		Source: v1.EventSource{
			Host:      nodeName,
			Component: globals.EventSourceComponentName,
		},
	}
}

func ValidateList(config string, validlist []string) bool {
	for _, ctype := range validlist {
		if ctype == config {
			return true
		}
	}
	return false
}

func checkInvalidPartitionType(computeType string, memoryType string) error {

	if !ValidateList(computeType, globals.ValidComputePartitions) {
		err := errors.New("not a valid profile. Invalid compute type")
		return err
	}
	if !ValidateList(memoryType, globals.ValidMemoryPartitions) {
		err := errors.New("not a valid profile. Invalid memory type")
		return err
	}
	return nil
}

func PartitionGPU(selectedProfile string) error {

	var profile *partition_pb.GPUConfigProfile
	var exists bool

	partStatus.SelectedProfile = selectedProfile
	partStatus.GPUStatus = nil
	partStatus.FinalStatus = "Failure"
	log.Println(logDivider)
	log.Printf("Partitioning the GPU\n")
	defer log.Println(logDivider)
	if _, err := os.Stat(globals.JsonFilePath); os.IsNotExist(err) {
		log.Printf("ConfigMap not present, please configure a configmap to proceed")
		partStatus.Reason = "Configmap does not exist"
		generateK8sEvent(errors.New("configmap not found"), globals.K8EventConfigMapNotPresent, partStatus)
		err = kc.AddNodeLabel(nodeName, "dcm.amd.com/gpu-config-profile-state", "failure")
		if err != nil {
			log.Printf("Error adding status node label: %s\n", err.Error())
		}
		return nil
	} else {
		log.Printf("Reading configmap: %v\n", globals.JsonFilePath)
	}

	var profiles partition_pb.GPUConfigProfiles
	file, _ := ioutil.ReadFile(globals.JsonFilePath)
	err := json.Unmarshal(file, &profiles)
	if err != nil {
		log_e.Errorf("Failed to unmarshal JSON: %v", err)
		partStatus.Reason = "Invalid JSON inside configmap"
		generateK8sEvent(errors.New("invalid json in configmap"), globals.K8EventInvalidJSONInConfigMap, partStatus)
		err = kc.AddNodeLabel(nodeName, "dcm.amd.com/gpu-config-profile-state", "failure")
		if err != nil {
			log.Printf("Error adding status node label: %s\n", err.Error())
		}
		return nil
	}

	profile, exists = profiles.ProfilesList[selectedProfile]
	if exists {
		log.Printf("Selected Profile %v found in the configmap.\n", selectedProfile)
	} else {
		log.Printf("Selected Profile %v not found.\n", selectedProfile)
		partStatus.Reason = "Profile does not exist in the configmap"
		generateK8sEvent(errors.New("profile not found"), globals.K8EventNonExistentProfile, partStatus)
		err = kc.AddNodeLabel(nodeName, "dcm.amd.com/gpu-config-profile-state", "failure")
		if err != nil {
			log.Printf("Error adding status node label: %s\n", err.Error())
		}
		return nil
	}

	// Initialize the AMD SMI library for GPU
	ret := C.amdsmi_init(C.AMDSMI_INIT_AMD_GPUS)
	if ret != C.AMDSMI_STATUS_SUCCESS {
		log_e.Errorf("Failed to initialize AMD SMI!")
		partStatus.Reason = "AMD-SMI API error : Failed to initialize AMD SMI!"
		generateK8sEvent(err, globals.K8EventAMDSMIAPIFailure, partStatus)
		err := kc.AddNodeLabel(nodeName, "dcm.amd.com/gpu-config-profile-state", "failure")
		if err != nil {
			log.Printf("Error adding status node label: %s\n", err.Error())
		}
		return nil
	}
	defer shutDownAMDSMI()
	amdSMIHelper(selectedProfile, profile)
	if partition_failed {
		return errors.New("partition failed")
	} else {
		return nil
	}
}

func printAndApplyLabelChanges(oldLabels, newLabels map[string]string) {
	// Check for added or updated labels
	for key, newVal := range newLabels {
		if key == globals.LabelKey && newVal != "" {
			if oldVal, exists := oldLabels[key]; !exists || oldVal != newVal {
				log.Printf("\nNEW TRIGGER ALERT FROM NODE LABELS\n")
				log.Printf("Label changed: %s\nOld value: %s\nNew value: %s\n", key, oldVal, newVal)
				selectedProfile, err := GetPartitionProfile()
				if err != nil {
					log_e.Errorf("err: %+v", err)
				}
				if selectedProfile != "" {
					TriggerRetryLoop(selectedProfile, "nodelabel watcher")
				}
			}
		}
	}

	// Check for removed labels
	for key, oldVal := range oldLabels {
		if _, exists := newLabels[key]; !exists {
			if key == globals.LabelKey {
				log.Printf("Label removed: %s\nOld value: %s\n", key, oldVal)
			}
		}
	}
}

func NodeLabelWatcher() {

	nodeInformer := kc.GetNodeInformer(nodeName)

	// Set up event handlers for the node informer
	nodeInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldNode := oldObj.(*v1.Node)
			newNode := newObj.(*v1.Node)
			if !reflect.DeepEqual(oldNode.Labels, newNode.Labels) {
				printAndApplyLabelChanges(oldNode.Labels, newNode.Labels)
			}
		},
	})

	// Start the informer
	stopCh := make(chan struct{})
	defer close(stopCh)

	go func() {
		// Creating a timer to prevent blockage of code execution
		timer := time.NewTimer(100 * time.Second)
		<-timer.C
		// Stop the Node Informer after the timer expires
	}()

	go nodeInformer.Run(stopCh)

	// Wait for the informer to sync
	if !cache.WaitForCacheSync(stopCh, nodeInformer.HasSynced) {
		log_e.Errorf("Failed to sync informers")
	}

	log.Print("Node Informer started and will run for 100 seconds.")
	// Keep the function running
	<-make(chan struct{})
}

func RetryPartition(ctx context.Context, selectedProfile string) {
	defer wg.Done()
	expiration := time.Now().Add(30 * time.Minute)
	count := 1

	for {
		select {
		case <-ctx.Done():
			log.Println("Aborting retry loop")
			return
		default:
			// Allow retry logic to continue if no cancellation signal is received
		}

		// from here this
		if time.Now().After(expiration) {
			generateK8sEvent(errors.New("partition failed"), globals.K8EventPartitionFailed, partStatus)
			log.Println("Retry loop expired after retrying for 30 mins")
			return
		}

		log.Printf("Calling PartitionGPU...\n")

		if err := PartitionGPU(selectedProfile); err != nil {
			log.Printf("Error occurred in PartitionGPU: %v\n", err)
			log.Println("Waiting for 1 minute before retrying...")
			if count == 1 {
				count = count + 1
				partStatus.FinalStatus = "Partition failed, retrying."
				partStatus.Reason = fmt.Sprintf("Partition retrying for profile: %v", selectedProfile)
				generateK8sEvent(errors.New("partition retrying"), globals.K8EventPartitionRetrying, partStatus)
			}
			// Wait for 1 minute or exit early if context is canceled
			select {
			case <-time.After(1 * time.Minute): // Wait 1 minute for retry
				// Proceed to the next iteration of the retry loop
			case <-ctx.Done(): // Exit the retry loop if context is canceled
				log.Println("Aborting retry loop during wait due to cancellation")
				return
			}
		} else {
			log.Println("PartitionGPU executed successfully")
			return
		}
	}
}

// Worker function to handle retry signals
func Worker() {
	for prof := range retryCh {
		mu.Lock()
		if cancelFunc != nil {
			log.Println("Calling cancelled")
			cancelFunc()
			mu.Unlock()
			wg.Wait()
			mu.Lock()
		}
		ctx, cancel := context.WithCancel(context.Background())
		cancelFunc = cancel
		wg.Add(1)

		log.Printf("New trigger, calling PartitionGPU.")
		go RetryPartition(ctx, prof)
		mu.Unlock()
	}
}

func TriggerRetryLoop(selectedProfile string, funcname string) {
	select {
	case retryCh <- selectedProfile: // Signal a retry request
		log.Printf("Triggering new retry loop from %s\n", funcname)
	default:
		log.Println("Retry loop already pending, ignoring trigger")
	}
}
