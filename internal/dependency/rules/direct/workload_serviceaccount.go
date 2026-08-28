package direct

import (
	"fmt"
	"github.com/vincent/KubeDepGuard/internal/dependency"
	referencecatalog "github.com/vincent/KubeDepGuard/internal/dependency/reference/catalog"
	"github.com/vincent/KubeDepGuard/internal/dependency/resolver"
	"github.com/vincent/KubeDepGuard/internal/dependency/rules"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
)

// WorkloadServiceAccountRule validates the effective ServiceAccount of PodSpec-based sources.
type WorkloadServiceAccountRule struct{}

func NewWorkloadServiceAccountRule() WorkloadServiceAccountRule { return WorkloadServiceAccountRule{} }
func (WorkloadServiceAccountRule) Name() string                 { return "MissingServiceAccount" }
func (WorkloadServiceAccountRule) Applies(r rules.Request) bool {
	return (r.Resource == "pods" || r.Resource == "deployments") && (r.Operation == rules.Create || r.Operation == rules.Update)
}
func (r WorkloadServiceAccountRule) Evaluate(req rules.Request, query resolver.Query) (rules.Result, error) {
	source, ok := req.Object.(v1.Object)
	if !ok {
		return rules.Result{}, fmt.Errorf("expected %s object", req.Resource)
	}
	refs, err := referencecatalog.DefaultDirectRegistry.Extract(sourceKind(req.Resource), req.Object)
	if err != nil {
		return rules.Result{}, err
	}
	accounts, err := query.List(resolver.ResourceScope{Kind: "ServiceAccount", Namespace: source.GetNamespace()})
	if err != nil {
		return rules.Result{}, err
	}
	available := map[string]bool{}
	for _, account := range accounts {
		available[account.GetName()] = true
	}
	var violations []dependency.Violation
	for _, ref := range refs {
		if ref.TargetKind == "ServiceAccount" && !available[ref.Name] {
			violations = append(violations, dependency.Violation{Rule: r.Name(), Resource: source.GetNamespace() + "/" + source.GetName(), Message: fmt.Sprintf("references ServiceAccount %q at %s which does not exist", ref.Name, ref.FieldPath)})
		}
	}
	return rules.Result{Violations: violations, Reject: len(violations) > 0 && dependency.ModeFor(source) == dependency.ModeEnforce}, nil
}
