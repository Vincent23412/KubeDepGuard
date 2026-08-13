package reference

import (
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
)

// Extractor owns typed field traversal for one source resource kind.
type Extractor interface {
	SourceKind() string
	Extract(runtime.Object) ([]Reference, error)
}

type Factory func() Extractor

// Registry permits several focused extractors for a source kind, such as Pod
// ConfigMap, Secret, and PVC extractors.
type Registry struct{ bySourceKind map[string][]Extractor }

func NewRegistry(factories ...Factory) *Registry {
	r := &Registry{bySourceKind: make(map[string][]Extractor, len(factories))}
	for _, factory := range factories {
		extractor := factory()
		if extractor == nil {
			panic("reference extractor factory returned nil")
		}
		r.bySourceKind[extractor.SourceKind()] = append(r.bySourceKind[extractor.SourceKind()], extractor)
	}
	return r
}

func (r *Registry) Extract(kind string, object runtime.Object) ([]Reference, error) {
	extractors, ok := r.bySourceKind[kind]
	if !ok {
		return nil, fmt.Errorf("no reference extractor registered for %s", kind)
	}
	var references []Reference
	for _, extractor := range extractors {
		extracted, err := extractor.Extract(object)
		if err != nil {
			return nil, err
		}
		references = append(references, extracted...)
	}
	return references, nil
}
