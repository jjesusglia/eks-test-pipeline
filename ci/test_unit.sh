#!/usr/bin/env bash
# ci/test_unit.sh — Run Go unit tests with optional coverage
# Usage: ci/test_unit.sh [--coverage] <test-dir>
# Example: ci/test_unit.sh test
# Example: ci/test_unit.sh --coverage test
#
# Reusable: no project-specific logic. Works for any Go test suite.
set -euo pipefail

COVERAGE=false
if [[ "${1:-}" == "--coverage" ]]; then
  COVERAGE=true
  shift
fi

TEST_DIR="${1:?Usage: ci/test_unit.sh [--coverage] <test-dir>}"

cd "$TEST_DIR"

if $COVERAGE; then
  go test -v -cover -coverprofile=coverage.out ./unit/...
  go tool cover -html=coverage.out -o coverage.html
  echo "Coverage report: ${TEST_DIR}/coverage.html"
else
  go test -v ./unit/...
fi
