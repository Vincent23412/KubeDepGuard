package conditional

import (
	"fmt"

	"github.com/vincent/KubeDepGuard/internal/dependency"
	referencecatalog "github.com/vincent/KubeDepGuard/internal/dependency/reference/catalog"
	serviceRef "github.com/vincent/KubeDepGuard/internal/dependency/reference/service"
	"github.com/vincent/KubeDepGuard/internal/dependency/resolver"
	"github.com/vincent/KubeDepGuard/internal/dependency/rules"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
)

// ServicePodRule validates the conditional Service -> Pod dependency.
// The Service selector is extracted by reference/service; this rule compares it
// with current Pod state obtained through the resolver.
type ServicePodRule struct{ ConditionalRule }

func NewServicePodRule() ServicePodRule {
	return ServicePodRule{ConditionalRule{RuleName: "EmptyServiceSelector"}}
}

func (r ServicePodRule) Validate(svc *corev1.Service, selectorRef serviceRef.Selector, candidates []resolver.Resource) []dependency.Violation {
	if len(selectorRef.Labels) == 0 {
		return nil
	}
	selector := labels.SelectorFromSet(selectorRef.Labels)
	for _, candidate := range candidates {
		if candidate.GetNamespace() == selectorRef.Namespace && selector.Matches(labels.Set(candidate.GetLabels())) {
			return nil
		}
	}
	return r.unsatisfied(svc, "selector matches no Pods")
}

func (r ServicePodRule) Applies(request rules.Request) bool {
	return (request.Resource == "services" && (request.Operation == rules.Create || request.Operation == rules.Update)) ||
		(request.Resource == "pods" && (request.Operation == rules.Update || request.Operation == rules.Delete))
}

func (r ServicePodRule) Evaluate(request rules.Request, query resolver.Query) (rules.Result, error) {
	switch {
	case request.Resource == "services":
		svc, ok := request.Object.(*corev1.Service)
		if !ok {
			return rules.Result{}, fmt.Errorf("expected Service object")
		}
		if dependency.ModeFor(svc) == dependency.ModeDisabled {
			return rules.Result{}, nil
		}
		selector, err := referencecatalog.ExtractServiceSelector(svc)
		if err != nil {
			return rules.Result{}, err
		}
		pods, err := query.List(resolver.ResourceScope{Kind: "Pod", Namespace: svc.Namespace})
		if err != nil {
			return rules.Result{}, err
		}
		violations := r.Validate(svc, selector, pods)
		return rules.Result{Violations: violations, Reject: len(violations) > 0 && dependency.ModeFor(svc) == dependency.ModeEnforce}, nil
	case request.Resource == "pods":
		pods, err := query.List(resolver.ResourceScope{Kind: "Pod", Namespace: request.Namespace})
		if err != nil {
			return rules.Result{}, err
		}
		candidates := make([]resolver.Resource, 0, len(pods))
		if request.Operation == rules.Update {
			updated, ok := request.Object.(*corev1.Pod)
			if !ok {
				return rules.Result{}, fmt.Errorf("expected Pod object")
			}
			found := false
			for _, candidate := range pods {
				if candidate.GetName() == request.Name {
					candidates = append(candidates, updated)
					found = true
					continue
				}
				candidates = append(candidates, candidate)
			}
			if !found {
				candidates = append(candidates, updated)
			}
		} else {
			foundPod := false
			for _, candidate := range pods {
				if candidate.GetName() == request.Name {
					foundPod = true
					continue
				}
				candidates = append(candidates, candidate)
			}
			if !foundPod {
				return rules.Result{}, nil
			}
		}
		services, err := query.List(resolver.ResourceScope{Kind: "Service", Namespace: request.Namespace})
		if err != nil {
			return rules.Result{}, err
		}
		var violations []dependency.Violation
		for _, resource := range services {
			svc, ok := resource.(*corev1.Service)
			if !ok {
				continue
			}
			if dependency.ModeFor(svc) != dependency.ModeEnforce {
				continue
			}
			selector, err := referencecatalog.ExtractServiceSelector(svc)
			if err != nil {
				return rules.Result{}, err
			}
			found := r.Validate(svc, selector, candidates)
			violations = append(violations, found...)
		}
		return rules.Result{Violations: violations, Reject: len(violations) > 0}, nil
	default:
		return rules.Result{}, nil
	}
}
