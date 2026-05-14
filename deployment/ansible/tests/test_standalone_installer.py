from __future__ import annotations

import os
import re
import unittest
from pathlib import Path

import yaml
from jinja2 import Environment, FileSystemLoader, StrictUndefined


ROOT = Path(__file__).resolve().parents[3]
COMPOSE_TEMPLATE = ROOT / "deployment" / "ansible" / "roles" / "deploy" / "templates" / "compose.yml.j2"
ENV_TEMPLATE = ROOT / "deployment" / "ansible" / "roles" / "deploy" / "templates" / "env.j2"
BROWSER_EXTENSION_WORKFLOW = ROOT / ".github" / "workflows" / "browser-extension.yml"
INSTALL_PLAYBOOK = ROOT / "deployment" / "ansible" / "playbooks" / "install.yml"
INSTALL_APPLY_TASKS = ROOT / "deployment" / "ansible" / "playbooks" / "tasks" / "install_apply.yml"
CERTIFICATE_TASKS = ROOT / "deployment" / "ansible" / "roles" / "certificates" / "tasks" / "main.yml"
PODMAN_SECRETS_TASKS = ROOT / "deployment" / "ansible" / "roles" / "podman_secrets" / "tasks" / "main.yml"
DEPLOY_APPLY_TASKS = ROOT / "deployment" / "ansible" / "roles" / "deploy" / "tasks" / "apply.yml"
DEPLOY_PLAYBOOK = ROOT / "deployment" / "ansible" / "playbooks" / "deploy.yml"
DEV_REFRESH_PLAYBOOK = ROOT / "deployment" / "ansible" / "playbooks" / "dev_refresh.yml"
DOCKER_BUILD_WORKFLOW = ROOT / ".github" / "workflows" / "docker-build.yml"
GATEWAYS_BUILD_WORKFLOW = ROOT / ".github" / "workflows" / "gateways-build.yml"
MAKEFILE = ROOT / "Makefile"


def _bool_filter(value: object) -> bool:
    if isinstance(value, bool):
        return value
    if value is None:
        return False
    if isinstance(value, (int, float)):
        return bool(value)
    return str(value).strip().lower() in {"1", "true", "yes", "on"}


def _regex_replace(value: object, pattern: str, replacement: str = "") -> str:
    return re.sub(pattern, replacement, str(value))


def _basename(value: object) -> str:
    return Path(str(value)).name


def _realpath(value: object) -> str:
    return os.path.realpath(str(value))


