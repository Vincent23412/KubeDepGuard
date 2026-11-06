package resolver

import (
	"context"
	"fmt"

	"k8s.io/client-go/informers"
	appsinformers "k8s.io/client-go/informers/apps/v1"
	coreinformers "k8s.io/client-go/informers/core/v1"
	networkinginformers "k8s.io/client-go/informers/networking/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

// MonitorInformerResolver owns the informer lifecycle and event sources used
// by the monitor. It also exposes the same cache-backed Query used by rules.
type MonitorInformerResolver struct {
	factory         informers.SharedInformerFactory
	deployments     appsinformers.DeploymentInformer
	pods            coreinformers.PodInformer
	configMaps      coreinformers.ConfigMapInformer
	secrets         coreinformers.SecretInformer
	pvcs            coreinformers.PersistentVolumeClaimInformer
	services        coreinformers.ServiceInformer
	ingresses       networkinginformers.IngressInformer
	networkPolicies networkinginformers.NetworkPolicyInformer
	query           *ListerResolver
	syncers         []cache.InformerSynced
}

// NewMonitorInformerResolver materializes every monitor informer before Start
// so callers can register event handlers before the factory begins running.
func NewMonitorInformerResolver(client kubernetes.Interface) *MonitorInformerResolver {
	factory := informers.NewSharedInformerFactory(client, 0)
	deployments := factory.Apps().V1().Deployments()
	pods := factory.Core().V1().Pods()
	configMaps := factory.Core().V1().ConfigMaps()
	secrets := factory.Core().V1().Secrets()
	pvcs := factory.Core().V1().PersistentVolumeClaims()
	services := factory.Core().V1().Services()
	ingresses := factory.Networking().V1().Ingresses()
	networkPolicies := factory.Networking().V1().NetworkPolicies()
	return &MonitorInformerResolver{
		factory:         factory,
		deployments:     deployments,
		pods:            pods,
		configMaps:      configMaps,
		secrets:         secrets,
		pvcs:            pvcs,
		services:        services,
		ingresses:       ingresses,
		networkPolicies: networkPolicies,
		query:           NewListerResolver(deployments.Lister(), pods.Lister(), configMaps.Lister(), secrets.Lister(), pvcs.Lister(), services.Lister(), ingresses.Lister(), networkPolicies.Lister()),
		syncers: []cache.InformerSynced{
			deployments.Informer().HasSynced,
			pods.Informer().HasSynced,
			configMaps.Informer().HasSynced,
			secrets.Informer().HasSynced,
			pvcs.Informer().HasSynced,
			services.Informer().HasSynced,
			ingresses.Informer().HasSynced,
			networkPolicies.Informer().HasSynced,
		},
	}
}

func (r *MonitorInformerResolver) Deployments() appsinformers.DeploymentInformer {
	return r.deployments
}

func (r *MonitorInformerResolver) Pods() coreinformers.PodInformer { return r.pods }

func (r *MonitorInformerResolver) ConfigMaps() coreinformers.ConfigMapInformer { return r.configMaps }

func (r *MonitorInformerResolver) Secrets() coreinformers.SecretInformer { return r.secrets }

func (r *MonitorInformerResolver) PersistentVolumeClaims() coreinformers.PersistentVolumeClaimInformer {
	return r.pvcs
}

func (r *MonitorInformerResolver) Services() coreinformers.ServiceInformer { return r.services }

func (r *MonitorInformerResolver) Ingresses() networkinginformers.IngressInformer { return r.ingresses }

func (r *MonitorInformerResolver) NetworkPolicies() networkinginformers.NetworkPolicyInformer {
	return r.networkPolicies
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
