package workload

import (
	"fmt"

	ref "github.com/vincent/KubeDepGuard/internal/dependency/reference"
	"github.com/vincent/KubeDepGuard/internal/dependency/reference/podspec"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// ConfigMapExtractor extracts ConfigMap references from a Pod workload source.
type ConfigMapExtractor struct{}

func NewPodConfigMapExtractor() ref.Extractor { return ConfigMapExtractor{} }
func (ConfigMapExtractor) SourceKind() string { return "Pod" }
func (ConfigMapExtractor) Extract(object runtime.Object) ([]ref.Reference, error) {
	pod, ok := object.(*corev1.Pod)
	if !ok {
		return nil, fmt.Errorf("Pod ConfigMap extractor received %T", object)
	}
	return podspec.ConfigMapReferences("Pod", pod.Namespace, "spec", &pod.Spec), nil
}
