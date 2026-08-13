package resolver

import (
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	corelisters "k8s.io/client-go/listers/core/v1"
)

// ListerResolver resolves supported targets from SharedInformer caches.
type ListerResolver struct{ configMaps corelisters.ConfigMapLister }

func NewListerResolver(configMaps corelisters.ConfigMapLister) *ListerResolver {
	return &ListerResolver{configMaps: configMaps}
}
func (r *ListerResolver) Exists(target Target) (bool, error) {
	switch target.Kind {
	case "ConfigMap":
		_, err := r.configMaps.ConfigMaps(target.Namespace).Get(target.Name)
		if err == nil {
			return true, nil
		}
		if apierrors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	default:
		return false, fmt.Errorf("unsupported target kind %q", target.Kind)
	}
}
