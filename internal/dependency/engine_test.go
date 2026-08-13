package dependency_test

import (
	"testing"

	ref "github.com/vincent/KubeDepGuard/internal/dependency/reference"
	"github.com/vincent/KubeDepGuard/internal/dependency/rules"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestConfigMapReferences(t *testing.T) {
	pod := &corev1.Pod{Spec: corev1.PodSpec{Volumes: []corev1.Volume{{VolumeSource: corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{LocalObjectReference: corev1.LocalObjectReference{Name: "volume"}}}}}, Containers: []corev1.Container{{EnvFrom: []corev1.EnvFromSource{{ConfigMapRef: &corev1.ConfigMapEnvSource{LocalObjectReference: corev1.LocalObjectReference{Name: "all-env"}}}}, Env: []corev1.EnvVar{{ValueFrom: &corev1.EnvVarSource{ConfigMapKeyRef: &corev1.ConfigMapKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: "one-env"}}}}}}}}}
	references, err := ref.DefaultRegistry.Extract("Pod", pod)
	if err != nil {
		t.Fatalf("extract references: %v", err)
	}
	if got := len(references); got != 3 {
		t.Fatalf("references = %d, want 3", got)
	}
	if references[0].TargetKind != "ConfigMap" || references[0].FieldPath == "" {
		t.Fatalf("reference missing target metadata: %#v", references[0])
	}
}

func TestServiceHasMatchingPod(t *testing.T) {
	svc := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Namespace: "test"}, Spec: corev1.ServiceSpec{Selector: map[string]string{"app": "api"}}}
	pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Namespace: "test", Labels: map[string]string{"app": "api"}}}
	rule := NewServicePodSelectorRule().(ServiceSelectorRule)
	if !rule.HasMatch(svc, []*corev1.Pod{pod}) {
		t.Fatal("expected match")
	}
}

func TestDefaultRegistryCategories(t *testing.T) {
	if got := len(rules.DefaultRegistry.DirectReferenceRules()); got != 1 {
		t.Fatalf("direct rule count = %d, want 1", got)
	}
	if got := len(rules.DefaultRegistry.ServiceConditionalRules()); got != 1 {
		t.Fatalf("conditional rule count = %d, want 1", got)
	}
}
