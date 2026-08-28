package monitor

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/vincent/KubeDepGuard/internal/dependency"
	"github.com/vincent/KubeDepGuard/internal/dependency/resolver"
	"github.com/vincent/KubeDepGuard/internal/dependency/rules"
	rulecatalog "github.com/vincent/KubeDepGuard/internal/dependency/rules/catalog"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	appsinformers "k8s.io/client-go/informers/apps/v1"
	coreinformers "k8s.io/client-go/informers/core/v1"
	discoveryinformers "k8s.io/client-go/informers/discovery/v1"
	networkinginformers "k8s.io/client-go/informers/networking/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
)

type Controller struct {
	deployments    appsinformers.DeploymentInformer
	ingresses      networkinginformers.IngressInformer
	pods           coreinformers.PodInformer
	services       coreinformers.ServiceInformer
	configMaps     coreinformers.ConfigMapInformer
	endpointSlices discoveryinformers.EndpointSliceInformer
	queue          workqueue.TypedRateLimitingInterface[string]
	recorder       record.EventRecorder
	log            *slog.Logger
	evaluator      *rules.Evaluator
	ready          chan struct{}
	readyOnce      sync.Once
}

// New registers monitor event handlers on the resolver before it is started.
func New(informers *resolver.MonitorInformerResolver, recorder record.EventRecorder, log *slog.Logger) *Controller {
	pods := informers.Pods()
	services := informers.Services()
	configMaps := informers.ConfigMaps()
	secrets := informers.Secrets()
	pvcs := informers.PersistentVolumeClaims()
	endpointSlices := informers.EndpointSlices()
	deployments := informers.Deployments()
	ingresses := informers.Ingresses()
	serviceAccounts := informers.ServiceAccounts()
	nodes := informers.Nodes()
	pvs := informers.PersistentVolumes()
	c := &Controller{deployments: deployments, ingresses: ingresses, pods: pods, services: services, configMaps: configMaps, endpointSlices: endpointSlices, queue: workqueue.NewTypedRateLimitingQueue(workqueue.DefaultTypedControllerRateLimiter[string]()), recorder: recorder, log: log, evaluator: rules.NewEvaluator(informers, rulecatalog.DefaultRegistry.AdmissionRules()), ready: make(chan struct{})}
	deployments.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{AddFunc: func(obj any) { c.enqueue("deployment", obj) }, UpdateFunc: func(_, obj any) { c.enqueue("deployment", obj) }})
	ingresses.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{AddFunc: func(obj any) { c.enqueue("ingress", obj) }, UpdateFunc: func(_, obj any) { c.enqueue("ingress", obj) }})
	pods.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(obj any) { c.enqueue("pod", obj); c.enqueueAllServices() },
		UpdateFunc: func(_, obj any) { c.enqueue("pod", obj); c.enqueueAllServices() },
		DeleteFunc: func(any) { c.enqueueAllServices() },
	})
	services.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{AddFunc: func(obj any) { c.enqueue("service", obj) }, UpdateFunc: func(_, obj any) { c.enqueue("service", obj) }, DeleteFunc: func(obj any) { c.enqueue("service", obj) }})
	configMaps.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{AddFunc: func(any) { c.enqueueAllPods(); c.enqueueAllDeployments() }, UpdateFunc: func(_, _ any) { c.enqueueAllPods(); c.enqueueAllDeployments() }, DeleteFunc: func(any) { c.enqueueAllPods(); c.enqueueAllDeployments() }})
	secrets.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{AddFunc: func(any) { c.enqueueAllPods(); c.enqueueAllDeployments() }, UpdateFunc: func(_, _ any) { c.enqueueAllPods(); c.enqueueAllDeployments() }, DeleteFunc: func(any) { c.enqueueAllPods(); c.enqueueAllDeployments() }})
	pvcs.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{AddFunc: func(any) { c.enqueueAllPods(); c.enqueueAllDeployments() }, UpdateFunc: func(_, _ any) { c.enqueueAllPods(); c.enqueueAllDeployments() }, DeleteFunc: func(any) { c.enqueueAllPods(); c.enqueueAllDeployments() }})
	serviceAccounts.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{AddFunc: func(any) { c.enqueueAllPods(); c.enqueueAllDeployments() }, UpdateFunc: func(_, _ any) { c.enqueueAllPods(); c.enqueueAllDeployments() }, DeleteFunc: func(any) { c.enqueueAllPods(); c.enqueueAllDeployments() }})
	nodes.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{AddFunc: func(any) { c.enqueueAllPods() }, UpdateFunc: func(_, _ any) { c.enqueueAllPods() }, DeleteFunc: func(any) { c.enqueueAllPods() }})
	pvs.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{AddFunc: func(any) { c.enqueueAllPods(); c.enqueueAllDeployments() }, UpdateFunc: func(_, _ any) { c.enqueueAllPods(); c.enqueueAllDeployments() }, DeleteFunc: func(any) { c.enqueueAllPods(); c.enqueueAllDeployments() }})
	endpointSlices.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{AddFunc: func(obj any) { c.enqueueEndpointService(obj) }, UpdateFunc: func(_, obj any) { c.enqueueEndpointService(obj) }, DeleteFunc: func(obj any) { c.enqueueEndpointService(obj) }})
	return c
}

// Ready closes after all Informer caches have synchronized.
func (c *Controller) Ready() <-chan struct{} { return c.ready }

func (c *Controller) Run(ctx context.Context, workers int) error {
	defer c.queue.ShutDown()
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
		result, err := c.evaluator.Evaluate(rules.Request{Resource: "pods", Operation: rules.Update, Namespace: namespace, Name: name, Object: pod})
		if err != nil {
			return err
		}
		for _, violation := range result.Violations {
			c.warn(pod, violation)
		}
	case "deployment":
		deployment, err := c.deployments.Lister().Deployments(namespace).Get(name)
		if err != nil {
			return nil
		}
		result, err := c.evaluator.Evaluate(rules.Request{Resource: "deployments", Operation: rules.Update, Namespace: namespace, Name: name, Object: deployment})
		if err != nil {
			return err
		}
		for _, violation := range result.Violations {
			c.warn(deployment, violation)
		}
	case "ingress":
		ingress, err := c.ingresses.Lister().Ingresses(namespace).Get(name)
		if err != nil {
			return nil
		}
		result, err := c.evaluator.Evaluate(rules.Request{Resource: "ingresses", Operation: rules.Update, Namespace: namespace, Name: name, Object: ingress})
		if err != nil {
			return err
		}
		for _, violation := range result.Violations {
			c.warn(ingress, violation)
		}
	case "service":
		svc, err := c.services.Lister().Services(namespace).Get(name)
		if err != nil {
			return nil
		}
		result, err := c.evaluator.Evaluate(rules.Request{Resource: "services", Operation: rules.Update, Namespace: namespace, Name: name, Object: svc})
		if err != nil {
			return err
		}
		for _, violation := range result.Violations {
			c.warn(svc, violation)
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

func (c *Controller) enqueueAllDeployments() {
	deployments, _ := c.deployments.Lister().List(labels.Everything())
	for _, deployment := range deployments {
		c.enqueue("deployment", deployment)
	}
}

// enqueueReferencingPods rechecks Pods when a target is created or updated.
func (c *Controller) enqueueAllPods() {
	pods, _ := c.pods.Lister().List(labels.Everything())
	for _, pod := range pods {
		c.enqueue("pod", pod)
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
	c.enqueueAllPods()
	c.enqueueAllDeployments()
	c.enqueueAllServices()
}
