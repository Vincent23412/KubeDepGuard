package resolver

import corev1 "k8s.io/api/core/v1"

// Query is the read-only cluster state available to dependency rules.
// Implementations may use informer caches, direct API calls, or test fakes.
type Query interface {
	TargetResolver
	GetPod(namespace, name string) (*corev1.Pod, error)
	ListPods(namespace string) ([]*corev1.Pod, error)
	ListServices(namespace string) ([]*corev1.Service, error)
}
