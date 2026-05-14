from __future__ import annotations

import unittest
from pathlib import Path


ROOT = Path(__file__).resolve().parents[3]


class ClientNginxConfigTest(unittest.TestCase):
    def test_map_assets_upstream_is_resolved_lazily(self) -> None:
        config_paths = [
            ROOT / "client" / "nginx.conf",
            ROOT / "client" / "nginx.dev.conf",
            ROOT / "deployment" / "ansible" / "roles" / "deploy" / "templates" / "client" / "nginx.https.conf.j2",
        ]

        for config_path in config_paths:
            with self.subTest(config=str(config_path.relative_to(ROOT))):
                config = config_path.read_text(encoding="utf-8")
                self.assertIn("set $map_assets_upstream http://${MAP_ASSETS_UPSTREAM_HOST}:8096;", config)
                self.assertIn("proxy_pass $map_assets_upstream;", config)
                self.assertNotIn("proxy_pass http://${MAP_ASSETS_UPSTREAM_HOST}:8096;", config)
                self.assertIn("location = /api/tunnel/connect", config)
                self.assertIn("set $tunnel_broker_upstream http://tunnel-broker:8092$request_uri;", config)
                self.assertIn("proxy_pass $tunnel_broker_upstream;", config)
