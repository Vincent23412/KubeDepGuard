package resolver

import (
	"github.com/vincent/KubeDepGuard/internal/dependency/resolver/lister"
	corelisters "k8s.io/client-go/listers/core/v1"
)

// The root aliases keep resolver's public contracts independent from the
// concrete lister package layout.
type ResourceScope = lister.ResourceScope
type Resource = lister.Resource
type ResourceLister = lister.ResourceLister
type ListerResolver = lister.ListerResolver
type ConfigMapResolver = lister.ConfigMapResolver
type PodResolver = lister.PodResolver
type ServiceResolver = lister.ServiceResolver

func NewListerResolver(pods corelisters.PodLister, configMaps corelisters.ConfigMapLister, services corelisters.ServiceLister) *ListerResolver {
	return lister.NewListerResolver(pods, configMaps, services)
}

func NewConfigMapResolver(configMaps corelisters.ConfigMapLister) *ConfigMapResolver {
	return lister.NewConfigMapResolver(configMaps)
}

func NewPodResolver(pods corelisters.PodLister) *PodResolver {
	return lister.NewPodResolver(pods)
}

func NewServiceResolver(services corelisters.ServiceLister) *ServiceResolver {
	return lister.NewServiceResolver(services)
}
