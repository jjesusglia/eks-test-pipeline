#!/usr/bin/env bash
# ci/validate.sh — Run terraform init + validate across multiple paths
# Usage: ci/validate.sh <space-separated-paths>
# Example: ci/validate.sh "modules/eks-cluster examples/vpc examples/eks"
#
# Reusable: no project-specific logic. Works for any Terraform project.
set -euo pipefail

PATHS="${1:?Usage: ci/validate.sh <paths>}"

for path in $PATHS; do
  echo "=== Validating: $path ==="
  (cd "$path" && rm -rf .terraform && terraform init -backend=false > /dev/null 2>&1 && terraform validate)
done
