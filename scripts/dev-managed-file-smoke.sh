#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "${repo_root}/scripts/lib/arsenale-cli-harness.sh"

default_state_home="${XDG_STATE_HOME:-$HOME/.local/state}"
default_dev_home="${ARSENALE_DEV_HOME:-$default_state_home/arsenale-dev}"
ca_cert="${ARSENALE_CA_CERT:-$repo_root/dev-certs/client/ca.pem}"
if [[ ! -f "${ca_cert}" && -f "${default_dev_home}/dev-certs/client/ca.pem" ]]; then
  ca_cert="${default_dev_home}/dev-certs/client/ca.pem"
fi

server_url="${ARSENALE_SERVER_URL:-https://localhost:3000}"
cli_bin="${ARSENALE_CLI_BIN:-$repo_root/build/go/arsenale-cli}"
smoke_suffix="$(date +%s)-$$"
tmp_dir="$(mktemp -d)"
evidence_dir="${repo_root}/.sisyphus/evidence"
created_connection_ids=()

cleanup() {
  for index in "${!created_connection_ids[@]}"; do
    local connection_id="${created_connection_ids[$index]}"
    if [[ -n "${connection_id}" ]]; then
      "${cli_bin}" --server "${server_url}" connection delete "${connection_id}" >/dev/null 2>&1 || true
    fi
  done
  rm -rf "${tmp_dir}"
}

trap cleanup EXIT

ensure_cli() {
  arsenale_cli_ensure_built "${repo_root}" "${cli_bin}"
}

run_cli() {
  arsenale_cli_run "${cli_bin}" "${server_url}" "$@"
}

write_json_payload() {
  local output_path="$1"
  local kind="$2"
  local connection_name="$3"
  local gateway_id="${4:-}"

  python3 - "${output_path}" "${kind}" "${connection_name}" "${gateway_id}" <<'PY'
import json
import sys

output_path = sys.argv[1]
kind = sys.argv[2]
connection_name = sys.argv[3]
gateway_id = sys.argv[4] if len(sys.argv) > 4 else ""

if kind == "ssh-positive":
    payload = {
        "name": connection_name,
        "type": "SSH",
        "host": "terminal-target",
        "port": 2224,
        "username": "acceptance",
        "password": "acceptance",
        "description": "Managed file smoke SSH connection",
    }
elif kind == "ssh-deny":
    payload = {
        "name": connection_name,
        "type": "SSH",
        "host": "terminal-target",
        "port": 2224,
        "username": "acceptance",
        "password": "acceptance",
        "description": "Managed file smoke SSH deny connection",
        "dlpPolicy": {
            "disableUpload": True,
            "disableDownload": True,
        },
    }
elif kind == "rdp-positive":
    payload = {
        "name": connection_name,
        "type": "RDP",
        "host": "rdp.invalid",
        "port": 3389,
        "username": "acceptance",
        "password": "acceptance",
        "gatewayId": gateway_id or None,
        "enableDrive": True,
        "description": "Managed file smoke RDP connection",
        "rdpSettings": {
            "ignoreCert": True,
        },
    }
else:
    raise SystemExit(f"unknown payload kind: {kind}")

if payload.get("gatewayId") is None:
    payload.pop("gatewayId", None)

with open(output_path, "w", encoding="utf-8") as handle:
    json.dump(payload, handle)
PY
}

create_connection() {
  local payload_path="$1"
  local output_path="$2"
  run_cli connection create --from-file "${payload_path}" -o json > "${output_path}"
  jq -e '.id | type == "string" and length > 0' "${output_path}" >/dev/null
  local connection_id
  connection_id="$(jq -r '.id' "${output_path}")"
  created_connection_ids+=("${connection_id}")
  printf '%s' "${connection_id}"
}

discover_connection_id() {
  local connections_json="$1"
  local connection_type="$2"
  jq -r --arg type "${connection_type}" '([.own[]?, .shared[]?, .team[]?] | map(select(.type == $type)) | .[0].id) // empty' "${connections_json}"
}

discover_gateway_id() {
  local gateways_json="$1"
  jq -r 'map(select(.type == "GUACD")) | ((map(select((.tunnelEnabled // false) == false))[0]) // .[0]).id // empty' "${gateways_json}"
}

