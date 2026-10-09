#!/usr/bin/env bash
set -euo pipefail

kind_cluster=${KIND_CLUSTER:-kube-dep-guard}
image_tag=${IMAGE_TAG:-dev}
expected_context="kind-${kind_cluster}"
test_namespace="kube-dep-guard-e2e-${RANDOM}${RANDOM}"
namespace_created=false

cleanup() {
	if [[ "$namespace_created" != true ]]; then
		return
	fi
	# Remove sources before their targets so enforce deletion rules cannot leave
	# the temporary namespace terminating.
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

wait_for_event() {
	local reason=$1
	local object_name=$2
	local deadline=$((SECONDS + 90))
	while (( SECONDS < deadline )); do
		if kubectl -n "$test_namespace" get events --field-selector "reason=${reason}" -o jsonpath='{range .items[*]}{.involvedObject.name}{"\n"}{end}' 2>/dev/null | grep -qx "$object_name"; then
			echo "ok: ${reason} event for ${object_name}"
			return
		fi
		sleep 2
	done
	fail "timed out waiting for ${reason} event for ${object_name}"
}

if [[ "$(kubectl config current-context)" != "$expected_context" ]]; then
	fail "current context must be ${expected_context}; found $(kubectl config current-context)"
fi

make IMAGE_TAG="$image_tag" KIND_CLUSTER="$kind_cluster" setup
kubectl -n kube-dep-guard-system rollout status deployment/kube-dep-guard-monitor --timeout=120s

kubectl create namespace "$test_namespace"
namespace_created=true
# The namespaceSelector is evaluated from the API server's namespace cache.
# Wait until the automatically assigned label is observable before sending the
# first admission request, otherwise the first request can be skipped while
# the newly created namespace is still propagating through that cache.
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
expect_rejected "enforce Pod with missing Secret" kubectl -n "$test_namespace" apply -f examples/failures/enforce-pod-missing-secret.yaml
expect_rejected "enforce Deployment with missing PVC" kubectl -n "$test_namespace" apply -f examples/failures/enforce-deployment-missing-pvc.yaml

expect_rejected "enforce Service with an empty selector" kubectl -n "$test_namespace" apply -f examples/failures/enforce-service-empty-selector.yaml
expect_rejected "enforce Ingress with missing Service" kubectl -n "$test_namespace" apply -f examples/failures/enforce-ingress-missing-service.yaml
expect_rejected "enforce Ingress with missing Service port" kubectl -n "$test_namespace" apply -f examples/failures/enforce-ingress-missing-service-port.yaml

kubectl -n "$test_namespace" apply -f examples/failures/enforce-configmap-delete-setup.yaml
expect_rejected "deleting ConfigMap referenced by enforce Deployment" kubectl -n "$test_namespace" delete configmap protected-config

kubectl -n "$test_namespace" apply -f examples/failures/enforce-secret-delete-setup.yaml
expect_rejected "deleting Secret referenced by enforce Pod" kubectl -n "$test_namespace" delete secret protected-credentials

kubectl -n "$test_namespace" apply -f examples/failures/enforce-pvc-delete-setup.yaml
expect_rejected "deleting PVC referenced by enforce Deployment" kubectl -n "$test_namespace" delete persistentvolumeclaim protected-data

kubectl -n "$test_namespace" apply -f examples/failures/enforce-last-pod-delete-setup.yaml
expect_rejected "deleting final Pod selected by enforce Service" kubectl -n "$test_namespace" delete pod enforce-service-target

echo "all kind integration tests passed"
