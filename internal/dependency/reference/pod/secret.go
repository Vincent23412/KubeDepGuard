package pod

import (
	"fmt"

	ref "github.com/vincent/KubeDepGuard/internal/dependency/reference"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// SecretExtractor extracts all Pod fields that directly reference Secrets.
type SecretExtractor struct{}

func NewSecretExtractor() ref.Extractor    { return SecretExtractor{} }
func (SecretExtractor) SourceKind() string { return "Pod" }
func (SecretExtractor) Extract(object runtime.Object) ([]ref.Reference, error) {
	pod, ok := object.(*corev1.Pod)
	if !ok {
		return nil, fmt.Errorf("Pod Secret extractor received %T", object)
	}
	var refs []ref.Reference
	add := func(name, path, key string, optional bool) {
		if name != "" {
			refs = append(refs, ref.Reference{SourceKind: "Pod", TargetKind: "Secret", Namespace: pod.Namespace, Name: name, FieldPath: path, Key: key, Optional: optional})
		}
	}
	for vi, volume := range pod.Spec.Volumes {
		if volume.Secret != nil {
			add(volume.Secret.SecretName, fmt.Sprintf("spec.volumes[%d].secret.secretName", vi), "", volume.Secret.Optional != nil && *volume.Secret.Optional)
		}
		if volume.Projected != nil {
			for si, source := range volume.Projected.Sources {
				if source.Secret != nil {
					add(source.Secret.Name, fmt.Sprintf("spec.volumes[%d].projected.sources[%d].secret.name", vi, si), "", source.Secret.Optional != nil && *source.Secret.Optional)
				}
			}
		}
	}
	for i, pullSecret := range pod.Spec.ImagePullSecrets {
		add(pullSecret.Name, fmt.Sprintf("spec.imagePullSecrets[%d].name", i), "", false)
	}
	extractContainers := func(containers []corev1.Container, base string) {
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
	extractContainers(pod.Spec.InitContainers, "spec.initContainers")
	extractContainers(pod.Spec.Containers, "spec.containers")
	return refs, nil
}
