# guacenc Compose Install

This directory runs the recording conversion service as a direct HTTPS service. Point the Arsenale control plane at it with `GUACENC_SERVICE_URL`, `GUACENC_AUTH_TOKEN`, `GUACENC_USE_TLS=true`, and the matching CA configuration.

```bash
cp .env.example .env
docker compose up -d
```

`GUACENC_AUTH_TOKEN` must match the token configured in the Arsenale control plane. The recordings volume must contain the same recording paths that the control plane asks guacenc to convert.
