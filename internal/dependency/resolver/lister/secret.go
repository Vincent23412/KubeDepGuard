package lister

import (
	"k8s.io/apimachinery/pkg/labels"
	corelisters "k8s.io/client-go/listers/core/v1"
)

// SecretResolver lists Secret resources from an Informer cache.
type SecretResolver struct{ secrets corelisters.SecretLister }

func NewSecretResolver(secrets corelisters.SecretLister) *SecretResolver {
	return &SecretResolver{secrets: secrets}
}

func (r *SecretResolver) Kind() string { return "Secret" }

func (r *SecretResolver) List(scope ResourceScope) ([]Resource, error) {
	items, err := r.secrets.Secrets(scope.Namespace).List(labels.Everything())
	if err != nil {
		return nil, err
	}
	resources := make([]Resource, 0, len(items))
	for _, item := range items {
		resources = append(resources, item)
	}
	return resources, nil
}
