package main

import (
	"crypto/tls"
	"flag"
	"log/slog"
	"net/http"
	"os"

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
	server := &http.Server{Addr: *addr, Handler: webhook.New(client, log), TLSConfig: &tls.Config{MinVersion: tls.VersionTLS12}}
	log.Info("starting admission webhook", "addr", *addr)
	if err := server.ListenAndServeTLS(*cert, *key); err != nil {
		log.Error("webhook stopped", "error", err)
		os.Exit(1)
	}
}
