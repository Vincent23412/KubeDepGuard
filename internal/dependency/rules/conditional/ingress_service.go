package conditional

import (
	"fmt"
	"github.com/vincent/KubeDepGuard/internal/dependency"
	referencecatalog "github.com/vincent/KubeDepGuard/internal/dependency/reference/catalog"
	"github.com/vincent/KubeDepGuard/internal/dependency/resolver"
	"github.com/vincent/KubeDepGuard/internal/dependency/rules"
	corev1 "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1"
)

// IngressServiceRule validates Ingress backend Services and their ports.
type IngressServiceRule struct{}

func NewIngressServiceRule() IngressServiceRule { return IngressServiceRule{} }
func (IngressServiceRule) Name() string         { return "MissingIngressService" }
func (IngressServiceRule) Applies(r rules.Request) bool {
	return r.Resource == "ingresses" && (r.Operation == rules.Create || r.Operation == rules.Update)
}
func (r IngressServiceRule) Evaluate(req rules.Request, query resolver.Query) (rules.Result, error) {
	ing, ok := req.Object.(*networking.Ingress)
	if !ok {
		return rules.Result{}, fmt.Errorf("expected Ingress object")
	}
	if dependency.ModeFor(ing) == dependency.ModeDisabled {
		return rules.Result{}, nil
	}
	refs, err := referencecatalog.DefaultDirectRegistry.Extract("Ingress", ing)
	if err != nil {
		return rules.Result{}, err
	}
	services, err := query.List(resolver.ResourceScope{Kind: "Service", Namespace: ing.Namespace})
	if err != nil {
		return rules.Result{}, err
	}
	byName := map[string]*corev1.Service{}
	for _, item := range services {
		if svc, ok := item.(*corev1.Service); ok {
			byName[svc.Name] = svc
		}
	}
	var violations []dependency.Violation
	for _, ref := range refs {
		svc := byName[ref.Name]
		if svc == nil {
			violations = append(violations, dependency.Violation{Rule: r.Name(), Resource: ing.Namespace + "/" + ing.Name, Message: fmt.Sprintf("references Service %q at %s which does not exist", ref.Name, ref.FieldPath)})
			continue
		}
		if ref.Key != "" && !servicePortExists(svc, ref.Key) {
			violations = append(violations, dependency.Violation{Rule: "MissingIngressServicePort", Resource: ing.Namespace + "/" + ing.Name, Message: fmt.Sprintf("references Service %q port %q at %s which does not exist", ref.Name, ref.Key, ref.FieldPath)})
		}
	}
	return rules.Result{Violations: violations, Reject: len(violations) > 0 && dependency.ModeFor(ing) == dependency.ModeEnforce}, nil
}

func servicePortExists(service *corev1.Service, port string) bool {
	for _, item := range service.Spec.Ports {
		if item.Name == port || fmt.Sprintf("%d", item.Port) == port {
			return true
		}
	}
	return false
}
