package workload

import (
	"fmt"
	ref "github.com/vincent/KubeDepGuard/internal/dependency/reference"
	"github.com/vincent/KubeDepGuard/internal/dependency/reference/podspec"
	apps "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type DeploymentServiceAccountExtractor struct{}

func NewDeploymentServiceAccountExtractor() ref.Extractor    { return DeploymentServiceAccountExtractor{} }
func (DeploymentServiceAccountExtractor) SourceKind() string { return "Deployment" }
func (DeploymentServiceAccountExtractor) Extract(object runtime.Object) ([]ref.Reference, error) {
	deployment, ok := object.(*apps.Deployment)
	if !ok {
		return nil, fmt.Errorf("Deployment ServiceAccount extractor received %T", object)
	}
	return podspec.ServiceAccountReferences("Deployment", deployment.Namespace, "spec.template.spec", &deployment.Spec.Template.Spec), nil
}
