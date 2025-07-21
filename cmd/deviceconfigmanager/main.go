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
package main

import (
	"log"
	"os"

	configmanager "github.com/ROCm/device-config-manager/pkg/config_manager"
)

var (
	Version   string
	BuildDate string
	GitCommit string
)

func main() {

	log.Printf("Version : %v", Version)
	log.Printf("BuildDate: %v", BuildDate)
	log.Printf("GitCommit: %v", GitCommit)

	if os.Getenv("KUBERNETES_SERVICE_HOST") != "" {
		log.Println("Running inside a Kubernetes pod")
	} else {
		log.Println("Not running inside a Kubernetes pod")
		<-make(chan struct{})
	}
	//Read profile from node labeller
	selectedProfile, err := configmanager.GetPartitionProfile()
	if err != nil {
		log.Printf("err: %+v", err)
		return
	}

	// Start the worker routine
	go configmanager.Worker()

	if selectedProfile != "" {
		configmanager.TriggerRetryLoop(selectedProfile, "initial partitioning")
	}

	// starting a seperate go routine for file watcher
	go configmanager.StartFileWatcher(selectedProfile)

	go configmanager.NodeLabelWatcher()

	// Keep the program running
	<-make(chan struct{})
}
