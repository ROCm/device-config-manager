/**
# Copyright (c) Advanced Micro Devices, Inc. All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the \"License\");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an \"AS IS\" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
**/

package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/kubectl/pkg/scheme"
)

type K8sClient struct {
	client *kubernetes.Clientset
}

type GPUConfigProfiles struct {
	ProfilesList map[string]*GPUConfigProfile `json:"gpu-config-profiles,omitempty"`
}

type ProfileConfig struct {
	ComputePartition string `json:"computePartition,omitempty"`
	MemoryPartition  string `json:"memoryPartition,omitempty"`
	NumGPUsAssigned  uint32 `json:"numGPUsAssigned,omitempty"`
}

type SkippedGPUs struct {
	Id []uint32 `json:"ids,omitempty"`
}

type GPUConfigProfile struct {
	Filters  *SkippedGPUs     `json:"skippedGPUs,omitempty"`
	Profiles []*ProfileConfig `json:"profiles,omitempty"`
}

func NewK8sClient(config *restclient.Config) (*K8sClient, error) {
	k8sc := K8sClient{}
	cs, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	k8sc.client = cs
	return &k8sc, nil
}

func (k *K8sClient) CreateNamespace(ctx context.Context, namespace string) error {
	namespaceObj := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
		Status: corev1.NamespaceStatus{},
	}
	_, err := k.client.CoreV1().Namespaces().Create(ctx, namespaceObj, metav1.CreateOptions{})
	return err
}

func (k *K8sClient) DeleteNamespace(ctx context.Context, namespace string) error {
	return k.client.CoreV1().Namespaces().Delete(ctx, namespace, metav1.DeleteOptions{})
}

func (k *K8sClient) GetPodsByLabel(ctx context.Context, namespace string, labelMap map[string]string) ([]corev1.Pod, error) {
	podList, err := k.client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(labelMap).String(),
	})
	if err != nil {
		return nil, err
	}
	return podList.Items, nil
}

func (k *K8sClient) GetNodesByLabel(ctx context.Context, labelMap map[string]string) ([]corev1.Node, error) {
	nodeList, err := k.client.CoreV1().Nodes().List(ctx, metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(labelMap).String(),
	})
	if err != nil {
		return nil, err
	}
	return nodeList.Items, nil
}

func (k *K8sClient) GetServiceByLabel(ctx context.Context, namespace string, labelMap map[string]string) ([]corev1.Service, error) {
	nodeList, err := k.client.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(labelMap).String(),
	})
	if err != nil {
		return nil, err
	}
	return nodeList.Items, nil
}

func (k *K8sClient) GetEndpointByLabel(ctx context.Context, namespace string, labelMap map[string]string) ([]corev1.Endpoints, error) {
	nodeList, err := k.client.CoreV1().Endpoints(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(labelMap).String(),
	})
	if err != nil {
		return nil, err
	}
	return nodeList.Items, nil
}

func (k *K8sClient) ValidatePod(ctx context.Context, namespace, podName string) error {
	pod, err := k.client.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("unexpected error getting pod %s; err: %w", podName, err)
	}

	for _, c := range pod.Status.ContainerStatuses {
		if c.State.Waiting != nil && c.State.Waiting.Reason == "CrashLoopBackOff" {
			return fmt.Errorf("pod %s in namespace %s is in CrashLoopBackOff", pod.Name, pod.Namespace)
		}
	}

	return nil
}

func (k *K8sClient) CreateConfigMap(ctx context.Context, namespace string, name string) error {
	skippedGPUs := &SkippedGPUs{
		Id: []uint32{},
	}

	skippedGPUs2 := &SkippedGPUs{
		Id: []uint32{2, 3},
	}

	profiles_set1 := []*ProfileConfig{
		{
			ComputePartition: "SPX",
			MemoryPartition:  "NPS1",
			NumGPUsAssigned:  8,
		},
	}

	profiles_set2 := []*ProfileConfig{
		{
			ComputePartition: "CPX",
			MemoryPartition:  "NPS1",
			NumGPUsAssigned:  1,
		},
		{
			ComputePartition: "DPX",
			MemoryPartition:  "NPS1",
			NumGPUsAssigned:  4,
		},
		{
			ComputePartition: "SPX",
			MemoryPartition:  "NPS1",
			NumGPUsAssigned:  1,
		},
	}

	profiles_set3 := []*ProfileConfig{
		{
			ComputePartition: "InvalidName",
			MemoryPartition:  "NPS1",
			NumGPUsAssigned:  8,
		},
	}

	profiles_set4 := []*ProfileConfig{
		{
			ComputePartition: "SPX",
			MemoryPartition:  "InvalidName",
			NumGPUsAssigned:  8,
		},
	}

	profiles_set5 := []*ProfileConfig{
		{
			ComputePartition: "SPX",
			MemoryPartition:  "NPS1",
			NumGPUsAssigned:  100,
		},
	}

	profiles_set6 := []*ProfileConfig{
		{
			ComputePartition: "CPX",
			MemoryPartition:  "NPS4",
		},
	}

	profiles_set7 := []*ProfileConfig{
		{
			ComputePartition: "DPX",
			MemoryPartition:  "NPS4",
			NumGPUsAssigned:  7,
		},
		{
			ComputePartition: "SPX",
			MemoryPartition:  "NPS1",
			NumGPUsAssigned:  1,
		},
	}

	profiles_set8 := []*ProfileConfig{
		{
			ComputePartition: "CPX",
			MemoryPartition:  "NPS4",
			NumGPUsAssigned:  8,
		},
	}

	profiles_set9 := []*ProfileConfig{
		{
			ComputePartition: "DPX",
			MemoryPartition:  "NPS2",
			NumGPUsAssigned:  8,
		},
	}

	profileslist := GPUConfigProfiles{
		ProfilesList: map[string]*GPUConfigProfile{
			"default": {
				Filters:  skippedGPUs,
				Profiles: profiles_set1,
			},
			"e2e_profile1": {
				Filters:  skippedGPUs2,
				Profiles: profiles_set2,
			},
			"e2e_profile2": {
				Filters:  skippedGPUs,
				Profiles: profiles_set6,
			},
			"inval_prof1": {
				Filters:  skippedGPUs,
				Profiles: profiles_set3,
			},
			"inval_prof2": {
				Filters:  skippedGPUs,
				Profiles: profiles_set4,
			},
			"inval_prof3": {
				Filters:  skippedGPUs,
				Profiles: profiles_set5,
			},
			"inval_prof4": {
				Filters:  skippedGPUs,
				Profiles: profiles_set7,
			},
			"nps4": {
				Filters:  skippedGPUs,
				Profiles: profiles_set8,
			},
			"nps2": {
				Filters:  skippedGPUs,
				Profiles: profiles_set9,
			},
		},
	}

	cfgData, _ := json.Marshal(profileslist)

	mcfgMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: map[string]string{
			"config.json": string(cfgData),
		},
	}

	_, err := k.client.CoreV1().ConfigMaps(namespace).Create(ctx, mcfgMap, metav1.CreateOptions{})
	if err != nil {
		log.Print("Configmap created successfully.\n")
	}
	return err
}

