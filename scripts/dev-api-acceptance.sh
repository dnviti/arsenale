#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
vault_file="${ARSENALE_VAULT_FILE:-$repo_root/deployment/ansible/inventory/group_vars/all/vault.yml}"

resolve_postgres_password() {
  if [[ -n "${ARSENALE_DB_PASSWORD:-}" ]]; then
    printf '%s' "${ARSENALE_DB_PASSWORD}"
    return
  fi

  python3 - "$vault_file" <<'PY'
import re
import sys
from pathlib import Path

vault_path = Path(sys.argv[1])
text = vault_path.read_text()
match = re.search(r'^vault_postgres_password: "([^"]+)"$', text, re.M)
if not match:
    raise SystemExit("could not read vault_postgres_password from " + str(vault_path))
print(match.group(1))
PY
}

postgres_password="$(resolve_postgres_password)"
ca_cert="${ARSENALE_CA_CERT:-$repo_root/dev-certs/client/ca.pem}"
api_base="${ARSENALE_API_BASE:-https://localhost:3000/api}"
cp_base="${ARSENALE_CP_BASE:-http://127.0.0.1:18080}"
controller_base="${ARSENALE_CONTROLLER_BASE:-http://127.0.0.1:18081}"
authz_base="${ARSENALE_AUTHZ_BASE:-http://127.0.0.1:18082}"
model_base="${ARSENALE_MODEL_BASE:-http://127.0.0.1:18083}"
tool_base="${ARSENALE_TOOL_BASE:-http://127.0.0.1:18084}"
agent_base="${ARSENALE_AGENT_BASE:-http://127.0.0.1:18085}"
memory_base="${ARSENALE_MEMORY_BASE:-http://127.0.0.1:18086}"
query_base="${ARSENALE_QUERY_BASE:-http://127.0.0.1:18093}"
desktop_base="${ARSENALE_DESKTOP_BASE:-http://127.0.0.1:18091}"
terminal_base="${ARSENALE_TERMINAL_BASE:-http://127.0.0.1:18090}"
runtime_base="${ARSENALE_RUNTIME_BASE:-http://127.0.0.1:18095}"
admin_email="${ARSENALE_ADMIN_EMAIL:-admin@example.com}"
admin_password="${ARSENALE_ADMIN_PASSWORD:-DevAdmin123!}"
db_user="${ARSENALE_DB_USER:-arsenale}"
db_name="${ARSENALE_DB_NAME:-arsenale}"
connection_name="Acceptance DB $(date +%s)"
ssh_connection_name="Acceptance SSH $(date +%s)"
ssh_tunnel_connection_name="Acceptance SSH Tunnel $(date +%s)"
rdp_connection_name="Acceptance RDP $(date +%s)"
dev_tunnel_managed_ssh_gateway_id="${DEV_TUNNEL_MANAGED_SSH_GATEWAY_ID:-11111111-1111-4111-8111-111111111111}"

access_token="${ARSENALE_ACCESS_TOKEN:-}"
tenant_id="${ARSENALE_TENANT_ID:-}"
ssh_connection_id=""
ssh_session_id=""
ssh_tunnel_connection_id=""
ssh_tunnel_session_id=""
connection_id=""
session_id=""
rdp_connection_id=""
rdp_session_id=""

