package main

import (
	"context"
	"crypto/tls"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/vincent/KubeDepGuard/internal/kube"
	"github.com/vincent/KubeDepGuard/internal/webhook"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
)

func main() {
	addr := flag.String("addr", ":8443", "HTTPS listen address")
	cert := flag.String("tls-cert-file", "/tls/tls.crt", "TLS certificate")
	key := flag.String("tls-private-key-file", "/tls/tls.key", "TLS private key")
	flag.Parse()
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	client, err := kube.InClusterClient()
	if err != nil {
		log.Error("create client", "error", err)
		os.Exit(1)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	factory := informers.NewSharedInformerFactory(client, 0)
	podInformer := factory.Core().V1().Pods()
	configMapInformer := factory.Core().V1().ConfigMaps()
	serviceInformer := factory.Core().V1().Services()
	// Informers are created lazily. Materialize them before Start so the factory
	// has all three cache controllers to run.
	podCache := podInformer.Informer()
	configMapCache := configMapInformer.Informer()
	serviceCache := serviceInformer.Informer()
	factory.Start(ctx.Done())
	if !cache.WaitForCacheSync(ctx.Done(), podCache.HasSynced, configMapCache.HasSynced, serviceCache.HasSynced) {
		log.Error("informer cache did not synchronize")
		os.Exit(1)
	}
	mux := http.NewServeMux()
	mux.Handle(webhook.ValidationPath, webhook.New(podInformer.Lister(), configMapInformer.Lister(), serviceInformer.Lister(), log))
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	server := &http.Server{Addr: *addr, Handler: mux, TLSConfig: &tls.Config{MinVersion: tls.VersionTLS12}}
	log.Info("starting admission webhook", "addr", *addr)
	if err := server.ListenAndServeTLS(*cert, *key); err != nil {
		log.Error("webhook stopped", "error", err)
		os.Exit(1)
	}
}
