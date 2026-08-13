package rules

import (
	"fmt"

	"github.com/vincent/KubeDepGuard/internal/dependency"
	"github.com/vincent/KubeDepGuard/internal/dependency/catalog"
	ref "github.com/vincent/KubeDepGuard/internal/dependency/reference"
	"github.com/vincent/KubeDepGuard/internal/dependency/resolver"
	corev1 "k8s.io/api/core/v1"
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

func (r PodConfigMapRule) Applies(request Request) bool {
	return (request.Resource == "pods" && (request.Operation == Create || request.Operation == Update)) ||
		(request.Resource == "configmaps" && request.Operation == Delete)
}

func (r PodConfigMapRule) Evaluate(request Request, query resolver.Query) (Result, error) {
	switch {
	case request.Resource == "pods":
		pod, ok := request.Object.(*corev1.Pod)
		if !ok {
			return Result{}, fmt.Errorf("expected Pod object")
		}
		if dependency.ModeFor(pod) == dependency.ModeDisabled {
			return Result{}, nil
		}
		references, err := catalog.DefaultReferenceRegistry.Extract("Pod", pod)
		if err != nil {
			return Result{}, err
		}
		violations := r.Validate(pod, references, query)
		return Result{Violations: violations, Reject: len(violations) > 0 && dependency.ModeFor(pod) == dependency.ModeEnforce}, nil
	case request.Resource == "configmaps":
		target := resolver.Target{Kind: r.TargetKind(), Namespace: request.Namespace, Name: request.Name}
		pods, err := query.ListPods(request.Namespace)
		if err != nil {
			return Result{}, err
		}
		var violations []dependency.Violation
		for _, pod := range pods {
			if dependency.ModeFor(pod) != dependency.ModeEnforce {
				continue
			}
			references, err := catalog.DefaultReferenceRegistry.Extract("Pod", pod)
			if err != nil {
				return Result{}, err
			}
			violations = append(violations, r.ValidateTargetDeletion(pod, references, target)...)
		}
		return Result{Violations: violations, Reject: len(violations) > 0}, nil
	default:
		return Result{}, nil
	}
}