cleanup() {
  if [[ -n "${ssh_session_id}" ]]; then
    curl --silent --show-error --fail \
      --cacert "${ca_cert}" \
      -H "authorization: Bearer ${access_token}" \
      -H 'content-type: application/json' \
      -d '{}' \
      "${api_base}/sessions/ssh/${ssh_session_id}/end" >/dev/null || true
  fi

  if [[ -n "${ssh_connection_id}" ]]; then
    curl --silent --show-error --fail \
      --cacert "${ca_cert}" \
      -H "authorization: Bearer ${access_token}" \
      -X DELETE \
      "${api_base}/connections/${ssh_connection_id}" >/dev/null || true
  fi

  if [[ -n "${ssh_tunnel_session_id}" ]]; then
    curl --silent --show-error --fail \
      --cacert "${ca_cert}" \
      -H "authorization: Bearer ${access_token}" \
      -H 'content-type: application/json' \
      -d '{}' \
      "${api_base}/sessions/ssh/${ssh_tunnel_session_id}/end" >/dev/null || true
  fi

  if [[ -n "${ssh_tunnel_connection_id}" ]]; then
    curl --silent --show-error --fail \
      --cacert "${ca_cert}" \
      -H "authorization: Bearer ${access_token}" \
      -X DELETE \
      "${api_base}/connections/${ssh_tunnel_connection_id}" >/dev/null || true
  fi

  if [[ -n "${rdp_session_id}" ]]; then
    curl --silent --show-error --fail \
      --cacert "${ca_cert}" \
      -H "authorization: Bearer ${access_token}" \
      -H 'content-type: application/json' \
      -d '{}' \
      "${api_base}/sessions/rdp/${rdp_session_id}/end" >/dev/null || true
  fi

  if [[ -n "${rdp_connection_id}" ]]; then
    curl --silent --show-error --fail \
      --cacert "${ca_cert}" \
      -H "authorization: Bearer ${access_token}" \
      -X DELETE \
      "${api_base}/connections/${rdp_connection_id}" >/dev/null || true
  fi

  if [[ -n "${session_id}" ]]; then
    curl --silent --show-error --fail \
      --cacert "${ca_cert}" \
      -H "authorization: Bearer ${access_token}" \
      -H 'content-type: application/json' \
      -d '{}' \
      "${api_base}/sessions/database/${session_id}/end" >/dev/null || true
  fi

  if [[ -n "${connection_id}" ]]; then
    curl --silent --show-error --fail \
      --cacert "${ca_cert}" \
      -H "authorization: Bearer ${access_token}" \
      -X DELETE \
      "${api_base}/connections/${connection_id}" >/dev/null || true
  fi
}
trap cleanup EXIT

echo '1. /api/ready'
curl --silent --show-error --fail --cacert "${ca_cert}" "${api_base}/ready" \
  | jq -e '.status == "ok"' >/dev/null

echo '2. login'
if [[ -z "${access_token}" ]]; then
  login_json="$(curl --silent --show-error --fail \
    --cacert "${ca_cert}" \
    -H 'content-type: application/json' \
    -d "{\"email\":\"${admin_email}\",\"password\":\"${admin_password}\"}" \
    "${api_base}/auth/login")"
  access_token="$(printf '%s' "${login_json}" | jq -r '.accessToken')"
  tenant_id="$(printf '%s' "${login_json}" | jq -r '.user.tenantId')"
fi
[[ -n "${access_token}" && "${access_token}" != "null" ]]
[[ -n "${tenant_id}" && "${tenant_id}" != "null" ]]

echo '3. Go control-plane'
curl --silent --show-error --fail "${cp_base}/v1/meta/service" \
  | jq -e '.service.name == "control-plane-api"' >/dev/null
curl --silent --show-error --fail "${cp_base}/v1/orchestrators" \
  | jq -e '.connections | length >= 1' >/dev/null

orchestrator_name="$(curl --silent --show-error --fail "${cp_base}/v1/orchestrators" | jq -r '.connections[0].name')"
[[ -n "${orchestrator_name}" && "${orchestrator_name}" != "null" ]]

echo '3.1 Go control-plane-controller'
curl --silent --show-error --fail "${controller_base}/v1/meta/service" \
  | jq -e '.service.name == "control-plane-controller"' >/dev/null
curl --silent --show-error --fail \
  -H 'content-type: application/json' \
  -d "{\"connectionName\":\"${orchestrator_name}\",\"workload\":{\"name\":\"acceptance-workload\",\"image\":\"ghcr.io/example/app:latest\",\"env\":{\"MODE\":\"dev\"},\"ports\":[{\"container\":8080,\"protocol\":\"tcp\"}],\"healthcheck\":{\"command\":[\"/bin/true\"],\"intervalSec\":10,\"timeoutSec\":5,\"retries\":3},\"oci\":{\"network\":\"bridge\"}}}" \
  "${controller_base}/v1/reconcile:plan" \
  | jq -e '.accepted == true and .connection.name == "'"${orchestrator_name}"'"' >/dev/null

