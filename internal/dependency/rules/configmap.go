package rules

import (
	"github.com/vincent/KubeDepGuard/internal/dependency"
	ref "github.com/vincent/KubeDepGuard/internal/dependency/reference"
	"github.com/vincent/KubeDepGuard/internal/dependency/resolver"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ConfigMapReferenceRule struct{ DirectReferenceRule }

func NewConfigMapReferenceRule() ConfigMapReferenceRule {
	return ConfigMapReferenceRule{DirectReferenceRule{RuleName: "MissingConfigMap", TargetResourceKind: "ConfigMap"}}
}
func (r ConfigMapReferenceRule) Validate(source metav1.Object, refs []ref.Reference, targets resolver.TargetResolver) []dependency.Violation {
	return r.missing(source, refs, targets)
}
