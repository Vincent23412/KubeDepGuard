package workload

import (
	"fmt"
	ref "github.com/vincent/KubeDepGuard/internal/dependency/reference"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type PodNodeExtractor struct{}

func NewPodNodeExtractor() ref.Extractor    { return PodNodeExtractor{} }
func (PodNodeExtractor) SourceKind() string { return "Pod" }
func (PodNodeExtractor) Extract(object runtime.Object) ([]ref.Reference, error) {
	pod, ok := object.(*corev1.Pod)
	if !ok {
		return nil, fmt.Errorf("Pod Node extractor received %T", object)
	}
	if pod.Spec.NodeName == "" {
		return nil, nil
	}
	return []ref.Reference{{SourceKind: "Pod", TargetKind: "Node", Name: pod.Spec.NodeName, FieldPath: "spec.nodeName"}}, nil
}
