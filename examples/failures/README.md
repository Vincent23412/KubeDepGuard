# E2E fixtures

These manifests are test fixtures consumed only by `scripts/test-e2e.sh`.
Each E2E run creates a disposable namespace and removes it on completion, so
the resources in these manifests do not remain in the target cluster.
