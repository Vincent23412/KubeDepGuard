package workload

import (
	"fmt"

	ref "github.com/vincent/KubeDepGuard/internal/dependency/reference"
	"github.com/vincent/KubeDepGuard/internal/dependency/reference/podspec"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// SecretExtractor extracts Secret references from a Pod workload source.
type SecretExtractor struct{}

func NewPodSecretExtractor() ref.Extractor { return SecretExtractor{} }
func (SecretExtractor) SourceKind() string { return "Pod" }
func (SecretExtractor) Extract(object runtime.Object) ([]ref.Reference, error) {
	pod, ok := object.(*corev1.Pod)
	if !ok {
		return nil, fmt.Errorf("Pod Secret extractor received %T", object)
	}
	return podspec.SecretReferences("Pod", pod.Namespace, "spec", &pod.Spec), nil
}
