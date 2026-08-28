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
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestConfigMapReferences(t *testing.T) {
	pod := &corev1.Pod{Spec: corev1.PodSpec{Volumes: []corev1.Volume{{VolumeSource: corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{LocalObjectReference: corev1.LocalObjectReference{Name: "volume"}}}}}, Containers: []corev1.Container{{EnvFrom: []corev1.EnvFromSource{{ConfigMapRef: &corev1.ConfigMapEnvSource{LocalObjectReference: corev1.LocalObjectReference{Name: "all-env"}}}}, Env: []corev1.EnvVar{{ValueFrom: &corev1.EnvVarSource{ConfigMapKeyRef: &corev1.ConfigMapKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: "one-env"}}}}}}}}}
	references, err := referencecatalog.DefaultDirectRegistry.Extract("Pod", pod)
	if err != nil {
		t.Fatalf("extract references: %v", err)
	}
	configMapReferences := 0
	for _, reference := range references {
		if reference.TargetKind == "ConfigMap" {
			configMapReferences++
		}
	}
	if got := configMapReferences; got != 3 {
		t.Fatalf("references = %d, want 3", got)
	}
	if references[0].TargetKind != "ConfigMap" || references[0].FieldPath == "" {
		t.Fatalf("reference missing target metadata: %#v", references[0])
	}
}

func TestSecretReferences(t *testing.T) {
	pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Namespace: "test"}, Spec: corev1.PodSpec{
		ImagePullSecrets: []corev1.LocalObjectReference{{Name: "pull"}},
		Volumes: []corev1.Volume{
			{Name: "secret", VolumeSource: corev1.VolumeSource{Secret: &corev1.SecretVolumeSource{SecretName: "volume"}}},
			{Name: "projected", VolumeSource: corev1.VolumeSource{Projected: &corev1.ProjectedVolumeSource{Sources: []corev1.VolumeProjection{{Secret: &corev1.SecretProjection{LocalObjectReference: corev1.LocalObjectReference{Name: "projected"}}}}}}},
		},
		Containers: []corev1.Container{{
			EnvFrom: []corev1.EnvFromSource{{SecretRef: &corev1.SecretEnvSource{LocalObjectReference: corev1.LocalObjectReference{Name: "all-env"}}}},
			Env:     []corev1.EnvVar{{ValueFrom: &corev1.EnvVarSource{SecretKeyRef: &corev1.SecretKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: "one-env"}, Key: "token"}}}},
		}},
	}}
	references, err := referencecatalog.DefaultDirectRegistry.Extract("Pod", pod)
	if err != nil {
		t.Fatalf("extract references: %v", err)
	}
	var secrets []ref.Reference
	for _, reference := range references {
		if reference.TargetKind == "Secret" {
			secrets = append(secrets, reference)
		}
	}
	if got := len(secrets); got != 5 {
		t.Fatalf("Secret references = %d, want 5", got)
	}
	if secrets[0].FieldPath == "" || secrets[0].Namespace != "test" {
		t.Fatalf("reference missing source metadata: %#v", secrets[0])
	}
}

func TestPersistentVolumeClaimReferences(t *testing.T) {
	pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Namespace: "test"}, Spec: corev1.PodSpec{Volumes: []corev1.Volume{{
		Name: "data", VolumeSource: corev1.VolumeSource{PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: "data-pvc"}},
	}}}}
	references, err := referencecatalog.DefaultDirectRegistry.Extract("Pod", pod)
	if err != nil {
		t.Fatalf("extract references: %v", err)
	}
	var found *ref.Reference
	for i := range references {
		if references[i].TargetKind == "PersistentVolumeClaim" {
			found = &references[i]
			break
		}
	}
	if found == nil || found.Name != "data-pvc" || found.FieldPath != "spec.volumes[0].persistentVolumeClaim.claimName" {
		t.Fatalf("PVC reference = %#v", found)
	}
}

