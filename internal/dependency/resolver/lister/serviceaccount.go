package lister

import (
	"k8s.io/apimachinery/pkg/labels"
	corelisters "k8s.io/client-go/listers/core/v1"
)

type ServiceAccountResolver struct {
	items corelisters.ServiceAccountLister
}

func NewServiceAccountResolver(items corelisters.ServiceAccountLister) *ServiceAccountResolver {
	return &ServiceAccountResolver{items: items}
}
func (r *ServiceAccountResolver) Kind() string { return "ServiceAccount" }
func (r *ServiceAccountResolver) List(scope ResourceScope) ([]Resource, error) {
	items, err := r.items.ServiceAccounts(scope.Namespace).List(labels.Everything())
	if err != nil {
		return nil, err
	}
	out := make([]Resource, 0, len(items))
	for _, item := range items {
		out = append(out, item)
	}
	return out, nil
}