def _render_compose(**overrides: object) -> dict[str, object]:
    env = Environment(
        loader=FileSystemLoader(str(COMPOSE_TEMPLATE.parent)),
        undefined=StrictUndefined,
        keep_trailing_newline=True,
        trim_blocks=False,
        lstrip_blocks=False,
    )
    env.filters["bool"] = _bool_filter
    env.filters["regex_replace"] = _regex_replace
    env.filters["basename"] = _basename
    env.filters["realpath"] = _realpath

    image_tag = str(overrides.get("arsenale_image_tag", "stable"))
    registry = str(overrides.get("arsenale_registry", "ghcr.io/dnviti/arsenale"))
    component_images = {
        "migrate": f"{registry}/control-plane-api:{image_tag}",
        "control-plane-api": f"{registry}/control-plane-api:{image_tag}",
        "control-plane-controller": f"{registry}/control-plane-controller:{image_tag}",
        "authz-pdp": f"{registry}/authz-pdp:{image_tag}",
        "model-gateway": f"{registry}/model-gateway:{image_tag}",
        "tool-gateway": f"{registry}/tool-gateway:{image_tag}",
        "terminal-broker": f"{registry}/terminal-broker:{image_tag}",
        "desktop-broker": f"{registry}/desktop-broker:{image_tag}",
        "tunnel-broker": f"{registry}/tunnel-broker:{image_tag}",
        "query-runner": f"{registry}/query-runner:{image_tag}",
        "map-assets": f"{registry}/map-assets:{image_tag}",
        "memory-service": f"{registry}/memory-service:{image_tag}",
        "agent-orchestrator": f"{registry}/agent-orchestrator:{image_tag}",
        "runtime-agent": f"{registry}/runtime-agent:{image_tag}",
        "client": f"{registry}/client:{image_tag}",
        "guacd": f"{registry}/guacd:{image_tag}",
        "guacenc": f"{registry}/guacenc:{image_tag}",
        "ssh-gateway": f"{registry}/ssh-gateway:{image_tag}",
        "db-proxy": f"{registry}/db-proxy:{image_tag}",
    }

    context: dict[str, object] = {
        "arsenale_env": "production",
        "_home": "/opt/arsenale",
        "_is_dev": False,
        "_build": False,
        "_client_bind_host": "0.0.0.0",
        "_public_url": "https://arsenale.example.com",
        "installer_runtime_assets_dir": "/opt/arsenale/config/installer-assets",
        "arsenale_registry": registry,
        "arsenale_image_tag": image_tag,
        "arsenale_postgres_image": "quay.io/sclorg/postgresql-16-c10s",
        "arsenale_postgres_data_dir": "/var/lib/pgsql/data",
        "arsenale_db_user": "arsenale",
        "arsenale_db_name": "arsenale",
        "arsenale_domain": "arsenale.example.com",
        "arsenale_cert_dir": "/opt/arsenale/certs",
        "arsenale_component_images": component_images,
        "arsenale_recording_enabled": True,
        "arsenale_service_bind_host": "0.0.0.0",
        "arsenale_client_port": 3000,
        "arsenale_ssh_port": 2222,
        "arsenale_control_plane_api_port": 18080,
        "arsenale_control_plane_controller_port": 18081,
        "arsenale_authz_pdp_port": 18082,
        "arsenale_model_gateway_port": 18083,
        "arsenale_tool_gateway_port": 18084,
        "arsenale_agent_orchestrator_port": 18085,
        "arsenale_memory_service_port": 18086,
        "arsenale_terminal_broker_port": 18090,
        "arsenale_desktop_broker_port": 18091,
        "arsenale_tunnel_broker_port": 18092,
        "arsenale_query_runner_port": 18093,
        "arsenale_runtime_agent_port": 18095,
        "arsenale_map_assets_port": 18096,
        "arsenale_dev_bootstrap_admin_email": "admin@example.com",
        "arsenale_dev_bootstrap_admin_username": "admin",
        "arsenale_dev_bootstrap_admin_password": "ArsenaleTemp91Qx",
        "arsenale_dev_bootstrap_tenant_name": "Development Environment",
        "arsenale_uid": "1000",
        "arsenale_container_dns_servers": [],
        "dev_sample_postgres_host": "dev-demo-postgres",
        "dev_sample_postgres_port": 5432,
        "dev_sample_postgres_database": "arsenale_demo",
        "dev_sample_postgres_user": "demo_pg_user",
        "dev_sample_postgres_password": "DemoPgPass123!",
        "dev_sample_postgres_ssl_mode": "disable",
        "dev_sample_mysql_host": "dev-demo-mysql",
        "dev_sample_mysql_port": 3306,
        "dev_sample_mysql_database": "arsenale_demo",
        "dev_sample_mysql_user": "demo_mysql_user",
        "dev_sample_mysql_password": "DemoMySqlPass123!",
        "dev_sample_mysql_root_password": "DemoMySqlRoot123!",
        "dev_sample_mongodb_host": "dev-demo-mongodb",
        "dev_sample_mongodb_port": 27017,
        "dev_sample_mongodb_database": "arsenale_demo",
        "dev_sample_mongodb_root_user": "demo_mongo_root",
        "dev_sample_mongodb_root_password": "DemoMongoRoot123!",
        "dev_sample_mongodb_user": "demo_mongo_user",
        "dev_sample_mongodb_password": "DemoMongoPass123!",
        "dev_sample_oracle_host": "dev-demo-oracle",
        "dev_sample_oracle_port": 1521,
        "dev_sample_oracle_service_name": "FREEPDB1",
        "dev_sample_oracle_user": "demo_oracle_user",
        "dev_sample_oracle_password": "DemoOraclePass123!",
        "dev_sample_oracle_system_password": "DemoOracleSys123!",
        "dev_sample_mssql_host": "dev-demo-mssql",
        "dev_sample_mssql_port": 1433,
        "dev_sample_mssql_database": "ArsenaleDemo",
        "dev_sample_mssql_user": "demo_mssql_user",
        "dev_sample_mssql_password": "DemoMssqlPass123!",
        "dev_sample_mssql_sa_password": "DemoMssqlSa123!",
        "arsenale_resource_limits": {
            "postgres": {"cpus": "1.0", "memory": "1g", "pids": 256},
            "guacd": {"cpus": "1.0", "memory": "512m", "pids": 256},
            "guacenc": {"cpus": "1.0", "memory": "768m", "pids": 256},
            "client": {"cpus": "0.75", "memory": "256m", "pids": 256},
            "go_service": {"cpus": "0.5", "memory": "512m", "pids": 128},
            "ssh_gateway": {"cpus": "0.5", "memory": "256m", "pids": 128},
            "db_proxy": {"cpus": "0.5", "memory": "256m", "pids": 128},
        },
    }
    omit_keys = set(overrides.pop("_omit_keys", []))
    context.update(overrides)
    for key in omit_keys:
        context.pop(str(key), None)

    rendered = env.get_template(COMPOSE_TEMPLATE.name).render(**context)
    return yaml.safe_load(rendered)


