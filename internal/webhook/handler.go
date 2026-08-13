package webhook

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/vincent/KubeDepGuard/internal/dependency"
	ref "github.com/vincent/KubeDepGuard/internal/dependency/reference"
	"github.com/vincent/KubeDepGuard/internal/dependency/resolver"
	"github.com/vincent/KubeDepGuard/internal/dependency/rules"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	corelisters "k8s.io/client-go/listers/core/v1"
)

const ValidationPath = "/validate/dependencies"

type Handler struct {
	pods     corelisters.PodLister
	resolver *resolver.ListerResolver
	services corelisters.ServiceLister
	log      *slog.Logger
}

func New(pods corelisters.PodLister, configMaps corelisters.ConfigMapLister, services corelisters.ServiceLister, log *slog.Logger) *Handler {
	return &Handler{pods: pods, resolver: resolver.NewListerResolver(configMaps), services: services, log: log}
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
	response := h.validate(review.Request)
	response.UID = review.Request.UID
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(admissionv1.AdmissionReview{TypeMeta: metav1.TypeMeta{APIVersion: "admission.k8s.io/v1", Kind: "AdmissionReview"}, Response: response})
}

func (h *Handler) validate(req *admissionv1.AdmissionRequest) *admissionv1.AdmissionResponse {
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
			current, err := h.pods.Pods(req.Namespace).Get(req.Name)
			if err != nil {
				return allow()
			}
			pod = *current
			services, err := h.services.Services(pod.Namespace).List(labels.Everything())
			if err != nil {
				return deny(fmt.Sprintf("list Services: %v", err))
			}
			pods, err := h.pods.Pods(pod.Namespace).List(labels.Everything())
			if err != nil {
				return deny(fmt.Sprintf("list Pods: %v", err))
			}
			remaining := make([]*corev1.Pod, 0, len(pods))
			for _, candidate := range pods {
				if candidate.Name != pod.Name {
					remaining = append(remaining, candidate)
				}
			}
			for _, svc := range services {
				if dependency.ModeFor(svc) == dependency.ModeEnforce {
					for _, rule := range rules.DefaultRegistry.ServiceConditionalRules() {
						if !rule.HasMatch(svc, remaining) {
							violations = append(violations, rule.Validate(svc, remaining)...)
							reject = true
						}
					}
				}
			}
		} else if err := json.Unmarshal(req.Object.Raw, &pod); err != nil {
			return deny(fmt.Sprintf("decode Pod: %v", err))
		}
		if req.Operation != admissionv1.Delete && dependency.ModeFor(&pod) != dependency.ModeDisabled {
			references, err := ref.DefaultRegistry.Extract("Pod", &pod)
			if err != nil {
				return deny(fmt.Sprintf("extract Pod references: %v", err))
			}
			for _, rule := range rules.DefaultRegistry.DirectReferenceRules() {
				violations = append(violations, rule.Validate(&pod, references, h.resolver)...)
			}
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
			pods, err := h.pods.Pods(service.Namespace).List(labels.Everything())
			if err != nil {
				return deny(fmt.Sprintf("list Pods: %v", err))
			}
			for _, rule := range rules.DefaultRegistry.ServiceConditionalRules() {
				violations = append(violations, rule.Validate(&service, pods)...)
			}
			reject = len(violations) > 0 && dependency.ModeFor(&service) == dependency.ModeEnforce
		}
	case "configmaps":
		if req.Operation != admissionv1.Delete {
			break
		}
		pods, err := h.pods.Pods(req.Namespace).List(labels.Everything())
		if err != nil {
			return deny(fmt.Sprintf("list Pods: %v", err))
		}
		for _, pod := range pods {
			if dependency.ModeFor(pod) != dependency.ModeEnforce {
				continue
			}
			for _, rule := range rules.DefaultRegistry.DirectReferenceRules() {
				if rule.TargetKind() != "ConfigMap" {
					continue
				}
				references, err := ref.DefaultRegistry.Extract("Pod", pod)
				if err != nil {
					return deny(fmt.Sprintf("extract Pod references: %v", err))
				}
				for _, ref := range references {
					if ref.TargetKind == rule.TargetKind() && ref.Name == req.Name {
						violations = append(violations, dependency.Violation{"ReferencedConfigMapDeletion", req.Namespace + "/" + req.Name, fmt.Sprintf("is still referenced by enforce Pod %s", pod.Name)})
						reject = true
					}
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
