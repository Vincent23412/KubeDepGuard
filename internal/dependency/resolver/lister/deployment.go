package lister

import (
	"k8s.io/apimachinery/pkg/labels"
	appslisters "k8s.io/client-go/listers/apps/v1"
)

// DeploymentResolver lists Deployment resources from an Informer cache.
type DeploymentResolver struct{ deployments appslisters.DeploymentLister }

func NewDeploymentResolver(deployments appslisters.DeploymentLister) *DeploymentResolver {
	return &DeploymentResolver{deployments: deployments}
}

func (r *DeploymentResolver) Kind() string { return "Deployment" }

func (r *DeploymentResolver) List(scope ResourceScope) ([]Resource, error) {
	items, err := r.deployments.Deployments(scope.Namespace).List(labels.Everything())
	if err != nil {
		return nil, err
	}
	resources := make([]Resource, 0, len(items))
	for _, item := range items {
		resources = append(resources, item)
	}
	return resources, nil
}