def _render_env(**overrides: object) -> dict[str, str]:
    env = Environment(
        loader=FileSystemLoader(str(ENV_TEMPLATE.parent)),
        undefined=StrictUndefined,
        keep_trailing_newline=True,
        trim_blocks=False,
        lstrip_blocks=False,
    )
    env.filters["bool"] = _bool_filter

    context: dict[str, object] = {
        "arsenale_db_user": "arsenale",
        "arsenale_db_name": "arsenale",
        "arsenale_node_env": "production",
        "arsenale_recording_enabled": True,
        "arsenale_self_signup_enabled": False,
        "arsenale_file_threat_scanner_mode": "builtin",
        "arsenale_shared_files_s3_bucket": "",
        "arsenale_shared_files_s3_region": "us-east-1",
        "arsenale_shared_files_s3_endpoint": "",
        "arsenale_shared_files_s3_access_key_id": "",
        "arsenale_shared_files_s3_prefix": "",
        "arsenale_shared_files_s3_force_path_style": False,
        "arsenale_shared_files_s3_auto_create_bucket": False,
        "arsenale_dev_shared_files_s3_bucket": "arsenale-shared-files",
        "arsenale_dev_shared_files_s3_region": "us-east-1",
        "arsenale_dev_shared_files_s3_endpoint": "http://shared-files-s3:9000",
        "arsenale_dev_shared_files_s3_access_key_id": "arsenale",
        "arsenale_dev_shared_files_s3_secret_access_key": "arsenale-dev-shared-files",
        "arsenale_dev_shared_files_s3_prefix": "staged",
        "arsenale_dev_shared_files_s3_force_path_style": True,
        "arsenale_dev_shared_files_s3_auto_create_bucket": True,
        "arsenale_dev_bootstrap_admin_email": "admin@example.com",
        "arsenale_dev_bootstrap_admin_password": "ArsenaleTemp91Qx",
        "arsenale_dev_bootstrap_admin_username": "admin",
        "arsenale_dev_bootstrap_tenant_name": "Development Environment",
        "arsenale_cert_dir": "/opt/arsenale/certs",
        "arsenale_dev_tunnel_fixtures_enabled": False,
        "arsenale_dev_demo_databases_enabled": False,
        "installer_runtime_env": {},
        "installer_services": [],
        "_is_dev": False,
    }
    context.update(overrides)

    rendered = env.get_template(ENV_TEMPLATE.name).render(**context)
    return dict(
        line.split("=", 1)
        for line in rendered.splitlines()
        if line.strip() and not line.lstrip().startswith("#") and "=" in line
    )


