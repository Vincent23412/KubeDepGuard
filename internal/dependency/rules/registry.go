// Package rules contains dependency validation semantics and the enabled-rule registry.
package rules

import (
	"github.com/vincent/KubeDepGuard/internal/dependency"
	ref "github.com/vincent/KubeDepGuard/internal/dependency/reference"
	serviceRef "github.com/vincent/KubeDepGuard/internal/dependency/reference/service"
	"github.com/vincent/KubeDepGuard/internal/dependency/resolver"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Rule interface{ Name() string }
type DirectRule interface {
	Rule
	TargetKind() string
	ValidateTargetDeletion(metav1.Object, []ref.Reference, resolver.ResourceScope, string) []dependency.Violation
}
type ServiceConditionalRule interface {
	Rule
	Validate(*corev1.Service, serviceRef.Selector, []resolver.Resource) []dependency.Violation
}

type Registry struct {
	direct      []DirectRule
	conditional []ServiceConditionalRule
	admission   []AdmissionRule
}

func (r *Registry) DirectReferenceRules() []DirectRule { return append([]DirectRule(nil), r.direct...) }
func (r *Registry) ServiceConditionalRules() []ServiceConditionalRule {
	return append([]ServiceConditionalRule(nil), r.conditional...)
}
func (r *Registry) AdmissionRules() []AdmissionRule {
	return append([]AdmissionRule(nil), r.admission...)
}

// NewRegistry classifies concrete rules by their supported validation style.
func NewRegistry(admissionRules ...AdmissionRule) *Registry {
	registry := &Registry{admission: append([]AdmissionRule(nil), admissionRules...)}
	for _, rule := range admissionRules {
		if direct, ok := rule.(DirectRule); ok {
			registry.direct = append(registry.direct, direct)
		}
		if conditional, ok := rule.(ServiceConditionalRule); ok {
			registry.conditional = append(registry.conditional, conditional)
		}
	}
	return registry
}
