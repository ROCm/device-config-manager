package k8e2e

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/stretchr/testify/assert"
	. "gopkg.in/check.v1"
	corev1 "k8s.io/api/core/v1"
)

var (
	dcmPod        *corev1.Pod
	configmapName = "test-e2e-config"
	nodePort      = 80
)

const GpuConfigProfileStateLabel = "dcm.amd.com/gpu-config-profile-state"
const GpuConfigProfileLabel = "dcm.amd.com/gpu-config-profile"

func (s *E2ESuite) addRemoveNodeLabels(nodeName string, selectedProfile string, computePartition bool) {
	ctx := context.Background()
	err := s.k8sclient.AddNodeLabel(ctx, nodeName, "dcm.amd.com/gpu-config-profile", selectedProfile)
	if err != nil {
		log.Printf("Error adding node lbels: %s\n", err.Error())
		return
	}
	if computePartition {
		time.Sleep(10 * time.Second)
	} else {
		// Memory partition requires reloading drivers which takes upto a minute
		time.Sleep(60 * time.Second)
	}

	// Allow partition to happen
	err = s.k8sclient.DeleteNodeLabel(ctx, nodeName, "dcm.amd.com/gpu-config-profile")
	if err != nil {
		log.Printf("Error removing node labels: %s\n", err.Error())
		return
	}
}

func (s *E2ESuite) getWorkerNode(c *C, ctx context.Context) string {
	labelMap := make(map[string]string)
	labelMap["feature.node.kubernetes.io/amd-gpu"] = "true"

	nodes, err := s.k8sclient.GetNodesByLabel(ctx, labelMap)
	if err != nil {
		log.Printf("Error getting nodes: %s\n", err.Error())
		assert.Fail(c, "Error getting worker node")
		return ""
	}
	if len(nodes) == 0 {
		log.Printf("No worker nodes present")
		assert.Fail(c, "Error getting worker node")
		return ""
	}
	worker_node := nodes[0].Name
	log.Printf("Selected Worker Node: %v", worker_node)

	return worker_node
}

func validateNodeLabels(c *C, labels map[string]string, negativeTC bool) {
	if len(labels) != 0 {
		gpuConfigProfileState := labels[GpuConfigProfileStateLabel]
		gpuConfigProfile := labels[GpuConfigProfileLabel]
		log.Printf("gpuConfigProfileState: %v\n", gpuConfigProfileState)
		log.Printf("gpuConfigProfile: %v\n", gpuConfigProfile)
		if negativeTC {
			if gpuConfigProfileState != "failure" {
				log.Printf("Negative test case failure: GPUConfigProfileState label reporting partition as success\n")
				assert.Fail(c, "Not expected (negative test case): GPUConfigProfileState -> Success")
			} else {
				log.Printf("Negative test case passed: GPUConfigProfileState label reporting partition as failure\n")
			}
		} else {
			if gpuConfigProfileState != "success" {
				log.Printf("GPUConfigProfileState label reporting partition as failure\n")
				assert.Fail(c, "GPUConfigProfileState -> Failure")
			} else {
				log.Printf("GPUConfigProfileState label reporting partition as success\n")
			}
		}
	}
}

