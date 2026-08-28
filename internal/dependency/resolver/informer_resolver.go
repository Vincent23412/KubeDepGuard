package resolver

import (
	"context"
	"fmt"

	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

// InformerResolver owns the shared informer factory used by admission rules.
// Its reads are served entirely from informer caches.
type InformerResolver struct {
	factory informers.SharedInformerFactory
	query   *ListerResolver
	syncers []cache.InformerSynced
}

// NewInformerResolver registers every resource currently supported by the
// dependency rule registry. Call Start before evaluating requests.
func NewInformerResolver(client kubernetes.Interface) *InformerResolver {
	factory := informers.NewSharedInformerFactory(client, 0)
	deployments := factory.Apps().V1().Deployments()
	pods := factory.Core().V1().Pods()
	configMaps := factory.Core().V1().ConfigMaps()
	secrets := factory.Core().V1().Secrets()
	pvcs := factory.Core().V1().PersistentVolumeClaims()
	services := factory.Core().V1().Services()
	return &InformerResolver{
		factory: factory,
		query:   NewListerResolver(deployments.Lister(), pods.Lister(), configMaps.Lister(), secrets.Lister(), pvcs.Lister(), services.Lister()),
		syncers: []cache.InformerSynced{
			deployments.Informer().HasSynced,
			pods.Informer().HasSynced,
			configMaps.Informer().HasSynced,
			secrets.Informer().HasSynced,
			pvcs.Informer().HasSynced,
			services.Informer().HasSynced,
		},
	}
}

// Start begins the owned informer factory and waits for every supported cache.
func (r *InformerResolver) Start(ctx context.Context) error {
	r.factory.Start(ctx.Done())
	if !cache.WaitForCacheSync(ctx.Done(), r.syncers...) {
		return fmt.Errorf("informer cache did not synchronize")
	}
	return nil
}

func (r *InformerResolver) List(scope ResourceScope) ([]Resource, error) {
	return r.query.List(scope)
}

var _ Query = (*InformerResolver)(nil)
