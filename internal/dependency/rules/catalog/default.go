// Package catalog assembles the concrete dependency rules enabled by default.
package catalog

import (
	"github.com/vincent/KubeDepGuard/internal/dependency/rules"
	"github.com/vincent/KubeDepGuard/internal/dependency/rules/conditional"
	"github.com/vincent/KubeDepGuard/internal/dependency/rules/direct"
)

var DefaultRegistry = rules.NewRegistry(
	direct.NewWorkloadConfigMapRule(),
	direct.NewWorkloadSecretRule(),
	direct.NewWorkloadPersistentVolumeClaimRule(),
	direct.NewWorkloadServiceAccountRule(),
	direct.NewPodNodeRule(),
	conditional.NewIngressServiceRule(),
	conditional.NewServicePodRule(),
)
