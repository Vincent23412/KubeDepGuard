package ingress

import (
	"fmt"
	ref "github.com/vincent/KubeDepGuard/internal/dependency/reference"
	networking "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type ServiceExtractor struct{}

func NewServiceExtractor() ref.Extractor    { return ServiceExtractor{} }
func (ServiceExtractor) SourceKind() string { return "Ingress" }
func (ServiceExtractor) Extract(object runtime.Object) ([]ref.Reference, error) {
	ingress, ok := object.(*networking.Ingress)
	if !ok {
		return nil, fmt.Errorf("Ingress Service extractor received %T", object)
	}
	refs := make([]ref.Reference, 0)
	add := func(name, path string, port networking.ServiceBackendPort) {
		if name != "" {
			key := ""
			if port.Name != "" {
				key = port.Name
			} else if port.Number != 0 {
				key = fmt.Sprintf("%d", port.Number)
			}
			refs = append(refs, ref.Reference{SourceKind: "Ingress", TargetKind: "Service", Namespace: ingress.Namespace, Name: name, FieldPath: path, Key: key})
		}
	}
	for ri, rule := range ingress.Spec.Rules {
		if rule.HTTP == nil {
			continue
		}
		for pi, path := range rule.HTTP.Paths {
			if path.Backend.Service != nil {
				add(path.Backend.Service.Name, fmt.Sprintf("spec.rules[%d].http.paths[%d].backend.service.name", ri, pi), path.Backend.Service.Port)
			}
		}
	}
	if ingress.Spec.DefaultBackend != nil && ingress.Spec.DefaultBackend.Service != nil {
		add(ingress.Spec.DefaultBackend.Service.Name, "spec.defaultBackend.service.name", ingress.Spec.DefaultBackend.Service.Port)
	}
	return refs, nil
}