echo '3.2 Go authz-pdp'
curl --silent --show-error --fail "${authz_base}/v1/meta/service" \
  | jq -e '.service.name == "authz-pdp"' >/dev/null
curl --silent --show-error --fail \
  -H 'content-type: application/json' \
  -d '{"subject":{"type":"agent_run","id":"run-1"},"action":"db.query.execute.write","resource":{"type":"database","id":"dev-postgres"}}' \
  "${authz_base}/v1/decide" \
  | jq -e '.effect == "deny" and (.obligations | any(.type == "require_approval"))' >/dev/null

echo '3.3 Go model-gateway'
curl --silent --show-error --fail "${model_base}/v1/meta/service" \
  | jq -e '.service.name == "model-gateway"' >/dev/null
curl --silent --show-error --fail "${model_base}/v1/providers" \
  | jq -e '.providers | any(.id == "openai")' >/dev/null
curl --silent --show-error --fail \
  -H 'content-type: application/json' \
  -d "{\"config\":{\"tenantId\":\"${tenant_id}\",\"provider\":\"openai\",\"modelId\":\"gpt-4o\",\"maxTokensPerRequest\":2048,\"dailyRequestLimit\":50,\"enabled\":true},\"apiKeyConfigured\":false}" \
  "${model_base}/v1/provider-configs:validate" \
  | jq -e '.valid == false and (.errors | any(. == "provider requires an API key"))' >/dev/null
curl --silent --show-error --fail \
  -X PUT \
  -H 'content-type: application/json' \
  -d '{"provider":"openai","apiKey":"acceptance-key","modelId":"gpt-4o-mini","maxTokensPerRequest":2048,"dailyRequestLimit":50,"enabled":true}' \
  "${model_base}/v1/provider-configs/${tenant_id}" \
  | jq -e '.config.provider == "openai" and .config.hasApiKey == true and .config.modelId == "gpt-4o-mini"' >/dev/null

echo '3.4 Go runtime-agent'
curl --silent --show-error --fail "${runtime_base}/v1/meta/service" \
  | jq -e '.service.name == "runtime-agent"' >/dev/null
curl --silent --show-error --fail \
  -H 'content-type: application/json' \
  -d '{"kind":"podman","workload":{"name":"acceptance-runtime","image":"ghcr.io/example/app:latest","env":{"MODE":"dev"},"ports":[{"container":8080,"protocol":"tcp"}],"healthcheck":{"command":["/bin/true"],"intervalSec":10,"timeoutSec":5,"retries":3},"oci":{"network":"bridge"}}}' \
  "${runtime_base}/v1/runtime/workloads:validate" \
  | jq -e '.valid == true' >/dev/null

echo '3.5 Go desktop-broker'
curl --silent --show-error --fail "${desktop_base}/v1/meta/service" \
  | jq -e '.service.name == "desktop-broker"' >/dev/null

echo '3.6 Go terminal-broker'
curl --silent --show-error --fail "${terminal_base}/v1/meta/service" \
  | jq -e '.service.name == "terminal-broker"' >/dev/null
curl --silent --show-error --fail "${terminal_base}/v1/session-protocol" \
  | jq -e '.webSocketPath == "/ws/terminal"' >/dev/null

echo '4. Go query-runner'
curl --silent --show-error --fail \
  -H 'content-type: application/json' \
  -d '{"sql":"select current_database() as database_name","maxRows":1}' \
  "${query_base}/v1/query-runs:execute" \
  | jq -e '.rowCount == 1' >/dev/null

curl --silent --show-error --fail \
  -H 'content-type: application/json' \
  -d '{"sql":"select current_database() as database_name"}' \
  "${query_base}/v1/query-plans:explain" \
  | jq -e '.supported == true and .format == "json"' >/dev/null

curl --silent --show-error --fail \
  -H 'content-type: application/json' \
  -d "{\"type\":\"database_version\",\"db\":{\"protocol\":\"postgresql\",\"host\":\"postgres\",\"port\":5432,\"database\":\"${db_name}\",\"sslMode\":\"require\",\"username\":\"${db_user}\",\"password\":\"${postgres_password}\"}}" \
  "${query_base}/v1/introspection:run" \
  | jq -e '.supported == true and (.data.version | type == "string")' >/dev/null

