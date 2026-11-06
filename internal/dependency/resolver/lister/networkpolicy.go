package lister

import (
	"k8s.io/apimachinery/pkg/labels"
	networkinglisters "k8s.io/client-go/listers/networking/v1"
)

// NetworkPolicyResolver exposes cached NetworkPolicies to conditional rules.
type NetworkPolicyResolver struct {
	items networkinglisters.NetworkPolicyLister
}

func NewNetworkPolicyResolver(items networkinglisters.NetworkPolicyLister) *NetworkPolicyResolver {
	return &NetworkPolicyResolver{items: items}
}

func (r *NetworkPolicyResolver) Kind() string { return "NetworkPolicy" }

func (r *NetworkPolicyResolver) List(scope ResourceScope) ([]Resource, error) {
	items, err := r.items.NetworkPolicies(scope.Namespace).List(labels.Everything())
	if err != nil {
		return nil, err
	}
	out := make([]Resource, 0, len(items))
	for _, item := range items {
		out = append(out, item)
	}
	return out, nil
}
