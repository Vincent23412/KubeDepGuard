package dependency_test

import (
	"testing"

	"github.com/vincent/KubeDepGuard/internal/dependency"
	"github.com/vincent/KubeDepGuard/internal/dependency/resolver"
	"github.com/vincent/KubeDepGuard/internal/dependency/rules"
	rulecatalog "github.com/vincent/KubeDepGuard/internal/dependency/rules/catalog"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const formalNamespace = "formal-cases"

type resourcesByScope map[resolver.ResourceScope][]resolver.Resource

func (r resourcesByScope) List(scope resolver.ResourceScope) ([]resolver.Resource, error) {
	return r[scope], nil
}

func evaluateFormal(t *testing.T, query resolver.Query, request rules.Request, wantReject bool) {
	t.Helper()
	result, err := rules.NewEvaluator(query, rulecatalog.DefaultRegistry.AdmissionRules()).Evaluate(request)
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if result.Reject != wantReject {
		t.Fatalf("reject = %t, want %t; violations = %#v", result.Reject, wantReject, result.Violations)
	}
}

func enforceMeta(name string) metav1.ObjectMeta {
	return metav1.ObjectMeta{Name: name, Namespace: formalNamespace, Annotations: map[string]string{dependency.PolicyAnnotation: string(dependency.ModeEnforce)}}
}

func deploymentWithConfigMap(name, configMap string) *appsv1.Deployment {
	return &appsv1.Deployment{ObjectMeta: enforceMeta(name), Spec: appsv1.DeploymentSpec{Template: corev1.PodTemplateSpec{Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "app", EnvFrom: []corev1.EnvFromSource{{ConfigMapRef: &corev1.ConfigMapEnvSource{LocalObjectReference: corev1.LocalObjectReference{Name: configMap}}}}}}}}}}
}

func deploymentWithPVC(name, claim string) *appsv1.Deployment {
	return &appsv1.Deployment{ObjectMeta: enforceMeta(name), Spec: appsv1.DeploymentSpec{Template: corev1.PodTemplateSpec{Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "app"}}, Volumes: []corev1.Volume{{Name: "data", VolumeSource: corev1.VolumeSource{PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: claim}}}}}}}}
}

func podWithSecret(name, secret string, labels map[string]string) *corev1.Pod {
	pod := &corev1.Pod{ObjectMeta: enforceMeta(name), Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "app", EnvFrom: []corev1.EnvFromSource{{SecretRef: &corev1.SecretEnvSource{LocalObjectReference: corev1.LocalObjectReference{Name: secret}}}}}}}}
	pod.Labels = labels
	return pod
}

func ingressWithService(name, service string, port int32) *networkingv1.Ingress {
	return &networkingv1.Ingress{ObjectMeta: enforceMeta(name), Spec: networkingv1.IngressSpec{Rules: []networkingv1.IngressRule{{IngressRuleValue: networkingv1.IngressRuleValue{HTTP: &networkingv1.HTTPIngressRuleValue{Paths: []networkingv1.HTTPIngressPath{{Backend: networkingv1.IngressBackend{Service: &networkingv1.IngressServiceBackend{Name: service, Port: networkingv1.ServiceBackendPort{Number: port}}}}}}}}}}}
}

func serviceWithSelector(name string) *corev1.Service {
	return &corev1.Service{ObjectMeta: enforceMeta(name), Spec: corev1.ServiceSpec{Selector: map[string]string{"app": "api"}}}
}

func scope(kind string) resolver.ResourceScope {
	return resolver.ResourceScope{Kind: kind, Namespace: formalNamespace}
}

func TestFormalDeploymentConfigMapCases(t *testing.T) {
	source := deploymentWithConfigMap("consumer", "settings")
	target := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "settings", Namespace: formalNamespace}}

	t.Run("source normal", func(t *testing.T) {
		evaluateFormal(t, resourcesByScope{scope("ConfigMap"): {target}}, rules.Request{Resource: "deployments", Operation: rules.Create, Namespace: formalNamespace, Name: source.Name, Object: source}, false)
	})
	t.Run("source conflict", func(t *testing.T) {
		evaluateFormal(t, resourcesByScope{}, rules.Request{Resource: "deployments", Operation: rules.Create, Namespace: formalNamespace, Name: source.Name, Object: source}, true)
	})
	t.Run("target normal", func(t *testing.T) {
		evaluateFormal(t, resourcesByScope{}, rules.Request{Resource: "configmaps", Operation: rules.Delete, Namespace: formalNamespace, Name: "unused"}, false)
	})
	t.Run("target conflict", func(t *testing.T) {
		evaluateFormal(t, resourcesByScope{scope("Deployment"): {source}}, rules.Request{Resource: "configmaps", Operation: rules.Delete, Namespace: formalNamespace, Name: target.Name}, true)
	})
}

