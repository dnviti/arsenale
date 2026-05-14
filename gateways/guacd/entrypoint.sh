#!/bin/sh
set -e

if [ "$(id -u)" = "0" ]; then
  TUNNEL_RUNTIME_DIR="/tmp/arsenale-tunnel"
  TLS_RUNTIME_DIR="/tmp/guacd-tls"
  mkdir -p "$TUNNEL_RUNTIME_DIR" "$TLS_RUNTIME_DIR"
  chmod 700 "$TUNNEL_RUNTIME_DIR" "$TLS_RUNTIME_DIR"
  chown guacd:guacd "$TUNNEL_RUNTIME_DIR" "$TLS_RUNTIME_DIR"

  if [ -n "${TUNNEL_CLIENT_CERT_FILE:-}" ] && [ -f "$TUNNEL_CLIENT_CERT_FILE" ]; then
    cp "$TUNNEL_CLIENT_CERT_FILE" "$TUNNEL_RUNTIME_DIR/client-cert.pem"
    chmod 644 "$TUNNEL_RUNTIME_DIR/client-cert.pem"
    chown guacd:guacd "$TUNNEL_RUNTIME_DIR/client-cert.pem"
    export TUNNEL_CLIENT_CERT_FILE="$TUNNEL_RUNTIME_DIR/client-cert.pem"
  fi

  if [ -n "${TUNNEL_CLIENT_KEY_FILE:-}" ] && [ -f "$TUNNEL_CLIENT_KEY_FILE" ]; then
    cp "$TUNNEL_CLIENT_KEY_FILE" "$TUNNEL_RUNTIME_DIR/client-key.pem"
    chmod 600 "$TUNNEL_RUNTIME_DIR/client-key.pem"
    chown guacd:guacd "$TUNNEL_RUNTIME_DIR/client-key.pem"
    export TUNNEL_CLIENT_KEY_FILE="$TUNNEL_RUNTIME_DIR/client-key.pem"
  fi

  if [ -n "${GUACD_SSL_CERT:-}" ] && [ -f "$GUACD_SSL_CERT" ]; then
    cp "$GUACD_SSL_CERT" "$TLS_RUNTIME_DIR/server-cert.pem"
    chmod 644 "$TLS_RUNTIME_DIR/server-cert.pem"
    chown guacd:guacd "$TLS_RUNTIME_DIR/server-cert.pem"
    export GUACD_SSL_CERT="$TLS_RUNTIME_DIR/server-cert.pem"
  fi

  if [ -n "${GUACD_SSL_KEY:-}" ] && [ -f "$GUACD_SSL_KEY" ]; then
    cp "$GUACD_SSL_KEY" "$TLS_RUNTIME_DIR/server-key.pem"
    chmod 600 "$TLS_RUNTIME_DIR/server-key.pem"
    chown guacd:guacd "$TLS_RUNTIME_DIR/server-key.pem"
    export GUACD_SSL_KEY="$TLS_RUNTIME_DIR/server-key.pem"
  fi

  exec su-exec guacd "$0" "$@"
fi

requested_home="${HOME:-/home/guacd}"
requested_config_home="${XDG_CONFIG_HOME:-$requested_home/.config}"

if mkdir -p "$requested_home" "$requested_config_home" 2>/dev/null; then
  export HOME="$requested_home"
  export XDG_CONFIG_HOME="$requested_config_home"
else
  export HOME="/tmp/guacd-home"
  export XDG_CONFIG_HOME="$HOME/.config"
  mkdir -p "$HOME" "$XDG_CONFIG_HOME"
fi

# Start zero-trust tunnel agent if configured (auto-activating, dormant if env vars absent)
if [ -x /usr/local/bin/tunnel-agent ]; then
  echo "Starting tunnel agent (dormant if TUNNEL_SERVER_URL not set)..."
  /usr/local/bin/tunnel-agent &
fi

guacd_bin="$(command -v guacd || true)"
if [ -z "$guacd_bin" ] && [ -x /opt/guacamole/sbin/guacd ]; then
  guacd_bin="/opt/guacamole/sbin/guacd"
fi
if [ -z "$guacd_bin" ]; then
  echo "guacd binary not found" >&2
  exit 127
fi

guacd_port="${GUACD_PORT:-4822}"
case "$guacd_port" in
  ''|*[!0-9]*)
    echo "GUACD_PORT must be a valid port number (1-65535)" >&2
    exit 1
    ;;
esac
if [ "$guacd_port" -lt 1 ] || [ "$guacd_port" -gt 65535 ]; then
  echo "GUACD_PORT must be a valid port number (1-65535)" >&2
  exit 1
fi

set -- "$guacd_bin" -b 0.0.0.0 -l "$guacd_port" -f

if [ "${GUACD_SSL:-false}" = "true" ]; then
  tls_cert_path="${GUACD_SSL_CERT:-}"
  tls_key_path="${GUACD_SSL_KEY:-}"

  if [ -n "${GUACD_SSL_CERT_PEM:-}" ] || [ -n "${GUACD_SSL_KEY_PEM:-}" ]; then
    tls_dir="/tmp/guacd-tls"
    mkdir -p "$tls_dir"

    if [ -n "${GUACD_SSL_CERT_PEM:-}" ]; then
      tls_cert_path="$tls_dir/server-cert.pem"
      printf '%s\n' "$GUACD_SSL_CERT_PEM" > "$tls_cert_path"
      chmod 0644 "$tls_cert_path"
    fi

    if [ -n "${GUACD_SSL_KEY_PEM:-}" ]; then
      tls_key_path="$tls_dir/server-key.pem"
      printf '%s\n' "$GUACD_SSL_KEY_PEM" > "$tls_key_path"
      chmod 0600 "$tls_key_path"
    fi
  fi

  if [ -z "$tls_cert_path" ] || [ -z "$tls_key_path" ]; then
    echo "GUACD_SSL=true requires GUACD_SSL_CERT and GUACD_SSL_KEY (or *_PEM variants)" >&2
    exit 1
  fi

  echo "Starting guacd with TLS..."
  exec "$@" -C "$tls_cert_path" -K "$tls_key_path"
fi

echo "Starting guacd..."
exec "$@"
