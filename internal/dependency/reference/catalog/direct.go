// Package catalog assembles the reference extractors enabled by default.
package catalog

import (
	ref "github.com/vincent/KubeDepGuard/internal/dependency/reference"
	"github.com/vincent/KubeDepGuard/internal/dependency/reference/pod"
)

// DefaultDirectRegistry contains every direct-reference field extractor.
var DefaultDirectRegistry = ref.NewRegistry(
	pod.NewConfigMapExtractor,
)
