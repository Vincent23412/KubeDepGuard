package reference

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// PodConfigMapExtractor owns Pod fields that reference ConfigMaps.
type PodConfigMapExtractor struct{}

func NewPodConfigMapExtractor() Extractor        { return PodConfigMapExtractor{} }
func (PodConfigMapExtractor) SourceKind() string { return "Pod" }
func (PodConfigMapExtractor) Extract(object runtime.Object) ([]Reference, error) {
	pod, ok := object.(*corev1.Pod)
	if !ok {
		return nil, fmt.Errorf("Pod extractor received %T", object)
	}
	var refs []Reference
	addConfigMap := func(name, path, key string, optional bool) {
		if name != "" {
			refs = append(refs, Reference{SourceKind: "Pod", TargetKind: "ConfigMap", Namespace: pod.Namespace, Name: name, FieldPath: path, Key: key, Optional: optional})
		}
	}
	for _, volume := range pod.Spec.Volumes {
		if volume.ConfigMap != nil {
			addConfigMap(volume.ConfigMap.Name, "spec.volumes["+volume.Name+"].configMap.name", "", volume.ConfigMap.Optional != nil && *volume.ConfigMap.Optional)
		}
	}
	extractContainers := func(containers []corev1.Container, base string) {
		for ci, container := range containers {
			for ei, from := range container.EnvFrom {
				if from.ConfigMapRef != nil {
					addConfigMap(from.ConfigMapRef.Name, fmt.Sprintf("%s[%d].envFrom[%d].configMapRef.name", base, ci, ei), "", from.ConfigMapRef.Optional != nil && *from.ConfigMapRef.Optional)
				}
			}
			for ei, env := range container.Env {
				if env.ValueFrom != nil && env.ValueFrom.ConfigMapKeyRef != nil {
					addConfigMap(env.ValueFrom.ConfigMapKeyRef.Name, fmt.Sprintf("%s[%d].env[%d].valueFrom.configMapKeyRef.name", base, ci, ei), env.ValueFrom.ConfigMapKeyRef.Key, env.ValueFrom.ConfigMapKeyRef.Optional != nil && *env.ValueFrom.ConfigMapKeyRef.Optional)
				}
			}
		}
	}
	extractContainers(pod.Spec.InitContainers, "spec.initContainers")
	extractContainers(pod.Spec.Containers, "spec.containers")
	return refs, nil
}
