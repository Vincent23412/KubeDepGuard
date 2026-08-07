package dependency

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
)

// ServiceSelectorRule validates the conditional Service -> Pod selector dependency.
// It embeds ConditionalRule to reuse the common unsatisfied-dependency behaviour.
type ServiceSelectorRule struct {
	ConditionalRule
}

var _ ServiceConditionalRule = ServiceSelectorRule{}

func NewServicePodSelectorRule() Rule {
	return ServiceSelectorRule{
		ConditionalRule: ConditionalRule{RuleName: "EmptyServiceSelector"},
	}
}

func (r ServiceSelectorRule) HasMatch(service *corev1.Service, pods []*corev1.Pod) bool {
	if len(service.Spec.Selector) == 0 {
		return true
	}
	selector := labels.SelectorFromSet(service.Spec.Selector)
	for _, pod := range pods {
		if pod.Namespace == service.Namespace && selector.Matches(labels.Set(pod.Labels)) {
			return true
		}
	}
	return false
}

func (r ServiceSelectorRule) Validate(service *corev1.Service, pods []*corev1.Pod) []Violation {
	if r.HasMatch(service, pods) {
		return nil
	}
	return r.Unsatisfied(service, "selector matches no Pods")
}
