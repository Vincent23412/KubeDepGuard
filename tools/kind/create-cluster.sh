#!/usr/bin/env bash
set -euo pipefail

cluster_name=${1:-kube-dep-guard}
kind create cluster --name "$cluster_name"
