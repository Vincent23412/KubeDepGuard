package dependency_test

import (
	"testing"

	"github.com/vincent/KubeDepGuard/internal/dependency/catalog"
	ref "github.com/vincent/KubeDepGuard/internal/dependency/reference"
	serviceRef "github.com/vincent/KubeDepGuard/internal/dependency/reference/service"
	"github.com/vincent/KubeDepGuard/internal/dependency/resolver"
	"github.com/vincent/KubeDepGuard/internal/dependency/rules"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestConfigMapReferences(t *testing.T) {
	pod := &corev1.Pod{Spec: corev1.PodSpec{Volumes: []corev1.Volume{{VolumeSource: corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{LocalObjectReference: corev1.LocalObjectReference{Name: "volume"}}}}}, Containers: []corev1.Container{{EnvFrom: []corev1.EnvFromSource{{ConfigMapRef: &corev1.ConfigMapEnvSource{LocalObjectReference: corev1.LocalObjectReference{Name: "all-env"}}}}, Env: []corev1.EnvVar{{ValueFrom: &corev1.EnvVarSource{ConfigMapKeyRef: &corev1.ConfigMapKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: "one-env"}}}}}}}}}
	references, err := catalog.DefaultReferenceRegistry.Extract("Pod", pod)
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
	selector, err := serviceRef.NewSelectorExtractor().Extract(svc)
	if err != nil {
		t.Fatalf("extract selector: %v", err)
	}
	violations, err := rules.NewServicePodRule().Validate(svc, selector, testPodLister{pods: []*corev1.Pod{pod}})
	if err != nil || len(violations) != 0 {
		t.Fatal("expected match")
	}
}

type testPodLister struct{ pods []*corev1.Pod }

func (l testPodLister) List(string) ([]*corev1.Pod, error) { return l.pods, nil }

var _ resolver.PodLister = testPodLister{}

func TestDefaultRegistryCategories(t *testing.T) {
	if got := len(rules.DefaultRegistry.DirectReferenceRules()); got != 1 {
		t.Fatalf("direct rule count = %d, want 1", got)
	}
	if got := len(rules.DefaultRegistry.ServiceConditionalRules()); got != 1 {
		t.Fatalf("conditional rule count = %d, want 1", got)
	}
}

func TestPodConfigMapRuleRejectsReferencedTargetDeletion(t *testing.T) {
	pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "test"}}
	refs := []ref.Reference{{SourceKind: "Pod", TargetKind: "ConfigMap", Namespace: "test", Name: "settings"}}
	violations := rules.NewPodConfigMapRule().ValidateTargetDeletion(pod, refs, resolver.Target{Kind: "ConfigMap", Namespace: "test", Name: "settings"})
	if got := len(violations); got != 1 {
		t.Fatalf("violations = %d, want 1", got)
	}
	if violations[0].Rule != "ReferencedConfigMapDeletion" {
		t.Fatalf("rule = %q", violations[0].Rule)
	}
}