func TestFormalPodSecretCases(t *testing.T) {
	source := podWithSecret("consumer", "credentials", nil)
	target := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "credentials", Namespace: formalNamespace}}

	t.Run("source normal", func(t *testing.T) {
		evaluateFormal(t, resourcesByScope{scope("Secret"): {target}}, rules.Request{Resource: "pods", Operation: rules.Create, Namespace: formalNamespace, Name: source.Name, Object: source}, false)
	})
	t.Run("source conflict", func(t *testing.T) {
		evaluateFormal(t, resourcesByScope{}, rules.Request{Resource: "pods", Operation: rules.Create, Namespace: formalNamespace, Name: source.Name, Object: source}, true)
	})
	t.Run("target normal", func(t *testing.T) {
		evaluateFormal(t, resourcesByScope{}, rules.Request{Resource: "secrets", Operation: rules.Delete, Namespace: formalNamespace, Name: "unused"}, false)
	})
	t.Run("target conflict", func(t *testing.T) {
		evaluateFormal(t, resourcesByScope{scope("Pod"): {source}}, rules.Request{Resource: "secrets", Operation: rules.Delete, Namespace: formalNamespace, Name: target.Name}, true)
	})
}

func TestFormalDeploymentPVCCases(t *testing.T) {
	source := deploymentWithPVC("consumer", "data")
	target := &corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: "data", Namespace: formalNamespace}}

	t.Run("source normal", func(t *testing.T) {
		evaluateFormal(t, resourcesByScope{scope("PersistentVolumeClaim"): {target}}, rules.Request{Resource: "deployments", Operation: rules.Create, Namespace: formalNamespace, Name: source.Name, Object: source}, false)
	})
	t.Run("source conflict", func(t *testing.T) {
		evaluateFormal(t, resourcesByScope{}, rules.Request{Resource: "deployments", Operation: rules.Create, Namespace: formalNamespace, Name: source.Name, Object: source}, true)
	})
	t.Run("target normal", func(t *testing.T) {
		evaluateFormal(t, resourcesByScope{}, rules.Request{Resource: "persistentvolumeclaims", Operation: rules.Delete, Namespace: formalNamespace, Name: "unused"}, false)
	})
	t.Run("target conflict", func(t *testing.T) {
		evaluateFormal(t, resourcesByScope{scope("Deployment"): {source}}, rules.Request{Resource: "persistentvolumeclaims", Operation: rules.Delete, Namespace: formalNamespace, Name: target.Name}, true)
	})
}

func TestFormalIngressServiceCases(t *testing.T) {
	source := ingressWithService("edge", "backend", 8080)
	target := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "backend", Namespace: formalNamespace}, Spec: corev1.ServiceSpec{Ports: []corev1.ServicePort{{Port: 8080}}}}

	t.Run("source normal", func(t *testing.T) {
		evaluateFormal(t, resourcesByScope{scope("Service"): {target}}, rules.Request{Resource: "ingresses", Operation: rules.Create, Namespace: formalNamespace, Name: source.Name, Object: source}, false)
	})
	t.Run("source conflict", func(t *testing.T) {
		evaluateFormal(t, resourcesByScope{}, rules.Request{Resource: "ingresses", Operation: rules.Create, Namespace: formalNamespace, Name: source.Name, Object: source}, true)
	})
	t.Run("target normal", func(t *testing.T) {
		evaluateFormal(t, resourcesByScope{}, rules.Request{Resource: "services", Operation: rules.Delete, Namespace: formalNamespace, Name: "unused"}, false)
	})
	t.Run("target conflict", func(t *testing.T) {
		evaluateFormal(t, resourcesByScope{scope("Ingress"): {source}}, rules.Request{Resource: "services", Operation: rules.Delete, Namespace: formalNamespace, Name: target.Name}, true)
	})
	t.Run("target port update conflict", func(t *testing.T) {
		updated := target.DeepCopy()
		updated.Spec.Ports = []corev1.ServicePort{{Port: 9090}}
		evaluateFormal(t, resourcesByScope{scope("Ingress"): {source}}, rules.Request{Resource: "services", Operation: rules.Update, Namespace: formalNamespace, Name: target.Name, Object: updated}, true)
	})
}

func TestFormalServicePodCases(t *testing.T) {
	source := serviceWithSelector("api")
	matching := podWithSecret("api-0", "", map[string]string{"app": "api"})
	other := podWithSecret("api-1", "", map[string]string{"app": "api"})

	t.Run("source normal", func(t *testing.T) {
		evaluateFormal(t, resourcesByScope{scope("Pod"): {matching}}, rules.Request{Resource: "services", Operation: rules.Create, Namespace: formalNamespace, Name: source.Name, Object: source}, false)
	})
	t.Run("source conflict", func(t *testing.T) {
		evaluateFormal(t, resourcesByScope{}, rules.Request{Resource: "services", Operation: rules.Create, Namespace: formalNamespace, Name: source.Name, Object: source}, true)
	})
	t.Run("target normal", func(t *testing.T) {
		evaluateFormal(t, resourcesByScope{scope("Pod"): {matching, other}, scope("Service"): {source}}, rules.Request{Resource: "pods", Operation: rules.Delete, Namespace: formalNamespace, Name: matching.Name}, false)
	})
	t.Run("target conflict", func(t *testing.T) {
		evaluateFormal(t, resourcesByScope{scope("Pod"): {matching}, scope("Service"): {source}}, rules.Request{Resource: "pods", Operation: rules.Delete, Namespace: formalNamespace, Name: matching.Name}, true)
	})
	t.Run("target label update conflict", func(t *testing.T) {
		updated := matching.DeepCopy()
		updated.Labels = map[string]string{"app": "other"}
		evaluateFormal(t, resourcesByScope{scope("Pod"): {matching}, scope("Service"): {source}}, rules.Request{Resource: "pods", Operation: rules.Update, Namespace: formalNamespace, Name: matching.Name, Object: updated}, true)
	})
}
