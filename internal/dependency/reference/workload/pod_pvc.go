package workload

import (
	"fmt"

	ref "github.com/vincent/KubeDepGuard/internal/dependency/reference"
	"github.com/vincent/KubeDepGuard/internal/dependency/reference/podspec"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// PersistentVolumeClaimExtractor extracts PVC references from a Pod workload source.
type PersistentVolumeClaimExtractor struct{}

func NewPodPersistentVolumeClaimExtractor() ref.Extractor { return PersistentVolumeClaimExtractor{} }
func (PersistentVolumeClaimExtractor) SourceKind() string { return "Pod" }
func (PersistentVolumeClaimExtractor) Extract(object runtime.Object) ([]ref.Reference, error) {
	pod, ok := object.(*corev1.Pod)
	if !ok {
		return nil, fmt.Errorf("Pod PVC extractor received %T", object)
	}
	return podspec.PersistentVolumeClaimReferences("Pod", pod.Namespace, "spec", &pod.Spec), nil
}
