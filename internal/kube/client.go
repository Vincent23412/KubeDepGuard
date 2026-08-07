// Package kube contains small Kubernetes integration helpers shared by the
// independently deployable webhook and monitor binaries.
package kube

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func InClusterClient() (kubernetes.Interface, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(config)
}
