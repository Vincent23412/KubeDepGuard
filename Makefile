IMAGE_TAG ?= dev
KIND_CLUSTER ?= kube-dep-guard

.PHONY: test build kind-create kind-load deploy undeploy

test:
	go test ./...

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
