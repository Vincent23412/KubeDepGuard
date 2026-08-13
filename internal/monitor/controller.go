package monitor

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/vincent/KubeDepGuard/internal/dependency"
	"github.com/vincent/KubeDepGuard/internal/dependency/catalog"
	serviceRef "github.com/vincent/KubeDepGuard/internal/dependency/reference/service"
	"github.com/vincent/KubeDepGuard/internal/dependency/resolver"
	"github.com/vincent/KubeDepGuard/internal/dependency/rules"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	coreinformers "k8s.io/client-go/informers/core/v1"
	discoveryinformers "k8s.io/client-go/informers/discovery/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
)

type Controller struct {
	pods           coreinformers.PodInformer
	services       coreinformers.ServiceInformer
	configMaps     coreinformers.ConfigMapInformer
	endpointSlices discoveryinformers.EndpointSliceInformer
	queue          workqueue.TypedRateLimitingInterface[string]
	recorder       record.EventRecorder
	log            *slog.Logger
	resolver       *resolver.ListerResolver
	podLister      resolver.PodLister
	ready          chan struct{}
	readyOnce      sync.Once
}

func NewWithInformers(pods coreinformers.PodInformer, services coreinformers.ServiceInformer, configMaps coreinformers.ConfigMapInformer, endpointSlices discoveryinformers.EndpointSliceInformer, recorder record.EventRecorder, log *slog.Logger) *Controller {
	c := &Controller{pods: pods, services: services, configMaps: configMaps, endpointSlices: endpointSlices, queue: workqueue.NewTypedRateLimitingQueue(workqueue.DefaultTypedControllerRateLimiter[string]()), recorder: recorder, log: log, resolver: resolver.NewListerResolver(configMaps.Lister()), podLister: resolver.NewInformerPodLister(pods.Lister()), ready: make(chan struct{})}
	pods.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(obj any) { c.enqueue("pod", obj); c.enqueueAllServices() },
		UpdateFunc: func(_, obj any) { c.enqueue("pod", obj); c.enqueueAllServices() },
		DeleteFunc: func(any) { c.enqueueAllServices() },
	})
	services.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{AddFunc: func(obj any) { c.enqueue("service", obj) }, UpdateFunc: func(_, obj any) { c.enqueue("service", obj) }, DeleteFunc: func(obj any) { c.enqueue("service", obj) }})
	configMaps.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{AddFunc: func(obj any) { c.enqueueAffectedPods(obj) }, UpdateFunc: func(_, obj any) { c.enqueueAffectedPods(obj) }, DeleteFunc: func(obj any) { c.enqueueAffectedPods(obj) }})
	endpointSlices.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{AddFunc: func(obj any) { c.enqueueEndpointService(obj) }, UpdateFunc: func(_, obj any) { c.enqueueEndpointService(obj) }, DeleteFunc: func(obj any) { c.enqueueEndpointService(obj) }})
	return c
}

// Ready closes after all Informer caches have synchronized.
func (c *Controller) Ready() <-chan struct{} { return c.ready }

func (c *Controller) Run(ctx context.Context, workers int) error {
	defer c.queue.ShutDown()
	if !cache.WaitForCacheSync(ctx.Done(), c.pods.Informer().HasSynced, c.services.Informer().HasSynced, c.configMaps.Informer().HasSynced, c.endpointSlices.Informer().HasSynced) {
		return fmt.Errorf("informer cache did not synchronize")
	}
	c.readyOnce.Do(func() { close(c.ready) })
	c.fullReconcile()
	for i := 0; i < workers; i++ {
		go func() {
			for c.processNext() {
			}
		}()
	}
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			c.fullReconcile()
		}
	}
}

func (c *Controller) processNext() bool {
	item, shutdown := c.queue.Get()
	if shutdown {
		return false
	}
	defer c.queue.Done(item)
	if err := c.reconcile(item); err != nil {
		c.log.Error("reconcile failed", "key", item, "error", err)
		c.queue.AddRateLimited(item)
	} else {
		c.queue.Forget(item)
	}
	return true
}

