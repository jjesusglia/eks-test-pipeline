#!/usr/bin/env bash
# scripts/test_versions.sh — Run tests across multiple versions in parallel
# Usage: ./scripts/test_versions.sh [--versions-file .task/versions.txt]
#
# Expects:
#   - VPC layer already deployed (outputs in .task/vpc.outputs.json)
#   - .task/versions.txt with one version per line
#   - PIPELINE_RUN_ID set in environment
#
# For each version: deploys EKS layer → runs tests → destroys EKS layer

set -euo pipefail

VERSIONS_FILE="${1:-.task/versions.txt}"

if [[ ! -f "$VERSIONS_FILE" ]]; then
  echo "Error: $VERSIONS_FILE not found. Run: task discover-versions" >&2
  exit 1
fi

VERSIONS=$(cat "$VERSIONS_FILE")
declare -a TEST_PIDS
declare -a TEST_VERSIONS
FAILED=()

echo "=== Parallel version testing ==="
echo "Versions: $VERSIONS"
echo "RunID: ${PIPELINE_RUN_ID:-unset}"
echo ""

for VERSION in $VERSIONS; do
  (
    export TF_VAR_cluster_version="$VERSION"
    task deploy -- eks
    cd test
    go test -v -timeout 35m -run TestEksClusterVersioned ./integration/... > ".eks-$VERSION.log" 2>&1
    EXIT_CODE=$?
    cd ..
    task destroy -- eks || true
    exit $EXIT_CODE
  ) &
  TEST_PIDS+=($!)
  TEST_VERSIONS+=("$VERSION")
  echo "Started version $VERSION (PID: ${TEST_PIDS[-1]})"
done

echo ""
echo "Waiting for tests..."

for i in "${!TEST_PIDS[@]}"; do
  PID=${TEST_PIDS[$i]}
  VER=${TEST_VERSIONS[$i]}
  if wait $PID 2>/dev/null; then
    echo "PASS: version $VER"
  else
    echo "FAIL: version $VER (see test/.eks-$VER.log)"
    FAILED+=("$VER")
  fi
done

echo ""
echo "=== Results ==="
if [ ${#FAILED[@]} -eq 0 ]; then
  echo "All version tests passed"
  exit 0
else
  echo "Failed versions: ${FAILED[*]}"
  exit 1
fi
