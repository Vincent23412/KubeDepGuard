// Package catalog assembles the reference extractors enabled by default.
package catalog

import (
	ref "github.com/vincent/KubeDepGuard/internal/dependency/reference"
	"github.com/vincent/KubeDepGuard/internal/dependency/reference/ingress"
	"github.com/vincent/KubeDepGuard/internal/dependency/reference/workload"
)

// DefaultDirectRegistry contains every direct-reference field extractor.
var DefaultDirectRegistry = ref.NewRegistry(
	workload.NewPodConfigMapExtractor,
	workload.NewPodSecretExtractor,
	workload.NewPodPersistentVolumeClaimExtractor,
	workload.NewDeploymentConfigMapExtractor,
	workload.NewDeploymentSecretExtractor,
	workload.NewDeploymentPersistentVolumeClaimExtractor,
	workload.NewPodServiceAccountExtractor,
	workload.NewDeploymentServiceAccountExtractor,
	workload.NewPodNodeExtractor,
	ingress.NewServiceExtractor,
)
