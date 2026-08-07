package dependency

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Rule is the common contract implemented by every dependency rule.
type Rule interface {
	Name() string
}

// RuleConstructor creates one concrete rule instance.
type RuleConstructor func() Rule

type RuleCategory string

const (
	DirectReferenceCategory RuleCategory = "direct-reference"
	ConditionalCategory     RuleCategory = "conditional"
)

// RuleFactory is the registry-facing factory contract. Its category makes the
// rule family explicit before the registry accepts the constructed rule.
type RuleFactory interface {
	Category() RuleCategory
	Create() Rule
}

// DirectReferenceRuleFactory registers rules derived from DirectReferenceRule.
type DirectReferenceRuleFactory struct{ constructor RuleConstructor }

func NewDirectReferenceRuleFactory(constructor RuleConstructor) DirectReferenceRuleFactory {
	return DirectReferenceRuleFactory{constructor: constructor}
}
func (f DirectReferenceRuleFactory) Category() RuleCategory { return DirectReferenceCategory }
func (f DirectReferenceRuleFactory) Create() Rule           { return f.constructor() }

// ConditionalRuleFactory registers rules derived from ConditionalRule.
type ConditionalRuleFactory struct{ constructor RuleConstructor }

func NewConditionalRuleFactory(constructor RuleConstructor) ConditionalRuleFactory {
	return ConditionalRuleFactory{constructor: constructor}
}
func (f ConditionalRuleFactory) Category() RuleCategory { return ConditionalCategory }
func (f ConditionalRuleFactory) Create() Rule           { return f.constructor() }

// Registry is the single place where enabled rules are assembled. Add a new
// factory to DefaultRegistry to make a rule available to both components.
type Registry struct {
	directRules      []PodDirectReferenceRule
	conditionalRules []ServiceConditionalRule
}

func NewRegistry(factories ...RuleFactory) *Registry {
	registry := &Registry{}
	seen := make(map[string]struct{}, len(factories))
	for _, factory := range factories {
		rule := factory.Create()
		if rule == nil {
			panic("dependency rule factory returned nil")
		}
		if _, exists := seen[rule.Name()]; exists {
			panic("duplicate dependency rule: " + rule.Name())
		}
		seen[rule.Name()] = struct{}{}
		switch factory.Category() {
		case DirectReferenceCategory:
			directRule, ok := rule.(PodDirectReferenceRule)
			if !ok {
				panic("direct reference factory produced incompatible rule: " + rule.Name())
			}
			registry.directRules = append(registry.directRules, directRule)
		case ConditionalCategory:
			conditionalRule, ok := rule.(ServiceConditionalRule)
			if !ok {
				panic("conditional factory produced incompatible rule: " + rule.Name())
			}
			registry.conditionalRules = append(registry.conditionalRules, conditionalRule)
		default:
			panic("unsupported dependency rule category: " + string(factory.Category()))
		}
	}
	return registry
}

// PodDirectReferenceRule is the common execution contract for direct rules
// sourced from a Pod. Future Pod -> Secret rules can implement this interface.
type PodDirectReferenceRule interface {
	Rule
	TargetKind() string
	References(*corev1.Pod) []string
	Validate(*corev1.Pod, TargetResolver) []Violation
}

// TargetResolver lets direct rules query targets without depending on an API
// client or Informer cache. The caller supplies the backing lookup.
type TargetResolver func(kind, name string) bool

func (r *Registry) PodDirectReferenceRules() []PodDirectReferenceRule {
	return append([]PodDirectReferenceRule(nil), r.directRules...)
}

// ServiceConditionalRule is the common execution contract for conditional
// rules inferred from a Service and its candidate Pods.
type ServiceConditionalRule interface {
	Rule
	HasMatch(*corev1.Service, []*corev1.Pod) bool
	Validate(*corev1.Service, []*corev1.Pod) []Violation
}

func (r *Registry) ServiceConditionalRules() []ServiceConditionalRule {
	return append([]ServiceConditionalRule(nil), r.conditionalRules...)
}

// DirectReferenceRule is the embeddable base for rules whose source resource
// explicitly names another resource, such as Pod -> ConfigMap.
type DirectReferenceRule struct {
	RuleName           string
	TargetResourceKind string
}

func (r DirectReferenceRule) Name() string       { return r.RuleName }
func (r DirectReferenceRule) TargetKind() string { return r.TargetResourceKind }

func (r DirectReferenceRule) Missing(source metav1.Object, references []string, exists TargetResolver) []Violation {
	var violations []Violation
	for _, reference := range references {
		if !exists(r.TargetKind(), reference) {
			violations = append(violations, Violation{
				Rule:     r.Name(),
				Resource: resourceKey(source),
				Message:  fmt.Sprintf("references %s %q which does not exist", r.TargetResourceKind, reference),
			})
		}
	}
	return violations
}

// ConditionalRule is the embeddable base for rules whose dependency is
// inferred from a predicate over other resources, such as Service -> Pod labels.
type ConditionalRule struct {
	RuleName string
}

func (r ConditionalRule) Name() string { return r.RuleName }

func (r ConditionalRule) Unsatisfied(source metav1.Object, message string) []Violation {
	return []Violation{{Rule: r.Name(), Resource: resourceKey(source), Message: message}}
}

func resourceKey(object metav1.Object) string {
	return object.GetNamespace() + "/" + object.GetName()
}

// DefaultRegistry lists the rules enabled by this MVP. Adding a new rule is a
// one-line factory registration here; Webhook and Monitor discover it by type.
var DefaultRegistry = NewRegistry(
	NewDirectReferenceRuleFactory(NewPodConfigMapRule),
	NewConditionalRuleFactory(NewServicePodSelectorRule),
)
