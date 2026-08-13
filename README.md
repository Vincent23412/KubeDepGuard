# KubeDepGuard

KubeDepGuard is a Kubernetes MVP for validating and observing two dependency types:

- Pod → ConfigMap direct references (`volumes`, `envFrom`, and `env.valueFrom`)
- Service → Pod selector dependencies, including Ready EndpointSlice availability

It runs two independently deployable components:

- **Admission Webhook** receives HTTPS `AdmissionReview` calls from the Kubernetes API Server and returns allow/deny.
- **Runtime Monitor** uses Informers and a rate-limited queue to report violations as Kubernetes Warning Events.

The shared code is intentionally small:

- `internal/dependency`: pure dependency parsing, matching, violations, and policy rules.
- `internal/kube`: in-cluster Kubernetes client construction.
- `internal/webhook` and `internal/monitor`: component-specific behaviour only.

The deployment layout follows the same boundary: `deploy/webhook`, `deploy/monitor`, and `deploy/shared`.

The Webhook exposes these HTTPS paths:

| Path | Purpose |
| --- | --- |
| `/validate/dependencies` | Kubernetes API Server AdmissionReview endpoint |
| `/healthz` | process health check |
| `/readyz` | readiness check after Informer cache synchronization |

The Monitor exposes `/healthz` and `/readyz` over its internal HTTP port `8080`.

## Direct-reference extension points

Direct dependency processing is split into three layers so Kubernetes field
traversal is not coupled to validation policy:

```text
typed resource → ReferenceExtractor → []Reference → DirectReferenceRule → TargetResolver
```

- `PodReferenceExtractor` owns paths such as `spec.volumes[].configMap` and
  environment-variable ConfigMap references.
- `Reference` retains the target kind, namespace, name, optional flag, key, and
  source `FieldPath` for precise diagnostics.
- `ConfigMapReferenceRule` only checks ConfigMap references; it does not inspect
  Pod fields.

To support a new source resource, register one extractor in
`DefaultExtractorRegistry`. To support a new target kind, add its direct rule
to `DefaultRegistry` and make the Webhook/Monitor resolver plus RBAC able to
read that target kind.

## Policy

Set `dependency.kubedepguard.io/mode` on a Pod or Service:

| Mode | Admission | Runtime Monitor |
| --- | --- | --- |
| `enforce` | reject a violation | emit Warning Event |
| `warn` (default) | allow and log | emit Warning Event |
| `disabled` | skip | skip |

For a ConfigMap deletion, an enforce Pod that references it blocks deletion. Deleting the final Pod selected by an enforce Service is also blocked.

## Local deployment with kind

Prerequisites: Go, Docker, kind, kubectl, and OpenSSL.

```sh
make test
make kind-create
make build
make kind-load
make deploy
```

`make deploy` creates a development CA and TLS Secret, waits for the webhook to be ready, then injects the CA bundle into `ValidatingWebhookConfiguration`. The generated private keys exist only in a temporary directory; the server certificate/key live in the Kubernetes Secret.

Try the rejecting path:

```sh
kubectl apply -f examples/failures/enforce-pod-missing-configmap.yaml
```

The request should be rejected because the referenced ConfigMap is absent. Change the annotation to `warn` to allow it, then inspect the runtime warning:

```sh
kubectl get events --sort-by=.lastTimestamp
```

Remove the installation with `make undeploy`.

More admission and runtime failure cases are in [examples/scenarios/README.md](examples/scenarios/README.md).

## Current MVP boundaries

- `failurePolicy: Ignore` is intentional: the API Server remains available when the webhook is unavailable, while the Runtime Monitor detects invalid observed state afterward.
- The Webhook and Monitor both use SharedInformer caches. The Webhook waits for its Pod, ConfigMap, and Service caches to synchronize before its HTTPS listener starts.
- The implementation excludes `kube-dep-guard-system` and `kube-system` from interception to avoid blocking core components or the webhook itself. The Webhook can only read its required resources; only the Monitor has Event write permission.
- Informer caches and queues are derived state only. There is no Pending State, leader election, multi-replica coordination, or auto-remediation yet.
