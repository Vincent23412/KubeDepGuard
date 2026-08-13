package rules

import (
	"github.com/vincent/KubeDepGuard/internal/dependency"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
)

type ServiceSelectorRule struct{ ConditionalRule }

func NewServiceSelectorRule() ServiceSelectorRule {
	return ServiceSelectorRule{ConditionalRule{RuleName: "EmptyServiceSelector"}}
}
func (r ServiceSelectorRule) HasMatch(svc *corev1.Service, pods []*corev1.Pod) bool {
	if len(svc.Spec.Selector) == 0 {
		return true
	}
	selector := labels.SelectorFromSet(svc.Spec.Selector)
	for _, pod := range pods {
		if pod.Namespace == svc.Namespace && selector.Matches(labels.Set(pod.Labels)) {
			return true
		}
	}
	return false
}
func (r ServiceSelectorRule) Validate(svc *corev1.Service, pods []*corev1.Pod) []dependency.Violation {
	if r.HasMatch(svc, pods) {
		return nil
	}
	return r.unsatisfied(svc, "selector matches no Pods")
}
