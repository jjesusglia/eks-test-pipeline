#!/usr/bin/env bash
# ci/cleanup.sh — Cloud-nuke cleanup with subcommands
# Usage:
#   ci/cleanup.sh fallback --region <r> --project <p> --run-id <id>
#   ci/cleanup.sh run      --region <r> [--force]
#   ci/cleanup.sh project  --region <r> --project <p> [--force]
#
# The 'fallback' subcommand is used by CI as a safety net (always --force).
# The 'run' subcommand cleans resources from the last run (uses .task/run-metadata.env).
# The 'project' subcommand cleans ALL resources for a project (any RunID).
#
# Reusable: reads .cloud-nuke-config.template.yml from repo root.
set -euo pipefail

SUBCOMMAND="${1:?Usage: ci/cleanup.sh <fallback|run|project> [options]}"
shift

# Defaults
REGION=""
PROJECT=""
RUN_ID=""
FORCE=false
NUKE_VERSION="${CLOUD_NUKE_VERSION:-v0.46.0}"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --region)  REGION="$2"; shift 2 ;;
    --project) PROJECT="$2"; shift 2 ;;
    --run-id)  RUN_ID="$2"; shift 2 ;;
    --force)   FORCE=true; shift ;;
    *)         echo "Unknown option: $1" >&2; exit 1 ;;
  esac
done

# Ensure cloud-nuke is available
ensure_cloud_nuke() {
  if ! command -v cloud-nuke &>/dev/null; then
    echo "Installing cloud-nuke ${NUKE_VERSION}..."
    curl -sL "https://github.com/gruntwork-io/cloud-nuke/releases/download/${NUKE_VERSION}/cloud-nuke_linux_amd64" -o /tmp/cloud-nuke
    chmod +x /tmp/cloud-nuke
    export PATH="/tmp:$PATH"
  fi
}

# Generate config from template with placeholder substitution
generate_config() {
  local pipeline="$1"
  local run_id="${2:-}"  # empty = delete RunID line (project-wide cleanup)

  mkdir -p .task

  if [[ -n "$run_id" ]]; then
    sed -e "s|PLACEHOLDER_PIPELINE|${pipeline}|g" \
        -e "s|PLACEHOLDER_RUN_ID|${run_id}|g" \
        .cloud-nuke-config.template.yml > .task/cloud-nuke-config.yml
  else
    sed -e "s|PLACEHOLDER_PIPELINE|${pipeline}|g" \
        -e "/RunID/d" \
        .cloud-nuke-config.template.yml > .task/cloud-nuke-config.yml
  fi
}

# Run cloud-nuke with generated config
run_nuke() {
  local force_flag=""
  if $FORCE; then
    force_flag="--force"
  else
    force_flag="--dry-run --force"
  fi

  # shellcheck disable=SC2086
  cloud-nuke aws --config .task/cloud-nuke-config.yml --region "$REGION" $force_flag
}

case "$SUBCOMMAND" in
  fallback)
    [[ -z "$REGION" ]] && { echo "Error: --region required" >&2; exit 1; }
    [[ -z "$PROJECT" ]] && { echo "Error: --project required" >&2; exit 1; }
    RUN_ID="${RUN_ID:-${PIPELINE_RUN_ID:-unknown}}"

    echo "=== Cleanup fallback: Pipeline=${PROJECT}, RunID=${RUN_ID} ==="
    ensure_cloud_nuke
    generate_config "$PROJECT" "$RUN_ID"
    FORCE=true run_nuke
    ;;

  run)
    [[ -z "$REGION" ]] && { echo "Error: --region required" >&2; exit 1; }

    if [[ -f .task/run-metadata.env ]]; then
      # shellcheck disable=SC1091
      source .task/run-metadata.env
    else
      echo "No run metadata found. Skipping."
      exit 0
    fi

    echo "=== Scope: Pipeline=${PIPELINE_TAG}, RunID=${PIPELINE_RUN_ID} ==="
    generate_config "$PIPELINE_TAG" "$PIPELINE_RUN_ID"
    run_nuke

    if ! $FORCE; then
      echo ""
      echo "DRY RUN. To delete: ci/cleanup.sh run --region $REGION --force"
    fi
    ;;

  project)
    [[ -z "$REGION" ]] && { echo "Error: --region required" >&2; exit 1; }
    [[ -z "$PROJECT" ]] && { echo "Error: --project required" >&2; exit 1; }

    echo "=== Scope: Pipeline=${PROJECT} (ALL runs) ==="
    generate_config "$PROJECT" ""
    run_nuke

    if ! $FORCE; then
      echo ""
      echo "DRY RUN. To delete: ci/cleanup.sh project --region $REGION --project $PROJECT --force"
    fi
    ;;

  *)
    echo "Unknown subcommand: $SUBCOMMAND" >&2
    echo "Usage: ci/cleanup.sh <fallback|run|project> [options]" >&2
    exit 1
    ;;
esac
