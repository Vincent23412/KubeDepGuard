package podspec

import (
	ref "github.com/vincent/KubeDepGuard/internal/dependency/reference"
	corev1 "k8s.io/api/core/v1"
)

// ServiceAccountReferences extracts the effective ServiceAccount name.
func ServiceAccountReferences(sourceKind, namespace, pathPrefix string, spec *corev1.PodSpec) []ref.Reference {
	name := spec.ServiceAccountName
	if name == "" {
		name = "default"
	}
	return []ref.Reference{{SourceKind: sourceKind, TargetKind: "ServiceAccount", Namespace: namespace, Name: name, FieldPath: pathPrefix + ".serviceAccountName"}}
}