discover_history_id() {
  local history_json="$1"
  local file_name="$2"
  jq -r --arg name "${file_name}" '[.[]? | select(.fileName == $name) | .id][0] // empty' "${history_json}"
}

assert_session_flags() {
  local session_json="$1"
  local expect_drive="$2"
  if [[ "${expect_drive}" == "ssh" ]]; then
    jq -e '.transport == "terminal-broker" and .sftpSupported == false and .fileBrowserSupported == true' "${session_json}" >/dev/null
  else
    jq -e '.enableDrive == true' "${session_json}" >/dev/null
  fi
}

assert_actions_present() {
  local audit_json="$1"
  shift
  for action in "$@"; do
    jq -e --arg action "${action}" '.data | any(.action == $action)' "${audit_json}" >/dev/null
  done
}

assert_actions_absent() {
  local audit_json="$1"
  shift
  for action in "$@"; do
    jq -e --arg action "${action}" '.data | all(.action != $action)' "${audit_json}" >/dev/null
  done
}

assert_output_contains() {
  local output="$1"
  local expected="$2"
  local label="$3"
  if [[ "${output}" != *"${expected}"* ]]; then
    echo "expected ${label} to contain: ${expected}" >&2
    echo "${output}" >&2
    exit 1
  fi
}

assert_json_array_length() {
  local json_path="$1"
  local expected_length="$2"
  jq -e --argjson expected "${expected_length}" 'length == $expected' "${json_path}" >/dev/null
}

assert_json_array_contains_field() {
  local json_path="$1"
  local field_name="$2"
  local expected_value="$3"
  jq -e --arg field "${field_name}" --arg value "${expected_value}" 'any(.[]; .[$field] == $value)' "${json_path}" >/dev/null
}

set_connection_transfer_retention_policy() {
  local connection_id="$1"
  python3 - "${connection_id}" <<'PY'
import shlex
import subprocess
import sys

connection_id = sys.argv[1]
sql = f'UPDATE "Connection" SET "transferRetentionPolicy" = \'{{"retainSuccessfulUploads":true}}\'::jsonb WHERE id = \'{connection_id}\';'
inner = 'PGPASSWORD=$(cat /run/secrets/postgres_password) psql -U arsenale -d arsenale -v ON_ERROR_STOP=1 -c ' + shlex.quote(sql)
subprocess.run(['podman', 'exec', 'arsenale-postgres', 'sh', '-lc', inner], check=True)
PY
}

assert_connection_transfer_retention_policy_true() {
  local connection_id="$1"
  python3 - "${connection_id}" <<'PY'
import json
import shlex
import subprocess
import sys

connection_id = sys.argv[1]
sql = f'SELECT "transferRetentionPolicy" FROM "Connection" WHERE id = \'{connection_id}\';'
inner = 'PGPASSWORD=$(cat /run/secrets/postgres_password) psql -U arsenale -d arsenale -tAc ' + shlex.quote(sql)
result = subprocess.run(['podman', 'exec', 'arsenale-postgres', 'sh', '-lc', inner], check=True, capture_output=True, text=True)
value = result.stdout.strip()
if value != '{"retainSuccessfulUploads": true}' and value != '{"retainSuccessfulUploads":true}':
    raise SystemExit(f'unexpected retention policy: {value!r}')
PY
}


assert_connection_transfer_retention_policy_default() {
  local connection_id="$1"
  python3 - "${connection_id}" <<'PY'
import shlex
import subprocess
import sys

connection_id = sys.argv[1]
sql = f"SELECT COALESCE(\"transferRetentionPolicy\"::text, 'NULL') FROM \"Connection\" WHERE id = '{connection_id}';"
inner = 'PGPASSWORD=$(cat /run/secrets/postgres_password) psql -U arsenale -d arsenale -tAc ' + shlex.quote(sql)
result = subprocess.run(['podman', 'exec', 'arsenale-postgres', 'sh', '-lc', inner], check=True, capture_output=True, text=True)
value = result.stdout.strip()
allowed = {'NULL', '{"retainSuccessfulUploads": false}', '{"retainSuccessfulUploads":false}'}
if value not in allowed:
    raise SystemExit(f'unexpected default retention policy: {value!r}')
PY
}

