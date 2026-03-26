#!/usr/bin/env bash
# Generates all development TLS certificates for Arsenale.
# Usage: ./generate-all-dev-certs.sh
#
# Creates certificates in:
#   infrastructure/gocache/certs/     — gocache gRPC mTLS (CA + server + client)
#   infrastructure/tunnel-certs/      — tunnel mTLS server
#   infrastructure/postgres-certs/    — PostgreSQL SSL
#   infrastructure/guacenc-certs/     — guacenc sidecar HTTPS
#   infrastructure/dev-server-certs/  — Express + guacamole-lite HTTPS

set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
DAYS=3650

generate_ca() {
  local dir="$1" cn="$2"
  openssl ecparam -genkey -name prime256v1 -out "$dir/ca-key.pem" 2>/dev/null
  openssl req -new -x509 -sha256 -key "$dir/ca-key.pem" -out "$dir/ca.pem" \
    -days "$DAYS" -subj "/CN=$cn/O=Arsenale" -batch 2>/dev/null
}

generate_server_cert() {
  local dir="$1" cn="$2" sans="$3"
  openssl ecparam -genkey -name prime256v1 -out "$dir/server-key.pem" 2>/dev/null
  openssl req -new -sha256 -key "$dir/server-key.pem" -out "$dir/server.csr" \
    -subj "/CN=$cn/O=Arsenale" -batch 2>/dev/null
  cat > "$dir/server-ext.cnf" <<EOF
subjectAltName = $sans
keyUsage = digitalSignature, keyEncipherment
extendedKeyUsage = serverAuth
EOF
  openssl x509 -req -sha256 -in "$dir/server.csr" -CA "$dir/ca.pem" -CAkey "$dir/ca-key.pem" \
    -CAcreateserial -out "$dir/server-cert.pem" -days "$DAYS" \
    -extfile "$dir/server-ext.cnf" 2>/dev/null
  rm -f "$dir"/*.csr "$dir"/*.cnf "$dir"/*.srl
}

generate_client_cert() {
  local dir="$1" cn="$2"
  openssl ecparam -genkey -name prime256v1 -out "$dir/client-key.pem" 2>/dev/null
  openssl req -new -sha256 -key "$dir/client-key.pem" -out "$dir/client.csr" \
    -subj "/CN=$cn/O=Arsenale" -batch 2>/dev/null
  cat > "$dir/client-ext.cnf" <<EOF
keyUsage = digitalSignature, keyEncipherment
extendedKeyUsage = clientAuth
EOF
  openssl x509 -req -sha256 -in "$dir/client.csr" -CA "$dir/ca.pem" -CAkey "$dir/ca-key.pem" \
    -CAcreateserial -out "$dir/client-cert.pem" -days "$DAYS" \
    -extfile "$dir/client-ext.cnf" 2>/dev/null
  rm -f "$dir"/*.csr "$dir"/*.cnf "$dir"/*.srl
}

# 1. gocache (reuse existing script if present)
echo "=== gocache mTLS ==="
GOCACHE_DIR="$SCRIPT_DIR/gocache/certs"
mkdir -p "$GOCACHE_DIR"
if [ -x "$GOCACHE_DIR/generate-dev-certs.sh" ]; then
  "$GOCACHE_DIR/generate-dev-certs.sh" "$GOCACHE_DIR"
else
  generate_ca "$GOCACHE_DIR" "gocache-dev-ca"
  generate_server_cert "$GOCACHE_DIR" "gocache" "DNS:gocache, DNS:localhost, IP:127.0.0.1, IP:::1"
  generate_client_cert "$GOCACHE_DIR" "arsenale-server"
fi
chmod 644 "$GOCACHE_DIR"/*-key.pem  # Rootless container UID 10001

# 2. tunnel
echo "=== Tunnel mTLS ==="
TUNNEL_DIR="$SCRIPT_DIR/tunnel-certs"
mkdir -p "$TUNNEL_DIR"
generate_ca "$TUNNEL_DIR" "tunnel-dev-ca"
generate_server_cert "$TUNNEL_DIR" "localhost" "DNS:localhost, IP:127.0.0.1, IP:::1"

# 3. PostgreSQL
echo "=== PostgreSQL SSL ==="
PG_DIR="$SCRIPT_DIR/postgres-certs"
mkdir -p "$PG_DIR"
generate_ca "$PG_DIR" "postgres-dev-ca"
generate_server_cert "$PG_DIR" "postgres" "DNS:postgres, DNS:localhost, IP:127.0.0.1"
chmod 600 "$PG_DIR/server-key.pem"  # PostgreSQL requires strict perms

# 4. guacenc
echo "=== Guacenc HTTPS ==="
GUACENC_DIR="$SCRIPT_DIR/guacenc-certs"
mkdir -p "$GUACENC_DIR"
generate_ca "$GUACENC_DIR" "guacenc-dev-ca"
generate_server_cert "$GUACENC_DIR" "guacenc" "DNS:guacenc, DNS:localhost, IP:127.0.0.1"

# 5. Dev server (Express + guacamole-lite)
echo "=== Dev Server HTTPS ==="
DEV_DIR="$SCRIPT_DIR/dev-server-certs"
mkdir -p "$DEV_DIR"
generate_ca "$DEV_DIR" "arsenale-dev-ca"
generate_server_cert "$DEV_DIR" "localhost" "DNS:localhost, IP:127.0.0.1, IP:::1"

echo ""
echo "=== All certificates generated ==="
echo "  gocache:    $GOCACHE_DIR"
echo "  tunnel:     $TUNNEL_DIR"
echo "  PostgreSQL: $PG_DIR"
echo "  guacenc:    $GUACENC_DIR"
echo "  dev-server: $DEV_DIR"
