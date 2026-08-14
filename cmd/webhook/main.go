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

	"github.com/vincent/KubeDepGuard/internal/dependency/resolver"
	"github.com/vincent/KubeDepGuard/internal/dependency/rules"
	rulecatalog "github.com/vincent/KubeDepGuard/internal/dependency/rules/catalog"
	"github.com/vincent/KubeDepGuard/internal/kube"
	"github.com/vincent/KubeDepGuard/internal/webhook"
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
	informerResolver := resolver.NewInformerResolver(client)
	if err := informerResolver.Start(ctx); err != nil {
		log.Error("start dependency resolver", "error", err)
		os.Exit(1)
	}
	mux := http.NewServeMux()
	evaluator := rules.NewEvaluator(informerResolver, rulecatalog.DefaultRegistry.AdmissionRules())
	mux.Handle(webhook.ValidationPath, webhook.New(evaluator, log))
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	server := &http.Server{Addr: *addr, Handler: mux, TLSConfig: &tls.Config{MinVersion: tls.VersionTLS12}}
	log.Info("starting admission webhook", "addr", *addr)
	if err := server.ListenAndServeTLS(*cert, *key); err != nil {
		log.Error("webhook stopped", "error", err)
		os.Exit(1)
	}
}