ensure_cli
run_cli health >/dev/null
run_cli whoami >/dev/null

mkdir -p "${evidence_dir}"

connections_json="${tmp_dir}/connections.json"
run_cli connection list -o json > "${connections_json}"

discovered_ssh_connection_id="$(discover_connection_id "${connections_json}" "SSH")"
discovered_rdp_connection_id="$(discover_connection_id "${connections_json}" "RDP")"

ssh_payload="${tmp_dir}/ssh-positive.json"
ssh_output="${tmp_dir}/ssh-positive-created.json"
write_json_payload "${ssh_payload}" "ssh-positive" "Managed File Smoke SSH ${smoke_suffix}"
ssh_connection_id="$(create_connection "${ssh_payload}" "${ssh_output}")"

ssh_payload_file="${tmp_dir}/ssh-payload.txt"
ssh_download_dir="${tmp_dir}/ssh-download"
ssh_workspace_dir="tmp/managed-ssh-${smoke_suffix}"
ssh_workspace_name="report-${smoke_suffix}.txt"
ssh_workspace_path="${ssh_workspace_dir}/${ssh_workspace_name}"
printf 'managed ssh smoke %s\n' "${smoke_suffix}" > "${ssh_payload_file}"
mkdir -p "${ssh_download_dir}"

run_cli file ssh mkdir --connection "${ssh_connection_id}" --path "${ssh_workspace_dir}"
run_cli file ssh upload --connection "${ssh_connection_id}" --file "${ssh_payload_file}" --to "${ssh_workspace_path}" -o json > "${tmp_dir}/ssh-upload.json"
run_cli file ssh list --connection "${ssh_connection_id}" --path "${ssh_workspace_dir}" -o json > "${tmp_dir}/ssh-list-after-upload.json"
jq -e --arg name "${ssh_workspace_name}" 'map(.name) | index($name) != null' "${tmp_dir}/ssh-list-after-upload.json" >/dev/null
run_cli file ssh download --connection "${ssh_connection_id}" --path "${ssh_workspace_path}" --dest "${ssh_download_dir}"
if [[ ! -f "${ssh_download_dir}/${ssh_workspace_name}" ]]; then
  echo "expected SSH download artifact" >&2
  exit 1
fi
run_cli audit connection "${ssh_connection_id}" -o json > "${tmp_dir}/ssh-audit.json"
assert_actions_present "${tmp_dir}/ssh-audit.json" FILE_LIST FILE_UPLOAD FILE_DOWNLOAD

gateways_json="${tmp_dir}/gateways.json"
run_cli gateway list -o json > "${gateways_json}"
guacd_gateway_id="$(discover_gateway_id "${gateways_json}")"

if [[ -z "${guacd_gateway_id}" ]]; then
  echo "could not discover a GUACD gateway for managed RDP smoke" >&2
  exit 1
fi

rdp_payload="${tmp_dir}/rdp-positive.json"
rdp_output="${tmp_dir}/rdp-positive-created.json"
write_json_payload "${rdp_payload}" "rdp-positive" "Managed File Smoke RDP ${smoke_suffix}" "${guacd_gateway_id}"
rdp_connection_id="$(create_connection "${rdp_payload}" "${rdp_output}")"

rdp_payload_file="${tmp_dir}/rdp-payload.txt"
rdp_download_dir="${tmp_dir}/rdp-download"
rdp_file_name="$(basename "${rdp_payload_file}")"
printf 'managed rdp smoke %s\n' "${smoke_suffix}" > "${rdp_payload_file}"
mkdir -p "${rdp_download_dir}"

run_cli file list --connection "${rdp_connection_id}" -o json > "${tmp_dir}/rdp-list-before.json"
jq -e 'type == "array"' "${tmp_dir}/rdp-list-before.json" >/dev/null

run_cli file upload --connection "${rdp_connection_id}" --file "${rdp_payload_file}" -o json > "${tmp_dir}/rdp-upload.json"
jq -e --arg name "${rdp_file_name}" 'map(.name) | index($name) != null' "${tmp_dir}/rdp-upload.json" >/dev/null

