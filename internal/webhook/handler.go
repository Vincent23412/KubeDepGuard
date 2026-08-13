package webhook

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/vincent/KubeDepGuard/internal/dependency/resolver"
	"github.com/vincent/KubeDepGuard/internal/dependency/rules"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corelisters "k8s.io/client-go/listers/core/v1"
)

const ValidationPath = "/validate/dependencies"

type Handler struct {
	evaluator *rules.Evaluator
	log       *slog.Logger
}

func New(pods corelisters.PodLister, configMaps corelisters.ConfigMapLister, services corelisters.ServiceLister, log *slog.Logger) *Handler {
	query := resolver.NewInformerQuery(pods, configMaps, services)
	return &Handler{evaluator: rules.NewEvaluator(query, rules.DefaultRegistry.AdmissionRules()), log: log}
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
	request, err := newRuleRequest(req)
	if err != nil {
		return deny(err.Error())
	}
	result, err := h.evaluator.Evaluate(request)
	if err != nil {
		return deny(fmt.Sprintf("evaluate dependency rules: %v", err))
	}
	for _, violation := range result.Violations {
		if !result.Reject {
			h.log.Warn("dependency violation allowed by warn policy", "rule", violation.Rule, "resource", violation.Resource)
		}
	}
	if len(result.Violations) > 0 && result.Reject {
		return deny(result.Violations[0].Message)
	}
	return allow()
}

func newRuleRequest(req *admissionv1.AdmissionRequest) (rules.Request, error) {
	request := rules.Request{Resource: req.Resource.Resource, Operation: rules.Operation(req.Operation), Namespace: req.Namespace, Name: req.Name}
	if req.Operation == admissionv1.Delete {
		return request, nil
	}
	switch req.Resource.Resource {
	case "pods":
		pod := &corev1.Pod{}
		if err := json.Unmarshal(req.Object.Raw, pod); err != nil {
			return rules.Request{}, fmt.Errorf("decode Pod: %w", err)
		}
		request.Object = pod
	case "services":
		service := &corev1.Service{}
		if err := json.Unmarshal(req.Object.Raw, service); err != nil {
			return rules.Request{}, fmt.Errorf("decode Service: %w", err)
		}
		request.Object = service
	}
	return request, nil
}

func deny(message string) *admissionv1.AdmissionResponse {
	return &admissionv1.AdmissionResponse{Allowed: false, Result: &metav1.Status{Message: message, Code: http.StatusForbidden}}
}
