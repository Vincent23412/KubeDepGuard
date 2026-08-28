package direct

import (
	"fmt"

	"github.com/vincent/KubeDepGuard/internal/dependency"
	ref "github.com/vincent/KubeDepGuard/internal/dependency/reference"
	referencecatalog "github.com/vincent/KubeDepGuard/internal/dependency/reference/catalog"
	"github.com/vincent/KubeDepGuard/internal/dependency/resolver"
	"github.com/vincent/KubeDepGuard/internal/dependency/rules"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PodSecretRule validates direct Pod -> Secret references.
type PodSecretRule struct{ DirectReferenceRule }

func NewPodSecretRule() PodSecretRule {
	return PodSecretRule{DirectReferenceRule{RuleName: "MissingSecret", TargetResourceKind: "Secret"}}
}

func (r PodSecretRule) ValidateTargetDeletion(source metav1.Object, refs []ref.Reference, scope resolver.ResourceScope, name string) []dependency.Violation {
	return r.referencedBy(source, refs, scope, name, "Pod")
}

func (r PodSecretRule) Applies(request rules.Request) bool {
	return (request.Resource == "pods" && (request.Operation == rules.Create || request.Operation == rules.Update)) ||
		(request.Resource == "secrets" && request.Operation == rules.Delete)
}

func (r PodSecretRule) Evaluate(request rules.Request, query resolver.Query) (rules.Result, error) {
	switch {
	case request.Resource == "pods":
		pod, ok := request.Object.(*corev1.Pod)
		if !ok {
			return rules.Result{}, fmt.Errorf("expected Pod object")
		}
		if dependency.ModeFor(pod) == dependency.ModeDisabled {
			return rules.Result{}, nil
		}
		references, err := referencecatalog.DefaultDirectRegistry.Extract("Pod", pod)
		if err != nil {
			return rules.Result{}, err
		}
		available, err := r.availableTargets(references, query)
		if err != nil {
			return rules.Result{}, err
		}
		violations := r.missing(pod, references, available)
		return rules.Result{Violations: violations, Reject: len(violations) > 0 && dependency.ModeFor(pod) == dependency.ModeEnforce}, nil
	case request.Resource == "secrets":
		pods, err := query.List(resolver.ResourceScope{Kind: "Pod", Namespace: request.Namespace})
		if err != nil {
			return rules.Result{}, err
		}
		var violations []dependency.Violation
		for _, resource := range pods {
			pod, ok := resource.(*corev1.Pod)
			if !ok || dependency.ModeFor(pod) != dependency.ModeEnforce {
				continue
			}
			references, err := referencecatalog.DefaultDirectRegistry.Extract("Pod", pod)
			if err != nil {
				return rules.Result{}, err
			}
			violations = append(violations, r.ValidateTargetDeletion(pod, references, resolver.ResourceScope{Kind: r.TargetKind(), Namespace: request.Namespace}, request.Name)...)
		}
		return rules.Result{Violations: violations, Reject: len(violations) > 0}, nil
	default:
		return rules.Result{}, nil
	}
}

func (r PodSecretRule) availableTargets(references []ref.Reference, query resolver.Query) (map[resolver.ResourceScope]map[string]struct{}, error) {
	available := make(map[resolver.ResourceScope]map[string]struct{})
	for _, reference := range references {
		if reference.TargetKind != r.TargetKind() || reference.Optional {
			continue
		}
		scope := resolver.ResourceScope{Kind: reference.TargetKind, Namespace: reference.Namespace}
		if _, listed := available[scope]; listed {
			continue
		}
		resources, err := query.List(scope)
		if err != nil {
			return nil, err
		}
		names := make(map[string]struct{}, len(resources))
		for _, resource := range resources {
			names[resource.GetName()] = struct{}{}
		}
		available[scope] = names
	}
	return available, nil
}
