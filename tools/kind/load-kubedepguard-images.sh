#!/usr/bin/env bash
set -euo pipefail

cluster_name=${1:-kube-dep-guard}
image_tag=${IMAGE_TAG:-dev}
kind load docker-image "kube-dep-guard-webhook:${image_tag}" --name "$cluster_name"
kind load docker-image "kube-dep-guard-monitor:${image_tag}" --name "$cluster_name"
