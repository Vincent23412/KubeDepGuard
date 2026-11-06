# kind development environment

This directory contains optional kind-only helpers. It is separate from
KubeDepGuard's application deployment and can be moved to the repository that
owns local cluster infrastructure.

```sh
./tools/kind/create-cluster.sh kube-dep-guard
IMAGE_TAG=dev ./tools/kind/load-kubedepguard-images.sh kube-dep-guard
```

For an existing cluster whose current context is `kind-kube-dep-guard`, use:

```sh
make kind-deploy
```

This builds the local images, imports them into the existing kind cluster, and
deploys KubeDepGuard. Use `make kind-e2e` to deploy and then run integration
tests.
