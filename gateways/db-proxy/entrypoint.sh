#!/bin/sh
set -e

if [ "$(id -u)" = "0" ]; then
  TUNNEL_RUNTIME_DIR="/tmp/arsenale-tunnel"
  mkdir -p "$TUNNEL_RUNTIME_DIR"
  chown dbproxy:dbproxy "$TUNNEL_RUNTIME_DIR"
  chmod 700 "$TUNNEL_RUNTIME_DIR"

  if [ -n "${TUNNEL_CLIENT_CERT_FILE:-}" ] && [ -f "$TUNNEL_CLIENT_CERT_FILE" ]; then
    cp "$TUNNEL_CLIENT_CERT_FILE" "$TUNNEL_RUNTIME_DIR/client-cert.pem"
    chown dbproxy:dbproxy "$TUNNEL_RUNTIME_DIR/client-cert.pem"
    chmod 644 "$TUNNEL_RUNTIME_DIR/client-cert.pem"
    export TUNNEL_CLIENT_CERT_FILE="$TUNNEL_RUNTIME_DIR/client-cert.pem"
  fi

  if [ -n "${TUNNEL_CLIENT_KEY_FILE:-}" ] && [ -f "$TUNNEL_CLIENT_KEY_FILE" ]; then
    cp "$TUNNEL_CLIENT_KEY_FILE" "$TUNNEL_RUNTIME_DIR/client-key.pem"
    chown dbproxy:dbproxy "$TUNNEL_RUNTIME_DIR/client-key.pem"
    chmod 600 "$TUNNEL_RUNTIME_DIR/client-key.pem"
    export TUNNEL_CLIENT_KEY_FILE="$TUNNEL_RUNTIME_DIR/client-key.pem"
  fi

  exec su-exec dbproxy "$0" "$@"
fi

export PORT="${DB_LISTEN_PORT:-5432}"

db_proxy_pid=""
tunnel_pid=""

shutdown() {
  if [ -n "$tunnel_pid" ] && kill -0 "$tunnel_pid" 2>/dev/null; then
    kill "$tunnel_pid" 2>/dev/null || true
  fi
  if [ -n "$db_proxy_pid" ] && kill -0 "$db_proxy_pid" 2>/dev/null; then
    kill "$db_proxy_pid" 2>/dev/null || true
  fi
  wait "$tunnel_pid" 2>/dev/null || true
  wait "$db_proxy_pid" 2>/dev/null || true
  exit 0
}

trap shutdown INT TERM

echo "[db-proxy] Starting middleware service on port ${PORT}..."
/usr/local/bin/db-proxy &
db_proxy_pid=$!

if [ -n "$TUNNEL_SERVER_URL" ] && [ -n "$TUNNEL_TOKEN" ]; then
  echo "[db-proxy] Starting tunnel agent..."
  /usr/local/bin/tunnel-agent &
  tunnel_pid=$!
fi

echo "[db-proxy] Database proxy gateway ready on port ${PORT}"

while :; do
  if ! kill -0 "$db_proxy_pid" 2>/dev/null; then
    wait "$db_proxy_pid" || true
    exit 1
  fi

  if [ -n "$tunnel_pid" ] && ! kill -0 "$tunnel_pid" 2>/dev/null; then
    wait "$tunnel_pid" || true
    exit 1
  fi

  sleep 1
done
