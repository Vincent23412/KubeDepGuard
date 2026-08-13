package resolver

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	corelisters "k8s.io/client-go/listers/core/v1"
)

// InformerQuery exposes the Kubernetes state needed by dependency rules.
// Its reads are served entirely from informer caches.
type InformerQuery struct {
	pods     corelisters.PodLister
	services corelisters.ServiceLister
	targets  *ListerResolver
}

func NewInformerQuery(pods corelisters.PodLister, configMaps corelisters.ConfigMapLister, services corelisters.ServiceLister) *InformerQuery {
	return &InformerQuery{pods: pods, services: services, targets: NewListerResolver(configMaps)}
}

func (q *InformerQuery) Exists(target Target) (bool, error) {
	return q.targets.Exists(target)
}

func (q *InformerQuery) GetPod(namespace, name string) (*corev1.Pod, error) {
	return q.pods.Pods(namespace).Get(name)
}

func (q *InformerQuery) ListPods(namespace string) ([]*corev1.Pod, error) {
	return q.pods.Pods(namespace).List(labels.Everything())
}

func (q *InformerQuery) ListServices(namespace string) ([]*corev1.Service, error) {
	return q.services.Services(namespace).List(labels.Everything())
}
