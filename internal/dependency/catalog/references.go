// Package catalog assembles the enabled dependency components.
package catalog

import (
	ref "github.com/vincent/KubeDepGuard/internal/dependency/reference"
	"github.com/vincent/KubeDepGuard/internal/dependency/reference/pod"
)

var DefaultReferenceRegistry = ref.NewRegistry(
	pod.NewConfigMapExtractor,
)
