package resolver

import (
	"context"
	"fmt"

	"k8s.io/client-go/informers"
	coreinformers "k8s.io/client-go/informers/core/v1"
	discoveryinformers "k8s.io/client-go/informers/discovery/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

// MonitorInformerResolver owns the informer lifecycle and event sources used
// by the monitor. It also exposes the same cache-backed Query used by rules.
type MonitorInformerResolver struct {
	factory        informers.SharedInformerFactory
	pods           coreinformers.PodInformer
	configMaps     coreinformers.ConfigMapInformer
	secrets        coreinformers.SecretInformer
	services       coreinformers.ServiceInformer
	endpointSlices discoveryinformers.EndpointSliceInformer
	query          *ListerResolver
	syncers        []cache.InformerSynced
}

// NewMonitorInformerResolver materializes every monitor informer before Start
// so callers can register event handlers before the factory begins running.
func NewMonitorInformerResolver(client kubernetes.Interface) *MonitorInformerResolver {
	factory := informers.NewSharedInformerFactory(client, 0)
	pods := factory.Core().V1().Pods()
	configMaps := factory.Core().V1().ConfigMaps()
	secrets := factory.Core().V1().Secrets()
	services := factory.Core().V1().Services()
	endpointSlices := factory.Discovery().V1().EndpointSlices()
	return &MonitorInformerResolver{
		factory:        factory,
		pods:           pods,
		configMaps:     configMaps,
		secrets:        secrets,
		services:       services,
		endpointSlices: endpointSlices,
		query:          NewListerResolver(pods.Lister(), configMaps.Lister(), secrets.Lister(), services.Lister()),
		syncers: []cache.InformerSynced{
			pods.Informer().HasSynced,
			configMaps.Informer().HasSynced,
			secrets.Informer().HasSynced,
			services.Informer().HasSynced,
			endpointSlices.Informer().HasSynced,
		},
	}
}

func (r *MonitorInformerResolver) Pods() coreinformers.PodInformer { return r.pods }

func (r *MonitorInformerResolver) ConfigMaps() coreinformers.ConfigMapInformer { return r.configMaps }

func (r *MonitorInformerResolver) Secrets() coreinformers.SecretInformer { return r.secrets }

func (r *MonitorInformerResolver) Services() coreinformers.ServiceInformer { return r.services }

func (r *MonitorInformerResolver) EndpointSlices() discoveryinformers.EndpointSliceInformer {
	return r.endpointSlices
}

// Start begins the owned informer factory and waits for every monitor cache.
func (r *MonitorInformerResolver) Start(ctx context.Context) error {
	r.factory.Start(ctx.Done())
	if !cache.WaitForCacheSync(ctx.Done(), r.syncers...) {
		return fmt.Errorf("monitor informer cache did not synchronize")
	}
	return nil
}

func (r *MonitorInformerResolver) List(scope ResourceScope) ([]Resource, error) {
	return r.query.List(scope)
}

var _ Query = (*MonitorInformerResolver)(nil)
