package workload

import (
	"fmt"
	ref "github.com/vincent/KubeDepGuard/internal/dependency/reference"
	"github.com/vincent/KubeDepGuard/internal/dependency/reference/podspec"
	apps "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// DeploymentSecretExtractor extracts Secret references from a Deployment template.
type DeploymentSecretExtractor struct{}

func NewDeploymentSecretExtractor() ref.Extractor    { return DeploymentSecretExtractor{} }
func (DeploymentSecretExtractor) SourceKind() string { return "Deployment" }
func (DeploymentSecretExtractor) Extract(object runtime.Object) ([]ref.Reference, error) {
	deployment, ok := object.(*apps.Deployment)
	if !ok {
		return nil, fmt.Errorf("Deployment Secret extractor received %T", object)
	}
	return podspec.SecretReferences("Deployment", deployment.Namespace, "spec.template.spec", &deployment.Spec.Template.Spec), nil
}
