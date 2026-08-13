// Package service contains field extractors for Service resources.
package service

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// Selector is the raw Service selector field. It contains no Pod-matching logic.
type Selector struct {
	Namespace string
	Name      string
	Labels    map[string]string
}

type SelectorExtractor struct{}

func NewSelectorExtractor() SelectorExtractor { return SelectorExtractor{} }
func (SelectorExtractor) Extract(object runtime.Object) (Selector, error) {
	svc, ok := object.(*corev1.Service)
	if !ok {
		return Selector{}, fmt.Errorf("Service selector extractor received %T", object)
	}
	labels := make(map[string]string, len(svc.Spec.Selector))
	for key, value := range svc.Spec.Selector {
		labels[key] = value
	}
	return Selector{Namespace: svc.Namespace, Name: svc.Name, Labels: labels}, nil
}
