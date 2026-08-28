package lister

import (
	"k8s.io/apimachinery/pkg/labels"
	corelisters "k8s.io/client-go/listers/core/v1"
)

type NodeResolver struct{ items corelisters.NodeLister }

func NewNodeResolver(items corelisters.NodeLister) *NodeResolver { return &NodeResolver{items: items} }
func (r *NodeResolver) Kind() string                             { return "Node" }
func (r *NodeResolver) List(scope ResourceScope) ([]Resource, error) {
	items, err := r.items.List(labels.Everything())
	if err != nil {
		return nil, err
	}
	out := make([]Resource, 0, len(items))
	for _, item := range items {
		out = append(out, item)
	}
	return out, nil
}