func TestDeploymentConfigMapReferences(t *testing.T) {
	deployment := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Namespace: "test"}, Spec: appsv1.DeploymentSpec{Template: corev1.PodTemplateSpec{
		Spec: corev1.PodSpec{Containers: []corev1.Container{{EnvFrom: []corev1.EnvFromSource{{ConfigMapRef: &corev1.ConfigMapEnvSource{LocalObjectReference: corev1.LocalObjectReference{Name: "app-config"}}}}}}},
	}}}
	references, err := referencecatalog.DefaultDirectRegistry.Extract("Deployment", deployment)
	if err != nil {
		t.Fatalf("extract references: %v", err)
	}
	if len(references) < 1 || references[0].SourceKind != "Deployment" || references[0].Name != "app-config" {
		t.Fatalf("references = %#v, want one Deployment ConfigMap reference", references)
	}
	if references[0].FieldPath != "spec.template.spec.containers[0].envFrom[0].configMapRef.name" {
		t.Fatalf("field path = %q", references[0].FieldPath)
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

func (testQuery) List(scope resolver.ResourceScope) ([]resolver.Resource, error) {
	if scope.Kind == "ServiceAccount" {
		return []resolver.Resource{&corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: "default", Namespace: scope.Namespace}}}, nil
	}
	return nil, nil
}

var _ resolver.Query = testQuery{}

func TestDefaultRegistryCategories(t *testing.T) {
	if got := len(rulecatalog.DefaultRegistry.DirectReferenceRules()); got != 3 {
		t.Fatalf("direct rule count = %d, want 3", got)
	}
	if got := len(rulecatalog.DefaultRegistry.ServiceConditionalRules()); got != 1 {
		t.Fatalf("conditional rule count = %d, want 1", got)
	}
}

func TestEvaluatorRejectsEnforcePodWithMissingSecret(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "api",
			Namespace:   "test",
			Annotations: map[string]string{dependency.PolicyAnnotation: string(dependency.ModeEnforce)},
		},
		Spec: corev1.PodSpec{Containers: []corev1.Container{{
			Name: "app",
			EnvFrom: []corev1.EnvFromSource{{
				SecretRef: &corev1.SecretEnvSource{LocalObjectReference: corev1.LocalObjectReference{Name: "missing"}},
			}},
		}}},
	}
	evaluator := rules.NewEvaluator(testQuery{}, rulecatalog.DefaultRegistry.AdmissionRules())
	result, err := evaluator.Evaluate(rules.Request{Resource: "pods", Operation: rules.Create, Namespace: pod.Namespace, Name: pod.Name, Object: pod})
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if !result.Reject || len(result.Violations) != 1 || result.Violations[0].Rule != "MissingSecret" {
		t.Fatalf("result = %#v, want one MissingSecret rejection", result)
	}
}

type pvcQuery struct{ pvc *corev1.PersistentVolumeClaim }

func (q pvcQuery) List(scope resolver.ResourceScope) ([]resolver.Resource, error) {
	if scope.Kind == "ServiceAccount" {
		return []resolver.Resource{&corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: "default", Namespace: scope.Namespace}}}, nil
	}
	if scope.Kind == "PersistentVolumeClaim" && q.pvc != nil {
		return []resolver.Resource{q.pvc}, nil
	}
	if scope.Kind == "PersistentVolume" && q.pvc != nil && q.pvc.Spec.VolumeName != "" {
		return []resolver.Resource{&corev1.PersistentVolume{ObjectMeta: metav1.ObjectMeta{Name: q.pvc.Spec.VolumeName}, Status: corev1.PersistentVolumeStatus{Phase: corev1.VolumeBound}}}, nil
	}
	return nil, nil
}

func TestEvaluatorRejectsEnforcePodWithUnboundPVC(t *testing.T) {
	pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "test", Annotations: map[string]string{dependency.PolicyAnnotation: string(dependency.ModeEnforce)}}, Spec: corev1.PodSpec{Volumes: []corev1.Volume{{
		Name: "data", VolumeSource: corev1.VolumeSource{PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: "data-pvc"}},
	}}, Containers: []corev1.Container{{Name: "app"}}}}
	pvc := &corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: "data-pvc", Namespace: "test"}, Status: corev1.PersistentVolumeClaimStatus{Phase: corev1.ClaimPending}}
	result, err := rules.NewEvaluator(pvcQuery{pvc: pvc}, rulecatalog.DefaultRegistry.AdmissionRules()).Evaluate(rules.Request{Resource: "pods", Operation: rules.Create, Namespace: "test", Name: "api", Object: pod})
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if !result.Reject || len(result.Violations) != 1 || result.Violations[0].Rule != "PersistentVolumeClaimNotBound" {
		t.Fatalf("result = %#v, want one PVC state rejection", result)
	}
}