run_cli file list --connection "${rdp_connection_id}" -o json > "${tmp_dir}/rdp-list-after-upload.json"
jq -e --arg name "${rdp_file_name}" 'map(.name) | index($name) != null' "${tmp_dir}/rdp-list-after-upload.json" >/dev/null

run_cli audit connection "${rdp_connection_id}" -o json > "${tmp_dir}/rdp-audit.json"
assert_actions_present "${tmp_dir}/rdp-audit.json" FILE_LIST FILE_UPLOAD
assert_connection_transfer_retention_policy_default "${rdp_connection_id}"

rdp_retention_payload="${tmp_dir}/rdp-retain.json"
rdp_retention_output="${tmp_dir}/rdp-retain-created.json"
write_json_payload "${rdp_retention_payload}" "rdp-positive" "Managed File Smoke RDP Retain ${smoke_suffix}" "${guacd_gateway_id}"
rdp_retention_connection_id="$(create_connection "${rdp_retention_payload}" "${rdp_retention_output}")"
set_connection_transfer_retention_policy "${rdp_retention_connection_id}"

rdp_retain_payload_file="${tmp_dir}/rdp-retain-payload.txt"
rdp_retain_file_name="$(basename "${rdp_retain_payload_file}")"
rdp_history_download_dir="${tmp_dir}/rdp-history-download"
printf 'managed rdp retain smoke %s\n' "${smoke_suffix}" > "${rdp_retain_payload_file}"
mkdir -p "${rdp_history_download_dir}"

run_cli file upload --connection "${rdp_retention_connection_id}" --file "${rdp_retain_payload_file}" -o json > "${tmp_dir}/rdp-retain-upload.json"
assert_connection_transfer_retention_policy_true "${rdp_retention_connection_id}"
run_cli file list --connection "${rdp_retention_connection_id}" -o json > "${tmp_dir}/rdp-retain-list-before.json"
jq -e 'type == "array"' "${tmp_dir}/rdp-retain-list-before.json" >/dev/null
run_cli file history list --connection "${rdp_retention_connection_id}" -o json > "${tmp_dir}/rdp-retain-history-list.json"
history_id="$(discover_history_id "${tmp_dir}/rdp-retain-history-list.json" "${rdp_retain_file_name}")"
if [[ -z "${history_id}" ]]; then
  echo "expected retained upload to appear in file history list" >&2
  exit 1
fi
run_cli file history download "${history_id}" --connection "${rdp_retention_connection_id}" --dest "${rdp_history_download_dir}"
if [[ ! -f "${rdp_history_download_dir}/${rdp_retain_file_name}" ]]; then
  echo "expected retained history download artifact" >&2
  exit 1
fi
run_cli audit connection "${rdp_retention_connection_id}" -o json > "${tmp_dir}/rdp-retain-audit.json"

deny_connection_payload="${tmp_dir}/ssh-deny.json"
deny_connection_output="${tmp_dir}/ssh-deny-created.json"
write_json_payload "${deny_connection_payload}" "ssh-deny" "Managed File Smoke SSH Deny ${smoke_suffix}"
deny_connection_id="$(create_connection "${deny_connection_payload}" "${deny_connection_output}")"

ssh_absolute_payload="${tmp_dir}/ssh-absolute.json"
ssh_absolute_output="${tmp_dir}/ssh-absolute-created.json"
write_json_payload "${ssh_absolute_payload}" "ssh-positive" "Managed File Smoke SSH Absolute ${smoke_suffix}"
ssh_absolute_connection_id="$(create_connection "${ssh_absolute_payload}" "${ssh_absolute_output}")"

deny_payload_file="${tmp_dir}/deny-payload.txt"
deny_remote_name="blocked-${smoke_suffix}.txt"
deny_workspace_dir="tmp/managed-deny-${smoke_suffix}"
deny_remote_path="${deny_workspace_dir}/${deny_remote_name}"
deny_download_dir="${tmp_dir}/deny-download"
deny_upload_output=""
deny_download_output=""
printf 'deny %s\n' "${smoke_suffix}" > "${deny_payload_file}"
mkdir -p "${deny_download_dir}"


