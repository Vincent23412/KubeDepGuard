package workload

import (
	"fmt"

	ref "github.com/vincent/KubeDepGuard/internal/dependency/reference"
	"github.com/vincent/KubeDepGuard/internal/dependency/reference/podspec"
	apps "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// DeploymentConfigMapExtractor extracts ConfigMap references from a
// Deployment's PodTemplateSpec.
type DeploymentConfigMapExtractor struct{}

func NewDeploymentConfigMapExtractor() ref.Extractor    { return DeploymentConfigMapExtractor{} }
func (DeploymentConfigMapExtractor) SourceKind() string { return "Deployment" }
func (DeploymentConfigMapExtractor) Extract(object runtime.Object) ([]ref.Reference, error) {
	deployment, ok := object.(*apps.Deployment)
	if !ok {
		return nil, fmt.Errorf("Deployment ConfigMap extractor received %T", object)
	}
	return podspec.ConfigMapReferences("Deployment", deployment.Namespace, "spec.template.spec", &deployment.Spec.Template.Spec), nil
}
