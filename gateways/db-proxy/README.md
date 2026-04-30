# DB Proxy Compose Install

This directory runs the database proxy gateway with the embedded tunnel agent. It is intended for database gateway hosts that connect outbound to an existing Arsenale server.

```bash
cp .env.example .env
arsenale --server https://arsenale.example.com gateway tunnel-token create <gateway-id> --bundle-dir .
docker compose --env-file .env --env-file tunnel.env up -d
```

The gateway container must have network access to the databases users will query through Arsenale.