func (s *E2ESuite) Test001FirstDeplymentDefaults(c *C) {
	ctx := context.Background()
	worker_node := s.getWorkerNode(c, ctx)
	// remove existing label for profile selection if any
	err := s.k8sclient.DeleteNodeLabel(ctx, worker_node, "dcm.amd.com/gpu-config-profile")
	if err != nil {
		log.Printf("Error removing node labels: %s\n", err.Error())
		return
	}
	log.Print("Testing helm install for exporter")
	fmt.Printf("image.repository=%v\n", s.registry)
	fmt.Printf("image.tag=%v\n", s.imageTag)
	fmt.Printf("configMap=%v\n", configmapName)
	fmt.Printf("platform=%v\n", s.platform)
	fmt.Printf("service.NodePort.nodePort=%d\n", nodePort)
	fmt.Printf("configMap=%v\n", configmapName)
	values := []string{
		fmt.Sprintf("image.repository=%v", s.registry),
		fmt.Sprintf("image.tag=%v", s.imageTag),
		fmt.Sprintf("service.NodePort.nodePort=%d", nodePort),
		fmt.Sprintf("service.type=NodePort"),
		fmt.Sprintf("configMap=%v", configmapName),
		"image.pullPolicy=IfNotPresent",
	}

	err = s.k8sclient.CreateConfigMap(ctx, s.ns, configmapName)
	if err != nil {
		log.Printf("Failed to create config map %v", configmapName)
	}
	rel, err := s.helmClient.InstallChart(ctx, s.helmChart, values)
	if err != nil {
		log.Printf("failed to install charts")
		assert.Fail(c, err.Error())
		return
	}
	log.Printf("helm installed configmanager relName :%v err:%v", rel, err)
	log.Printf("sleep for 20s for pod to be ready")
	time.Sleep(20 * time.Second)
	labelMap := map[string]string{"app": "amdgpu-device-config-manager"}
	assert.Eventually(c, func() bool {
		pods, err := s.k8sclient.GetPodsByLabel(ctx, s.ns, labelMap)
		if err != nil {
			log.Printf("label get pod err %v", err)
			return false
		}
		log.Printf("pods : %+v", pods)
		if len(pods) >= 1 {
			for _, pod := range pods {
				if pod.Status.Phase == "Running" {
					dcmPod = &pod
					break
				}
			}
			return true
		}
		return false
	}, 2*time.Minute, 10*time.Second)
	assert.Eventually(c, func() bool {
		err := s.k8sclient.ValidatePod(ctx, s.ns, dcmPod.Name)
		if err != nil {
			log.Printf("label get pod err %v", err)
			return false
		}
		return true
	}, 10*time.Second, 1*time.Second)

	log.Print("Successfully deployed DCM Pod")
}

func (s *E2ESuite) Test002DCMDefaultPartitioning(c *C) {
	ctx := context.Background()

	worker_node := s.getWorkerNode(c, ctx)
	log.Printf("Adding node label to select profile: default\n")
	s.addRemoveNodeLabels(worker_node, "default", true)
	labels, err := s.k8sclient.GetNodeLabel(ctx, worker_node)
	if err != nil {
		log.Printf("Error in getting node labels")
		assert.Fail(c, err.Error())
		return
	}
	validateNodeLabels(c, labels, false)
}

func (s *E2ESuite) Test003DCMHeterogenousPartitioning(c *C) {
	ctx := context.Background()

	worker_node := s.getWorkerNode(c, ctx)
	log.Printf("Adding node label to select profile: e2e_profile1\n")
	s.addRemoveNodeLabels(worker_node, "e2e_profile1", true)
	labels, err := s.k8sclient.GetNodeLabel(ctx, worker_node)
	if err != nil {
		log.Printf("Error in getting node labels")
		assert.Fail(c, err.Error())
		return
	}
	validateNodeLabels(c, labels, false)
}

func (s *E2ESuite) Test004DCMInvalidProfiles(c *C) {
	ctx := context.Background()

	labelMap := make(map[string]string)
	labelMap["feature.node.kubernetes.io/amd-gpu"] = "true"

	worker_node := s.getWorkerNode(c, ctx)
	log.Printf("Adding node label to select profile: e2e_profile2")
	s.addRemoveNodeLabels(worker_node, "e2e_profile2", true)
	labels, err := s.k8sclient.GetNodeLabel(ctx, worker_node)
	if err != nil {
		log.Printf("Error in getting node labels")
		assert.Fail(c, err.Error())
		return
	}
	validateNodeLabels(c, labels, true)
}

func (s *E2ESuite) Test005DCMInvalidComputeType(c *C) {
	ctx := context.Background()

	labelMap := make(map[string]string)
	labelMap["feature.node.kubernetes.io/amd-gpu"] = "true"

	worker_node := s.getWorkerNode(c, ctx)
	log.Printf("Adding node label to select profile: inval_prof1")
	s.addRemoveNodeLabels(worker_node, "inval_prof1", true)
	labels, err := s.k8sclient.GetNodeLabel(ctx, worker_node)
	if err != nil {
		log.Printf("Error in getting node labels")
		assert.Fail(c, err.Error())
		return
	}
	validateNodeLabels(c, labels, true)
}

