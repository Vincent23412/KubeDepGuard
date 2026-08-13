package rules

import (
	"fmt"

	"github.com/vincent/KubeDepGuard/internal/dependency"
	"github.com/vincent/KubeDepGuard/internal/dependency/resolver"
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

type Result struct {
	Violations []dependency.Violation
	Reject     bool
}

type AdmissionRule interface {
	Rule
	Applies(Request) bool
	Evaluate(Request, resolver.Query) (Result, error)
}

// Evaluator runs every rule relevant to one admission operation.
type Evaluator struct {
	query resolver.Query
	rules []AdmissionRule
}

func NewEvaluator(query resolver.Query, rules []AdmissionRule) *Evaluator {
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
