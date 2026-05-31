#!/usr/bin/env bash
set -euo pipefail

REF="${1:-main}"
DEST_DIR="$(cd "$(dirname "$0")" && pwd)"
URL="https://raw.githubusercontent.com/fleetdm/fleet/${REF}/schema/osquery_fleet_schema.json"
TMP="$(mktemp "${DEST_DIR}/osquery_fleet_schema.json.XXXXXX")"
trap 'rm -f "${TMP}"' EXIT

echo "Fetching schema at ref ${REF}..."
curl -fsSL "${URL}" -o "${TMP}"
jq empty "${TMP}"

mv "${TMP}" "${DEST_DIR}/osquery_fleet_schema.json"
trap - EXIT

echo "Updated osquery_fleet_schema.json"
