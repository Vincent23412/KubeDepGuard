package podspec

import (
	"fmt"
	ref "github.com/vincent/KubeDepGuard/internal/dependency/reference"
	corev1 "k8s.io/api/core/v1"
)

// SecretReferences extracts Secret references from any PodSpec owner.
func SecretReferences(sourceKind, namespace, pathPrefix string, spec *corev1.PodSpec) []ref.Reference {
	var refs []ref.Reference
	add := func(name, path, key string, optional bool) {
		if name != "" {
			refs = append(refs, ref.Reference{SourceKind: sourceKind, TargetKind: "Secret", Namespace: namespace, Name: name, FieldPath: path, Key: key, Optional: optional})
		}
	}
	for vi, volume := range spec.Volumes {
		if volume.Secret != nil {
			add(volume.Secret.SecretName, fmt.Sprintf("%s.volumes[%d].secret.secretName", pathPrefix, vi), "", volume.Secret.Optional != nil && *volume.Secret.Optional)
		}
		if volume.Projected != nil {
			for si, source := range volume.Projected.Sources {
				if source.Secret != nil {
					add(source.Secret.Name, fmt.Sprintf("%s.volumes[%d].projected.sources[%d].secret.name", pathPrefix, vi, si), "", source.Secret.Optional != nil && *source.Secret.Optional)
				}
			}
		}
	}
	for i, pullSecret := range spec.ImagePullSecrets {
		add(pullSecret.Name, fmt.Sprintf("%s.imagePullSecrets[%d].name", pathPrefix, i), "", false)
	}
	extract := func(containers []corev1.Container, base string) {
		for ci, container := range containers {
			for ei, from := range container.EnvFrom {
				if from.SecretRef != nil {
					add(from.SecretRef.Name, fmt.Sprintf("%s[%d].envFrom[%d].secretRef.name", base, ci, ei), "", from.SecretRef.Optional != nil && *from.SecretRef.Optional)
				}
			}
			for ei, env := range container.Env {
				if env.ValueFrom != nil && env.ValueFrom.SecretKeyRef != nil {
					add(env.ValueFrom.SecretKeyRef.Name, fmt.Sprintf("%s[%d].env[%d].valueFrom.secretKeyRef.name", base, ci, ei), env.ValueFrom.SecretKeyRef.Key, env.ValueFrom.SecretKeyRef.Optional != nil && *env.ValueFrom.SecretKeyRef.Optional)
				}
			}
		}
	}
	extract(spec.InitContainers, pathPrefix+".initContainers")
	extract(spec.Containers, pathPrefix+".containers")
	return refs
}
