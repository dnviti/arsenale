#!/usr/bin/env bash
# update-geodb.sh — Download or update the MaxMind GeoLite2-City database.
#
# Usage:
#   ./scripts/update-geodb.sh [LICENSE_KEY] [OUTPUT_DIR]
#
# Arguments:
#   LICENSE_KEY  Your MaxMind licence key (free account required).
#                Falls back to the MAXMIND_LICENSE_KEY env var.
#   OUTPUT_DIR   Directory to store the .mmdb file (default: ./data/geoip).
#
# After downloading, set GEOIP_DB_PATH in .env to the resulting .mmdb path.
# Re-run periodically (e.g. weekly cron) to keep the database current.

set -euo pipefail

LICENSE_KEY="${1:-${MAXMIND_LICENSE_KEY:-}}"
OUTPUT_DIR="${2:-./data/geoip}"

if [ -z "$LICENSE_KEY" ]; then
  echo "Error: MaxMind licence key required."
  echo "  Sign up for a free GeoLite2 account at:"
  echo "    https://dev.maxmind.com/geoip/geolite2-free-geolocation-data"
  echo ""
  echo "Usage: $0 <LICENSE_KEY> [OUTPUT_DIR]"
  echo "  or set MAXMIND_LICENSE_KEY env var"
  exit 1
fi

EDITION="GeoLite2-City"
URL="https://download.maxmind.com/app/geoip_download?edition_id=${EDITION}&license_key=${LICENSE_KEY}&suffix=tar.gz"

mkdir -p "$OUTPUT_DIR"
TMPFILE=$(mktemp)
trap 'rm -f "$TMPFILE"' EXIT

echo "Downloading ${EDITION} database..."
curl -fsSL "$URL" -o "$TMPFILE"

echo "Extracting..."
tar -xzf "$TMPFILE" -C "$OUTPUT_DIR" --strip-components=1 --wildcards '*/*.mmdb'

MMDB_PATH=$(find "$OUTPUT_DIR" -name '*.mmdb' -print -quit)
echo ""
echo "Done! Database saved to: ${MMDB_PATH}"
echo ""
echo "Add to your .env:"
echo "  GEOIP_DB_PATH=${MMDB_PATH}"