curl --silent --show-error --fail \
  -H 'content-type: application/json' \
  -d "{\"target\":{\"protocol\":\"postgresql\",\"host\":\"postgres\",\"port\":5432,\"database\":\"${db_name}\",\"sslMode\":\"require\",\"username\":\"${db_user}\",\"password\":\"${postgres_password}\"}}" \
  "${query_base}/v1/schema:fetch" \
  | jq -e '.tables | length > 0' >/dev/null

echo '5. Go tool-gateway'
curl --silent --show-error --fail "${tool_base}/v1/capabilities" \
  | jq -e '.capabilities | length >= 1' >/dev/null

curl --silent --show-error --fail \
  -H 'content-type: application/json' \
  -d '{"capability":"db.query.execute.readonly","authz":{"subject":{"type":"system","id":"acceptance"},"resource":{"type":"database","id":"control-plane"}},"input":{"sql":"select current_database() as database_name","maxRows":1}}' \
  "${tool_base}/v1/tool-calls:execute" \
  | jq -e '.decision.effect == "allow" and .output.rowCount == 1' >/dev/null

curl --silent --show-error --fail \
  -H 'content-type: application/json' \
  -d "{\"capability\":\"db.schema.read\",\"authz\":{\"subject\":{\"type\":\"system\",\"id\":\"acceptance\"},\"resource\":{\"type\":\"database\",\"id\":\"dev-postgres\"}},\"input\":{\"target\":{\"protocol\":\"postgresql\",\"host\":\"postgres\",\"port\":5432,\"database\":\"${db_name}\",\"sslMode\":\"require\",\"username\":\"${db_user}\",\"password\":\"${postgres_password}\"}}}" \
  "${tool_base}/v1/tool-calls:execute" \
  | jq -e '.decision.effect == "allow" and (.output.tables | length > 0)' >/dev/null

curl --silent --show-error --fail \
  -H 'content-type: application/json' \
  -d "{\"capability\":\"db.introspection.read\",\"authz\":{\"subject\":{\"type\":\"system\",\"id\":\"acceptance\"},\"resource\":{\"type\":\"database\",\"id\":\"dev-postgres\"}},\"input\":{\"type\":\"database_version\",\"db\":{\"protocol\":\"postgresql\",\"host\":\"postgres\",\"port\":5432,\"database\":\"${db_name}\",\"sslMode\":\"require\",\"username\":\"${db_user}\",\"password\":\"${postgres_password}\"}}}" \
  "${tool_base}/v1/tool-calls:execute" \
  | jq -e '.decision.effect == "allow" and (.output.data.version | type == "string")' >/dev/null

echo '5.1 Go memory-service'
memory_namespace_key="$(curl --silent --show-error --fail \
  -X PUT \
  -H 'content-type: application/json' \
  -d '{"tenantId":"acceptance","scope":"agent","agentId":"agent-acceptance","type":"episodic","name":"default"}' \
  "${memory_base}/v1/memory/namespaces" \
  | jq -r '.namespace.key')"
[[ -n "${memory_namespace_key}" && "${memory_namespace_key}" != "null" ]]

curl --silent --show-error --fail \
  -H 'content-type: application/json' \
  -d '{"namespace":{"tenantId":"acceptance","scope":"agent","agentId":"agent-acceptance","type":"episodic","name":"default"},"content":"validated schema fetch path","summary":"schema fetch validation","metadata":{"source":"acceptance"}}' \
  "${memory_base}/v1/memory/items" \
  | jq -e '.item.namespaceKey == "'"${memory_namespace_key}"'"' >/dev/null

curl --silent --show-error --fail \
  "${memory_base}/v1/memory/items?namespaceKey=${memory_namespace_key}" \
  | jq -e '.items | length >= 1' >/dev/null

echo '5.2 Go agent-orchestrator'
agent_run_id="$(curl --silent --show-error --fail \
  -H 'content-type: application/json' \
  -d '{"tenantId":"acceptance","definitionId":"ops-agent","trigger":"acceptance","goals":["validate infra APIs"],"requestedCapabilities":["db.schema.read","gateway.scale"]}' \
  "${agent_base}/v1/agent-runs" \
  | jq -r '.run.id')"
