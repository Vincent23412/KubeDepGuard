package rules

import (
	"fmt"

	"github.com/vincent/KubeDepGuard/internal/dependency"
	ref "github.com/vincent/KubeDepGuard/internal/dependency/reference"
	"github.com/vincent/KubeDepGuard/internal/dependency/resolver"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type DirectReferenceRule struct{ RuleName, TargetResourceKind string }

func (r DirectReferenceRule) Name() string       { return r.RuleName }
func (r DirectReferenceRule) TargetKind() string { return r.TargetResourceKind }

func (r DirectReferenceRule) missing(source metav1.Object, refs []ref.Reference, targets resolver.TargetResolver) []dependency.Violation {
	var out []dependency.Violation
	for _, x := range refs {
		if x.TargetKind != r.TargetKind() || x.Optional {
			continue
		}
		exists, err := targets.Exists(resolver.Target{Kind: x.TargetKind, Namespace: x.Namespace, Name: x.Name})
		if err != nil || !exists {
			message := fmt.Sprintf("references %s %q at %s which does not exist", x.TargetKind, x.Name, x.FieldPath)
			if err != nil {
				message = fmt.Sprintf("cannot resolve %s %q at %s: %v", x.TargetKind, x.Name, x.FieldPath, err)
			}
			out = append(out, dependency.Violation{Rule: r.Name(), Resource: source.GetNamespace() + "/" + source.GetName(), Message: message})
		}
	}
	return out
}

// referencedBy reports an inbound dependency while a target is being deleted.
// Concrete rules supply the source kind so the message remains meaningful.
func (r DirectReferenceRule) referencedBy(source metav1.Object, refs []ref.Reference, target resolver.Target, sourceKind string) []dependency.Violation {
	for _, x := range refs {
		if x.TargetKind == r.TargetKind() && x.Namespace == target.Namespace && x.Name == target.Name {
			return []dependency.Violation{{
				Rule:     "Referenced" + r.TargetKind() + "Deletion",
				Resource: target.Namespace + "/" + target.Name,
				Message:  fmt.Sprintf("is still referenced by enforce %s %s", sourceKind, source.GetName()),
			}}
		}
	}
	return nil
}
