package rules

import (
	"github.com/vincent/KubeDepGuard/internal/dependency"
	serviceRef "github.com/vincent/KubeDepGuard/internal/dependency/reference/service"
	"github.com/vincent/KubeDepGuard/internal/dependency/resolver"
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

func (r ServicePodRule) Validate(svc *corev1.Service, selectorRef serviceRef.Selector, pods resolver.PodLister) ([]dependency.Violation, error) {
	if len(selectorRef.Labels) == 0 {
		return nil, nil
	}
	candidates, err := pods.List(selectorRef.Namespace)
	if err != nil {
		return nil, err
	}
	selector := labels.SelectorFromSet(selectorRef.Labels)
	for _, pod := range candidates {
		if pod.Namespace == selectorRef.Namespace && selector.Matches(labels.Set(pod.Labels)) {
			return nil, nil
		}
	}
	return r.unsatisfied(svc, "selector matches no Pods"), nil
}