[[ -n "${agent_run_id}" && "${agent_run_id}" != "null" ]]

curl --silent --show-error --fail \
  "${agent_base}/v1/agent-runs/${agent_run_id}" \
  | jq -e '.run.requiresApproval == true and .run.status == "queued"' >/dev/null

curl --silent --show-error --fail \
  "${agent_base}/v1/agent-runs?tenantId=acceptance" \
  | jq -e '.runs | any(.id == "'"${agent_run_id}"'")' >/dev/null

echo '5.3 Go tool-gateway memory capabilities'
curl --silent --show-error --fail \
  -H 'content-type: application/json' \
  -d '{"capability":"memory.write","authz":{"subject":{"type":"agent_run","id":"run-acceptance"},"resource":{"type":"memory_namespace","id":"agent-acceptance/default"},"context":{"approved":"true"}},"input":{"namespace":{"tenantId":"acceptance","scope":"agent","agentId":"agent-acceptance","type":"episodic","name":"default"},"content":"tool-gateway memory write","summary":"gateway memory validation","metadata":{"source":"tool-gateway"}}}' \
  "${tool_base}/v1/tool-calls:execute" \
  | jq -e '.decision.effect == "allow" and .output.item.namespaceKey == "'"${memory_namespace_key}"'"' >/dev/null

curl --silent --show-error --fail \
  -H 'content-type: application/json' \
  -d "{\"capability\":\"memory.read\",\"authz\":{\"subject\":{\"type\":\"agent_run\",\"id\":\"run-acceptance\"},\"resource\":{\"type\":\"memory_namespace\",\"id\":\"${memory_namespace_key}\"}},\"input\":{\"namespaceKey\":\"${memory_namespace_key}\"}}" \
  "${tool_base}/v1/tool-calls:execute" \
  | jq -e '.decision.effect == "allow" and (.output.items | length >= 1)' >/dev/null

echo '5.4 Go tool-gateway SSH grant + terminal broker flow'
terminal_grant_json="$(curl --silent --show-error --fail \
  -H 'content-type: application/json' \
  -d '{"capability":"connection.connect.ssh","authz":{"subject":{"type":"agent_run","id":"run-acceptance"},"resource":{"type":"connection","id":"terminal-target"},"context":{"approved":"true"}},"input":{"sessionId":"terminal-acceptance","connectionId":"terminal-target","userId":"acceptance","expiresAt":"2030-01-01T00:00:00Z","target":{"host":"terminal-target","port":2224,"username":"acceptance","password":"acceptance"},"terminal":{"term":"xterm-256color","cols":80,"rows":24}}}' \
  "${tool_base}/v1/tool-calls:execute")"
printf '%s' "${terminal_grant_json}" \
  | jq -e '.decision.effect == "allow" and (.output.token | type == "string") and (.output.webSocketUrl | startswith("ws://"))' >/dev/null
terminal_ws_url="$(printf '%s' "${terminal_grant_json}" | jq -r '.output.webSocketUrl')"
[[ -n "${terminal_ws_url}" && "${terminal_ws_url}" != "null" ]]

TERMINAL_WS_URL="${terminal_ws_url}" node <<'NODE'
const ws = new WebSocket(process.env.TERMINAL_WS_URL);
let ready = false;
let sawOutput = false;
let buffer = '';
const timeout = setTimeout(() => {
  console.error('terminal broker websocket timed out');
  process.exit(1);
}, 15000);

ws.onmessage = (event) => {
  const message = JSON.parse(String(event.data));
  if (message.type === 'ready' && !ready) {
    ready = true;
    ws.send(JSON.stringify({ type: 'input', data: 'echo acceptance-terminal && exit\n' }));
    return;
  }

  if (message.type === 'data') {
    buffer += message.data || '';
    if (buffer.includes('acceptance-terminal')) {
      sawOutput = true;
    }
    return;
  }

  if (message.type === 'error') {
    clearTimeout(timeout);
    console.error(message.message || message.code || 'terminal broker error');
    process.exit(1);
  }

  if (message.type === 'closed') {
    clearTimeout(timeout);
    process.exit(sawOutput ? 0 : 1);
  }
};

