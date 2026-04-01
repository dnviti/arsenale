#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
REPO_ROOT=$(cd -- "$SCRIPT_DIR/.." && pwd)
BACKEND_DIR="$REPO_ROOT/backend"

if command -v go >/dev/null 2>&1; then
  cd "$BACKEND_DIR"
  exec go "$@"
fi

if command -v podman >/dev/null 2>&1; then
  CONTAINER_RUNTIME=podman
elif command -v docker >/dev/null 2>&1; then
  CONTAINER_RUNTIME=docker
else
  echo "go is not installed and no container runtime is available" >&2
  exit 127
fi

GO_IMAGE=${GO_TOOLCHAIN_IMAGE:-docker.io/library/golang:1.25.8-alpine}
GO_CMD=(/usr/local/go/bin/go "$@")
printf -v GO_CMD_STRING '%q ' "${GO_CMD[@]}"

exec "$CONTAINER_RUNTIME" run --rm \
  -v "$BACKEND_DIR:/src" \
  -w /src \
  "$GO_IMAGE" \
  sh -lc "${GO_CMD_STRING% }"
