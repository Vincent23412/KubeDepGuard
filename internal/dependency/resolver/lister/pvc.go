package lister

import (
	"k8s.io/apimachinery/pkg/labels"
	corelisters "k8s.io/client-go/listers/core/v1"
)

// PersistentVolumeClaimResolver lists PVC resources from an Informer cache.
type PersistentVolumeClaimResolver struct {
	pvcs corelisters.PersistentVolumeClaimLister
}

func NewPersistentVolumeClaimResolver(pvcs corelisters.PersistentVolumeClaimLister) *PersistentVolumeClaimResolver {
	return &PersistentVolumeClaimResolver{pvcs: pvcs}
}

func (r *PersistentVolumeClaimResolver) Kind() string { return "PersistentVolumeClaim" }

func (r *PersistentVolumeClaimResolver) List(scope ResourceScope) ([]Resource, error) {
	items, err := r.pvcs.PersistentVolumeClaims(scope.Namespace).List(labels.Everything())
	if err != nil {
		return nil, err
	}
	resources := make([]Resource, 0, len(items))
	for _, item := range items {
		resources = append(resources, item)
	}
	return resources, nil
}
