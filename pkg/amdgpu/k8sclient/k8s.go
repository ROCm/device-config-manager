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

package k8sclient

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

type K8sClient struct {
	sync.Mutex
	ctx       context.Context
	clientset *kubernetes.Clientset
}

func (k *K8sClient) init() error {
	k.Lock()
	defer k.Unlock()

	config, err := rest.InClusterConfig()
	if err != nil {
		log.Printf("k8s cluster config error %v", err)
		return err
	}
	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Printf("clientset from config failed %v", err)
		return err
	}

	k.clientset = clientset
	return nil
}

func NewClient(ctx context.Context) *K8sClient {
	return &K8sClient{
		ctx: ctx,
	}
}

func (k *K8sClient) reConnect() error {
	if k.clientset == nil {
		return k.init()
	}
	return nil
}

func IsKMMDriverEnabled() bool {
	if os.Getenv("KMM_DRIVER_ENABLED") != "" {
		if strings.ToLower(os.Getenv("KMM_DRIVER_ENABLED")) == "true" {
			return true
		}
		return false
	}
	return false
}

func GetNodeName() string {
	if os.Getenv("DS_NODE_NAME") != "" {
		return os.Getenv("DS_NODE_NAME")
	}
	return ""
}

func GetPodName() string {
	if os.Getenv("POD_NAME") != "" {
		return os.Getenv("POD_NAME")
	}
	return ""
}

func GetPodNameSpace() string {
	if os.Getenv("POD_NAMESPACE") != "" {
		return os.Getenv("POD_NAMESPACE")
	}
	return ""
}

func (k *K8sClient) GetNodeLabel(nodeName string) (map[string]string, error) {
	k.reConnect()
	k.Lock()
	defer k.Unlock()
	ctx, cancel := context.WithCancel(k.ctx)
	defer cancel()

	retries := 10
	var err error
	var node *v1.Node

	for i := range retries {
		node, err = k.clientset.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
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

func (k *K8sClient) GetNodeInformer(nodeName string) cache.SharedIndexInformer {
	k.reConnect()
	k.Lock()
	defer k.Unlock()

	// Create a shared informer factory
	factory := informers.NewSharedInformerFactoryWithOptions(k.clientset, 0,
		informers.WithTweakListOptions(func(options *metav1.ListOptions) {
			options.FieldSelector = fmt.Sprintf("metadata.name=%s", nodeName) // Filter by node name
		}),
	)

	// Create a node informer
	nodeInformer := factory.Core().V1().Nodes().Informer()

	return nodeInformer
}

func (k *K8sClient) DeleteNodeModulesConfig(nodeName string) error {
	k.reConnect()
	k.Lock()
	defer k.Unlock()
	ctx, cancel := context.WithCancel(k.ctx)
	defer cancel()

	if nodeName == "" {
		log.Printf("k8s client got empty node name, skip deleting NodeModulesConfig")
		return fmt.Errorf("k8s client received empty node name")
	}

	// Use dynamic client to delete the custom resource
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Printf("failed to get in-cluster config: %v", err)
		return err
	}
	dynClient, err := dynamic.NewForConfig(config)
	if err != nil {
		log.Printf("failed to create dynamic client: %v", err)
		return err
	}
	gvr := schema.GroupVersionResource{
		Group:    "kmm.sigs.x-k8s.io",
		Version:  "v1beta1",
		Resource: "nodemodulesconfigs",
	}
	err = dynClient.Resource(gvr).Delete(ctx, nodeName, metav1.DeleteOptions{})
	if err != nil {
		log.Printf("failed to delete NodeModulesConfig for node %s, err: %v", nodeName, err)
		return err
	}

	log.Printf("NodeModulesConfig for node %s deleted successfully", nodeName)
	return nil
}

func (k *K8sClient) CreateEvent(evtObj *v1.Event) error {
	k.reConnect()
	k.Lock()
	defer k.Unlock()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if evtObj == nil {
		log.Printf("k8s client got empty event object, skip genreating k8s event")
		return fmt.Errorf("k8s client received empty event object")
	}

	if _, err := k.clientset.CoreV1().Events(evtObj.Namespace).Create(ctx, evtObj, metav1.CreateOptions{}); err != nil {
		log.Printf("failed to generate event %+v, err: %+v", evtObj, err)
		return err
	}

	return nil
}

func (k *K8sClient) GetDaemonSets() []string {
	k.reConnect()
	k.Lock()
	defer k.Unlock()
	ctx, cancel := context.WithCancel(k.ctx)
	defer cancel()

	daemonsetlist := make([]string, 0)
	daemonSets, err := k.clientset.AppsV1().DaemonSets(metav1.NamespaceAll).List(ctx, metav1.ListOptions{})

	if err != nil {
		log.Printf("k8s internal daemonset get failed %v", err)
		return daemonsetlist
	}

	for _, ds := range daemonSets.Items {
		daemonsetlist = append(daemonsetlist, ds.Name)
	}

	return daemonsetlist
}

func (k *K8sClient) GetPods(nodeName string) []string {
	k.reConnect()
	k.Lock()
	defer k.Unlock()
	ctx, cancel := context.WithCancel(k.ctx)
	defer cancel()

	podNames := []string{}

	// List all pods across all namespaces
	pods, err := k.clientset.CoreV1().Pods(metav1.NamespaceAll).List(ctx, metav1.ListOptions{})
	if err != nil {
		log.Printf("Failed to list pods: %v", err)
		return podNames
	}

	for _, pod := range pods.Items {
		if pod.Spec.NodeName == nodeName {
			podNames = append(podNames, pod.Name)
		}
	}

	return podNames
}

func (k *K8sClient) AddNodeLabel(nodeName string, key string, value string) error {
	k.reConnect()
	k.Lock()
	defer k.Unlock()
	ctx, cancel := context.WithCancel(k.ctx)
	defer cancel()

	retries := 10
	var err error
	var node *v1.Node

	for i := range retries {
		node, err = k.clientset.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
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
		_, err = k.clientset.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{})
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