ws.onerror = (event) => {
  clearTimeout(timeout);
  console.error(event.error?.message || 'terminal broker socket error');
  process.exit(1);
};

ws.onclose = () => {
  clearTimeout(timeout);
  process.exit(sawOutput ? 0 : 1);
};
NODE

echo '6. public SSH session flow'
create_ssh_connection_json="$(curl --silent --show-error --fail \
  --cacert "${ca_cert}" \
  -H "authorization: Bearer ${access_token}" \
  -H 'content-type: application/json' \
  -d "{\"name\":\"${ssh_connection_name}\",\"type\":\"SSH\",\"host\":\"terminal-target\",\"port\":2224,\"username\":\"acceptance\",\"password\":\"acceptance\"}" \
  "${api_base}/connections")"
ssh_connection_id="$(printf '%s' "${create_ssh_connection_json}" | jq -r '.id')"
[[ -n "${ssh_connection_id}" && "${ssh_connection_id}" != "null" ]]

start_ssh_session_json="$(curl --silent --show-error --fail \
  --cacert "${ca_cert}" \
  -H "authorization: Bearer ${access_token}" \
  -H 'content-type: application/json' \
  -d "{\"connectionId\":\"${ssh_connection_id}\"}" \
  "${api_base}/sessions/ssh")"
ssh_session_id="$(printf '%s' "${start_ssh_session_json}" | jq -r '.sessionId')"
ssh_transport="$(printf '%s' "${start_ssh_session_json}" | jq -r '.transport')"
ssh_ws_url="$(printf '%s' "${start_ssh_session_json}" | jq -r '.webSocketUrl')"
[[ "${ssh_transport}" == "terminal-broker" ]]
[[ -n "${ssh_session_id}" && "${ssh_session_id}" != "null" ]]
[[ -n "${ssh_ws_url}" && "${ssh_ws_url}" != "null" ]]

NODE_TLS_REJECT_UNAUTHORIZED=0 SSH_WS_URL="${ssh_ws_url}" node <<'NODE'
const ws = new WebSocket(process.env.SSH_WS_URL);
let ready = false;
let sawOutput = false;
let buffer = '';
const timeout = setTimeout(() => {
  console.error('public ssh websocket timed out');
  process.exit(1);
}, 15000);

ws.onmessage = (event) => {
  const message = JSON.parse(String(event.data));
  if (message.type === 'ready' && !ready) {
    ready = true;
    ws.send(JSON.stringify({ type: 'input', data: 'echo acceptance-public-ssh && exit\n' }));
    return;
  }

  if (message.type === 'data') {
    buffer += message.data || '';
    if (buffer.includes('acceptance-public-ssh')) {
      sawOutput = true;
    }
    return;
  }

  if (message.type === 'error') {
    clearTimeout(timeout);
    console.error(message.message || message.code || 'public ssh broker error');
    process.exit(1);
  }
};

ws.onerror = (event) => {
  clearTimeout(timeout);
  console.error(event.error?.message || 'public ssh socket error');
  process.exit(1);
};

ws.onclose = () => {
  clearTimeout(timeout);
  process.exit(sawOutput ? 0 : 1);
};
NODE

echo '6.1. public SSH session flow via tunnel-backed managed gateway'
create_ssh_tunnel_connection_json="$(curl --silent --show-error --fail \
  --cacert "${ca_cert}" \
  -H "authorization: Bearer ${access_token}" \
  -H 'content-type: application/json' \
  -d "{\"name\":\"${ssh_tunnel_connection_name}\",\"type\":\"SSH\",\"host\":\"terminal-target\",\"port\":2224,\"username\":\"acceptance\",\"password\":\"acceptance\",\"gatewayId\":\"${dev_tunnel_managed_ssh_gateway_id}\"}" \
  "${api_base}/connections")"
ssh_tunnel_connection_id="$(printf '%s' "${create_ssh_tunnel_connection_json}" | jq -r '.id')"
[[ -n "${ssh_tunnel_connection_id}" && "${ssh_tunnel_connection_id}" != "null" ]]

