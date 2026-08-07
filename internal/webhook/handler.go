package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/vincent/KubeDepGuard/internal/dependency"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type Handler struct {
	client kubernetes.Interface
	log    *slog.Logger
}

func New(client kubernetes.Interface, log *slog.Logger) *Handler {
	return &Handler{client: client, log: log}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var review admissionv1.AdmissionReview
	if err := json.Unmarshal(body, &review); err != nil || review.Request == nil {
		http.Error(w, "invalid AdmissionReview", http.StatusBadRequest)
		return
	}
	response := h.validate(r.Context(), review.Request)
	response.UID = review.Request.UID
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(admissionv1.AdmissionReview{TypeMeta: metav1.TypeMeta{APIVersion: "admission.k8s.io/v1", Kind: "AdmissionReview"}, Response: response})
}

func (h *Handler) validate(ctx context.Context, req *admissionv1.AdmissionRequest) *admissionv1.AdmissionResponse {
	allow := func() *admissionv1.AdmissionResponse { return &admissionv1.AdmissionResponse{Allowed: true} }
	if req.Namespace == "kube-dep-guard-system" {
		return allow()
	}
	var violations []dependency.Violation
	reject := false
	switch req.Resource.Resource {
	case "pods":
		var pod corev1.Pod
		if req.Operation == admissionv1.Delete {
			current, err := h.client.CoreV1().Pods(req.Namespace).Get(ctx, req.Name, metav1.GetOptions{})
			if err != nil {
				return allow()
			}
			pod = *current
			services, err := h.client.CoreV1().Services(pod.Namespace).List(ctx, metav1.ListOptions{})
			if err != nil {
				return deny(fmt.Sprintf("list Services: %v", err))
			}
			pods, err := h.client.CoreV1().Pods(pod.Namespace).List(ctx, metav1.ListOptions{})
			if err != nil {
				return deny(fmt.Sprintf("list Pods: %v", err))
			}
			remaining := make([]*corev1.Pod, 0, len(pods.Items))
			for i := range pods.Items {
				if pods.Items[i].Name != pod.Name {
					remaining = append(remaining, &pods.Items[i])
				}
			}
			for i := range services.Items {
				svc := &services.Items[i]
				if dependency.ModeFor(svc) == dependency.ModeEnforce && !dependency.ServiceHasMatchingPod(svc, remaining) {
					violations = append(violations, dependency.EmptyServiceSelector(svc, remaining)...)
					reject = true
				}
			}
		} else if err := json.Unmarshal(req.Object.Raw, &pod); err != nil {
			return deny(fmt.Sprintf("decode Pod: %v", err))
		}
		if req.Operation != admissionv1.Delete && dependency.ModeFor(&pod) != dependency.ModeDisabled {
			violations = append(violations, dependency.MissingConfigMaps(&pod, func(name string) bool {
				_, err := h.client.CoreV1().ConfigMaps(pod.Namespace).Get(ctx, name, metav1.GetOptions{})
				return err == nil
			})...)
			reject = len(violations) > 0 && dependency.ModeFor(&pod) == dependency.ModeEnforce
		}
	case "services":
		if req.Operation == admissionv1.Delete {
			return allow()
		}
		var service corev1.Service
		if err := json.Unmarshal(req.Object.Raw, &service); err != nil {
			return deny(fmt.Sprintf("decode Service: %v", err))
		}
		if dependency.ModeFor(&service) != dependency.ModeDisabled {
			list, err := h.client.CoreV1().Pods(service.Namespace).List(ctx, metav1.ListOptions{})
			if err != nil {
				return deny(fmt.Sprintf("list Pods: %v", err))
			}
			pods := make([]*corev1.Pod, 0, len(list.Items))
			for i := range list.Items {
				pods = append(pods, &list.Items[i])
			}
			violations = append(violations, dependency.EmptyServiceSelector(&service, pods)...)
			reject = len(violations) > 0 && dependency.ModeFor(&service) == dependency.ModeEnforce
		}
	case "configmaps":
		if req.Operation != admissionv1.Delete {
			break
		}
		pods, err := h.client.CoreV1().Pods(req.Namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			return deny(fmt.Sprintf("list Pods: %v", err))
		}
		for i := range pods.Items {
			pod := &pods.Items[i]
			if dependency.ModeFor(pod) != dependency.ModeEnforce {
				continue
			}
			for _, ref := range dependency.ConfigMapReferences(pod) {
				if ref == req.Name {
					violations = append(violations, dependency.Violation{"ReferencedConfigMapDeletion", req.Namespace + "/" + req.Name, fmt.Sprintf("is still referenced by enforce Pod %s", pod.Name)})
					reject = true
				}
			}
		}
	}
	for _, violation := range violations {
		if !reject {
			h.log.Warn("dependency violation allowed by warn policy", "rule", violation.Rule, "resource", violation.Resource)
		}
	}
	// The resource mode is evaluated per object. An enforce object rejects all violations.
	if len(violations) > 0 && reject {
		return deny(violations[0].Message)
	}
	return allow()
}

func deny(message string) *admissionv1.AdmissionResponse {
	return &admissionv1.AdmissionResponse{Allowed: false, Result: &metav1.Status{Message: message, Code: http.StatusForbidden}}
}
