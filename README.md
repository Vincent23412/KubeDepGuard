# KubeDepGuard

KubeDepGuard validates and observes the following dependency types:

- Deployment → ConfigMap direct references
- Deployment → Secret direct references
- Deployment → PersistentVolumeClaim direct references
- Ingress → Service direct references
- Service → Pod selector dependencies
- NetworkPolicy → Pod selector dependencies

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

- Deployment extractors own paths in `spec.template.spec`, including volume
  and environment-variable ConfigMap references.
- `Reference` retains the target kind, namespace, name, optional flag, key, and
  source `FieldPath` for precise diagnostics.
- Each direct-reference rule only checks its configured target kind.

To support a new source resource, register one extractor in
`DefaultExtractorRegistry`. To support a new target kind, add its direct rule
to `DefaultRegistry` and make the Webhook/Monitor resolver plus RBAC able to
read that target kind.

## Policy

Set `dependency.kubedepguard.io/mode` on a source resource:

| Mode | Admission | Runtime Monitor |
| --- | --- | --- |
| `enforce` | reject a violation | emit Warning Event |
| `warn` (default) | allow and log | emit Warning Event |
| `disabled` | skip | skip |

For a ConfigMap deletion, an enforce Deployment that references it blocks deletion. Deleting the final Pod selected by an enforce Service or NetworkPolicy is also blocked.

## Deployment to an existing cluster

Prerequisites: kubectl and OpenSSL. The cluster and application images are
provided by the environment that owns the cluster.

```sh
make test
make build
make deploy
```

`make deploy` creates a development CA and TLS Secret, waits for the webhook to be ready, then injects the CA bundle into `ValidatingWebhookConfiguration`. The generated private keys exist only in a temporary directory; the server certificate/key live in the Kubernetes Secret. For optional local kind helpers, see [tools/kind](tools/kind/README.md).

For an already-running kind cluster, `make kind-deploy` builds the local images,
imports them into the current kind cluster, and deploys KubeDepGuard. Use
`make kind-e2e` to deploy and run the integration suite in one command.

Run `make test-e2e` against an installed instance to exercise the admission
relationships. Its YAML fixtures are under [examples/failures](examples/failures/README.md)
and are applied only to a disposable namespace that the test removes.

Remove the installation with `make undeploy`.

## Current MVP boundaries

- `failurePolicy: Ignore` is intentional: the API Server remains available when the webhook is unavailable, while the Runtime Monitor detects invalid observed state afterward.
- The Webhook and Monitor both use SharedInformer caches and wait for all relationship resources to synchronize before processing requests.
- The implementation excludes `kube-dep-guard-system` and `kube-system` from interception to avoid blocking core components or the webhook itself. The Webhook can only read its required resources; only the Monitor has Event write permission.
- Informer caches and queues are derived state only. There is no Pending State, leader election, multi-replica coordination, or auto-remediation yet.
