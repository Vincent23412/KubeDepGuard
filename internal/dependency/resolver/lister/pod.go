package lister

import (
	"k8s.io/apimachinery/pkg/labels"
	corelisters "k8s.io/client-go/listers/core/v1"
)

// PodResolver lists Pod resources from the current cluster state.
type PodResolver struct{ pods corelisters.PodLister }

func NewPodResolver(pods corelisters.PodLister) *PodResolver {
	return &PodResolver{pods: pods}
}
func (r *PodResolver) Kind() string { return "Pod" }
func (r *PodResolver) List(scope ResourceScope) ([]Resource, error) {
	items, err := r.pods.Pods(scope.Namespace).List(labels.Everything())
	if err != nil {
		return nil, err
	}
	resources := make([]Resource, 0, len(items))
	for _, item := range items {
		resources = append(resources, item)
	}
	return resources, nil
}
