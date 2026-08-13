package rules

import (
	"fmt"

	"github.com/vincent/KubeDepGuard/internal/dependency"
	"github.com/vincent/KubeDepGuard/internal/dependency/resolver"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type Operation string

const (
	Create Operation = "CREATE"
	Update Operation = "UPDATE"
	Delete Operation = "DELETE"
)

// Request is the transport-independent admission operation being evaluated.
type Request struct {
	Resource  string
	Operation Operation
	Namespace string
	Name      string
	Object    runtime.Object
}

// Query is the read-only Kubernetes state available to rules.
type Query interface {
	resolver.TargetResolver
	GetPod(namespace, name string) (*corev1.Pod, error)
	ListPods(namespace string) ([]*corev1.Pod, error)
	ListServices(namespace string) ([]*corev1.Service, error)
}

type Result struct {
	Violations []dependency.Violation
	Reject     bool
}

type AdmissionRule interface {
	Rule
	Applies(Request) bool
	Evaluate(Request, Query) (Result, error)
}

// Evaluator runs every rule relevant to one admission operation.
type Evaluator struct {
	query Query
	rules []AdmissionRule
}

func NewEvaluator(query Query, rules []AdmissionRule) *Evaluator {
	return &Evaluator{query: query, rules: append([]AdmissionRule(nil), rules...)}
}

func (e *Evaluator) Evaluate(request Request) (Result, error) {
	var result Result
	for _, rule := range e.rules {
		if !rule.Applies(request) {
			continue
		}
		found, err := rule.Evaluate(request, e.query)
		if err != nil {
			return Result{}, fmt.Errorf("%s: %w", rule.Name(), err)
		}
		result.Violations = append(result.Violations, found.Violations...)
		result.Reject = result.Reject || found.Reject
	}
	return result, nil
}
