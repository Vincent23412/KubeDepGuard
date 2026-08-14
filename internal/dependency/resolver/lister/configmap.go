package lister

import (
	"k8s.io/apimachinery/pkg/labels"
	corelisters "k8s.io/client-go/listers/core/v1"
)

// ConfigMapResolver lists ConfigMap resources from an Informer cache.
type ConfigMapResolver struct{ configMaps corelisters.ConfigMapLister }

func NewConfigMapResolver(configMaps corelisters.ConfigMapLister) *ConfigMapResolver {
	return &ConfigMapResolver{configMaps: configMaps}
}

func (r *ConfigMapResolver) Kind() string { return "ConfigMap" }

func (r *ConfigMapResolver) List(scope ResourceScope) ([]Resource, error) {
	items, err := r.configMaps.ConfigMaps(scope.Namespace).List(labels.Everything())
	if err != nil {
		return nil, err
	}
	resources := make([]Resource, 0, len(items))
	for _, item := range items {
		resources = append(resources, item)
	}
	return resources, nil
}