func (s *E2ESuite) Test006DCMInvalidMemoryType(c *C) {
	ctx := context.Background()

	labelMap := make(map[string]string)
	labelMap["feature.node.kubernetes.io/amd-gpu"] = "true"

	worker_node := s.getWorkerNode(c, ctx)
	log.Printf("Adding node label to select profile: inval_prof2")
	s.addRemoveNodeLabels(worker_node, "inval_prof2", true)
	labels, err := s.k8sclient.GetNodeLabel(ctx, worker_node)
	if err != nil {
		log.Printf("Error in getting node labels")
		assert.Fail(c, err.Error())
		return
	}
	validateNodeLabels(c, labels, true)
}

func (s *E2ESuite) Test007DCMInvalidGPUCount(c *C) {
	ctx := context.Background()

	labelMap := make(map[string]string)
	labelMap["feature.node.kubernetes.io/amd-gpu"] = "true"

	worker_node := s.getWorkerNode(c, ctx)
	log.Printf("Adding node label to select profile: inval_prof3")
	s.addRemoveNodeLabels(worker_node, "inval_prof3", true)
	labels, err := s.k8sclient.GetNodeLabel(ctx, worker_node)
	if err != nil {
		log.Printf("Error in getting node labels")
		assert.Fail(c, err.Error())
		return
	}
	validateNodeLabels(c, labels, true)
}

func (s *E2ESuite) Test008DCMInvalidMemoryCombination(c *C) {
	ctx := context.Background()

	labelMap := make(map[string]string)
	labelMap["feature.node.kubernetes.io/amd-gpu"] = "true"

	worker_node := s.getWorkerNode(c, ctx)
	log.Printf("Adding node label to select profile: inval_prof4")
	s.addRemoveNodeLabels(worker_node, "inval_prof4", true)
	labels, err := s.k8sclient.GetNodeLabel(ctx, worker_node)
	if err != nil {
		log.Printf("Error in getting node labels")
		assert.Fail(c, err.Error())
		return
	}
	validateNodeLabels(c, labels, true)
}

func (s *E2ESuite) Test009DCMNPS4Partitioning(c *C) {
	ctx := context.Background()

	worker_node := s.getWorkerNode(c, ctx)
	log.Printf("Adding node label to select profile: nps4\n")
	s.addRemoveNodeLabels(worker_node, "nps4", false)
	labels, err := s.k8sclient.GetNodeLabel(ctx, worker_node)
	if err != nil {
		log.Printf("Error in getting node labels")
		assert.Fail(c, err.Error())
		return
	}
	validateNodeLabels(c, labels, false)
}

func (s *E2ESuite) Test010DCMNPS2Partitioning(c *C) {
	ctx := context.Background()

	worker_node := s.getWorkerNode(c, ctx)
	log.Printf("Adding node label to select profile: nps2\n")
	s.addRemoveNodeLabels(worker_node, "nps2", false)
	labels, err := s.k8sclient.GetNodeLabel(ctx, worker_node)
	if err != nil {
		log.Printf("Error in getting node labels")
		assert.Fail(c, err.Error())
		return
	}
	validateNodeLabels(c, labels, false)
}

func (s *E2ESuite) Test010DCMNPS2Partitioning(c *C) {
	ctx := context.Background()

	worker_node := s.getWorkerNode(c, ctx)
	log.Printf("Adding node label to select profile: nps2\n")
	s.addRemoveNodeLabels(worker_node, "nps2", false)
	labels, err := s.k8sclient.GetNodeLabel(ctx, worker_node)
	if err != nil {
		log.Printf("Error in getting node labels")
		assert.Fail(c, err.Error())
		return
	}
	validateNodeLabels(c, labels, false)
}
