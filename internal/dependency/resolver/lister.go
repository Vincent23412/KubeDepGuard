package resolver

import (
	"fmt"

	corelisters "k8s.io/client-go/listers/core/v1"
)

// TargetKindResolver handles a single Kubernetes target kind.
type TargetKindResolver interface {
	Kind() string
	Exists(Target) (bool, error)
}

// ListerResolver dispatches lookups to kind-specific Informer resolvers.
type ListerResolver struct{ byKind map[string]TargetKindResolver }

func NewListerResolver(configMaps corelisters.ConfigMapLister) *ListerResolver {
	configMapResolver := NewConfigMapResolver(configMaps)
	return &ListerResolver{byKind: map[string]TargetKindResolver{configMapResolver.Kind(): configMapResolver}}
}
func (r *ListerResolver) Exists(target Target) (bool, error) {
	kindResolver, ok := r.byKind[target.Kind]
	if !ok {
		return false, fmt.Errorf("unsupported target kind %q", target.Kind)
	}
	return kindResolver.Exists(target)
}
