// Package rules contains dependency validation semantics and the enabled-rule registry.
package rules

import (
	"github.com/vincent/KubeDepGuard/internal/dependency"
	ref "github.com/vincent/KubeDepGuard/internal/dependency/reference"
	"github.com/vincent/KubeDepGuard/internal/dependency/resolver"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Rule interface{ Name() string }
type DirectRule interface {
	Rule
	TargetKind() string
	Validate(metav1.Object, []ref.Reference, resolver.TargetResolver) []dependency.Violation
}
type ServiceConditionalRule interface {
	Rule
	HasMatch(*corev1.Service, []*corev1.Pod) bool
	Validate(*corev1.Service, []*corev1.Pod) []dependency.Violation
}

type Registry struct {
	direct      []DirectRule
	conditional []ServiceConditionalRule
}

func (r *Registry) DirectReferenceRules() []DirectRule { return append([]DirectRule(nil), r.direct...) }
func (r *Registry) ServiceConditionalRules() []ServiceConditionalRule {
	return append([]ServiceConditionalRule(nil), r.conditional...)
}

var DefaultRegistry = &Registry{direct: []DirectRule{NewConfigMapReferenceRule()}, conditional: []ServiceConditionalRule{NewServiceSelectorRule()}}
