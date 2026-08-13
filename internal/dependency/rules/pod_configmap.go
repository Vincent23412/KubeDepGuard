package rules

import (
	"github.com/vincent/KubeDepGuard/internal/dependency"
	ref "github.com/vincent/KubeDepGuard/internal/dependency/reference"
	"github.com/vincent/KubeDepGuard/internal/dependency/resolver"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PodConfigMapRule validates direct Pod -> ConfigMap references.
type PodConfigMapRule struct{ DirectReferenceRule }

func NewPodConfigMapRule() PodConfigMapRule {
	return PodConfigMapRule{DirectReferenceRule{RuleName: "MissingConfigMap", TargetResourceKind: "ConfigMap"}}
}

func (r PodConfigMapRule) Validate(source metav1.Object, refs []ref.Reference, targets resolver.TargetResolver) []dependency.Violation {
	return r.missing(source, refs, targets)
}

func (r PodConfigMapRule) ValidateTargetDeletion(source metav1.Object, refs []ref.Reference, target resolver.Target) []dependency.Violation {
	return r.referencedBy(source, refs, target, "Pod")
}
