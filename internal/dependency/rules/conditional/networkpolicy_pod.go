package conditional

import (
	"fmt"

	"github.com/vincent/KubeDepGuard/internal/dependency"
	"github.com/vincent/KubeDepGuard/internal/dependency/resolver"
	"github.com/vincent/KubeDepGuard/internal/dependency/rules"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

// NetworkPolicyPodRule validates the conditional NetworkPolicy -> Pod
// dependency represented by NetworkPolicy.spec.podSelector.
type NetworkPolicyPodRule struct{ ConditionalRule }

func NewNetworkPolicyPodRule() NetworkPolicyPodRule {
	return NetworkPolicyPodRule{ConditionalRule{RuleName: "EmptyNetworkPolicySelector"}}
}

func (r NetworkPolicyPodRule) Validate(policy *networkingv1.NetworkPolicy, candidates []resolver.Resource) ([]dependency.Violation, error) {
	selector, err := metav1.LabelSelectorAsSelector(&policy.Spec.PodSelector)
	if err != nil {
		return nil, fmt.Errorf("invalid NetworkPolicy pod selector: %w", err)
	}
	for _, candidate := range candidates {
		if candidate.GetNamespace() == policy.Namespace && selector.Matches(labels.Set(candidate.GetLabels())) {
			return nil, nil
		}
	}
	return r.unsatisfied(policy, "pod selector matches no Pods"), nil
}

func (r NetworkPolicyPodRule) Applies(request rules.Request) bool {
	return (request.Resource == "networkpolicies" && (request.Operation == rules.Create || request.Operation == rules.Update)) ||
		(request.Resource == "pods" && (request.Operation == rules.Update || request.Operation == rules.Delete))
}

func (r NetworkPolicyPodRule) Evaluate(request rules.Request, query resolver.Query) (rules.Result, error) {
	switch request.Resource {
	case "networkpolicies":
		policy, ok := request.Object.(*networkingv1.NetworkPolicy)
		if !ok {
			return rules.Result{}, fmt.Errorf("expected NetworkPolicy object")
		}
		if dependency.ModeFor(policy) == dependency.ModeDisabled {
			return rules.Result{}, nil
		}
		pods, err := query.List(resolver.ResourceScope{Kind: "Pod", Namespace: policy.Namespace})
		if err != nil {
			return rules.Result{}, err
		}
		violations, err := r.Validate(policy, pods)
		if err != nil {
			return rules.Result{}, err
		}
		return rules.Result{Violations: violations, Reject: len(violations) > 0 && dependency.ModeFor(policy) == dependency.ModeEnforce}, nil
	case "pods":
		candidates, found, err := candidatePodsAfterChange(request, query)
		if err != nil || !found {
			return rules.Result{}, err
		}
		policies, err := query.List(resolver.ResourceScope{Kind: "NetworkPolicy", Namespace: request.Namespace})
		if err != nil {
			return rules.Result{}, err
		}
		var violations []dependency.Violation
		for _, resource := range policies {
			policy, ok := resource.(*networkingv1.NetworkPolicy)
			if !ok || dependency.ModeFor(policy) != dependency.ModeEnforce {
				continue
			}
			found, err := r.Validate(policy, candidates)
			if err != nil {
				return rules.Result{}, err
			}
			violations = append(violations, found...)
		}
		return rules.Result{Violations: violations, Reject: len(violations) > 0}, nil
	default:
		return rules.Result{}, nil
	}
}

// candidatePodsAfterChange returns the post-request Pod set for an update or
// delete, so selector rules can evaluate the prospective cluster state.
func candidatePodsAfterChange(request rules.Request, query resolver.Query) ([]resolver.Resource, bool, error) {
	pods, err := query.List(resolver.ResourceScope{Kind: "Pod", Namespace: request.Namespace})
	if err != nil {
		return nil, false, err
	}
	candidates := make([]resolver.Resource, 0, len(pods))
	found := false
	if request.Operation == rules.Update {
		updated, ok := request.Object.(*corev1.Pod)
		if !ok {
			return nil, false, fmt.Errorf("expected Pod object")
		}
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
			found = true
		}
		return candidates, found, nil
	}
	for _, candidate := range pods {
		if candidate.GetName() == request.Name {
			found = true
			continue
		}
		candidates = append(candidates, candidate)
	}
	return candidates, found, nil
}
