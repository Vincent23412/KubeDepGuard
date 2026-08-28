package direct

import (
	"fmt"

	"github.com/vincent/KubeDepGuard/internal/dependency"
	ref "github.com/vincent/KubeDepGuard/internal/dependency/reference"
	"github.com/vincent/KubeDepGuard/internal/dependency/resolver"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type DirectReferenceRule struct{ RuleName, TargetResourceKind string }

func (r DirectReferenceRule) Name() string       { return r.RuleName }
func (r DirectReferenceRule) TargetKind() string { return r.TargetResourceKind }

func (r DirectReferenceRule) missing(source metav1.Object, refs []ref.Reference, available map[resolver.ResourceScope]map[string]struct{}) []dependency.Violation {
	var out []dependency.Violation
	for _, x := range refs {
		if x.TargetKind != r.TargetKind() || x.Optional {
			continue
		}
		scope := resolver.ResourceScope{Kind: x.TargetKind, Namespace: x.Namespace}
		if _, exists := available[scope][x.Name]; !exists {
			message := fmt.Sprintf("references %s %q at %s which does not exist", x.TargetKind, x.Name, x.FieldPath)
			out = append(out, dependency.Violation{Rule: r.Name(), Resource: source.GetNamespace() + "/" + source.GetName(), Message: message})
		}
	}
	return out
}

func (r DirectReferenceRule) missingKeys(source metav1.Object, refs []ref.Reference, query resolver.Query) ([]dependency.Violation, error) {
	var out []dependency.Violation
	checked := map[resolver.ResourceScope]map[string]struct{}{}
	found := map[resolver.ResourceScope]map[string]struct{}{}
	for _, x := range refs {
		if x.TargetKind != r.TargetKind() || x.Optional || x.Key == "" {
			continue
		}
		scope := resolver.ResourceScope{Kind: x.TargetKind, Namespace: x.Namespace}
		if checked[scope] == nil {
			items, err := query.List(scope)
			if err != nil {
				return nil, err
			}
			checked[scope] = map[string]struct{}{}
			found[scope] = map[string]struct{}{}
			for _, item := range items {
				if item.GetName() != x.Name {
					continue
				}
				found[scope][x.Name] = struct{}{}
				var exists bool
				switch obj := item.(type) {
				case *corev1.ConfigMap:
					_, exists = obj.Data[x.Key]
					if !exists {
						_, exists = obj.BinaryData[x.Key]
					}
				case *corev1.Secret:
					_, exists = obj.Data[x.Key]
				}
				if exists {
					checked[scope][x.Name+"/"+x.Key] = struct{}{}
				}
			}
		}
		if _, exists := found[scope][x.Name]; !exists {
			continue
		}
		if _, exists := checked[scope][x.Name+"/"+x.Key]; !exists {
			out = append(out, dependency.Violation{Rule: r.Name() + "Key", Resource: source.GetNamespace() + "/" + source.GetName(), Message: fmt.Sprintf("references %s %q key %q at %s which does not exist", x.TargetKind, x.Name, x.Key, x.FieldPath)})
		}
	}
	return out, nil
}

// referencedBy reports an inbound dependency while a target is being deleted.
// Concrete rules supply the source kind so the message remains meaningful.
func (r DirectReferenceRule) referencedBy(source metav1.Object, refs []ref.Reference, scope resolver.ResourceScope, name, sourceKind string) []dependency.Violation {
	for _, x := range refs {
		if x.TargetKind == r.TargetKind() && x.Namespace == scope.Namespace && x.Name == name {
			return []dependency.Violation{{
				Rule:     "Referenced" + r.TargetKind() + "Deletion",
				Resource: scope.Namespace + "/" + name,
				Message:  fmt.Sprintf("is still referenced by enforce %s %s", sourceKind, source.GetName()),
			}}
		}
	}
	return nil
}
