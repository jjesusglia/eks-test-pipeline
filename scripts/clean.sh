#!/usr/bin/env bash
# scripts/clean.sh — Clean local Terraform state, caches, and temp files
# Usage: scripts/clean.sh [--all]
#   --all  Also removes ~/.tflint.d cache
#
# Example: scripts/clean.sh
# Example: scripts/clean.sh --all
set -euo pipefail

TEST_DIR="${TEST_DIR:-test}"

echo "Cleaning Terraform state and caches..."
find . -type d -name ".terraform" -exec rm -rf {} + 2>/dev/null || true
find . -type f -name ".terraform.lock.hcl" -delete 2>/dev/null || true
find . -type f -name "*.tfstate*" -delete 2>/dev/null || true
rm -rf .task

if [[ -d "$TEST_DIR" ]]; then
  (cd "$TEST_DIR" && go clean -cache -testcache 2>/dev/null || true)
fi

if [[ "${1:-}" == "--all" ]]; then
  echo "Deep cleaning TFLint cache..."
  rm -rf ~/.tflint.d 2>/dev/null || true
fi

echo "Clean complete."
