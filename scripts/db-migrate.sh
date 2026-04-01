#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
COMPOSE_FILE="${ARSENALE_COMPOSE_FILE:-$PROJECT_ROOT/docker-compose.yml}"
POSTGRES_SERVICE="${ARSENALE_POSTGRES_SERVICE:-postgres}"
MIGRATE_SERVICE="${ARSENALE_MIGRATE_SERVICE:-migrate}"

detect_runtime() {
  if [[ -n "${CONTAINER_RUNTIME:-}" ]]; then
    printf '%s\n' "$CONTAINER_RUNTIME"
    return 0
  fi
  if command -v podman >/dev/null 2>&1; then
    printf 'podman\n'
    return 0
  fi
  if command -v docker >/dev/null 2>&1; then
    printf 'docker\n'
    return 0
  fi
  printf 'No supported container runtime found (podman/docker).\n' >&2
  return 1
}

COMMAND="${1:-up}"
shift || true

if [[ -n "${DATABASE_URL:-}" || -n "${DATABASE_URL_FILE:-}" ]]; then
  exec "$PROJECT_ROOT/scripts/run-go.sh" run ./cmd/migrate "$COMMAND" "$@"
fi

if [[ ! -f "$COMPOSE_FILE" ]]; then
  printf 'Compose file not found: %s\nRun make dev or set ARSENALE_COMPOSE_FILE.\n' "$COMPOSE_FILE" >&2
  exit 1
fi

RUNTIME="$(detect_runtime)"

"$RUNTIME" compose -f "$COMPOSE_FILE" up -d "$POSTGRES_SERVICE"
exec "$RUNTIME" compose -f "$COMPOSE_FILE" run --rm "$MIGRATE_SERVICE" "$COMMAND" "$@"
