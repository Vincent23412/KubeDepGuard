package resolver

import (
	"github.com/vincent/KubeDepGuard/internal/dependency/resolver/lister"
	appslisters "k8s.io/client-go/listers/apps/v1"
	corelisters "k8s.io/client-go/listers/core/v1"
	networkinglisters "k8s.io/client-go/listers/networking/v1"
)

// The root aliases keep resolver's public contracts independent from the
// concrete lister package layout.
type ResourceScope = lister.ResourceScope
type Resource = lister.Resource
type ResourceLister = lister.ResourceLister
type ListerResolver = lister.ListerResolver
type ConfigMapResolver = lister.ConfigMapResolver
type SecretResolver = lister.SecretResolver
type PersistentVolumeClaimResolver = lister.PersistentVolumeClaimResolver
type DeploymentResolver = lister.DeploymentResolver
type PodResolver = lister.PodResolver
type ServiceResolver = lister.ServiceResolver
type ServiceAccountResolver = lister.ServiceAccountResolver
type NodeResolver = lister.NodeResolver
type PersistentVolumeResolver = lister.PersistentVolumeResolver
type IngressResolver = lister.IngressResolver

func NewListerResolver(deployments appslisters.DeploymentLister, pods corelisters.PodLister, configMaps corelisters.ConfigMapLister, secrets corelisters.SecretLister, pvcs corelisters.PersistentVolumeClaimLister, services corelisters.ServiceLister, serviceAccounts corelisters.ServiceAccountLister, nodes corelisters.NodeLister, pvs corelisters.PersistentVolumeLister, ingresses networkinglisters.IngressLister) *ListerResolver {
	return lister.NewListerResolver(deployments, pods, configMaps, secrets, pvcs, services, serviceAccounts, nodes, pvs, ingresses)
}

func NewDeploymentResolver(deployments appslisters.DeploymentLister) *DeploymentResolver {
	return lister.NewDeploymentResolver(deployments)
}

func NewSecretResolver(secrets corelisters.SecretLister) *SecretResolver {
	return lister.NewSecretResolver(secrets)
}

func NewPersistentVolumeClaimResolver(pvcs corelisters.PersistentVolumeClaimLister) *PersistentVolumeClaimResolver {
	return lister.NewPersistentVolumeClaimResolver(pvcs)
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