start_ssh_tunnel_session_json="$(curl --silent --show-error --fail \
  --cacert "${ca_cert}" \
  -H "authorization: Bearer ${access_token}" \
  -H 'content-type: application/json' \
  -d "{\"connectionId\":\"${ssh_tunnel_connection_id}\"}" \
  "${api_base}/sessions/ssh")"
ssh_tunnel_session_id="$(printf '%s' "${start_ssh_tunnel_session_json}" | jq -r '.sessionId')"
ssh_tunnel_transport="$(printf '%s' "${start_ssh_tunnel_session_json}" | jq -r '.transport')"
ssh_tunnel_ws_url="$(printf '%s' "${start_ssh_tunnel_session_json}" | jq -r '.webSocketUrl')"
[[ "${ssh_tunnel_transport}" == "terminal-broker" ]]
[[ -n "${ssh_tunnel_session_id}" && "${ssh_tunnel_session_id}" != "null" ]]
[[ -n "${ssh_tunnel_ws_url}" && "${ssh_tunnel_ws_url}" != "null" ]]

NODE_TLS_REJECT_UNAUTHORIZED=0 SSH_WS_URL="${ssh_tunnel_ws_url}" node <<'NODE'
const ws = new WebSocket(process.env.SSH_WS_URL);
let ready = false;
let sawOutput = false;
let buffer = '';
const timeout = setTimeout(() => {
  console.error('public tunnel ssh websocket timed out');
  process.exit(1);
}, 15000);

ws.onmessage = (event) => {
  const message = JSON.parse(String(event.data));
  if (message.type === 'ready' && !ready) {
    ready = true;
    ws.send(JSON.stringify({ type: 'input', data: 'echo acceptance-tunnel-ssh && exit\n' }));
    return;
  }

  if (message.type === 'data') {
    buffer += message.data || '';
    if (buffer.includes('acceptance-tunnel-ssh')) {
      sawOutput = true;
    }
    return;
  }

  if (message.type === 'error') {
    clearTimeout(timeout);
    console.error(message.message || message.code || 'public tunnel ssh broker error');
    process.exit(1);
  }
};

ws.onerror = (event) => {
  clearTimeout(timeout);
  console.error(event.error?.message || 'public tunnel ssh socket error');
  process.exit(1);
};

ws.onclose = () => {
  clearTimeout(timeout);
  process.exit(sawOutput ? 0 : 1);
};
NODE

echo '7. public DB session flow'
create_connection_json="$(curl --silent --show-error --fail \
  --cacert "${ca_cert}" \
  -H "authorization: Bearer ${access_token}" \
  -H 'content-type: application/json' \
  -d "{\"name\":\"${connection_name}\",\"type\":\"DATABASE\",\"host\":\"postgres\",\"port\":5432,\"username\":\"${db_user}\",\"password\":\"${postgres_password}\",\"dbSettings\":{\"protocol\":\"postgresql\",\"databaseName\":\"${db_name}\",\"sslMode\":\"require\"}}" \
  "${api_base}/connections")"
connection_id="$(printf '%s' "${create_connection_json}" | jq -r '.id')"
[[ -n "${connection_id}" && "${connection_id}" != "null" ]]

create_session_json="$(curl --silent --show-error --fail \
  --cacert "${ca_cert}" \
  -H "authorization: Bearer ${access_token}" \
  -H 'content-type: application/json' \
  -d "{\"connectionId\":\"${connection_id}\"}" \
  "${api_base}/sessions/database")"
session_id="$(printf '%s' "${create_session_json}" | jq -r '.sessionId')"
[[ -n "${session_id}" && "${session_id}" != "null" ]]

curl --silent --show-error --fail \
  --cacert "${ca_cert}" \
  -H "authorization: Bearer ${access_token}" \
  -H 'content-type: application/json' \
  -d '{"sql":"select current_database() as database_name"}' \
  "${api_base}/sessions/database/${session_id}/query" \
  | jq -e ".rowCount == 1 and .rows[0].database_name == \"${db_name}\"" >/dev/null

