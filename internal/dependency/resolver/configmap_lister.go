package resolver

import (
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	corelisters "k8s.io/client-go/listers/core/v1"
)

// ConfigMapResolver resolves ConfigMap targets from an Informer cache.
type ConfigMapResolver struct{ configMaps corelisters.ConfigMapLister }

func NewConfigMapResolver(configMaps corelisters.ConfigMapLister) *ConfigMapResolver {
	return &ConfigMapResolver{configMaps: configMaps}
}

func (r *ConfigMapResolver) Kind() string { return "ConfigMap" }

func (r *ConfigMapResolver) Exists(target Target) (bool, error) {
	_, err := r.configMaps.ConfigMaps(target.Namespace).Get(target.Name)
	if err == nil {
		return true, nil
	}
	if apierrors.IsNotFound(err) {
		return false, nil
	}
	return false, err
}
