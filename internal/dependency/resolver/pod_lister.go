package resolver

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	corelisters "k8s.io/client-go/listers/core/v1"
)

// PodLister reads candidate Pods from the current cluster state.
type PodLister interface {
	List(namespace string) ([]*corev1.Pod, error)
}

type InformerPodLister struct{ pods corelisters.PodLister }

func NewInformerPodLister(pods corelisters.PodLister) *InformerPodLister {
	return &InformerPodLister{pods: pods}
}
func (l *InformerPodLister) List(namespace string) ([]*corev1.Pod, error) {
	return l.pods.Pods(namespace).List(labels.Everything())
}
