# SSH Gateway Compose Install

This directory contains a standalone Docker Compose entrypoint for running the SSH gateway against an existing Arsenale server.

```bash
cp .env.example .env
arsenale --server https://arsenale.example.com gateway tunnel-token create <gateway-id> --bundle-dir .
docker compose --env-file .env --env-file tunnel.env up -d
```

Use the default `compose.yml` for a simple tunnel-backed SSH bastion. Set `SSH_AUTHORIZED_KEYS` to at least one public key, or SSH access stays closed.

For managed key push, add the override and provide the gRPC certificate files listed in `.env`:

```bash
docker compose --env-file .env --env-file tunnel.env -f compose.yml -f compose.managed-key.yml up -d
```