func (k *K8sClient) UpdateConfigMap(ctx context.Context, namespace string, name string, json string) error {
	mcfgMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: map[string]string{
			"config.json": json,
		},
	}

	_, err := k.client.CoreV1().ConfigMaps(namespace).Update(ctx, mcfgMap, metav1.UpdateOptions{})
	return err
}

func (k *K8sClient) DeleteConfigMap(ctx context.Context, namespace string, name string) error {
	return k.client.CoreV1().ConfigMaps(namespace).Delete(ctx, name, metav1.DeleteOptions{})
}

func (k *K8sClient) ExecCmdOnPod(ctx context.Context, rc *restclient.Config, pod *corev1.Pod, container, execCmd string) (string, error) {
	if pod == nil {
		return "", fmt.Errorf("No pod specified")
	}
	req := k.client.CoreV1().RESTClient().Post().Resource("pods").Name(pod.Name).Namespace(pod.Namespace).SubResource("exec")
	req.VersionedParams(&corev1.PodExecOptions{
		Container: container,
		Command:   []string{"/bin/sh", "-c", execCmd},
		Stdin:     false,
		Stdout:    true,
		Stderr:    false,
		TTY:       false,
	}, scheme.ParameterCodec)
	executor, err := remotecommand.NewSPDYExecutor(rc, "POST", req.URL())
	if err != nil {
		return "", fmt.Errorf("failed to create command executor. Error:%v", err)
	}
	buf := &bytes.Buffer{}
	err = executor.StreamWithContext(context.Background(), remotecommand.StreamOptions{
		Stdout: buf,
		Tty:    false,
	})
	if err != nil {
		return "", fmt.Errorf("failed to run command on pod %v. Error:%v", pod.Name, err)
	}

	return buf.String(), nil
}

func (k *K8sClient) AddNodeLabel(ctx context.Context, nodeName string, key string, value string) error {

	retries := 10
	var err error
	var node *corev1.Node

	for i := range retries {
		node, err = k.client.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
		if err == nil {
			break
		}

		log.Printf("k8s get node API failed (attempt %d/%d): %v", i+1, retries, err)
		time.Sleep(10 * time.Second)
	}

	if err != nil {
		return err
	}

	node.Labels[key] = value

	for i := range retries {
		_, err = k.client.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{})
		if err == nil {
			break
		}

		log.Printf("k8s update node API failed (attempt %d/%d): %v", i+1, retries, err)
		time.Sleep(10 * time.Second)
	}

	if err != nil {
		return err
	}

	log.Printf("Gpu-config-profile-state label added successfully")
	return nil
}

func (k *K8sClient) DeleteNodeLabel(ctx context.Context, nodeName string, key string) error {

	node, err := k.client.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		panic(err.Error())
	}

	// Remove a label to the node
	delete(node.Labels, key)

	// Update the node object with the new label
	_, err = k.client.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{})
	if err != nil {
		panic(err.Error())
	}

	log.Printf("Label removed successfully")
	if err != nil {
		return fmt.Errorf("failed to remove node label to node: %v", err)
	}
	return nil
}

func (k *K8sClient) GetNodeLabel(ctx context.Context, nodeName string) (map[string]string, error) {

	retries := 10
	var err error
	var node *corev1.Node

	for i := range retries {
		node, err = k.client.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
		if err == nil {
			break
		} else {
			log.Printf("k8s get node API failed (attempt %d/%d): %v", i+1, retries, err)
			time.Sleep(30 * time.Second)
		}
	}
	if err != nil {
		log.Printf("k8s internal node get failed %v", err)
		return make(map[string]string), err
	}
	return node.Labels, nil
}
