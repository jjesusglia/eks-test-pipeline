#!/usr/bin/env bash
# scripts/discover_versions.sh — Discover supported versions from AWS
# Usage: scripts/discover_versions.sh --addon <name> --region <r> --min <version> [--max-count <n>] [--output <file>]
# Example: scripts/discover_versions.sh --addon vpc-cni --region us-west-1 --min 1.31
#
# Project-specific: uses AWS EKS addon API to discover supported cluster versions.
set -euo pipefail

ADDON=""
REGION=""
MIN_VERSION=""
MAX_COUNT=4
OUTPUT_FILE=".task/versions.txt"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --addon)     ADDON="$2"; shift 2 ;;
    --region)    REGION="$2"; shift 2 ;;
    --min)       MIN_VERSION="$2"; shift 2 ;;
    --max-count) MAX_COUNT="$2"; shift 2 ;;
    --output)    OUTPUT_FILE="$2"; shift 2 ;;
    *)           echo "Unknown option: $1" >&2; exit 1 ;;
  esac
done

[[ -z "$ADDON" ]] && { echo "Error: --addon required" >&2; exit 1; }
[[ -z "$REGION" ]] && { echo "Error: --region required" >&2; exit 1; }
[[ -z "$MIN_VERSION" ]] && { echo "Error: --min required" >&2; exit 1; }

VERSIONS=$(aws eks describe-addon-versions --addon-name "$ADDON" \
  --region "$REGION" \
  --query 'addons[].addonVersions[].compatibilities[].clusterVersion' \
  --output text 2>/dev/null | tr '\t' '\n' | sort -V | uniq | \
  awk -v min="$MIN_VERSION" '$0 >= min' | head -"$MAX_COUNT") || \
VERSIONS="1.31 1.32"

mkdir -p "$(dirname "$OUTPUT_FILE")"
echo "$VERSIONS" > "$OUTPUT_FILE"
echo "Discovered versions: $(cat "$OUTPUT_FILE")"
