package direct

import (
	"fmt"

	"github.com/vincent/KubeDepGuard/internal/dependency"
	"github.com/vincent/KubeDepGuard/internal/dependency/resolver"
	"github.com/vincent/KubeDepGuard/internal/dependency/rules"
	corev1 "k8s.io/api/core/v1"
)

// PodNodeRule validates an explicitly selected Node exists and is schedulable.
// The dependency is direct because Pod.spec.nodeName names the target Node;
// the target's current state is validated as part of the same rule.
type PodNodeRule struct{}

func NewPodNodeRule() PodNodeRule { return PodNodeRule{} }
func (PodNodeRule) Name() string  { return "UnavailableNode" }
func (PodNodeRule) Applies(r rules.Request) bool {
	return r.Resource == "pods" && (r.Operation == rules.Create || r.Operation == rules.Update)
}

func (r PodNodeRule) Evaluate(req rules.Request, query resolver.Query) (rules.Result, error) {
	pod, ok := req.Object.(*corev1.Pod)
	if !ok {
		return rules.Result{}, fmt.Errorf("expected Pod object")
	}
	if dependency.ModeFor(pod) == dependency.ModeDisabled || pod.Spec.NodeName == "" {
		return rules.Result{}, nil
	}
	nodes, err := query.List(resolver.ResourceScope{Kind: "Node"})
	if err != nil {
		return rules.Result{}, err
	}
	var node *corev1.Node
	for _, item := range nodes {
		if item.GetName() == pod.Spec.NodeName {
			node, _ = item.(*corev1.Node)
			break
		}
	}
	if node == nil {
		return rules.Result{Violations: []dependency.Violation{{Rule: r.Name(), Resource: pod.Namespace + "/" + pod.Name, Message: fmt.Sprintf("references Node %q at spec.nodeName which does not exist", pod.Spec.NodeName)}}, Reject: dependency.ModeFor(pod) == dependency.ModeEnforce}, nil
	}
	if node.Spec.Unschedulable {
		return rules.Result{Violations: []dependency.Violation{{Rule: "UnschedulableNode", Resource: pod.Namespace + "/" + pod.Name, Message: fmt.Sprintf("references Node %q which is unschedulable", node.Name)}}, Reject: dependency.ModeFor(pod) == dependency.ModeEnforce}, nil
	}
	return rules.Result{}, nil
}