curl --silent --show-error --fail \
  --cacert "${ca_cert}" \
  -H "authorization: Bearer ${access_token}" \
  -H 'content-type: application/json' \
  -d '{"sql":"select current_database() as database_name"}' \
  "${api_base}/sessions/database/${session_id}/explain" \
  | jq -e '.supported == true and .format == "json"' >/dev/null

curl --silent --show-error --fail \
  --cacert "${ca_cert}" \
  -H "authorization: Bearer ${access_token}" \
  -H 'content-type: application/json' \
  -d '{"type":"database_version"}' \
  "${api_base}/sessions/database/${session_id}/introspect" \
  | jq -e '.supported == true and (.data.version | type == "string")' >/dev/null

curl --silent --show-error --fail \
  --cacert "${ca_cert}" \
  -H "authorization: Bearer ${access_token}" \
  -H 'content-type: application/json' \
  -d '{"type":"table_schema","target":"orchestrator_connections"}' \
  "${api_base}/sessions/database/${session_id}/introspect" \
  | jq -e '.supported == true and (.data | length > 0)' >/dev/null

curl --silent --show-error --fail \
  --cacert "${ca_cert}" \
  -H "authorization: Bearer ${access_token}" \
  "${api_base}/sessions/database/${session_id}/schema" \
  | jq -e '.tables | length > 0' >/dev/null

echo '7. public desktop broker flow'
guacd_gateway_id="$(curl --silent --show-error --fail \
  --cacert "${ca_cert}" \
  -H "authorization: Bearer ${access_token}" \
  "${api_base}/gateways" \
  | jq -r '(map(select(.type == "GUACD")) | ((map(select((.tunnelEnabled // false) == false))[0]) // .[0]).id) // empty')"
[[ -n "${guacd_gateway_id}" ]]

create_rdp_connection_json="$(curl --silent --show-error --fail \
  --cacert "${ca_cert}" \
  -H "authorization: Bearer ${access_token}" \
  -H 'content-type: application/json' \
  -d "{\"name\":\"${rdp_connection_name}\",\"type\":\"RDP\",\"host\":\"rdp.invalid\",\"port\":3389,\"username\":\"acceptance\",\"password\":\"acceptance\",\"gatewayId\":\"${guacd_gateway_id}\",\"rdpSettings\":{\"ignoreCert\":true}}" \
  "${api_base}/connections")"
rdp_connection_id="$(printf '%s' "${create_rdp_connection_json}" | jq -r '.id')"
[[ -n "${rdp_connection_id}" && "${rdp_connection_id}" != "null" ]]

create_rdp_session_json="$(curl --silent --show-error --fail \
  --cacert "${ca_cert}" \
  -H "authorization: Bearer ${access_token}" \
  -H 'content-type: application/json' \
  -d "{\"connectionId\":\"${rdp_connection_id}\"}" \
  "${api_base}/sessions/rdp")"
rdp_session_id="$(printf '%s' "${create_rdp_session_json}" | jq -r '.sessionId')"
rdp_token="$(printf '%s' "${create_rdp_session_json}" | jq -r '.token')"
[[ -n "${rdp_session_id}" && "${rdp_session_id}" != "null" ]]
[[ -n "${rdp_token}" && "${rdp_token}" != "null" ]]

NODE_TLS_REJECT_UNAUTHORIZED=0 RDP_TOKEN="${rdp_token}" node <<'NODE'
const url = new URL('wss://localhost:3000/guacamole/');
url.searchParams.set('token', process.env.RDP_TOKEN);
let opened = false;
const timeout = setTimeout(() => {
  console.error('desktop broker websocket timed out');
  process.exit(1);
}, 10000);

const ws = new WebSocket(url);

ws.onopen = () => {
  opened = true;
};

ws.onmessage = () => {
  clearTimeout(timeout);
  ws.close();
};

ws.onclose = () => {
  clearTimeout(timeout);
  process.exit(opened ? 0 : 1);
};

ws.onerror = (event) => {
  if (!opened) {
    clearTimeout(timeout);
    console.error(event.error?.message || 'desktop broker socket error');
    process.exit(1);
  }
};
NODE

echo 'acceptance-ok'
