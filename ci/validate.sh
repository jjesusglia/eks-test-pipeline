#!/usr/bin/env bash
# ci/validate.sh — Run terraform init (+ optional validate) across multiple paths
# Usage: ci/validate.sh [--init-only] <space-separated-paths>
# Example: ci/validate.sh "modules/eks-cluster examples/vpc examples/eks"
# Example: ci/validate.sh --init-only "modules/eks-cluster examples/vpc"
#
# Reusable: no project-specific logic. Works for any Terraform project.
set -euo pipefail

INIT_ONLY=false
if [[ "${1:-}" == "--init-only" ]]; then
  INIT_ONLY=true
  shift
fi

PATHS="${1:?Usage: ci/validate.sh [--init-only] <paths>}"

for path in $PATHS; do
  if $INIT_ONLY; then
    echo "=== Initializing: $path ==="
    (cd "$path" && terraform init -backend=false > /dev/null 2>&1)
  else
    echo "=== Validating: $path ==="
    (cd "$path" && rm -rf .terraform && terraform init -backend=false > /dev/null 2>&1 && terraform validate)
  fi
done
