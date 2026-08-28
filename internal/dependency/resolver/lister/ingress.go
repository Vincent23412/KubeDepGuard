package lister

import (
	"k8s.io/apimachinery/pkg/labels"
	networkinglisters "k8s.io/client-go/listers/networking/v1"
)

type IngressResolver struct {
	items networkinglisters.IngressLister
}

func NewIngressResolver(items networkinglisters.IngressLister) *IngressResolver {
	return &IngressResolver{items: items}
}
func (r *IngressResolver) Kind() string { return "Ingress" }
func (r *IngressResolver) List(scope ResourceScope) ([]Resource, error) {
	items, err := r.items.Ingresses(scope.Namespace).List(labels.Everything())
	if err != nil {
		return nil, err
	}
	out := make([]Resource, 0, len(items))
	for _, item := range items {
		out = append(out, item)
	}
	return out, nil
}
