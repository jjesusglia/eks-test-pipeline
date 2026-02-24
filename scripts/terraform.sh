#!/usr/bin/env bash
# scripts/terraform.sh — Dynamic terraform executor for any layer
# Usage: ./scripts/terraform.sh <action> <layer_dir> [extra_args...]
# Actions: init, plan, apply, destroy, output
#
# Pipeline tags are automatically injected via TF_VAR_pipeline_tags.
# Every resource gets tagged for identification and safe cleanup.
#
# Required env vars (set by Taskfile global env block):
#   PROJECT_NAME     — from Taskfile vars
#   PIPELINE_RUN_ID  — generated once by deploy-all, shared across layers

set -euo pipefail

ACTION="${1:?Usage: terraform.sh <action> <layer_dir>}"
LAYER_DIR="${2:?Usage: terraform.sh <action> <layer_dir>}"
shift 2

# ===== AUTOMATIC PIPELINE TAGGING =====
# Tags injected:
#   Pipeline    — PROJECT_NAME from Taskfile (identifies which project)
#   RunID       — Unique per run (CI: github run ID, local: timestamp)
#   Environment — "ci" or "local"

# Detect environment
if [[ -n "${GITHUB_RUN_ID:-}" ]]; then
  ENVIRONMENT="ci"
  RUN_ID="${GITHUB_RUN_ID}"
else
  ENVIRONMENT="local"
  # PIPELINE_RUN_ID is set by deploy-all to ensure all layers share the same ID.
  # If not set (e.g., running task deploy -- vpc directly), auto-generate one.
  if [[ -z "${PIPELINE_RUN_ID:-}" ]]; then
    PIPELINE_RUN_ID="local-$(date +%Y%m%d-%H%M%S)"
    echo "--- Auto-generated PIPELINE_RUN_ID: $PIPELINE_RUN_ID"
    echo "--- Tip: Use 'task deploy-all' to share a single RunID across all layers."
  fi
  RUN_ID="${PIPELINE_RUN_ID}"
fi

# Validate PROJECT_NAME to prevent JSON injection
if [[ -z "${PROJECT_NAME:-}" ]]; then
  echo "ERROR: PROJECT_NAME env var is not set. Check Taskfile env block." >&2
  exit 1
fi
if [[ "$PROJECT_NAME" =~ [\"\\] ]]; then
  echo "ERROR: PROJECT_NAME must not contain quotes or backslashes" >&2
  exit 1
fi

# Generate a short unique hash from RunID for resource naming (avoids conflicts)
RUN_HASH=$(echo -n "$RUN_ID" | md5sum 2>/dev/null | cut -c1-6 || echo -n "$RUN_ID" | md5 2>/dev/null | cut -c1-6 || echo "${RUN_ID: -6}")

# Export pipeline tags as TF_VAR (terraform auto-reads TF_VAR_* env vars)
export TF_VAR_pipeline_tags="{\"Pipeline\":\"${PROJECT_NAME}\",\"RunID\":\"${RUN_ID}\",\"Environment\":\"${ENVIRONMENT}\"}"
export TF_VAR_pipeline_run_hash="$RUN_HASH"

# Individual values for cloud-nuke and other scripts
export PIPELINE_TAG="${PROJECT_NAME}"
export PIPELINE_ENVIRONMENT="${ENVIRONMENT}"

echo "--- Pipeline Tags: Pipeline=${PROJECT_NAME}, RunID=${RUN_ID}, Environment=${ENVIRONMENT}"

# Source layer outputs from previous layers (if any exist in .task/*.outputs.json)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
for output_file in .task/*.outputs.json; do
  [[ -f "$output_file" ]] || continue
  echo "--- Loading outputs from: $output_file"
  eval "$("${SCRIPT_DIR}/load_layer_outputs.sh" "$output_file")"
done

case "$ACTION" in
  init)
    terraform -chdir="$LAYER_DIR" init -input=false "$@"
    ;;
  plan)
    terraform -chdir="$LAYER_DIR" plan -input=false "$@"
    ;;
  apply)
    terraform -chdir="$LAYER_DIR" init -input=false
    terraform -chdir="$LAYER_DIR" apply -input=false -auto-approve "$@"
    # Save outputs for downstream layers
    mkdir -p .task
    local_layer_name=$(basename "$LAYER_DIR")
    terraform -chdir="$LAYER_DIR" output -json > ".task/${local_layer_name}.outputs.json" 2>/dev/null || true
    # Save run metadata for cleanup scripts
    echo "PIPELINE_TAG=${PROJECT_NAME}" > .task/run-metadata.env
    echo "PIPELINE_RUN_ID=${RUN_ID}" >> .task/run-metadata.env
    echo "PIPELINE_ENVIRONMENT=${ENVIRONMENT}" >> .task/run-metadata.env
    ;;
  destroy)
    terraform -chdir="$LAYER_DIR" destroy -input=false -auto-approve "$@"
    ;;
  output)
    terraform -chdir="$LAYER_DIR" output -json "$@"
    ;;
  *)
    echo "Unknown action: $ACTION" >&2
    exit 1
    ;;
esac
