package lister

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// ResourceScope identifies the collection of resources a rule needs to inspect.
type ResourceScope struct{ Kind, Namespace string }

// Resource is the common Kubernetes object surface used by dependency rules.
type Resource interface {
	runtime.Object
	metav1.Object
}

// ResourceLister lists resources of one Kubernetes kind from a backing store.
type ResourceLister interface {
	Kind() string
	List(ResourceScope) ([]Resource, error)
}
