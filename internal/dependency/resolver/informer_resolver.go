package resolver

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
)

// InformerResolver owns the shared informer factory used by admission rules.
// Its reads are served entirely from informer caches.
type InformerResolver struct {
	factory  informers.SharedInformerFactory
	pods     corelisters.PodLister
	services corelisters.ServiceLister
	targets  *ListerResolver
	syncers  []cache.InformerSynced
}

// NewInformerResolver registers every resource currently supported by the
// dependency rule registry. Call Start before evaluating requests.
func NewInformerResolver(client kubernetes.Interface) *InformerResolver {
	factory := informers.NewSharedInformerFactory(client, 0)
	pods := factory.Core().V1().Pods()
	configMaps := factory.Core().V1().ConfigMaps()
	services := factory.Core().V1().Services()
	return &InformerResolver{
		factory:  factory,
		pods:     pods.Lister(),
		services: services.Lister(),
		targets:  NewListerResolver(configMaps.Lister()),
		syncers: []cache.InformerSynced{
			pods.Informer().HasSynced,
			configMaps.Informer().HasSynced,
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

func (r *InformerResolver) Exists(target Target) (bool, error) {
	return r.targets.Exists(target)
}

func (r *InformerResolver) GetPod(namespace, name string) (*corev1.Pod, error) {
	return r.pods.Pods(namespace).Get(name)
}

func (r *InformerResolver) ListPods(namespace string) ([]*corev1.Pod, error) {
	return r.pods.Pods(namespace).List(labels.Everything())
}

func (r *InformerResolver) ListServices(namespace string) ([]*corev1.Service, error) {
	return r.services.Services(namespace).List(labels.Everything())
}

var _ Query = (*InformerResolver)(nil)
