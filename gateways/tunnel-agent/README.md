# Tunnel Agent Compose Install

This directory runs only the standalone tunnel agent. Use it when the gateway service already exists outside this compose project.

```bash
cp .env.example .env
arsenale --server https://arsenale.example.com gateway tunnel-token create <gateway-id> --bundle-dir .
docker compose --env-file .env --env-file tunnel.env up -d
```

Set `TUNNEL_LOCAL_HOST` and `TUNNEL_LOCAL_PORT` to the service the agent should proxy. For a host service on Docker Desktop or modern Docker Engine, `host.docker.internal` is usually the correct host value.
