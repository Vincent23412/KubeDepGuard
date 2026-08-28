package podspec

import (
	"fmt"
	ref "github.com/vincent/KubeDepGuard/internal/dependency/reference"
	corev1 "k8s.io/api/core/v1"
)

// PersistentVolumeClaimReferences extracts PVC volume references from any PodSpec owner.
func PersistentVolumeClaimReferences(sourceKind, namespace, pathPrefix string, spec *corev1.PodSpec) []ref.Reference {
	refs := make([]ref.Reference, 0)
	for i, volume := range spec.Volumes {
		if volume.PersistentVolumeClaim == nil || volume.PersistentVolumeClaim.ClaimName == "" {
			continue
		}
		refs = append(refs, ref.Reference{SourceKind: sourceKind, TargetKind: "PersistentVolumeClaim", Namespace: namespace, Name: volume.PersistentVolumeClaim.ClaimName, FieldPath: fmt.Sprintf("%s.volumes[%d].persistentVolumeClaim.claimName", pathPrefix, i)})
	}
	return refs
}
