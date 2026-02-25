#!/usr/bin/env bash
# scripts/test_versions.sh — Run tests across multiple versions in parallel
# Usage: ./scripts/test_versions.sh [versions_file]
#
# Expects:
#   - VPC layer already deployed (outputs in .task/vpc.outputs.json)
#   - versions file with one version per line (default: .task/versions.txt)
#   - PIPELINE_RUN_ID set in environment
#
# For each version: copies fixture → deploys EKS → runs tests → destroys EKS
# Each version gets its own working directory to avoid terraform state lock conflicts.

set -euo pipefail

VERSIONS_FILE="${1:-.task/versions.txt}"
FIXTURE_SRC="examples/eks"
WORKDIR=".task/versions"

if [[ ! -f "$VERSIONS_FILE" ]]; then
  echo "Error: $VERSIONS_FILE not found. Run: task discover-versions" >&2
  exit 1
fi

# Load only VPC outputs (not EKS — each version deploys its own EKS)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
if [[ -f .task/vpc.outputs.json ]]; then
  eval "$("${SCRIPT_DIR}/load_layer_outputs.sh" .task/vpc.outputs.json)"
fi
# Remove stale EKS outputs from previous runs
rm -f .task/eks.outputs.json

# Ensure required env vars are set (normally from Taskfile env block)
# If running outside of Taskfile, set defaults from common project convention
export PROJECT_NAME="${PROJECT_NAME:-$(grep 'PROJECT_NAME:' Taskfile.yml 2>/dev/null | head -1 | awk '{print $2}')}"
if [[ -z "$PROJECT_NAME" ]]; then
  echo "Error: PROJECT_NAME not set. Run via 'task test-run-versions' or export PROJECT_NAME." >&2
  exit 1
fi

VERSIONS=$(cat "$VERSIONS_FILE")
declare -a TEST_PIDS
declare -a TEST_VERSIONS
FAILED=()

# Clean previous version workdirs
rm -rf "$WORKDIR"
mkdir -p "$WORKDIR"

# Generate a shared RunID if not set
export PIPELINE_RUN_ID="${PIPELINE_RUN_ID:-local-$(date +%Y%m%d-%H%M%S)}"
# Short hash for unique resource names
RUN_HASH=$(echo -n "$PIPELINE_RUN_ID" | md5sum 2>/dev/null | cut -c1-4 || echo -n "$PIPELINE_RUN_ID" | md5 2>/dev/null | cut -c1-4 || echo "${PIPELINE_RUN_ID: -4}")

echo "=== Parallel version testing ==="
echo "Versions: $VERSIONS"
echo "RunID: $PIPELINE_RUN_ID"
echo "Project: $PROJECT_NAME"
echo ""

for VERSION in $VERSIONS; do
  VERSION_DIR="${WORKDIR}/eks-${VERSION//\./-}"

  # Copy only .tf files to isolated directory (clean state, no stale .terraform or tfstate)
  mkdir -p "$VERSION_DIR"
  cp "$FIXTURE_SRC"/*.tf "$VERSION_DIR/"

  # Fix relative module source path — now runs from .task/versions/eks-X-XX/ not examples/eks/
  ABSOLUTE_MODULE_PATH="$(cd "$(dirname "$0")/.." && pwd)/modules/eks-cluster"
  sed -i.bak "s|source.*=.*\"../../modules/eks-cluster\"|source = \"${ABSOLUTE_MODULE_PATH}\"|" "$VERSION_DIR/main.tf"
  rm -f "$VERSION_DIR/main.tf.bak"

  (
    export TF_VAR_cluster_version="$VERSION"
    VERSION_SLUG="${VERSION//\./-}"
    # Set both cluster_name AND pipeline_run_hash='' to prevent double-hashing
    # The run hash is already in the tags for identification
    export TF_VAR_cluster_name="test-eks-${VERSION_SLUG}-${RUN_HASH}"
    export TF_VAR_pipeline_run_hash=""

    # Deploy EKS in isolated directory
    echo "[$VERSION] Deploying EKS (cluster: $TF_VAR_cluster_name)..."
    ./scripts/terraform.sh apply "$VERSION_DIR"

    # Run tests
    echo "[$VERSION] Running tests..."
    cd test
    go test -v -timeout 35m -run TestEksClusterVersioned ./integration/... > ".eks-$VERSION.log" 2>&1
    EXIT_CODE=$?
    cd ..

    # Destroy EKS
    echo "[$VERSION] Destroying EKS..."
    ./scripts/terraform.sh destroy "$VERSION_DIR" || true

    exit $EXIT_CODE
  ) &
  TEST_PIDS+=($!)
  TEST_VERSIONS+=("$VERSION")
  echo "Started version $VERSION (PID: ${TEST_PIDS[-1]}) in $VERSION_DIR"
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

# Keep version workdirs on failure for 'task destroy-versions' cleanup
if [ ${#FAILED[@]} -eq 0 ]; then
  rm -rf "$WORKDIR"
fi

echo ""
echo "=== Results ==="
if [ ${#FAILED[@]} -eq 0 ]; then
  echo "All version tests passed"
  exit 0
else
  echo "Failed versions: ${FAILED[*]}"
  exit 1
fi
