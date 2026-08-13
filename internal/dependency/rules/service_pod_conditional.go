package rules

import (
	"fmt"

	"github.com/vincent/KubeDepGuard/internal/dependency"
	serviceRef "github.com/vincent/KubeDepGuard/internal/dependency/reference/service"
	"github.com/vincent/KubeDepGuard/internal/dependency/resolver"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
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

func (r ServicePodRule) Applies(request Request) bool {
	return (request.Resource == "services" && (request.Operation == Create || request.Operation == Update)) ||
		(request.Resource == "pods" && request.Operation == Delete)
}

func (r ServicePodRule) Evaluate(request Request, query resolver.Query) (Result, error) {
	switch {
	case request.Resource == "services":
		svc, ok := request.Object.(*corev1.Service)
		if !ok {
			return Result{}, fmt.Errorf("expected Service object")
		}
		if dependency.ModeFor(svc) == dependency.ModeDisabled {
			return Result{}, nil
		}
		selector, err := serviceRef.NewSelectorExtractor().Extract(svc)
		if err != nil {
			return Result{}, err
		}
		violations, err := r.Validate(svc, selector, queryPodLister{query})
		if err != nil {
			return Result{}, err
		}
		return Result{Violations: violations, Reject: len(violations) > 0 && dependency.ModeFor(svc) == dependency.ModeEnforce}, nil
	case request.Resource == "pods":
		pod, err := query.GetPod(request.Namespace, request.Name)
		if apierrors.IsNotFound(err) {
			return Result{}, nil
		}
		if err != nil {
			return Result{}, err
		}
		services, err := query.ListServices(pod.Namespace)
		if err != nil {
			return Result{}, err
		}
		pods, err := query.ListPods(pod.Namespace)
		if err != nil {
			return Result{}, err
		}
		remaining := make([]*corev1.Pod, 0, len(pods))
		for _, candidate := range pods {
			if candidate.Name != pod.Name {
				remaining = append(remaining, candidate)
			}
		}
		var violations []dependency.Violation
		for _, svc := range services {
			if dependency.ModeFor(svc) != dependency.ModeEnforce {
				continue
			}
			selector, err := serviceRef.NewSelectorExtractor().Extract(svc)
			if err != nil {
				return Result{}, err
			}
			found, err := r.Validate(svc, selector, podSliceLister{pods: remaining})
			if err != nil {
				return Result{}, err
			}
			violations = append(violations, found...)
		}
		return Result{Violations: violations, Reject: len(violations) > 0}, nil
	default:
		return Result{}, nil
	}
}

type queryPodLister struct{ query resolver.Query }

func (l queryPodLister) List(namespace string) ([]*corev1.Pod, error) {
	return l.query.ListPods(namespace)
}

type podSliceLister struct{ pods []*corev1.Pod }

func (l podSliceLister) List(string) ([]*corev1.Pod, error) { return l.pods, nil }
