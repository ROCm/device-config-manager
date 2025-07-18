package k8e2e

import (
	"github.com/ROCm/device-config-manager/test/k8s-e2e/clients"
	restclient "k8s.io/client-go/rest"
)

// E2ESuite e2e config
type E2ESuite struct {
	k8sclient  *clients.K8sClient
	helmClient *clients.HelmClient
	restConfig *restclient.Config
	registry   string
	helmChart  string
	imageTag   string
	ns         string
	kubeconfig string
	platform   string
}