class StandaloneInstallerTemplateTest(unittest.TestCase):
    def test_production_compose_uses_registry_images_and_installer_assets(self) -> None:
        compose = _render_compose()
        services = compose["services"]

        self.assertEqual(services["control-plane-api"]["image"], "ghcr.io/dnviti/arsenale/control-plane-api:stable")
        self.assertEqual(services["tunnel-broker"]["image"], "ghcr.io/dnviti/arsenale/tunnel-broker:stable")
        self.assertEqual(services["client"]["image"], "ghcr.io/dnviti/arsenale/client:stable")
        self.assertEqual(services["guacd"]["image"], "ghcr.io/dnviti/arsenale/guacd:stable")
        self.assertEqual(services["ssh-gateway"]["image"], "ghcr.io/dnviti/arsenale/ssh-gateway:stable")
        self.assertNotIn("build", services["control-plane-api"])
        self.assertNotIn("build", services["migrate"])
        self.assertEqual(services["control-plane-api"]["environment"]["SSH_PROXY_ENABLED"], "true")
        self.assertEqual(services["control-plane-api"]["environment"]["SSH_PROXY_PORT"], "2222")
        self.assertEqual(services["control-plane-api"]["environment"]["SSH_PROXY_PUBLIC_HOST"], "")
        self.assertEqual(services["control-plane-api"]["environment"]["SSH_PROXY_PUBLIC_PORT"], "")
        self.assertEqual(services["control-plane-api"]["environment"]["SSH_PROXY_TOKEN_TTL_SECONDS"], "300")
        self.assertEqual(services["control-plane-api"]["environment"]["SELF_SIGNUP_ENABLED"], "${SELF_SIGNUP_ENABLED:-false}")
        self.assertEqual(
            services["control-plane-api"]["environment"]["ARSENALE_DIRECT_ROUTING_ENABLED"],
            "${ARSENALE_DIRECT_ROUTING_ENABLED:-false}",
        )
        self.assertEqual(
            services["control-plane-api"]["environment"]["GATEWAY_ROUTING_MODE"],
            "${GATEWAY_ROUTING_MODE:-gateway-mandatory}",
        )
        self.assertIn("0.0.0.0:2222:2222", services["control-plane-api"]["ports"])
        self.assertNotIn("ports", services["ssh-gateway"])

        postgres_volumes = services["postgres"]["volumes"]
        self.assertIn(
            "/opt/arsenale/config/installer-assets/postgres/pg_hba.conf:/etc/postgresql/pg_hba.conf:ro",
            postgres_volumes,
        )
        self.assertIn(
            "/opt/arsenale/config/installer-assets/postgres/entrypoint.sh:/usr/local/bin/arsenale-postgres-entrypoint.sh:ro",
            postgres_volumes,
        )
        self.assertTrue(all("/opt/arsenale/arsenale/" not in entry for entry in postgres_volumes))
        self.assertEqual(
            services["client"]["volumes"],
            [
                "/opt/arsenale/certs/client:/certs:ro",
                "/opt/arsenale/config/installer-assets/client/nginx.https.conf:/etc/nginx/templates/default.conf.template:ro",
            ],
        )
        self.assertEqual(
            services["client"]["healthcheck"]["test"],
            ["CMD-SHELL", "curl --silent --show-error --fail --insecure https://localhost:8080/health >/dev/null || exit 1"],
        )
        self.assertIn("net-egress", services["guacd"]["networks"])
        self.assertIn("net-egress", services["ssh-gateway"]["networks"])
        self.assertIn("net-egress", services["query-runner"]["networks"])
        for service_name in [
            "client",
            "desktop-broker",
            "guacd",
            "guacenc",
            "migrate",
            "query-runner",
            "ssh-gateway",
            "terminal-broker",
            "tunnel-broker",
        ]:
            self.assertEqual(services[service_name]["group_add"], ["keep-groups"])

    def test_production_compose_renders_selected_extended_runtime_services(self) -> None:
        selected_services = [
            "postgres",
            "migrate",
            "redis",
            "control-plane-api",
            "authz-pdp",
            "client",
            "guacd",
            "desktop-broker",
            "terminal-broker",
            "ssh-gateway",
            "shared-files-s3",
            "map-assets",
            "guacenc",
            "query-runner",
            "model-gateway",
            "tool-gateway",
            "memory-service",
            "tunnel-broker",
            "control-plane-controller",
            "runtime-agent",
            "agent-orchestrator",
        ]
        compose = _render_compose(installer_services=selected_services)
        services = compose["services"]

        self.assertEqual(set(selected_services), set(services))
        self.assertEqual(services["client"]["depends_on"]["map-assets"]["condition"], "service_started")
        for service_name in [
            "agent-orchestrator",
            "control-plane-controller",
            "desktop-broker",
            "memory-service",
            "model-gateway",
            "query-runner",
            "terminal-broker",
            "tunnel-broker",
        ]:
            self.assertEqual(services[service_name]["group_add"], ["keep-groups"])

    def test_production_compose_honors_pinned_release_image_tag(self) -> None:
        compose = _render_compose(arsenale_image_tag="1.8.0")
        services = compose["services"]

        self.assertEqual(services["control-plane-api"]["image"], "ghcr.io/dnviti/arsenale/control-plane-api:1.8.0")
        self.assertEqual(services["tunnel-broker"]["image"], "ghcr.io/dnviti/arsenale/tunnel-broker:1.8.0")
        self.assertEqual(services["client"]["image"], "ghcr.io/dnviti/arsenale/client:1.8.0")
        self.assertEqual(services["guacd"]["image"], "ghcr.io/dnviti/arsenale/guacd:1.8.0")

    def test_production_podman_compose_dependencies_do_not_block_on_health_wait(self) -> None:
        compose = _render_compose(installer_services=["shared-files-s3"])
        services = compose["services"]

        self.assertEqual(services["control-plane-api"]["depends_on"]["postgres"]["condition"], "service_started")
        self.assertEqual(services["control-plane-api"]["depends_on"]["redis"]["condition"], "service_started")
        self.assertEqual(services["client"]["depends_on"]["control-plane-api"]["condition"], "service_started")
        self.assertNotIn("terminal-target", services)
        self.assertNotIn("terminal-target", services["terminal-broker"]["depends_on"])

    def test_production_http_public_url_keeps_plain_client_listener(self) -> None:
        compose = _render_compose(_public_url="http://arsenale.example.com:3000")
        client = compose["services"]["client"]

        self.assertNotIn("volumes", client)
        self.assertEqual(
            client["healthcheck"]["test"],
            ["CMD-SHELL", "curl --silent --show-error --fail http://127.0.0.1:8080/health >/dev/null || exit 1"],
        )

    def test_production_compose_can_render_bundled_shared_file_storage(self) -> None:
        compose = _render_compose(installer_services=["shared-files-s3"])
        services = compose["services"]
        env = services["shared-files-s3"]["environment"]

        self.assertIn("shared-files-s3", services)
        self.assertEqual(services["shared-files-s3"]["image"], "quay.io/minio/minio:latest")
        self.assertEqual(services["shared-files-s3"]["volumes"], ["shared_files_s3_data:/data"])
        self.assertEqual(services["shared-files-s3"]["secrets"], ["shared_files_s3_secret_access_key"])
        self.assertEqual(env["MINIO_ROOT_PASSWORD_FILE"], "/run/secrets/shared_files_s3_secret_access_key")
        self.assertNotIn("MINIO_ROOT_PASSWORD", env)
        self.assertEqual(
            services["control-plane-api"]["environment"]["ARSENALE_INSTALL_MODE"],
            "${ARSENALE_INSTALL_MODE:-production}",
        )
        self.assertNotIn("control-plane-controller", services)

    def test_production_compose_does_not_require_dev_demo_database_vars(self) -> None:
        dev_demo_keys = [
            "dev_sample_postgres_host",
            "dev_sample_postgres_port",
            "dev_sample_postgres_database",
            "dev_sample_postgres_user",
            "dev_sample_postgres_password",
            "dev_sample_postgres_ssl_mode",
            "dev_sample_mysql_host",
            "dev_sample_mysql_port",
            "dev_sample_mysql_database",
            "dev_sample_mysql_user",
            "dev_sample_mysql_password",
            "dev_sample_mongodb_host",
            "dev_sample_mongodb_port",
            "dev_sample_mongodb_database",
            "dev_sample_mongodb_user",
            "dev_sample_mongodb_password",
            "dev_sample_oracle_host",
            "dev_sample_oracle_port",
            "dev_sample_oracle_service_name",
            "dev_sample_oracle_user",
            "dev_sample_oracle_password",
            "dev_sample_mssql_host",
            "dev_sample_mssql_port",
            "dev_sample_mssql_database",
            "dev_sample_mssql_user",
            "dev_sample_mssql_password",
        ]

        compose = _render_compose(installer_services=["shared-files-s3"], _omit_keys=dev_demo_keys)
        env = compose["services"]["control-plane-api"]["environment"]

        self.assertNotIn("DEV_SAMPLE_POSTGRES_HOST", env)
        self.assertNotIn("dev-demo-postgres", compose["services"])

    def test_development_compose_keeps_local_builds(self) -> None:
        compose = _render_compose(
            arsenale_env="development",
            _home="/workspace/arsenale/deployment/ansible/playbooks/../../..",
            _is_dev=True,
            _build=True,
            arsenale_source_root="/workspace/arsenale",
            installer_runtime_assets_dir="/workspace/arsenale/config/installer-assets",
            arsenale_cert_dir="/workspace/arsenale/dev-certs",
        )
        services = compose["services"]

        self.assertEqual(services["control-plane-api"]["build"]["context"], "/workspace/arsenale")
        self.assertEqual(services["control-plane-api"]["image"], "localhost/arsenale_control-plane-api:latest")
        self.assertEqual(services["client"]["build"]["dockerfile"], "client/Dockerfile")
        self.assertEqual(services["client"]["image"], "localhost/arsenale_client:latest")
        self.assertEqual(
            services["control-plane-api"]["environment"]["ORCHESTRATOR_SSH_GATEWAY_IMAGE"],
            "localhost/arsenale_ssh-gateway:latest",
        )
        self.assertIn("multi_tenancy", services["control-plane-api"]["environment"]["ARSENALE_INSTALL_CAPABILITIES"])
        self.assertEqual(
            services["control-plane-api"]["environment"]["FEATURE_MULTI_TENANCY_ENABLED"],
            "${FEATURE_MULTI_TENANCY_ENABLED:-true}",
        )
        self.assertEqual(
            services["control-plane-api"]["environment"]["RECORDING_ENABLED"],
            "${FEATURE_RECORDINGS_ENABLED:-true}",
        )
        self.assertEqual(services["control-plane-api"]["environment"]["SSH_PROXY_ENABLED"], "true")
        self.assertEqual(services["control-plane-api"]["environment"]["SSH_PROXY_PORT"], "2222")
        self.assertEqual(services["control-plane-api"]["environment"]["SSH_PROXY_PUBLIC_HOST"], "")
        self.assertEqual(services["control-plane-api"]["environment"]["SSH_PROXY_PUBLIC_PORT"], "")
        self.assertEqual(services["control-plane-api"]["environment"]["SSH_PROXY_TOKEN_TTL_SECONDS"], "300")
        self.assertEqual(services["control-plane-api"]["environment"]["SELF_SIGNUP_ENABLED"], "${SELF_SIGNUP_ENABLED:-false}")
        self.assertEqual(services["control-plane-api"]["environment"]["SHARED_FILES_S3_BUCKET"], "${SHARED_FILES_S3_BUCKET:-}")
        self.assertEqual(services["control-plane-api"]["environment"]["SHARED_FILES_S3_ENDPOINT"], "${SHARED_FILES_S3_ENDPOINT:-}")
        self.assertIn("0.0.0.0:2222:2222", services["control-plane-api"]["ports"])
        self.assertIn("shared-files-s3", services)
        self.assertEqual(services["shared-files-s3"]["image"], "quay.io/minio/minio:latest")
        self.assertEqual(services["shared-files-s3"]["volumes"], ["shared_files_s3_data:/data"])
        self.assertEqual(services["shared-files-s3"]["secrets"], ["shared_files_s3_secret_access_key"])
        self.assertNotIn("MINIO_ROOT_PASSWORD", services["shared-files-s3"]["environment"])
        self.assertEqual(
            services["shared-files-s3"]["healthcheck"]["test"],
            ["CMD-SHELL", "curl --silent --fail http://127.0.0.1:9000/minio/health/live >/dev/null || exit 1"],
        )
        self.assertEqual(services["dev-demo-oracle"]["mem_limit"], "8g")
        self.assertEqual(services["dev-demo-oracle"]["shm_size"], "1g")
        self.assertEqual(
            services["postgres"]["volumes"][1],
            "/workspace/arsenale/config/installer-assets/postgres/pg_hba.conf:/etc/postgresql/pg_hba.conf:ro",
        )
        self.assertIn("net-egress", services["control-plane-api"]["networks"])
        self.assertIn("net-egress", services["guacd"]["networks"])
        self.assertIn("net-egress", services["ssh-gateway"]["networks"])
        self.assertIn("net-egress", services["query-runner"]["networks"])
        for service_name in [
            "agent-orchestrator",
            "client",
            "control-plane-controller",
            "desktop-broker",
            "guacd",
            "guacenc",
            "memory-service",
            "migrate",
            "model-gateway",
            "query-runner",
            "ssh-gateway",
            "terminal-broker",
            "tunnel-broker",
            "dev-tunnel-ssh-gateway",
            "dev-tunnel-guacd",
            "dev-tunnel-db-proxy",
        ]:
            self.assertEqual(services[service_name]["group_add"], ["keep-groups"])

    def test_development_compose_can_disable_dev_fixtures(self) -> None:
        compose = _render_compose(
            arsenale_env="development",
            _home="/workspace/arsenale/deployment/ansible/playbooks/../../..",
            _is_dev=True,
            _build=True,
            arsenale_source_root="/workspace/arsenale",
            installer_runtime_assets_dir="/workspace/arsenale/config/installer-assets",
            arsenale_cert_dir="/workspace/arsenale/dev-certs",
            arsenale_dev_fixture_targets_enabled=False,
            arsenale_dev_demo_databases_enabled=False,
            arsenale_dev_tunnel_fixtures_enabled=False,
        )
        services = compose["services"]

        self.assertNotIn("terminal-target", services)
        self.assertNotIn("dev-demo-postgres", services)
        self.assertNotIn("dev-tunnel-ssh-gateway", services)
        self.assertEqual(
            services["control-plane-api"]["environment"]["FEATURE_MULTI_TENANCY_ENABLED"],
            "${FEATURE_MULTI_TENANCY_ENABLED:-true}",
        )
        self.assertEqual(services["control-plane-api"]["environment"]["DEV_BOOTSTRAP_TUNNEL_FIXTURES_ENABLED"], "false")
        self.assertEqual(services["control-plane-api"]["environment"]["DEV_BOOTSTRAP_DEMO_DATABASES_ENABLED"], "false")

    def test_env_file_renders_self_signup_toggle(self) -> None:
        self.assertEqual(_render_env()["SELF_SIGNUP_ENABLED"], "false")
        self.assertEqual(_render_env(arsenale_self_signup_enabled=True)["SELF_SIGNUP_ENABLED"], "true")
        self.assertEqual(
            _render_env(installer_runtime_env={"SELF_SIGNUP_ENABLED": "true"})["SELF_SIGNUP_ENABLED"],
            "true",
        )

    def test_env_file_defaults_to_gateway_mandatory_routing(self) -> None:
        env = _render_env()
        self.assertEqual(env["ARSENALE_DIRECT_ROUTING_ENABLED"], "false")
        self.assertEqual(env["ARSENALE_ZERO_TRUST_ENABLED"], "true")
        self.assertEqual(env["GATEWAY_ROUTING_MODE"], "gateway-mandatory")

        direct_env = _render_env(
            installer_runtime_env={
                "ARSENALE_DIRECT_ROUTING_ENABLED": "true",
                "GATEWAY_ROUTING_MODE": "direct-allowed",
            }
        )
        self.assertEqual(direct_env["ARSENALE_DIRECT_ROUTING_ENABLED"], "true")
        self.assertEqual(direct_env["GATEWAY_ROUTING_MODE"], "direct-allowed")

    def test_env_file_does_not_render_shared_files_secret(self) -> None:
        self.assertNotIn("SHARED_FILES_S3_SECRET_ACCESS_KEY", _render_env(installer_services=["shared-files-s3"]))


