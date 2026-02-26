#!/usr/bin/env bash
# scripts/deploy_layers.sh — Deploy, destroy, or plan all layers in sequence
# Usage: scripts/deploy_layers.sh <apply|destroy|plan> <layers> [prefix]
# Example: scripts/deploy_layers.sh apply "vpc eks" examples
# Example: scripts/deploy_layers.sh destroy "vpc eks" examples
#
# For 'apply': layers are processed left-to-right, PIPELINE_RUN_ID is auto-generated if not set.
# For 'destroy': layers are processed right-to-left (reverse order), errors are tolerated.
# For 'plan': layers are processed left-to-right.
set -euo pipefail

ACTION="${1:?Usage: scripts/deploy_layers.sh <apply|destroy|plan> <layers> [prefix]}"
LAYERS="${2:?Usage: scripts/deploy_layers.sh <apply|destroy|plan> <layers> [prefix]}"
PREFIX="${3:-examples}"

if [[ -z "$LAYERS" ]]; then
  echo "No layers configured, skipping ${ACTION}."
  exit 0
fi

# For apply, generate a shared RunID if not set
if [[ "$ACTION" == "apply" ]]; then
  export PIPELINE_RUN_ID="${PIPELINE_RUN_ID:-local-$(date +%Y%m%d-%H%M%S)}"
  echo "=== Deploy RunID: $PIPELINE_RUN_ID ==="
fi

# Build ordered layer list
if [[ "$ACTION" == "destroy" ]]; then
  # Reverse order for destroy
  ORDERED=$(echo "$LAYERS" | tr ' ' '\n' | awk '{a[NR]=$0} END{for(i=NR;i>=1;i--)print a[i]}')
else
  ORDERED="$LAYERS"
fi

for layer in $ORDERED; do
  echo "=== ${ACTION^} layer: $layer ==="
  if [[ "$ACTION" == "destroy" ]]; then
    ./scripts/terraform.sh "$ACTION" "${PREFIX}/${layer}" || true
  else
    ./scripts/terraform.sh "$ACTION" "${PREFIX}/${layer}"
  fi
done
