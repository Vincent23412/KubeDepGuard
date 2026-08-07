package dependency

import corev1 "k8s.io/api/core/v1"

// ConfigMapReferenceRule validates Pod -> ConfigMap direct references.
// It embeds DirectReferenceRule to reuse the common missing-target behaviour.
type ConfigMapReferenceRule struct {
	DirectReferenceRule
}

var _ PodDirectReferenceRule = ConfigMapReferenceRule{}

func NewPodConfigMapRule() Rule {
	return ConfigMapReferenceRule{
		DirectReferenceRule: DirectReferenceRule{RuleName: "MissingConfigMap", TargetResourceKind: "ConfigMap"},
	}
}

func (r ConfigMapReferenceRule) References(pod *corev1.Pod) []string {
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

func (r ConfigMapReferenceRule) Validate(pod *corev1.Pod, exists TargetResolver) []Violation {
	return r.Missing(pod, r.References(pod), exists)
}
