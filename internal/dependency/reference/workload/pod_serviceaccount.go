package workload

import (
	"fmt"
	ref "github.com/vincent/KubeDepGuard/internal/dependency/reference"
	"github.com/vincent/KubeDepGuard/internal/dependency/reference/podspec"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type PodServiceAccountExtractor struct{}

func NewPodServiceAccountExtractor() ref.Extractor    { return PodServiceAccountExtractor{} }
func (PodServiceAccountExtractor) SourceKind() string { return "Pod" }
func (PodServiceAccountExtractor) Extract(object runtime.Object) ([]ref.Reference, error) {
	pod, ok := object.(*corev1.Pod)
	if !ok {
		return nil, fmt.Errorf("Pod ServiceAccount extractor received %T", object)
	}
	return podspec.ServiceAccountReferences("Pod", pod.Namespace, "spec", &pod.Spec), nil
}
