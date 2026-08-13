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
	Validate(metav1.Object, []ref.Reference, resolver.TargetResolver) []dependency.Violation
	ValidateTargetDeletion(metav1.Object, []ref.Reference, resolver.Target) []dependency.Violation
}
type ServiceConditionalRule interface {
	Rule
	Validate(*corev1.Service, serviceRef.Selector, resolver.PodLister) ([]dependency.Violation, error)
}

type Registry struct {
	direct      []DirectRule
	conditional []ServiceConditionalRule
}

func (r *Registry) DirectReferenceRules() []DirectRule { return append([]DirectRule(nil), r.direct...) }
func (r *Registry) ServiceConditionalRules() []ServiceConditionalRule {
	return append([]ServiceConditionalRule(nil), r.conditional...)
}

var DefaultRegistry = &Registry{direct: []DirectRule{NewPodConfigMapRule()}, conditional: []ServiceConditionalRule{NewServicePodRule()}}