deny_upload_output="$(run_cli file ssh upload --connection "${deny_connection_id}" --file "${deny_payload_file}" --to "${deny_remote_path}" 2>&1 || true)"
if [[ -z "${deny_upload_output}" ]]; then
  echo "expected SSH upload deny path to fail" >&2
  exit 1
fi
assert_output_contains "${deny_upload_output}" "File upload is disabled by organization policy" "SSH upload deny output"


deny_download_output="$(run_cli file ssh download --connection "${deny_connection_id}" --path "${deny_remote_path}" --dest "${deny_download_dir}" 2>&1 || true)"
if [[ -z "${deny_download_output}" ]]; then
  echo "expected SSH download deny path to fail" >&2
  exit 1
fi
assert_output_contains "${deny_download_output}" "File download is disabled by organization policy" "SSH download deny output"

if [[ -e "${deny_download_dir}/${deny_remote_name}" ]]; then
  echo "expected SSH download deny path to leave no local artifact" >&2
  exit 1
fi

run_cli audit connection "${deny_connection_id}" -o json > "${tmp_dir}/ssh-deny-audit.json"
assert_actions_absent "${tmp_dir}/ssh-deny-audit.json" FILE_UPLOAD FILE_DOWNLOAD
deny_audit_records="$(jq -c '(.data // .)' "${tmp_dir}/ssh-deny-audit.json")"

absolute_path_upload_output="$(run_cli file ssh upload --connection "${ssh_absolute_connection_id}" --file "${ssh_payload_file}" --to "/tmp/managed-ssh-${smoke_suffix}.txt" 2>&1 || true)"
if [[ -z "${absolute_path_upload_output}" ]]; then
  echo "expected absolute path upload to fail" >&2
  exit 1
fi
assert_output_contains "${absolute_path_upload_output}" "Only sandbox-relative paths are allowed; remote filesystem browsing is disabled." "absolute path upload output"

absolute_path_list_output="$(run_cli file ssh list --connection "${ssh_absolute_connection_id}" --path "/" 2>&1 || true)"
if [[ -z "${absolute_path_list_output}" ]]; then
  echo "expected absolute path list to fail" >&2
  exit 1
fi
assert_output_contains "${absolute_path_list_output}" "Only sandbox-relative paths are allowed; remote filesystem browsing is disabled." "absolute path list output"

run_cli audit connection "${ssh_absolute_connection_id}" -o json > "${tmp_dir}/ssh-absolute-audit.json"
assert_actions_absent "${tmp_dir}/ssh-absolute-audit.json" FILE_UPLOAD FILE_DOWNLOAD

cat > "${evidence_dir}/task-8-sandbox-docs-smoke.txt" <<EOF
managed-file-smoke-ok
ssh-connection=${ssh_connection_id}
rdp-connection=${rdp_connection_id}
cleanup-after-success-default=true
rdp-retention-policy=enabled
  history-connection=${rdp_retention_connection_id}
  history-command=file history list --connection ${rdp_retention_connection_id}
  history-id=${history_id}
  history-file=${rdp_retain_file_name}
  history-download=${rdp_history_download_dir}/${rdp_retain_file_name}
  ssh-audit-actions=$(jq -r '.data[].action' "${tmp_dir}/ssh-audit.json" | tr '\n' ' ')
  retain-history-actions=$(jq -r '.data[].action' "${tmp_dir}/rdp-retain-audit.json" | tr '\n' ' ')
  rdp-audit-actions=$(jq -r '.data[].action' "${tmp_dir}/rdp-audit.json" | tr '\n' ' ')
  deny-audit-actions=$(jq -r '.data[].action' "${tmp_dir}/ssh-deny-audit.json" | tr '\n' ' ')
  absolute-audit-actions=$(jq -r '.data[].action' "${tmp_dir}/ssh-absolute-audit.json" | tr '\n' ' ')
  absolute-upload-output=$(printf '%s' "${absolute_path_upload_output}" | tr '\n' ' ')
  absolute-list-output=$(printf '%s' "${absolute_path_list_output}" | tr '\n' ' ')
EOF

echo "managed-file-smoke-ok"