func TestEvaluatorAllowsEnforcePodWithBoundPVC(t *testing.T) {
	pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "test", Annotations: map[string]string{dependency.PolicyAnnotation: string(dependency.ModeEnforce)}}, Spec: corev1.PodSpec{Volumes: []corev1.Volume{{
		Name: "data", VolumeSource: corev1.VolumeSource{PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: "data-pvc"}},
	}}, Containers: []corev1.Container{{Name: "app"}}}}
	pvc := &corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: "data-pvc", Namespace: "test"}, Spec: corev1.PersistentVolumeClaimSpec{VolumeName: "data-pv"}, Status: corev1.PersistentVolumeClaimStatus{Phase: corev1.ClaimBound}}
	result, err := rules.NewEvaluator(pvcQuery{pvc: pvc}, rulecatalog.DefaultRegistry.AdmissionRules()).Evaluate(rules.Request{Resource: "pods", Operation: rules.Create, Namespace: "test", Name: "api", Object: pod})
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if len(result.Violations) != 0 || result.Reject {
		t.Fatalf("result = %#v, want no PVC violation", result)
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

func TestEvaluatorRejectsEnforceDeploymentWithMissingConfigMap(t *testing.T) {
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "frontend", Namespace: "test", Annotations: map[string]string{dependency.PolicyAnnotation: string(dependency.ModeEnforce)}},
		Spec: appsv1.DeploymentSpec{Template: corev1.PodTemplateSpec{Spec: corev1.PodSpec{Containers: []corev1.Container{{
			Name: "app", EnvFrom: []corev1.EnvFromSource{{ConfigMapRef: &corev1.ConfigMapEnvSource{LocalObjectReference: corev1.LocalObjectReference{Name: "missing"}}}},
		}}}}},
	}
	result, err := rules.NewEvaluator(testQuery{}, rulecatalog.DefaultRegistry.AdmissionRules()).Evaluate(rules.Request{Resource: "deployments", Operation: rules.Create, Namespace: "test", Name: "frontend", Object: deployment})
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if !result.Reject || len(result.Violations) != 1 || result.Violations[0].Rule != "MissingConfigMap" {
		t.Fatalf("result = %#v, want one MissingConfigMap rejection", result)
	}
}

func TestWorkloadConfigMapRuleRejectsReferencedTargetDeletion(t *testing.T) {
	pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "test"}}
	refs := []ref.Reference{{SourceKind: "Pod", TargetKind: "ConfigMap", Namespace: "test", Name: "settings"}}
	violations := direct.NewWorkloadConfigMapRule().ValidateTargetDeletion(pod, refs, resolver.ResourceScope{Kind: "ConfigMap", Namespace: "test"}, "settings")
	if got := len(violations); got != 1 {
		t.Fatalf("violations = %d, want 1", got)
	}
	if violations[0].Rule != "ReferencedConfigMapDeletion" {
		t.Fatalf("rule = %q", violations[0].Rule)
	}
}

func TestWorkloadSecretRuleRejectsReferencedTargetDeletion(t *testing.T) {
	pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "test"}}
	refs := []ref.Reference{{SourceKind: "Pod", TargetKind: "Secret", Namespace: "test", Name: "credentials"}}
	violations := direct.NewWorkloadSecretRule().ValidateTargetDeletion(pod, refs, resolver.ResourceScope{Kind: "Secret", Namespace: "test"}, "credentials")
	if got := len(violations); got != 1 {
		t.Fatalf("violations = %d, want 1", got)
	}
	if violations[0].Rule != "ReferencedSecretDeletion" {
		t.Fatalf("rule = %q", violations[0].Rule)
	}
}
