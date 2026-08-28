package workload

import (
	"fmt"
	ref "github.com/vincent/KubeDepGuard/internal/dependency/reference"
	"github.com/vincent/KubeDepGuard/internal/dependency/reference/podspec"
	apps "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// DeploymentPersistentVolumeClaimExtractor extracts PVC references from a Deployment template.
type DeploymentPersistentVolumeClaimExtractor struct{}

func NewDeploymentPersistentVolumeClaimExtractor() ref.Extractor {
	return DeploymentPersistentVolumeClaimExtractor{}
}
func (DeploymentPersistentVolumeClaimExtractor) SourceKind() string { return "Deployment" }
func (DeploymentPersistentVolumeClaimExtractor) Extract(object runtime.Object) ([]ref.Reference, error) {
	deployment, ok := object.(*apps.Deployment)
	if !ok {
		return nil, fmt.Errorf("Deployment PVC extractor received %T", object)
	}
	return podspec.PersistentVolumeClaimReferences("Deployment", deployment.Namespace, "spec.template.spec", &deployment.Spec.Template.Spec), nil
}
