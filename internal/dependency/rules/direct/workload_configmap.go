package direct

import (
	"fmt"

	"github.com/vincent/KubeDepGuard/internal/dependency"
	ref "github.com/vincent/KubeDepGuard/internal/dependency/reference"
	referencecatalog "github.com/vincent/KubeDepGuard/internal/dependency/reference/catalog"
	"github.com/vincent/KubeDepGuard/internal/dependency/resolver"
	"github.com/vincent/KubeDepGuard/internal/dependency/rules"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// WorkloadConfigMapRule validates direct PodSpec-based workload -> ConfigMap references.
type WorkloadConfigMapRule struct{ DirectReferenceRule }

func NewWorkloadConfigMapRule() WorkloadConfigMapRule {
	return WorkloadConfigMapRule{DirectReferenceRule{RuleName: "MissingConfigMap", TargetResourceKind: "ConfigMap"}}
}

func (r WorkloadConfigMapRule) ValidateTargetDeletion(source metav1.Object, refs []ref.Reference, scope resolver.ResourceScope, name string) []dependency.Violation {
	return r.referencedBy(source, refs, scope, name, "Deployment")
}

func (r WorkloadConfigMapRule) Applies(request rules.Request) bool {
	return (request.Resource == "deployments" && (request.Operation == rules.Create || request.Operation == rules.Update)) ||
		(request.Resource == "configmaps" && request.Operation == rules.Delete)
}

func (r WorkloadConfigMapRule) Evaluate(request rules.Request, query resolver.Query) (rules.Result, error) {
	switch {
	case request.Resource == "deployments":
		source, ok := request.Object.(metav1.Object)
		if !ok {
			return rules.Result{}, fmt.Errorf("expected %s object", request.Resource)
		}
		if dependency.ModeFor(source) == dependency.ModeDisabled {
			return rules.Result{}, nil
		}
		references, err := referencecatalog.DefaultDirectRegistry.Extract(sourceKind(request.Resource), request.Object)
		if err != nil {
			return rules.Result{}, err
		}
		available, err := r.availableTargets(references, query)
		if err != nil {
			return rules.Result{}, err
		}
		violations := r.missing(source, references, available)
		keyViolations, err := r.missingKeys(source, references, query)
		if err != nil {
			return rules.Result{}, err
		}
		violations = append(violations, keyViolations...)
		return rules.Result{Violations: violations, Reject: len(violations) > 0 && dependency.ModeFor(source) == dependency.ModeEnforce}, nil
	case request.Resource == "configmaps":
		var violations []dependency.Violation
		for _, kind := range []string{"Deployment"} {
			resources, err := query.List(resolver.ResourceScope{Kind: kind, Namespace: request.Namespace})
			if err != nil {
				return rules.Result{}, err
			}
			for _, resource := range resources {
				source, ok := resource.(metav1.Object)
				if !ok || dependency.ModeFor(source) != dependency.ModeEnforce {
					continue
				}
				references, err := referencecatalog.DefaultDirectRegistry.Extract(kind, resource)
				if err != nil {
					return rules.Result{}, err
				}
				violations = append(violations, r.referencedBy(source, references, resolver.ResourceScope{Kind: r.TargetKind(), Namespace: request.Namespace}, request.Name, kind)...)
			}
		}
		return rules.Result{Violations: violations, Reject: len(violations) > 0}, nil
	default:
		return rules.Result{}, nil
	}
}

func sourceKind(resource string) string {
	if resource == "deployments" {
		return "Deployment"
	}
	return "Pod"
}

func (r WorkloadConfigMapRule) availableTargets(references []ref.Reference, query resolver.Query) (map[resolver.ResourceScope]map[string]struct{}, error) {
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
