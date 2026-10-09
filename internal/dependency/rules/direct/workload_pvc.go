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

// WorkloadPersistentVolumeClaimRule validates PodSpec-based workload -> PVC references and PVC state.
type WorkloadPersistentVolumeClaimRule struct{ DirectReferenceRule }

func NewWorkloadPersistentVolumeClaimRule() WorkloadPersistentVolumeClaimRule {
	return WorkloadPersistentVolumeClaimRule{DirectReferenceRule{RuleName: "MissingPersistentVolumeClaim", TargetResourceKind: "PersistentVolumeClaim"}}
}

func (r WorkloadPersistentVolumeClaimRule) ValidateTargetDeletion(source metav1.Object, refs []ref.Reference, scope resolver.ResourceScope, name string) []dependency.Violation {
	return r.referencedBy(source, refs, scope, name, "Deployment")
}

func (r WorkloadPersistentVolumeClaimRule) Applies(request rules.Request) bool {
	return (request.Resource == "deployments" && (request.Operation == rules.Create || request.Operation == rules.Update)) ||
		(request.Resource == "persistentvolumeclaims" && request.Operation == rules.Delete)
}

func (r WorkloadPersistentVolumeClaimRule) Evaluate(request rules.Request, query resolver.Query) (rules.Result, error) {
	if request.Resource == "deployments" {
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
		resources, err := query.List(resolver.ResourceScope{Kind: r.TargetKind(), Namespace: source.GetNamespace()})
		if err != nil {
			return rules.Result{}, err
		}
		available := map[resolver.ResourceScope]map[string]struct{}{
			{Kind: r.TargetKind(), Namespace: source.GetNamespace()}: {},
		}
		for _, resource := range resources {
			available[resolver.ResourceScope{Kind: r.TargetKind(), Namespace: source.GetNamespace()}][resource.GetName()] = struct{}{}
		}
		violations := r.missing(source, references, available)
		return rules.Result{Violations: violations, Reject: len(violations) > 0 && dependency.ModeFor(source) == dependency.ModeEnforce}, nil
	}
	if request.Resource != "persistentvolumeclaims" {
		return rules.Result{}, nil
	}
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
}
