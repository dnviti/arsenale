# RD Gateway Compose Install

This directory runs the native RD Gateway service as a direct inbound HTTPS service. It is the exception to the default outbound tunnel pattern because native RDP clients connect to this service directly.

```bash
cp .env.example .env
docker compose up -d
```

`RDGW_ARSENALE_API_URL` must point to the Arsenale control-plane API, and `RDGW_API_TOKEN` must be a valid token accepted by that API.
