IMAGE_TAG ?= dev
KIND_CLUSTER ?= kube-dep-guard

.PHONY: test test-e2e test-all build deploy undeploy kind-deploy kind-e2e

test:
	go test ./...

# test-e2e validates an installed KubeDepGuard instance in the current context.
# The cluster and application images are supplied by the environment owner.
test-e2e:
	./scripts/test-e2e.sh

test-all: test test-e2e

build:
	docker build -t kube-dep-guard-webhook:$(IMAGE_TAG) .
	docker build -t kube-dep-guard-monitor:$(IMAGE_TAG) -f Dockerfile.monitor .

deploy:
	./scripts/deploy.sh

# kind-deploy targets an already-running kind cluster. It builds the local
# images, imports them into the cluster runtime, then deploys KubeDepGuard.
kind-deploy:
	@test "$$(kubectl config current-context)" = "kind-$(KIND_CLUSTER)" || (echo "current context must be kind-$(KIND_CLUSTER)" >&2; exit 1)
	$(MAKE) build
	IMAGE_TAG=$(IMAGE_TAG) ./tools/kind/load-kubedepguard-images.sh $(KIND_CLUSTER)
	$(MAKE) deploy

kind-e2e: kind-deploy
	$(MAKE) test-e2e

undeploy:
	kubectl delete validatingwebhookconfiguration kube-dep-guard --ignore-not-found
	kubectl delete namespace kube-dep-guard-system --ignore-not-found
