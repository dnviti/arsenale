#!/usr/bin/env bash
# Generate self-signed CA + server certificate for PostgreSQL SSL (development only).
# Usage: ./generate-dev-certs.sh [output-dir]
#
# Produces:
#   ca.pem / ca-key.pem              — Certificate Authority
#   server-cert.pem / server-key.pem — PostgreSQL server

set -euo pipefail

OUT="${1:-.}"
mkdir -p "$OUT"
DAYS=3650

echo "==> Generating PostgreSQL CA..."
openssl ecparam -genkey -name prime256v1 -out "$OUT/ca-key.pem" 2>/dev/null
openssl req -new -x509 -sha256 -key "$OUT/ca-key.pem" -out "$OUT/ca.pem" \
  -days "$DAYS" -subj "/CN=postgres-dev-ca/O=Arsenale" -batch 2>/dev/null

echo "==> Generating PostgreSQL server certificate..."
openssl ecparam -genkey -name prime256v1 -out "$OUT/server-key.pem" 2>/dev/null
openssl req -new -sha256 -key "$OUT/server-key.pem" -out "$OUT/server.csr" \
  -subj "/CN=postgres/O=Arsenale" -batch 2>/dev/null

cat > "$OUT/server-ext.cnf" <<EOF
subjectAltName = DNS:postgres, DNS:localhost, IP:127.0.0.1
keyUsage = digitalSignature, keyEncipherment
extendedKeyUsage = serverAuth
EOF

openssl x509 -req -sha256 -in "$OUT/server.csr" -CA "$OUT/ca.pem" -CAkey "$OUT/ca-key.pem" \
  -CAcreateserial -out "$OUT/server-cert.pem" -days "$DAYS" \
  -extfile "$OUT/server-ext.cnf" 2>/dev/null

rm -f "$OUT"/*.csr "$OUT"/*.cnf "$OUT"/*.srl
# PostgreSQL requires key to be readable only by owner (or container user)
chmod 600 "$OUT"/server-key.pem
chmod 644 "$OUT"/ca.pem "$OUT"/server-cert.pem

echo "==> Done. PostgreSQL certificates in: $OUT"
