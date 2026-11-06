#!/usr/bin/env bash
set -euo pipefail

test_namespace="kube-dep-guard-e2e-${RANDOM}${RANDOM}"
namespace_created=false

cleanup() {
	if [[ "$namespace_created" != true ]]; then
		return
	fi
	# Remove sources before their targets so enforce deletion rules cannot leave
	# the temporary namespace terminating.
	kubectl -n "$test_namespace" delete ingresses,networkpolicies --all --ignore-not-found --wait=true >/dev/null 2>&1 || true
	kubectl -n "$test_namespace" delete services --all --ignore-not-found --wait=true >/dev/null 2>&1 || true
	kubectl -n "$test_namespace" delete deployments,pods --all --ignore-not-found --wait=true >/dev/null 2>&1 || true
	kubectl -n "$test_namespace" delete configmaps,secrets,persistentvolumeclaims --all --ignore-not-found --wait=true >/dev/null 2>&1 || true
	kubectl delete namespace "$test_namespace" --ignore-not-found --wait=true --timeout=60s >/dev/null 2>&1 || true
}
trap cleanup EXIT

fail() {
	echo "test failed: $*" >&2
	exit 1
}

expect_rejected() {
	local description=$1
	shift
	if "$@" >/tmp/kube-dep-guard-e2e-command.out 2>&1; then
		cat /tmp/kube-dep-guard-e2e-command.out >&2
		fail "${description}: request was allowed"
	fi
	if ! grep -q "Forbidden\|denied\|references\|selector matches" /tmp/kube-dep-guard-e2e-command.out; then
		cat /tmp/kube-dep-guard-e2e-command.out >&2
		fail "${description}: request failed for an unexpected reason"
	fi
	echo "ok: ${description}"
}

if ! kubectl get namespace kube-dep-guard-system >/dev/null 2>&1; then
	fail "KubeDepGuard is not deployed in the current Kubernetes context; run make deploy first"
fi
kubectl -n kube-dep-guard-system rollout status deployment/kube-dep-guard-webhook --timeout=120s
kubectl -n kube-dep-guard-system rollout status deployment/kube-dep-guard-monitor --timeout=120s

kubectl create namespace "$test_namespace"
namespace_created=true
# The namespaceSelector is evaluated from the API server's namespace cache.
namespace_deadline=$((SECONDS + 30))
while (( SECONDS < namespace_deadline )); do
	if [[ "$(kubectl get namespace "$test_namespace" -o jsonpath='{.metadata.labels.kubernetes\.io/metadata\.name}' 2>/dev/null)" == "$test_namespace" ]]; then
		break
	fi
	sleep 1
done
if [[ "$(kubectl get namespace "$test_namespace" -o jsonpath='{.metadata.labels.kubernetes\.io/metadata\.name}' 2>/dev/null)" != "$test_namespace" ]]; then
	fail "namespace selector label did not become available"
fi
sleep 2

expect_rejected "enforce Deployment with missing ConfigMap" kubectl -n "$test_namespace" apply -f examples/failures/enforce-deployment-missing-configmap.yaml
expect_rejected "enforce Deployment with missing Secret" kubectl -n "$test_namespace" apply -f examples/failures/enforce-deployment-missing-secret.yaml
expect_rejected "enforce Deployment with missing PVC" kubectl -n "$test_namespace" apply -f examples/failures/enforce-deployment-missing-pvc.yaml
expect_rejected "enforce Service with an empty selector" kubectl -n "$test_namespace" apply -f examples/failures/enforce-service-empty-selector.yaml
expect_rejected "enforce NetworkPolicy with no matching Pod" kubectl -n "$test_namespace" apply -f examples/failures/enforce-networkpolicy-no-matching-pod.yaml
expect_rejected "enforce Ingress with missing Service" kubectl -n "$test_namespace" apply -f examples/failures/enforce-ingress-missing-service.yaml
expect_rejected "enforce Ingress with missing Service port" kubectl -n "$test_namespace" apply -f examples/failures/enforce-ingress-missing-service-port.yaml

kubectl -n "$test_namespace" apply -f examples/failures/enforce-configmap-delete-setup.yaml
expect_rejected "deleting ConfigMap referenced by enforce Deployment" kubectl -n "$test_namespace" delete configmap protected-config
kubectl -n "$test_namespace" apply -f examples/failures/enforce-secret-delete-setup.yaml
expect_rejected "deleting Secret referenced by enforce Deployment" kubectl -n "$test_namespace" delete secret protected-credentials
kubectl -n "$test_namespace" apply -f examples/failures/enforce-pvc-delete-setup.yaml
expect_rejected "deleting PVC referenced by enforce Deployment" kubectl -n "$test_namespace" delete persistentvolumeclaim protected-data
kubectl -n "$test_namespace" apply -f examples/failures/enforce-last-pod-delete-setup.yaml
expect_rejected "updating a Service selector to remove its final Pod" kubectl -n "$test_namespace" patch service enforce-service-target --type=merge -p '{"spec":{"selector":{"app":"does-not-exist"}}}'
expect_rejected "updating the final Service Pod label" kubectl -n "$test_namespace" label pod enforce-service-target app=other --overwrite
expect_rejected "deleting final Pod selected by enforce Service" kubectl -n "$test_namespace" delete pod enforce-service-target

kubectl -n "$test_namespace" apply -f examples/failures/enforce-networkpolicy-last-pod-delete-setup.yaml
expect_rejected "updating a NetworkPolicy selector to remove its final Pod" kubectl -n "$test_namespace" patch networkpolicy enforce-networkpolicy-target --type=merge -p '{"spec":{"podSelector":{"matchLabels":{"app":"does-not-exist"}}}}'
expect_rejected "updating the final NetworkPolicy Pod label" kubectl -n "$test_namespace" label pod enforce-networkpolicy-target app=other --overwrite
expect_rejected "deleting final Pod selected by enforce NetworkPolicy" kubectl -n "$test_namespace" delete pod enforce-networkpolicy-target

kubectl -n "$test_namespace" apply -f examples/failures/enforce-ingress-service-delete-setup.yaml
expect_rejected "removing an enforce Ingress backend port" kubectl -n "$test_namespace" patch service protected-ingress-backend --type=json -p '[{"op":"replace","path":"/spec/ports","value":[{"port":9090,"targetPort":8080}]}]'
expect_rejected "deleting Service referenced by enforce Ingress" kubectl -n "$test_namespace" delete service protected-ingress-backend

echo "all Kubernetes integration tests passed"
