# guacd Compose Install

This directory runs the Arsenale `guacd` image with the embedded tunnel agent. It is intended for remote RDP/VNC gateway hosts that should connect outbound to an existing Arsenale server.

```bash
cp .env.example .env
arsenale --server https://arsenale.example.com gateway tunnel-token create <gateway-id> --bundle-dir .
docker compose --env-file .env --env-file tunnel.env up -d
```

`GUACD_TLS_CERT_FILE` and `GUACD_TLS_KEY_FILE` must point to a certificate and key mounted into the container. The matching Arsenale desktop broker must trust the certificate chain.
