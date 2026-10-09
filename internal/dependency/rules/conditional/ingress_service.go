package conditional

import (
	"fmt"
	"github.com/vincent/KubeDepGuard/internal/dependency"
	ref "github.com/vincent/KubeDepGuard/internal/dependency/reference"
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
	return (r.Resource == "ingresses" && (r.Operation == rules.Create || r.Operation == rules.Update)) ||
		(r.Resource == "services" && (r.Operation == rules.Update || r.Operation == rules.Delete))
}
func (r IngressServiceRule) Evaluate(req rules.Request, query resolver.Query) (rules.Result, error) {
	if req.Resource == "services" {
		return r.validateServiceChange(req, query)
	}
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
	violations := validateIngressReferences(ing, refs, byName)
	return rules.Result{Violations: violations, Reject: len(violations) > 0 && dependency.ModeFor(ing) == dependency.ModeEnforce}, nil
}

func (r IngressServiceRule) validateServiceChange(req rules.Request, query resolver.Query) (rules.Result, error) {
	ingresses, err := query.List(resolver.ResourceScope{Kind: "Ingress", Namespace: req.Namespace})
	if err != nil {
		return rules.Result{}, err
	}
	var updated *corev1.Service
	if req.Operation == rules.Update {
		var ok bool
		updated, ok = req.Object.(*corev1.Service)
		if !ok {
			return rules.Result{}, fmt.Errorf("expected Service object")
		}
	}
	var violations []dependency.Violation
	for _, item := range ingresses {
		ing, ok := item.(*networking.Ingress)
		if !ok || dependency.ModeFor(ing) != dependency.ModeEnforce {
			continue
		}
		refs, err := referencecatalog.DefaultDirectRegistry.Extract("Ingress", ing)
		if err != nil {
			return rules.Result{}, err
		}
		violations = append(violations, referencedServiceViolations(ing, refs, req.Name, updated, req.Operation)...)
	}
	return rules.Result{Violations: violations, Reject: len(violations) > 0}, nil
}

func validateIngressReferences(ing *networking.Ingress, refs []ref.Reference, services map[string]*corev1.Service) []dependency.Violation {
	var violations []dependency.Violation
	for _, reference := range refs {
		svc := services[reference.Name]
		if svc == nil {
			violations = append(violations, dependency.Violation{Rule: "MissingIngressService", Resource: ing.Namespace + "/" + ing.Name, Message: fmt.Sprintf("references Service %q at %s which does not exist", reference.Name, reference.FieldPath)})
			continue
		}
		if reference.Key != "" && !servicePortExists(svc, reference.Key) {
			violations = append(violations, dependency.Violation{Rule: "MissingIngressServicePort", Resource: ing.Namespace + "/" + ing.Name, Message: fmt.Sprintf("references Service %q port %q at %s which does not exist", reference.Name, reference.Key, reference.FieldPath)})
		}
	}
	return violations
}

func referencedServiceViolations(ing *networking.Ingress, refs []ref.Reference, name string, updated *corev1.Service, operation rules.Operation) []dependency.Violation {
	for _, reference := range refs {
		if reference.TargetKind != "Service" || reference.Name != name {
			continue
		}
		if operation == rules.Delete {
			return []dependency.Violation{{Rule: "ReferencedServiceDeletion", Resource: ing.Namespace + "/" + ing.Name, Message: fmt.Sprintf("references Service %q which is being deleted", name)}}
		}
		if reference.Key != "" && !servicePortExists(updated, reference.Key) {
			return []dependency.Violation{{Rule: "ReferencedServicePortUpdate", Resource: ing.Namespace + "/" + ing.Name, Message: fmt.Sprintf("references Service %q port %q which is being removed", name, reference.Key)}}
		}
	}
	return nil
}

func servicePortExists(service *corev1.Service, port string) bool {
	for _, item := range service.Spec.Ports {
		if item.Name == port || fmt.Sprintf("%d", item.Port) == port {
			return true
		}
	}
	return false
}
