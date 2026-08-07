package dependency

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

const PolicyAnnotation = "dependency.kubedepguard.io/mode"

type Mode string

const (
	ModeEnforce  Mode = "enforce"
	ModeWarn     Mode = "warn"
	ModeDisabled Mode = "disabled"
)

type Violation struct {
	Rule     string
	Resource string
	Message  string
}

// ModeFor uses warn as the safe migration default for resources without an annotation.
func ModeFor(meta metav1.Object) Mode {
	switch Mode(meta.GetAnnotations()[PolicyAnnotation]) {
	case ModeEnforce, ModeDisabled:
		return Mode(meta.GetAnnotations()[PolicyAnnotation])
	default:
		return ModeWarn
	}
}

func ConfigMapReferences(pod *corev1.Pod) []string {
	seen := make(map[string]struct{})
	add := func(name string) {
		if name != "" {
			seen[name] = struct{}{}
		}
	}
	for _, volume := range pod.Spec.Volumes {
		if volume.ConfigMap != nil {
			add(volume.ConfigMap.Name)
		}
	}
	containers := append([]corev1.Container{}, pod.Spec.InitContainers...)
	containers = append(containers, pod.Spec.Containers...)
	for _, container := range containers {
		for _, from := range container.EnvFrom {
			if from.ConfigMapRef != nil {
				add(from.ConfigMapRef.Name)
			}
		}
		for _, env := range container.Env {
			if env.ValueFrom != nil && env.ValueFrom.ConfigMapKeyRef != nil {
				add(env.ValueFrom.ConfigMapKeyRef.Name)
			}
		}
	}
	refs := make([]string, 0, len(seen))
	for name := range seen {
		refs = append(refs, name)
	}
	return refs
}

func MissingConfigMaps(pod *corev1.Pod, exists func(string) bool) []Violation {
	var violations []Violation
	for _, name := range ConfigMapReferences(pod) {
		if !exists(name) {
			violations = append(violations, Violation{"MissingConfigMap", pod.Namespace + "/" + pod.Name, fmt.Sprintf("references ConfigMap %q which does not exist", name)})
		}
	}
	return violations
}

func ServiceHasMatchingPod(service *corev1.Service, pods []*corev1.Pod) bool {
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

func EmptyServiceSelector(service *corev1.Service, pods []*corev1.Pod) []Violation {
	if ServiceHasMatchingPod(service, pods) {
		return nil
	}
	return []Violation{{"EmptyServiceSelector", service.Namespace + "/" + service.Name, "selector matches no Pods"}}
}
