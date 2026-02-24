#!/usr/bin/env bash
# scripts/load_layer_outputs.sh — Convert terraform output -json to TF_VAR_* exports
# Usage: eval "$(./scripts/load_layer_outputs.sh <outputs.json>)"
#
# Handles string, number, and bool outputs directly.
# Lists/maps are exported as JSON strings (terraform accepts JSON for complex types).

set -euo pipefail

OUTPUT_FILE="${1:?Usage: load_layer_outputs.sh <file.json>}"

if [[ ! -f "$OUTPUT_FILE" ]]; then
  echo "# No outputs file: $OUTPUT_FILE" >&2
  exit 0
fi

# Skip empty or invalid JSON files
if [[ ! -s "$OUTPUT_FILE" ]] || ! jq empty "$OUTPUT_FILE" 2>/dev/null; then
  echo "# Empty or invalid outputs file: $OUTPUT_FILE" >&2
  exit 0
fi

# Iterate terraform output keys and export as TF_VAR_*
jq -r '
  to_entries[] |
  "export TF_VAR_\(.key)=\(
    if (.value.value | type) == "string" then
      .value.value | @sh
    elif (.value.value | type) == "number" or (.value.value | type) == "boolean" then
      .value.value | tostring | @sh
    else
      .value.value | tojson | @sh
    end
  )"
' "$OUTPUT_FILE"
