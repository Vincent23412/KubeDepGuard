package main

import (
	"context"
	"flag"
	"log/slog"
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
	if err := controller.Run(ctx, *workers); err != nil {
		log.Error("monitor stopped", "error", err)
		os.Exit(1)
	}
}

func scheme() *runtime.Scheme { s := runtime.NewScheme(); _ = corev1.AddToScheme(s); return s }
