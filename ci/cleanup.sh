#!/usr/bin/env bash
# ci/cleanup.sh — Cloud-nuke cleanup with subcommands
# Usage:
#   ci/cleanup.sh run     [--region <r>] [--project <p>] [--run-id <id>] [--force]
#   ci/cleanup.sh project [--region <r>] [--project <p>] [--force]
#
# The 'run' subcommand cleans resources from a specific run. It resolves
# --project and --run-id from (in order): flags, env vars (PROJECT_NAME,
# PIPELINE_RUN_ID), or .task/run-metadata.env.
# The 'project' subcommand cleans ALL resources for a project (any RunID).
#
# Dry-run by default. Add --force to actually delete resources.
# Reusable: reads .cloud-nuke-config.template.yml from repo root.
set -euo pipefail

SUBCOMMAND="${1:?Usage: ci/cleanup.sh <run|project> [options]}"
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

# Resolve project and run-id from flags → env vars → metadata file
resolve_run_metadata() {
  # Try env vars if flags not set
  [[ -z "$PROJECT" ]] && PROJECT="${PROJECT_NAME:-}"
  [[ -z "$RUN_ID" ]] && RUN_ID="${PIPELINE_RUN_ID:-}"

  # Fall back to metadata file
  if [[ -z "$PROJECT" || -z "$RUN_ID" ]] && [[ -f .task/run-metadata.env ]]; then
    # shellcheck disable=SC1091
    source .task/run-metadata.env
    [[ -z "$PROJECT" ]] && PROJECT="${PIPELINE_TAG:-}"
    [[ -z "$RUN_ID" ]] && RUN_ID="${PIPELINE_RUN_ID:-}"
  fi
}

case "$SUBCOMMAND" in
  run)
    [[ -z "$REGION" ]] && { echo "Error: --region required" >&2; exit 1; }

    resolve_run_metadata

    [[ -z "$PROJECT" ]] && { echo "Error: could not resolve project (use --project, PROJECT_NAME env, or .task/run-metadata.env)" >&2; exit 1; }
    [[ -z "$RUN_ID" ]] && { echo "Error: could not resolve run-id (use --run-id, PIPELINE_RUN_ID env, or .task/run-metadata.env)" >&2; exit 1; }

    echo "=== Cleanup run: Pipeline=${PROJECT}, RunID=${RUN_ID} ==="
    ensure_cloud_nuke
    generate_config "$PROJECT" "$RUN_ID"
    run_nuke

    if ! $FORCE; then
      echo ""
      echo "DRY RUN. To delete: task cleanup-run -- force"
    fi
    ;;

  project)
    [[ -z "$REGION" ]] && { echo "Error: --region required" >&2; exit 1; }
    [[ -z "$PROJECT" ]] && PROJECT="${PROJECT_NAME:-}"
    [[ -z "$PROJECT" ]] && { echo "Error: --project required" >&2; exit 1; }

    echo "=== Cleanup project: Pipeline=${PROJECT} (ALL runs) ==="
    ensure_cloud_nuke
    generate_config "$PROJECT" ""
    run_nuke

    if ! $FORCE; then
      echo ""
      echo "DRY RUN. To delete: task cleanup-project -- force"
    fi
    ;;

  *)
    echo "Unknown subcommand: $SUBCOMMAND" >&2
    echo "Usage: ci/cleanup.sh <run|project> [options]" >&2
    exit 1
    ;;
esac
