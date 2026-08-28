// Package podspec contains reference extractors that operate on a PodSpec.
package podspec

import (
	"fmt"

	ref "github.com/vincent/KubeDepGuard/internal/dependency/reference"
	corev1 "k8s.io/api/core/v1"
)

// ConfigMapReferences extracts ConfigMap references from any PodSpec owner.
// pathPrefix identifies where the PodSpec lives (for example "spec" or
// "spec.template.spec") and keeps diagnostics tied to the source object.
func ConfigMapReferences(sourceKind, namespace, pathPrefix string, spec *corev1.PodSpec) []ref.Reference {
	var refs []ref.Reference
	add := func(name, path, key string, optional bool) {
		if name != "" {
			refs = append(refs, ref.Reference{SourceKind: sourceKind, TargetKind: "ConfigMap", Namespace: namespace, Name: name, FieldPath: path, Key: key, Optional: optional})
		}
	}
	for vi, volume := range spec.Volumes {
		if volume.ConfigMap != nil {
			add(volume.ConfigMap.Name, fmt.Sprintf("%s.volumes[%d].configMap.name", pathPrefix, vi), "", volume.ConfigMap.Optional != nil && *volume.ConfigMap.Optional)
		}
	}
	extractContainers := func(containers []corev1.Container, base string) {
		for ci, container := range containers {
			for ei, from := range container.EnvFrom {
				if from.ConfigMapRef != nil {
					add(from.ConfigMapRef.Name, fmt.Sprintf("%s[%d].envFrom[%d].configMapRef.name", base, ci, ei), "", from.ConfigMapRef.Optional != nil && *from.ConfigMapRef.Optional)
				}
			}
			for ei, env := range container.Env {
				if env.ValueFrom != nil && env.ValueFrom.ConfigMapKeyRef != nil {
					add(env.ValueFrom.ConfigMapKeyRef.Name, fmt.Sprintf("%s[%d].env[%d].valueFrom.configMapKeyRef.name", base, ci, ei), env.ValueFrom.ConfigMapKeyRef.Key, env.ValueFrom.ConfigMapKeyRef.Optional != nil && *env.ValueFrom.ConfigMapKeyRef.Optional)
				}
			}
		}
	}
	extractContainers(spec.InitContainers, pathPrefix+".initContainers")
	extractContainers(spec.Containers, pathPrefix+".containers")
	return refs
}
