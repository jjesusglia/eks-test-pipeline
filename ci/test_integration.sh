#!/usr/bin/env bash
# ci/test_integration.sh — Run Go integration tests with configurable timeout and pattern
# Usage: ci/test_integration.sh [options] <test-dir>
#   -run <pattern>       Test name pattern (e.g., TestEksClusterComplete)
#   -timeout <duration>  Test timeout (default: 55m)
#   <test-dir>           Directory containing Go tests
#
# Example: ci/test_integration.sh -run TestEksClusterComplete -timeout 60m test
# Example: ci/test_integration.sh test
#
# Reusable: no project-specific logic. Works for any Go integration test suite.
set -euo pipefail

TIMEOUT="55m"
PATTERN=""
TEST_DIR=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    -run)     PATTERN="-run $2"; shift 2 ;;
    -timeout) TIMEOUT="$2"; shift 2 ;;
    *)        TEST_DIR="$1"; shift ;;
  esac
done

TEST_DIR="${TEST_DIR:?Usage: ci/test_integration.sh [options] <test-dir>}"

cd "$TEST_DIR"

echo "--- Running: go test ${PATTERN:+$PATTERN }-timeout $TIMEOUT ./integration/..."
# shellcheck disable=SC2086
go test -v $PATTERN -timeout "$TIMEOUT" ./integration/...
