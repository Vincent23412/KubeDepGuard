package lister

import (
	"fmt"

	appslisters "k8s.io/client-go/listers/apps/v1"
	corelisters "k8s.io/client-go/listers/core/v1"
	networkinglisters "k8s.io/client-go/listers/networking/v1"
)

// ListerResolver dispatches collection reads to kind-specific cache listers.
type ListerResolver struct{ byKind map[string]ResourceLister }

func NewListerResolver(deployments appslisters.DeploymentLister, pods corelisters.PodLister, configMaps corelisters.ConfigMapLister, secrets corelisters.SecretLister, pvcs corelisters.PersistentVolumeClaimLister, services corelisters.ServiceLister, serviceAccounts corelisters.ServiceAccountLister, nodes corelisters.NodeLister, pvs corelisters.PersistentVolumeLister, ingresses networkinglisters.IngressLister) *ListerResolver {
	deploymentResolver := NewDeploymentResolver(deployments)
	podResolver := NewPodResolver(pods)
	configMapResolver := NewConfigMapResolver(configMaps)
	secretResolver := NewSecretResolver(secrets)
	pvcResolver := NewPersistentVolumeClaimResolver(pvcs)
	serviceResolver := NewServiceResolver(services)
	serviceAccountResolver := NewServiceAccountResolver(serviceAccounts)
	nodeResolver := NewNodeResolver(nodes)
	pvResolver := NewPersistentVolumeResolver(pvs)
	ingressResolver := NewIngressResolver(ingresses)
	return &ListerResolver{byKind: map[string]ResourceLister{
		deploymentResolver.Kind():     deploymentResolver,
		podResolver.Kind():            podResolver,
		configMapResolver.Kind():      configMapResolver,
		secretResolver.Kind():         secretResolver,
		pvcResolver.Kind():            pvcResolver,
		serviceResolver.Kind():        serviceResolver,
		serviceAccountResolver.Kind(): serviceAccountResolver,
		nodeResolver.Kind():           nodeResolver,
		pvResolver.Kind():             pvResolver,
		ingressResolver.Kind():        ingressResolver,
	}}
}
func (r *ListerResolver) List(scope ResourceScope) ([]Resource, error) {
	kindResolver, ok := r.byKind[scope.Kind]
	if !ok {
		return nil, fmt.Errorf("unsupported resource kind %q", scope.Kind)
	}
	return kindResolver.List(scope)
}
