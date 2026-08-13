// Package pod contains direct-reference extractors for Pod fields.
package pod

import (
	"fmt"

	ref "github.com/vincent/KubeDepGuard/internal/dependency/reference"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// ConfigMapExtractor extracts all Pod fields that reference ConfigMaps.
type ConfigMapExtractor struct{}

func NewConfigMapExtractor() ref.Extractor    { return ConfigMapExtractor{} }
func (ConfigMapExtractor) SourceKind() string { return "Pod" }
func (ConfigMapExtractor) Extract(object runtime.Object) ([]ref.Reference, error) {
	pod, ok := object.(*corev1.Pod)
	if !ok {
		return nil, fmt.Errorf("Pod ConfigMap extractor received %T", object)
	}
	var refs []ref.Reference
	add := func(name, path, key string, optional bool) {
		if name != "" {
			refs = append(refs, ref.Reference{SourceKind: "Pod", TargetKind: "ConfigMap", Namespace: pod.Namespace, Name: name, FieldPath: path, Key: key, Optional: optional})
		}
	}
	for _, volume := range pod.Spec.Volumes {
		if volume.ConfigMap != nil {
			add(volume.ConfigMap.Name, "spec.volumes["+volume.Name+"].configMap.name", "", volume.ConfigMap.Optional != nil && *volume.ConfigMap.Optional)
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
	extractContainers(pod.Spec.InitContainers, "spec.initContainers")
	extractContainers(pod.Spec.Containers, "spec.containers")
	return refs, nil
}
