package lister

import (
	"k8s.io/apimachinery/pkg/labels"
	corelisters "k8s.io/client-go/listers/core/v1"
)

// ServiceResolver lists Service resources from the current cluster state.
type ServiceResolver struct{ services corelisters.ServiceLister }

func NewServiceResolver(services corelisters.ServiceLister) *ServiceResolver {
	return &ServiceResolver{services: services}
}

func (r *ServiceResolver) Kind() string { return "Service" }
func (r *ServiceResolver) List(scope ResourceScope) ([]Resource, error) {
	items, err := r.services.Services(scope.Namespace).List(labels.Everything())
	if err != nil {
		return nil, err
	}
	resources := make([]Resource, 0, len(items))
	for _, item := range items {
		resources = append(resources, item)
	}
	return resources, nil
}