func (c *Controller) reconcile(key string) error {
	parts := strings.SplitN(key, ":", 2)
	if len(parts) != 2 {
		return nil
	}
	namespace, name, err := cache.SplitMetaNamespaceKey(parts[1])
	if err != nil {
		return err
	}
	switch parts[0] {
	case "pod":
		pod, err := c.pods.Lister().Pods(namespace).Get(name)
		if err != nil {
			return nil
		}
		if dependency.ModeFor(pod) == dependency.ModeDisabled {
			return nil
		}
		references, err := catalog.DefaultReferenceRegistry.Extract("Pod", pod)
		if err != nil {
			return err
		}
		for _, rule := range rules.DefaultRegistry.DirectReferenceRules() {
			for _, v := range rule.Validate(pod, references, c.resolver) {
				c.warn(pod, v)
			}
		}
	case "service":
		svc, err := c.services.Lister().Services(namespace).Get(name)
		if err != nil {
			return nil
		}
		if dependency.ModeFor(svc) == dependency.ModeDisabled {
			return nil
		}
		selector, err := serviceRef.NewSelectorExtractor().Extract(svc)
		if err != nil {
			return err
		}
		for _, rule := range rules.DefaultRegistry.ServiceConditionalRules() {
			found, err := rule.Validate(svc, selector, c.podLister)
			if err != nil {
				return err
			}
			for _, v := range found {
				c.warn(svc, v)
			}
		}
		slices, _ := c.endpointSlices.Lister().EndpointSlices(namespace).List(labels.Everything())
		if len(svc.Spec.Selector) > 0 && !hasReadyEndpoint(svc, slices) {
			c.warn(svc, dependency.Violation{"NoReadyEndpoint", namespace + "/" + name, "has no ready EndpointSlice endpoint"})
		}
	}
	return nil
}

func hasReadyEndpoint(service *corev1.Service, slices []*discoveryv1.EndpointSlice) bool {
	for _, slice := range slices {
		if slice.Labels[discoveryv1.LabelServiceName] == service.Name {
			for _, endpoint := range slice.Endpoints {
				if endpoint.Conditions.Ready != nil && *endpoint.Conditions.Ready {
					return true
				}
			}
		}
	}
	return false
}

func (c *Controller) warn(object runtime.Object, v dependency.Violation) {
	c.recorder.Eventf(object, corev1.EventTypeWarning, v.Rule, "%s", v.Message)
}

func (c *Controller) enqueue(kind string, obj any) {
	if key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj); err == nil {
		c.queue.Add(kind + ":" + key)
	}
}
func (c *Controller) enqueueAllServices() {
	services, _ := c.services.Lister().List(labels.Everything())
	for _, svc := range services {
		c.enqueue("service", svc)
	}
}
func (c *Controller) enqueueAffectedPods(obj any) {
	cm, ok := obj.(*corev1.ConfigMap)
	if !ok {
		return
	}
	pods, _ := c.pods.Lister().Pods(cm.Namespace).List(labels.Everything())
	for _, pod := range pods {
		references, err := catalog.DefaultReferenceRegistry.Extract("Pod", pod)
		if err != nil {
			continue
		}
		for _, rule := range rules.DefaultRegistry.DirectReferenceRules() {
			if rule.TargetKind() != "ConfigMap" {
				continue
			}
			for _, ref := range references {
				if ref.TargetKind == rule.TargetKind() && ref.Name == cm.Name {
					c.enqueue("pod", pod)
					break
				}
			}
		}
	}
}
func (c *Controller) enqueueEndpointService(obj any) {
	slice, ok := obj.(*discoveryv1.EndpointSlice)
	if !ok {
		return
	}
	name := slice.Labels[discoveryv1.LabelServiceName]
	if name != "" {
		c.queue.Add("service:" + slice.Namespace + "/" + name)
	}
}
func (c *Controller) fullReconcile() {
	pods, _ := c.pods.Lister().List(labels.Everything())
	for _, pod := range pods {
		c.enqueue("pod", pod)
	}
	c.enqueueAllServices()
}
