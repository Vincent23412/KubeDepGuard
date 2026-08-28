package lister

import (
	"k8s.io/apimachinery/pkg/labels"
	corelisters "k8s.io/client-go/listers/core/v1"
)

type PersistentVolumeResolver struct {
	items corelisters.PersistentVolumeLister
}

func NewPersistentVolumeResolver(items corelisters.PersistentVolumeLister) *PersistentVolumeResolver {
	return &PersistentVolumeResolver{items: items}
}
func (r *PersistentVolumeResolver) Kind() string { return "PersistentVolume" }
func (r *PersistentVolumeResolver) List(scope ResourceScope) ([]Resource, error) {
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