class StandaloneInstallerConfigTest(unittest.TestCase):
    def test_non_dev_playbooks_default_to_prebuilt_images(self) -> None:
        install_text = INSTALL_PLAYBOOK.read_text(encoding="utf-8")
        deploy_text = DEPLOY_PLAYBOOK.read_text(encoding="utf-8")

        self.assertIn('_build: "{{ arsenale_build_images | default(false) }}"', install_text)
        self.assertIn('_build: "{{ true if _is_dev | bool else (arsenale_build_images | default(false)) }}"', deploy_text)

    def test_install_profile_maps_ip_geolocation_capability(self) -> None:
        install_text = INSTALL_APPLY_TASKS.read_text(encoding="utf-8")
        capabilities_section = install_text[
            install_text.index("'capabilities': {") : install_text.index("'routing': {")
        ]

        self.assertIn("'ip_geolocation': ('ip_geolocation' in installer_capability_selection)", capabilities_section)

    def test_runtime_service_private_keys_stay_group_scoped(self) -> None:
        cert_text = CERTIFICATE_TASKS.read_text(encoding="utf-8")

        self.assertIn('_runtime_key_mode: "0640"', cert_text)
        self.assertIn('_guacd_runtime_key_mode: "0640"', cert_text)
        self.assertIn("name: Ensure certificate root is private to the runtime owner", cert_text)
        self.assertNotIn('_runtime_key_mode: "0644"', cert_text)
        self.assertIn("_client_certificate_host_is_ipv4", cert_text)
        self.assertIn("regex_search('^\\\\d+\\\\.\\\\d+\\\\.\\\\d+\\\\.\\\\d+$')", cert_text)

    def test_bundled_production_shared_files_secret_requires_vault_value(self) -> None:
        secret_text = PODMAN_SECRETS_TASKS.read_text(encoding="utf-8")

        self.assertIn("name: Require shared-files S3 secret for bundled production storage", secret_text)
        self.assertIn("Set vault_shared_files_s3_secret_access_key", secret_text)
        self.assertIn("arsenale_dev_shared_files_s3_secret_access_key if ((_is_dev", secret_text)
        self.assertNotIn("or ('shared-files-s3' in (installer_services", secret_text)

    def test_full_force_recreate_refreshes_postgres_before_migrations(self) -> None:
        apply_text = DEPLOY_APPLY_TASKS.read_text(encoding="utf-8")

        self.assertLess(
            apply_text.index("name: Recreate PostgreSQL before schema bootstrap on full force-recreate applies"),
            apply_text.index("name: Run database migrations via Podman service runner"),
        )
        self.assertIn("{{ _compose_cmd }} up -d --force-recreate postgres", apply_text)
        self.assertIn("compose_force_recreate | default(false) | bool", apply_text)

    def test_full_force_recreate_removes_containers_before_plain_stack_up(self) -> None:
        apply_text = DEPLOY_APPLY_TASKS.read_text(encoding="utf-8")

        self.assertLess(
            apply_text.index("name: Remove existing compose containers before full force-recreate applies"),
            apply_text.index("name: Deploy Arsenale stack"),
        )
        deploy_section = apply_text[apply_text.index("name: Deploy Arsenale stack") :]
        deploy_section = deploy_section[: deploy_section.index("- name: Refresh selected Arsenale services")]
        self.assertIn("reversed(list(services.items()))", apply_text)
        self.assertIn("{{ _compose_cmd }} up -d --remove-orphans", deploy_section)
        self.assertNotIn("--force-recreate", deploy_section)

    def test_targeted_force_recreate_removes_selected_containers_before_refresh(self) -> None:
        apply_text = DEPLOY_APPLY_TASKS.read_text(encoding="utf-8")

        self.assertLess(
            apply_text.index("name: Remove selected compose containers before targeted force-recreate applies"),
            apply_text.index("name: Refresh selected Arsenale services"),
        )
        targeted_section = apply_text[
            apply_text.index("name: Remove selected compose containers before targeted force-recreate applies") :
            apply_text.index("name: Refresh selected Arsenale services")
        ]
        self.assertIn("targets = set(json.loads(sys.argv[1]))", targeted_section)
        self.assertIn("if service_name in targets:", targeted_section)
        self.assertIn("rm -f --depend", targeted_section)
        self.assertIn("compose_force_recreate | default(false) | bool", targeted_section)
        self.assertIn("(compose_target_services | default([]) | length) > 0", targeted_section)
        self.assertLess(
            apply_text.index("name: Restore compose stack after targeted force-recreate removal"),
            apply_text.index("name: Refresh selected Arsenale services"),
        )
        refresh_section = apply_text[apply_text.index("name: Refresh selected Arsenale services") :]
        self.assertIn("not (compose_force_recreate | default(false) | bool)", refresh_section)

    def test_install_playbook_failure_path_uses_one_summary_variable(self) -> None:
        install_text = INSTALL_APPLY_TASKS.read_text(encoding="utf-8")
        rescue_match = re.search(r"rescue:\n(?P<body>.*?)(?:\n  always:|\Z)", install_text, re.S)
        self.assertIsNotNone(rescue_match)
        rescue_block = rescue_match.group("body")

        self.assertIn(
            '{ src: "{{ playbook_dir }}/../scripts/install_failure.py", dest: "{{ installer_scripts_dir }}/install_failure.py" }',
            install_text,
        )
        self.assertIn("name: Summarize installer failure for operator", rescue_block)
        self.assertIn("name: Summarize installer failure for encrypted log", rescue_block)
        self.assertIn("python3", rescue_block)
        self.assertIn("{{ installer_scripts_dir }}/install_failure.py", rescue_block)
        self.assertIn("--detail", rescue_block)
        self.assertIn("changed_when: false", rescue_block)
        self.assertIn("failed_when: false", rescue_block)
        self.assertIn("no_log: true", rescue_block)
        self.assertIn("installer_failure_message", rescue_block)
        self.assertIn("installer_failure_log_message", rescue_block)
        self.assertEqual(rescue_block.count("installer_failure_message:"), 1)
        self.assertEqual(rescue_block.count("installer_failure_log_message:"), 1)
        message_section = rescue_block[
            rescue_block.index("installer_failure_message: >-") : rescue_block.index("installer_failure_log_message: >-")
        ]
        detail_section = rescue_block[rescue_block.index("installer_failure_log_message: >-") :]
        self.assertIn("ansible_failed_result.msg | default('installer apply failed')", message_section)
        self.assertNotIn("ansible_failed_result.stderr", message_section)
        self.assertIn(
            "ansible_failed_result.msg | default(ansible_failed_result.stderr | default('installer apply failed'))",
            detail_section,
        )
        self.assertIn('error: "{{ installer_failure_log_message }}"', rescue_block)
        self.assertIn('msg: "{{ installer_failure_message }}"', rescue_block)
        self.assertNotIn('error: "{{ installer_failure_message }}"', rescue_block)

        failure_section = rescue_block[rescue_block.index("name: Persist encrypted failure installer artifacts"):]
        failure_section = failure_section[: failure_section.index("- name: Surface installer failure")]
        failure_artifacts = failure_section.split("installer_write_artifacts:", 1)[1]
        self.assertIn("profile:", failure_artifacts)
        self.assertIn("status:", failure_artifacts)
        self.assertIn("log:", failure_artifacts)
        self.assertNotIn("state:", failure_artifacts)
        self.assertNotIn("rendered:", failure_artifacts)

    def test_dev_state_defaults_to_external_home_while_building_from_repo(self) -> None:
        install_text = INSTALL_PLAYBOOK.read_text(encoding="utf-8")
        deploy_text = DEPLOY_PLAYBOOK.read_text(encoding="utf-8")
        makefile_text = MAKEFILE.read_text(encoding="utf-8")

        self.assertIn("ARSENALE_DEV_HOME ?= $(ARSENALE_STATE_HOME)/arsenale-dev", makefile_text)
        self.assertIn("DEFAULT_INSTALL_PASSWORD_FILE := $(abspath $(ARSENALE_DEV_HOME)/install/password.txt)", makefile_text)
        self.assertIn("DEV_HOME_FLAG := -e arsenale_dev_home=$(ARSENALE_DEV_HOME)", makefile_text)

        self.assertIn("_dev_home: \"{{ arsenale_dev_home | default(", install_text)
        self.assertIn("_home: \"{{ _dev_home }}\"", install_text)
        self.assertIn("arsenale_source_root: \"{{ _repo_root }}\"", install_text)

        self.assertIn("_dev_home: \"{{ arsenale_dev_home | default(", deploy_text)
        self.assertIn("_home: \"{{ _dev_home if _is_dev | bool else (arsenale_home | default('/opt/arsenale')) }}\"", deploy_text)
        self.assertIn(
            "arsenale_source_root: \"{{ _repo_root if _is_dev | bool else (arsenale_home | default('/opt/arsenale')) + '/arsenale' }}\"",
            deploy_text,
        )

    def test_makefile_supports_service_scoped_dev_refresh(self) -> None:
        makefile_text = MAKEFILE.read_text(encoding="utf-8")

        self.assertIn("DEV_REFRESH_SELECTORS := client gateways control-plane", makefile_text)
        self.assertIn("playbooks/dev_refresh.yml", makefile_text)
        self.assertIn("make dev control-plane-api query-runner", makefile_text)

    def test_dev_refresh_playbook_reuses_saved_installer_artifacts(self) -> None:
        playbook_text = DEV_REFRESH_PLAYBOOK.read_text(encoding="utf-8")

        self.assertIn("name: install_artifacts", playbook_text)
        self.assertIn("resolve-dev-refresh", playbook_text)
        self.assertIn("compose_build_target_services", playbook_text)
        self.assertIn("compose_target_services", playbook_text)

    def test_ci_publishes_required_installer_images(self) -> None:
        browser_extension = yaml.safe_load(BROWSER_EXTENSION_WORKFLOW.read_text(encoding="utf-8"))
        browser_triggers = browser_extension.get("on", browser_extension.get(True))
        self.assertEqual(browser_triggers["push"]["branches"], ["develop", "main"])
        self.assertEqual(browser_triggers["pull_request"]["branches"], ["develop", "main"])

        docker_build = yaml.safe_load(DOCKER_BUILD_WORKFLOW.read_text(encoding="utf-8"))
        docker_steps = docker_build["jobs"]["build-and-scan"]["steps"]
        docker_meta = next(step for step in docker_steps if step.get("name") == "Extract metadata")
        docker_tags = docker_meta["with"]["tags"]
        docker_publish_profile = next(step for step in docker_steps if step.get("name") == "Determine publish profile")
        self.assertEqual(docker_publish_profile["id"], "publish_profile")
        self.assertIn('git branch -r --contains "${GITHUB_SHA}"', docker_publish_profile["run"])
        self.assertIn("origin/main", docker_publish_profile["run"])
        docker_publish = next(step for step in docker_steps if step.get("name") == "Push to registry")
        self.assertIn("steps.publish_profile.outputs.semver_allowed == 'true'", docker_publish["if"])
        self.assertNotIn("type=ref,event=branch", docker_tags)
        self.assertIn("type=ref,event=pr", docker_tags)
        self.assertIn(
            "type=raw,value=latest,enable=${{ github.event_name == 'push' && github.ref == 'refs/heads/develop' }}",
            docker_tags,
        )
        self.assertIn(
            "type=raw,value=stable,enable=${{ github.event_name == 'push' && github.ref == 'refs/heads/main' }}",
            docker_tags,
        )
        self.assertIn(
            "type=semver,pattern={{version}},enable=${{ github.event_name == 'push' && startsWith(github.ref, 'refs/tags/v') && steps.publish_profile.outputs.semver_allowed == 'true' }}",
            docker_tags,
        )
        self.assertIn(
            "type=semver,pattern={{major}}.{{minor}},enable=${{ github.event_name == 'push' && startsWith(github.ref, 'refs/tags/v') && steps.publish_profile.outputs.semver_allowed == 'true' }}",
            docker_tags,
        )

        docker_services = {
            entry["name"]
            for entry in docker_build["jobs"]["build-and-scan"]["strategy"]["matrix"]["service"]
        }
        self.assertTrue(
            {
                "control-plane-api",
                "control-plane-controller",
                "authz-pdp",
                "model-gateway",
                "tool-gateway",
                "agent-orchestrator",
                "memory-service",
                "terminal-broker",
                "desktop-broker",
                "tunnel-broker",
                "query-runner",
                "map-assets",
                "runtime-agent",
                "client",
            }.issubset(docker_services)
        )

        gateways_build = yaml.safe_load(GATEWAYS_BUILD_WORKFLOW.read_text(encoding="utf-8"))
        gateway_steps = gateways_build["jobs"]["build-and-scan"]["steps"]
        gateway_publish_profile = next(step for step in gateway_steps if step.get("name") == "Determine publish profile")
        self.assertEqual(gateway_publish_profile["id"], "publish_profile")
        self.assertIn('git branch -r --contains "${GITHUB_SHA}"', gateway_publish_profile["run"])
        self.assertIn("origin/main", gateway_publish_profile["run"])
        gateway_publish = next(step for step in gateway_steps if step.get("name") == "Push to registry")
        self.assertIn("steps.publish_profile.outputs.semver_allowed == 'true'", gateway_publish["if"])
        gateway_services = {
            entry["name"]
            for entry in gateways_build["jobs"]["build-and-scan"]["strategy"]["matrix"]["gateway"]
        }
        self.assertTrue({"guacd", "guacenc", "ssh-gateway", "db-proxy"}.issubset(gateway_services))


if __name__ == "__main__":
    unittest.main()
