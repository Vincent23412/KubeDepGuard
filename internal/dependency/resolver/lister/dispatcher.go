package lister

import (
	"fmt"

	appslisters "k8s.io/client-go/listers/apps/v1"
	corelisters "k8s.io/client-go/listers/core/v1"
	networkinglisters "k8s.io/client-go/listers/networking/v1"
)

// ListerResolver dispatches collection reads to kind-specific cache listers.
type ListerResolver struct{ byKind map[string]ResourceLister }

func NewListerResolver(deployments appslisters.DeploymentLister, pods corelisters.PodLister, configMaps corelisters.ConfigMapLister, secrets corelisters.SecretLister, pvcs corelisters.PersistentVolumeClaimLister, services corelisters.ServiceLister, ingresses networkinglisters.IngressLister, networkPolicies networkinglisters.NetworkPolicyLister) *ListerResolver {
	deploymentResolver := NewDeploymentResolver(deployments)
	podResolver := NewPodResolver(pods)
	configMapResolver := NewConfigMapResolver(configMaps)
	secretResolver := NewSecretResolver(secrets)
	pvcResolver := NewPersistentVolumeClaimResolver(pvcs)
	serviceResolver := NewServiceResolver(services)
	ingressResolver := NewIngressResolver(ingresses)
	networkPolicyResolver := NewNetworkPolicyResolver(networkPolicies)
	return &ListerResolver{byKind: map[string]ResourceLister{
		deploymentResolver.Kind():    deploymentResolver,
		podResolver.Kind():           podResolver,
		configMapResolver.Kind():     configMapResolver,
		secretResolver.Kind():        secretResolver,
		pvcResolver.Kind():           pvcResolver,
		serviceResolver.Kind():       serviceResolver,
		ingressResolver.Kind():       ingressResolver,
		networkPolicyResolver.Kind(): networkPolicyResolver,
	}}
}
func (r *ListerResolver) List(scope ResourceScope) ([]Resource, error) {
	kindResolver, ok := r.byKind[scope.Kind]
	if !ok {
		return nil, fmt.Errorf("unsupported resource kind %q", scope.Kind)
	}
	return kindResolver.List(scope)
}
