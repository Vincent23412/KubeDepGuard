package dependency_test

import (
	"testing"

	"github.com/vincent/KubeDepGuard/internal/dependency"
	ref "github.com/vincent/KubeDepGuard/internal/dependency/reference"
	referencecatalog "github.com/vincent/KubeDepGuard/internal/dependency/reference/catalog"
	"github.com/vincent/KubeDepGuard/internal/dependency/resolver"
	"github.com/vincent/KubeDepGuard/internal/dependency/rules"
	rulecatalog "github.com/vincent/KubeDepGuard/internal/dependency/rules/catalog"
	"github.com/vincent/KubeDepGuard/internal/dependency/rules/conditional"
	"github.com/vincent/KubeDepGuard/internal/dependency/rules/direct"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestConfigMapReferences(t *testing.T) {
	pod := &corev1.Pod{Spec: corev1.PodSpec{Volumes: []corev1.Volume{{VolumeSource: corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{LocalObjectReference: corev1.LocalObjectReference{Name: "volume"}}}}}, Containers: []corev1.Container{{EnvFrom: []corev1.EnvFromSource{{ConfigMapRef: &corev1.ConfigMapEnvSource{LocalObjectReference: corev1.LocalObjectReference{Name: "all-env"}}}}, Env: []corev1.EnvVar{{ValueFrom: &corev1.EnvVarSource{ConfigMapKeyRef: &corev1.ConfigMapKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: "one-env"}}}}}}}}}
	references, err := referencecatalog.DefaultDirectRegistry.Extract("Pod", pod)
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
	selector, err := referencecatalog.ExtractServiceSelector(svc)
	if err != nil {
		t.Fatalf("extract selector: %v", err)
	}
	violations := conditional.NewServicePodRule().Validate(svc, selector, []resolver.Resource{pod})
	if len(violations) != 0 {
		t.Fatal("expected match")
	}
}

type testQuery struct{}

func (testQuery) List(resolver.ResourceScope) ([]resolver.Resource, error) { return nil, nil }

var _ resolver.Query = testQuery{}

func TestDefaultRegistryCategories(t *testing.T) {
	if got := len(rulecatalog.DefaultRegistry.DirectReferenceRules()); got != 1 {
		t.Fatalf("direct rule count = %d, want 1", got)
	}
	if got := len(rulecatalog.DefaultRegistry.ServiceConditionalRules()); got != 1 {
		t.Fatalf("conditional rule count = %d, want 1", got)
	}
}

func TestEvaluatorRejectsEnforcePodWithMissingConfigMap(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "api",
			Namespace:   "test",
			Annotations: map[string]string{dependency.PolicyAnnotation: string(dependency.ModeEnforce)},
		},
		Spec: corev1.PodSpec{Containers: []corev1.Container{{
			Name: "app",
			EnvFrom: []corev1.EnvFromSource{{
				ConfigMapRef: &corev1.ConfigMapEnvSource{LocalObjectReference: corev1.LocalObjectReference{Name: "missing"}},
			}},
		}}},
	}
	evaluator := rules.NewEvaluator(testQuery{}, rulecatalog.DefaultRegistry.AdmissionRules())
	result, err := evaluator.Evaluate(rules.Request{Resource: "pods", Operation: rules.Create, Namespace: pod.Namespace, Name: pod.Name, Object: pod})
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if !result.Reject || len(result.Violations) != 1 {
		t.Fatalf("result = %#v, want one rejected violation", result)
	}
}

func TestPodConfigMapRuleRejectsReferencedTargetDeletion(t *testing.T) {
	pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "test"}}
	refs := []ref.Reference{{SourceKind: "Pod", TargetKind: "ConfigMap", Namespace: "test", Name: "settings"}}
	violations := direct.NewPodConfigMapRule().ValidateTargetDeletion(pod, refs, resolver.ResourceScope{Kind: "ConfigMap", Namespace: "test"}, "settings")
	if got := len(violations); got != 1 {
		t.Fatalf("violations = %d, want 1", got)
	}
	if violations[0].Rule != "ReferencedConfigMapDeletion" {
		t.Fatalf("rule = %q", violations[0].Rule)
	}
}
