#!/usr/bin/env bash
set -euo pipefail

REF="${1:-main}"
DEST_DIR="$(cd "$(dirname "$0")" && pwd)"
URL="https://raw.githubusercontent.com/fleetdm/fleet/${REF}/schema/osquery_fleet_schema.json"

echo "Fetching schema at ref ${REF}..."
curl -fsSL "${URL}" -o "${DEST_DIR}/osquery_fleet_schema.json.tmp"
jq empty "${DEST_DIR}/osquery_fleet_schema.json.tmp"

SHA="$(curl -fsSL "https://api.github.com/repos/fleetdm/fleet/commits/${REF}" | jq -r .sha)"

mv "${DEST_DIR}/osquery_fleet_schema.json.tmp" "${DEST_DIR}/osquery_fleet_schema.json"
jq --arg sha "${SHA}" \
  --arg url "https://github.com/fleetdm/fleet/blob/${SHA}/schema/osquery_fleet_schema.json" \
  '.fleet_ref = $sha | .fleet_url = $url' \
  "${DEST_DIR}/sources.json" > "${DEST_DIR}/sources.json.tmp"
mv "${DEST_DIR}/sources.json.tmp" "${DEST_DIR}/sources.json"

echo "Updated to fleet@${SHA}"
