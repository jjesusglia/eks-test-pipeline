#!/usr/bin/env bash
# ci/version_check.sh — Check required tool versions
# Usage: ci/version_check.sh
#
# Reusable: checks common Terraform project tools.
set -euo pipefail

echo "Checking tool versions..."

check_tool() {
  local name="$1"
  shift
  printf "  %-12s " "${name}:"
  local output
  if output=$("$@" 2>&1); then
    echo "$output" | head -1
  else
    echo "not installed"
  fi
}

check_tool "Terraform" terraform version
check_tool "Go" go version
check_tool "TFLint" tflint --version
check_tool "Trivy" trivy --version
check_tool "Task" task --version
check_tool "jq" jq --version
check_tool "pre-commit" pre-commit --version
