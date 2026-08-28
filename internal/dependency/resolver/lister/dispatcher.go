package lister

import (
	"fmt"

	corelisters "k8s.io/client-go/listers/core/v1"
)

// ListerResolver dispatches collection reads to kind-specific cache listers.
type ListerResolver struct{ byKind map[string]ResourceLister }

func NewListerResolver(pods corelisters.PodLister, configMaps corelisters.ConfigMapLister, secrets corelisters.SecretLister, services corelisters.ServiceLister) *ListerResolver {
	podResolver := NewPodResolver(pods)
	configMapResolver := NewConfigMapResolver(configMaps)
	secretResolver := NewSecretResolver(secrets)
	serviceResolver := NewServiceResolver(services)
	return &ListerResolver{byKind: map[string]ResourceLister{
		podResolver.Kind():       podResolver,
		configMapResolver.Kind(): configMapResolver,
		secretResolver.Kind():    secretResolver,
		serviceResolver.Kind():   serviceResolver,
	}}
}
func (r *ListerResolver) List(scope ResourceScope) ([]Resource, error) {
	kindResolver, ok := r.byKind[scope.Kind]
	if !ok {
		return nil, fmt.Errorf("unsupported resource kind %q", scope.Kind)
	}
	return kindResolver.List(scope)
}
