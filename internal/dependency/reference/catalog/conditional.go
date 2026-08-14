package catalog

import (
	serviceRef "github.com/vincent/KubeDepGuard/internal/dependency/reference/service"
	"k8s.io/apimachinery/pkg/runtime"
)

// DefaultServiceSelectorExtractor extracts the conditional reference data used
// by Service-to-Pod rules. Further conditional extractors belong in this file.
var DefaultServiceSelectorExtractor = serviceRef.NewSelectorExtractor()

func ExtractServiceSelector(object runtime.Object) (serviceRef.Selector, error) {
	return DefaultServiceSelectorExtractor.Extract(object)
}
