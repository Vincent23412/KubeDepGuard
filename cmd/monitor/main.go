package main

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/vincent/KubeDepGuard/internal/kube"
	"github.com/vincent/KubeDepGuard/internal/monitor"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/informers"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/record"
)

func main() {
	workers := flag.Int("workers", 2, "number of reconcile workers")
	healthAddr := flag.String("health-addr", ":8080", "HTTP health endpoint listen address")
	flag.Parse()
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	client, err := kube.InClusterClient()
	if err != nil {
		log.Error("create client", "error", err)
		os.Exit(1)
	}
	broadcaster := record.NewBroadcaster(record.WithContext(context.Background()))
	broadcaster.StartStructuredLogging(0)
	broadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: client.CoreV1().Events("")})
	recorder := broadcaster.NewRecorder(scheme(), corev1.EventSource{Component: "kube-dep-guard-monitor"})
	factory := informers.NewSharedInformerFactory(client, 0)
	controller := monitor.NewWithInformers(factory.Core().V1().Pods(), factory.Core().V1().Services(), factory.Core().V1().ConfigMaps(), factory.Discovery().V1().EndpointSlices(), recorder, log)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	factory.Start(ctx.Done())
	healthServer := newHealthServer(*healthAddr, controller.Ready())
	go func() {
		if err := healthServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("health server stopped", "error", err)
			stop()
		}
	}()
	go func() { <-ctx.Done(); _ = healthServer.Shutdown(context.Background()) }()
	if err := controller.Run(ctx, *workers); err != nil {
		log.Error("monitor stopped", "error", err)
		os.Exit(1)
	}
}

func scheme() *runtime.Scheme {
	s := runtime.NewScheme()
	_ = corev1.AddToScheme(s)
	return s
}

func newHealthServer(addr string, ready <-chan struct{}) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, _ *http.Request) {
		select {
		case <-ready:
			w.WriteHeader(http.StatusOK)
		default:
			http.Error(w, "informer cache is not synchronized", http.StatusServiceUnavailable)
		}
	})
	return &http.Server{Addr: addr, Handler: mux}
}
