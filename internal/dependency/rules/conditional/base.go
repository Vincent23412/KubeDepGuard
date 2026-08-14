package conditional

import (
	"github.com/vincent/KubeDepGuard/internal/dependency"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ConditionalRule struct{ RuleName string }

func (r ConditionalRule) Name() string { return r.RuleName }
func (r ConditionalRule) unsatisfied(source metav1.Object, message string) []dependency.Violation {
	return []dependency.Violation{{Rule: r.Name(), Resource: source.GetNamespace() + "/" + source.GetName(), Message: message}}
}
