package dependency

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

const PolicyAnnotation = "dependency.kubedepguard.io/mode"

type Mode string

const (
	ModeEnforce  Mode = "enforce"
	ModeWarn     Mode = "warn"
	ModeDisabled Mode = "disabled"
)

type Violation struct {
	Rule     string
	Resource string
	Message  string
}

// ModeFor uses warn as the safe migration default for resources without an annotation.
func ModeFor(meta metav1.Object) Mode {
	switch Mode(meta.GetAnnotations()[PolicyAnnotation]) {
	case ModeEnforce, ModeDisabled:
		return Mode(meta.GetAnnotations()[PolicyAnnotation])
	default:
		return ModeWarn
	}
}
