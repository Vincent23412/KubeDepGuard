IMAGE_TAG ?= dev
KIND_CLUSTER ?= kube-dep-guard

.PHONY: test test-e2e test-all setup build kind-create kind-load deploy undeploy

test:
	go test ./...

# test-e2e rebuilds and deploys the current images to the selected kind cluster.
# It removes its own temporary namespace even when an assertion fails.
test-e2e:
	KIND_CLUSTER=$(KIND_CLUSTER) IMAGE_TAG=$(IMAGE_TAG) ./scripts/test-kind.sh

test-all: test test-e2e

# setup prepares an existing kind cluster with the current local images.
setup:
	$(MAKE) build
	$(MAKE) kind-load
	$(MAKE) deploy

build:
	docker build -t kube-dep-guard-webhook:$(IMAGE_TAG) .
	docker build -t kube-dep-guard-monitor:$(IMAGE_TAG) -f Dockerfile.monitor .

kind-create:
	kind create cluster --name $(KIND_CLUSTER)

kind-load:
	kind load docker-image kube-dep-guard-webhook:$(IMAGE_TAG) --name $(KIND_CLUSTER)
	kind load docker-image kube-dep-guard-monitor:$(IMAGE_TAG) --name $(KIND_CLUSTER)

deploy:
	./scripts/deploy-kind.sh

undeploy:
	kubectl delete validatingwebhookconfiguration kube-dep-guard --ignore-not-found
	kubectl delete namespace kube-dep-guard-system --ignore-not-found
