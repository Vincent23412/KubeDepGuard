#!/usr/bin/env bash
set -euo pipefail

namespace=kube-dep-guard-system
service=kube-dep-guard-webhook
tmp_dir=$(mktemp -d)
trap 'rm -rf "$tmp_dir"' EXIT

kubectl apply -f deploy/shared/namespace.yaml
kubectl apply -f deploy/webhook/rbac.yaml
kubectl apply -f deploy/monitor/rbac.yaml

cat >"$tmp_dir/openssl.cnf" <<EOF
[req]
distinguished_name = dn
req_extensions = req_ext
prompt = no
[dn]
CN = ${service}.${namespace}.svc
[req_ext]
subjectAltName = @alt_names
[alt_names]
DNS.1 = ${service}
DNS.2 = ${service}.${namespace}
DNS.3 = ${service}.${namespace}.svc
EOF

openssl genrsa -out "$tmp_dir/ca.key" 2048
openssl req -x509 -new -nodes -key "$tmp_dir/ca.key" -subj '/CN=kube-dep-guard-ca' -days 365 -out "$tmp_dir/ca.crt"
openssl genrsa -out "$tmp_dir/tls.key" 2048
openssl req -new -key "$tmp_dir/tls.key" -out "$tmp_dir/tls.csr" -config "$tmp_dir/openssl.cnf"
openssl x509 -req -in "$tmp_dir/tls.csr" -CA "$tmp_dir/ca.crt" -CAkey "$tmp_dir/ca.key" -CAcreateserial -days 365 -extensions req_ext -extfile "$tmp_dir/openssl.cnf" -out "$tmp_dir/tls.crt"

kubectl -n "$namespace" create secret tls kube-dep-guard-webhook-tls --cert="$tmp_dir/tls.crt" --key="$tmp_dir/tls.key" --dry-run=client -o yaml | kubectl apply -f -
kubectl apply -f deploy/webhook/service.yaml
kubectl apply -f deploy/webhook/deployment.yaml
kubectl apply -f deploy/monitor/deployment.yaml
# A TLS Secret update does not restart the Pod, and the Go server reads its
# certificate only at startup. Restart so the serving certificate matches the
# CA bundle patched into the ValidatingWebhookConfiguration below.
kubectl -n "$namespace" rollout restart deployment/kube-dep-guard-webhook
kubectl -n "$namespace" rollout status deployment/kube-dep-guard-webhook --timeout=120s
kubectl apply -f deploy/webhook/validating-webhook.yaml
ca_bundle=$(base64 <"$tmp_dir/ca.crt" | tr -d '\n')
kubectl patch validatingwebhookconfiguration kube-dep-guard --type=json -p="[{\"op\":\"replace\",\"path\":\"/webhooks/0/clientConfig/caBundle\",\"value\":\"${ca_bundle}\"}]"
kubectl -n "$namespace" rollout restart deployment/kube-dep-guard-monitor
kubectl -n "$namespace" rollout status deployment/kube-dep-guard-monitor --timeout=120s
