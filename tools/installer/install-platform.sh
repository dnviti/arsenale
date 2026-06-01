#!/usr/bin/env bash
# install-platform.sh — One-command installer for the Arsenale platform.
#
# Downloads the published Ansible installer bundle, installs prerequisites, and
# runs the production installer (deployment/ansible/playbooks/install.yml) — all
# without cloning the repository.
#
#   curl -fsSL https://raw.githubusercontent.com/dnviti/arsenale/main/tools/installer/install-platform.sh | bash
#
# Configuration (all optional) via environment variables:
#   ARSENALE_DOMAIN            Public domain (default: localhost)
#   ARSENALE_INSTALL_PASSWORD  Technician password that encrypts installer state
#                              (default: prompted, or random in non-interactive mode)
#   ARSENALE_VAULT_PASSWORD    Ansible Vault password (default: random, stored on disk)
#   ARSENALE_CAPABILITIES      installer_capabilities_csv (comma-separated)
#   ARSENALE_DIRECT_GATEWAY    installer_direct_gateway (true/false)
#   ARSENALE_ZERO_TRUST        installer_zero_trust (true/false)
#   ARSENALE_VERSION           Release to install (latest|cli-dev|vX.Y.Z; default: latest)
#   ARSENALE_SECRETS_FILE      Path to a filled SECRETS.env for optional OAuth/SMTP secrets
#   ARSENALE_INSTALL_DIR       Where the bundle is extracted (default: /opt/arsenale/installer)
#   ARSENALE_HOST              Target host for the install (default: localhost)
#   ARSENALE_DEPLOY_USER       SSH user on the target host (default: current user)
#   ARSENALE_NONINTERACTIVE    Set to 1 to never prompt (fail/auto-generate instead)
#   ARSENALE_SKIP_PREREQS      Set to 1 to skip auto-installing system prerequisites
#   ARSENALE_REPO              GitHub repo (default: dnviti/arsenale)

set -euo pipefail

repo="${ARSENALE_REPO:-dnviti/arsenale}"
version="${ARSENALE_VERSION:-latest}"
install_dir="${ARSENALE_INSTALL_DIR:-/opt/arsenale/installer}"
domain="${ARSENALE_DOMAIN:-localhost}"
target_host="${ARSENALE_HOST:-localhost}"
noninteractive="${ARSENALE_NONINTERACTIVE:-0}"
skip_prereqs="${ARSENALE_SKIP_PREREQS:-0}"
need_become_pass=0
tmp_dir="$(mktemp -d)"

cleanup() {
  rm -rf "$tmp_dir"
}
trap cleanup EXIT

log() { printf '==> %s\n' "$1"; }
warn() { printf 'warning: %s\n' "$1" >&2; }
die() {
  printf 'error: %s\n' "$1" >&2
  exit 1
}

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || die "required command not found: $1"
}

# Run a command as root: directly if already root, otherwise via sudo.
as_root() {
  if [ "$(id -u)" -eq 0 ]; then
    "$@"
  else
    sudo "$@"
  fi
}

normalize_os() {
  case "$(uname -s)" in
    Linux) printf 'linux' ;;
    Darwin) printf 'darwin' ;;
    *) die "unsupported operating system: $(uname -s)" ;;
  esac
}

normalize_arch() {
  case "$(uname -m)" in
    x86_64 | amd64) printf 'amd64' ;;
    arm64 | aarch64) printf 'arm64' ;;
    *) die "unsupported CPU architecture: $(uname -m)" ;;
  esac
}

latest_version() {
  curl -fsSL "https://api.github.com/repos/${repo}/releases/latest" |
    sed -nE 's/.*"tag_name"[[:space:]]*:[[:space:]]*"v?([^"]+)".*/\1/p' |
    head -1
}

is_development_version() {
  case "$1" in
    dev | develop | development | cli-dev) return 0 ;;
    *) return 1 ;;
  esac
}

# Resolves $1 into release_tag / archive_version / display_version.
resolve_release() {
  local requested="$1"
  release_tag=""
  archive_version=""
  display_version=""

  if [ -z "$requested" ] || [ "$requested" = "latest" ]; then
    local resolved
    resolved="$(latest_version)"
    [ -n "$resolved" ] || die "could not resolve latest Arsenale release"
    release_tag="v${resolved}"
    archive_version="$resolved"
    display_version="$resolved"
    return
  fi

  if is_development_version "$requested"; then
    release_tag="cli-dev"
    archive_version="develop"
    display_version="develop"
    return
  fi

  local normalized="${requested#v}"
  release_tag="v${normalized}"
  archive_version="$normalized"
  display_version="$normalized"
}

verify_checksum() {
  local archive_name="$1"
  local checksums_file="$2"
  local expected
  expected="$(awk -v archive="$archive_name" '$2 == archive || $2 == "*" archive { print $1; exit }' "$checksums_file")"
  [ -n "$expected" ] || die "checksum for ${archive_name} not found"

  local actual
  if command -v sha256sum >/dev/null 2>&1; then
    actual="$(sha256sum "$tmp_dir/$archive_name" | awk '{print $1}')"
  else
    require_cmd shasum
    actual="$(shasum -a 256 "$tmp_dir/$archive_name" | awk '{print $1}')"
  fi

  [ "$actual" = "$expected" ] || die "checksum mismatch for ${archive_name}"
}

# ── Prerequisites ────────────────────────────────────────────────────────────

detect_pkg_mgr() {
  for mgr in apt-get dnf pacman brew; do
    if command -v "$mgr" >/dev/null 2>&1; then
      printf '%s' "$mgr"
      return
    fi
  done
  printf ''
}

install_packages() {
  local mgr="$1"
  shift
  case "$mgr" in
    apt-get)
      as_root apt-get update -y
      as_root apt-get install -y "$@"
      ;;
    dnf)
      as_root dnf install -y "$@"
      ;;
    pacman)
      as_root pacman -Sy --noconfirm "$@"
      ;;
    brew)
      brew install "$@"
      ;;
  esac
}

ensure_prerequisites() {
  if [ "$skip_prereqs" = "1" ]; then
    log "Skipping prerequisite auto-install (ARSENALE_SKIP_PREREQS=1)."
    require_cmd ansible-playbook
    require_cmd ansible-galaxy
    return
  fi

  local mgr
  mgr="$(detect_pkg_mgr)"
  local -a missing=()
  command -v ansible-playbook >/dev/null 2>&1 || missing+=(ansible)
  command -v podman >/dev/null 2>&1 || missing+=(podman)
  command -v openssl >/dev/null 2>&1 || missing+=(openssl)
  command -v git >/dev/null 2>&1 || missing+=(git)
  command -v ssh >/dev/null 2>&1 || missing+=(openssh-client)
  # A localhost install connects over SSH to localhost, so it also needs an SSH
  # server (sshd), not just the client.
  if [ "$target_host" = "localhost" ] || [ "$target_host" = "127.0.0.1" ]; then
    if ! command -v sshd >/dev/null 2>&1 && [ ! -x /usr/sbin/sshd ] && [ ! -x /usr/bin/sshd ]; then
      missing+=(openssh-server)
    fi
  fi

  if [ "${#missing[@]}" -eq 0 ]; then
    log "All system prerequisites already present."
    return
  fi

  [ -n "$mgr" ] || die "no supported package manager (apt-get/dnf/pacman/brew) found; install ${missing[*]} manually or set ARSENALE_SKIP_PREREQS=1"

  log "Installing prerequisites via ${mgr}: ${missing[*]}"

  # Map generic names to package names per manager.
  local -a pkgs=()
  local item
  for item in "${missing[@]}"; do
    case "$item:$mgr" in
      ansible:apt-get) pkgs+=(ansible) ;;
      ansible:dnf) pkgs+=(ansible-core) ;;
      ansible:pacman) pkgs+=(ansible) ;;
      ansible:brew) pkgs+=(ansible) ;;
      openssh-client:apt-get) pkgs+=(openssh-client) ;;
      openssh-client:dnf) pkgs+=(openssh-clients) ;;
      openssh-client:pacman) pkgs+=(openssh) ;;
      openssh-client:brew) pkgs+=(openssh) ;;
      openssh-server:apt-get) pkgs+=(openssh-server) ;;
      openssh-server:dnf) pkgs+=(openssh-server) ;;
      openssh-server:pacman) pkgs+=(openssh) ;;
      openssh-server:brew) pkgs+=(openssh) ;;
      *) pkgs+=("$item") ;;
    esac
  done

  install_packages "$mgr" "${pkgs[@]}"

  command -v ansible-playbook >/dev/null 2>&1 || die "ansible installation failed; install Ansible manually and re-run"
}

# ── SSH-to-target + sudo preflight ───────────────────────────────────────────
# The production play uses connection: ssh to the target host and become: true.
# For a localhost install we must be able to ssh to localhost non-interactively
# and run passwordless sudo.

ensure_local_ssh() {
  [ "$target_host" = "localhost" ] || [ "$target_host" = "127.0.0.1" ] || return 0

  if ssh -o BatchMode=yes -o ConnectTimeout=5 localhost true >/dev/null 2>&1; then
    log "Passwordless SSH to localhost confirmed."
    return
  fi

  log "Configuring passwordless SSH to localhost..."
  local key="$HOME/.ssh/id_ed25519"
  mkdir -p "$HOME/.ssh"
  chmod 700 "$HOME/.ssh"
  [ -f "$key" ] || ssh-keygen -t ed25519 -N "" -f "$key" >/dev/null
  if ! grep -qF "$(cat "$key.pub")" "$HOME/.ssh/authorized_keys" 2>/dev/null; then
    cat "$key.pub" >>"$HOME/.ssh/authorized_keys"
  fi
  chmod 600 "$HOME/.ssh/authorized_keys"

  # Ensure sshd is running where we manage it.
  if ! ssh -o BatchMode=yes -o ConnectTimeout=5 localhost true >/dev/null 2>&1; then
    if command -v systemctl >/dev/null 2>&1; then
      as_root systemctl enable --now ssh 2>/dev/null ||
        as_root systemctl enable --now sshd 2>/dev/null || true
    fi
  fi

  ssh -o BatchMode=yes -o ConnectTimeout=5 localhost true >/dev/null 2>&1 ||
    die "cannot SSH to localhost non-interactively. Ensure an SSH server is running and your key is authorized, then re-run."
}

ensure_sudo() {
  if [ "$(id -u)" -eq 0 ] || sudo -n true >/dev/null 2>&1; then
    log "Root privileges available without a password prompt."
    return
  fi
  if [ "$noninteractive" = "1" ]; then
    die "passwordless sudo is required for a non-interactive install but is unavailable. Configure NOPASSWD sudo for $(id -un) and re-run."
  fi
  warn "The installer needs sudo to install Podman and create the arsenale user."
  sudo -v || die "sudo authentication failed"
  # Ansible's become step runs under its own session, so a cached sudo timestamp
  # is not enough — have ansible-playbook prompt for the become password too.
  need_become_pass=1
}

# ── Download + extract bundle ────────────────────────────────────────────────

download_bundle() {
  resolve_release "$version"
  local base="https://github.com/${repo}/releases/download/${release_tag}"
  local archive="arsenale-installer_${archive_version}.tar.gz"

  log "Downloading Arsenale installer ${display_version}..."
  curl -fsSLo "$tmp_dir/$archive" "${base}/${archive}"
  curl -fsSLo "$tmp_dir/checksums_sha256.txt" "${base}/checksums_sha256.txt"
  verify_checksum "$archive" "$tmp_dir/checksums_sha256.txt"

  log "Extracting bundle to ${install_dir}..."
  as_root mkdir -p "$install_dir"
  # The archive root is deployment/ansible/; strip both components so playbooks/,
  # roles/, inventory/ land directly under $install_dir.
  as_root tar -xzf "$tmp_dir/$archive" -C "$install_dir" --strip-components=2

  # Re-own to the invoking user so subsequent steps (vault gen, ansible) don't
  # need root inside the bundle.
  if [ "$(id -u)" -ne 0 ]; then
    as_root chown -R "$(id -u):$(id -g)" "$install_dir"
  fi

  # Expose post-install operations (status/backup/rotate/...) without the repo.
  if [ -f "$install_dir/Makefile.bundle" ]; then
    cp "$install_dir/Makefile.bundle" "$install_dir/Makefile"
  fi
}

# ── Secrets (idempotent — create-if-absent) ──────────────────────────────────

prepare_secrets() {
  local vault_pass_file="$install_dir/.vault-pass"
  local vault_file="$install_dir/inventory/group_vars/all/vault.yml"
  local secrets_file="$install_dir/SECRETS.env"
  local password_file="$install_dir/install/password.txt"

  # Vault password file — reused across runs so the encrypted vault stays readable.
  if [ ! -f "$vault_pass_file" ]; then
    if [ -n "${ARSENALE_VAULT_PASSWORD:-}" ]; then
      printf '%s' "$ARSENALE_VAULT_PASSWORD" >"$vault_pass_file"
    else
      openssl rand -hex 32 >"$vault_pass_file"
    fi
    chmod 600 "$vault_pass_file"
  fi
  export ANSIBLE_VAULT_PASSWORD_FILE="$vault_pass_file"

  # Optional OAuth/SMTP secrets supplied by the user. Merge the non-empty values
  # over the existing SECRETS.env (or the template on a fresh install) so the
  # already-generated required secrets (JWT/postgres/...) are preserved rather
  # than overwritten, then force a vault refresh so the new values take effect.
  local secrets_updated=0
  if [ -n "${ARSENALE_SECRETS_FILE:-}" ]; then
    [ -f "$ARSENALE_SECRETS_FILE" ] || die "ARSENALE_SECRETS_FILE not found: $ARSENALE_SECRETS_FILE"
    local base="$secrets_file"
    [ -f "$base" ] || base="$install_dir/SECRETS.env.example"
    python3 - "$base" "$ARSENALE_SECRETS_FILE" "$secrets_file" <<'PY'
import sys
base, overlay, out = sys.argv[1], sys.argv[2], sys.argv[3]
def load(path):
    pairs = {}
    try:
        with open(path) as fh:
            for line in fh:
                s = line.strip()
                if not s or s.startswith("#") or "=" not in s:
                    continue
                k, v = s.split("=", 1)
                pairs[k] = v
    except FileNotFoundError:
        pass
    return pairs
merged = load(base)
for key, value in load(overlay).items():
    if value != "":
        merged[key] = value
with open(out, "w") as fh:
    for key, value in merged.items():
        fh.write(f"{key}={value}\n")
PY
    chmod 600 "$secrets_file"
    secrets_updated=1
  fi

  # Generate the encrypted vault on first run, or when new secrets were supplied.
  # generate-vault.sh keeps non-empty required secrets as-is, so a refresh adds
  # the new values without rotating JWT/postgres/... on a live stack.
  if [ ! -f "$vault_file" ]; then
    log "Generating secrets and encrypting the Ansible Vault..."
    (cd "$install_dir" && ./scripts/generate-vault.sh)
  elif [ "$secrets_updated" = "1" ]; then
    log "Refreshing the encrypted vault with the supplied secrets..."
    (cd "$install_dir" && ./scripts/generate-vault.sh)
  else
    log "Existing vault found — reusing current secrets."
  fi

  # Technician (installer) password file.
  if [ ! -f "$password_file" ]; then
    mkdir -p "$install_dir/install"
    if [ -n "${ARSENALE_INSTALL_PASSWORD:-}" ]; then
      printf '%s' "$ARSENALE_INSTALL_PASSWORD" >"$password_file"
    elif [ "$noninteractive" = "1" ]; then
      (umask 177 && openssl rand -hex 24 >"$password_file")
      # Do not echo the secret (it would land in shell history / CI logs); point
      # the operator at the protected file instead.
      warn "Generated a random technician password and saved it to ${password_file} (mode 0600)."
      warn "Copy it somewhere safe — it is required to reconfigure or recover this install."
    else
      local pw1 pw2
      printf 'Set a technician password (encrypts installer state): '
      read -rs pw1
      printf '\nConfirm technician password: '
      read -rs pw2
      printf '\n'
      [ "$pw1" = "$pw2" ] || die "passwords did not match"
      [ -n "$pw1" ] || die "technician password cannot be empty"
      printf '%s' "$pw1" >"$password_file"
    fi
    chmod 600 "$password_file"
  fi
}

# ── Run the real installer ───────────────────────────────────────────────────

run_installer() {
  local -a extra=(
    --vault-password-file "$install_dir/.vault-pass"
    -e "install_password_file=$install_dir/install/password.txt"
    -e "installer_mode=production"
    -e "installer_backend=podman"
    -e "arsenale_domain=${domain}"
    -e "arsenale_public_url=https://${domain}"
  )
  [ -n "${ARSENALE_CAPABILITIES:-}" ] && extra+=(-e "installer_capabilities_csv=${ARSENALE_CAPABILITIES}")
  [ -n "${ARSENALE_DIRECT_GATEWAY:-}" ] && extra+=(-e "installer_direct_gateway=${ARSENALE_DIRECT_GATEWAY}")
  [ -n "${ARSENALE_ZERO_TRUST:-}" ] && extra+=(-e "installer_zero_trust=${ARSENALE_ZERO_TRUST}")
  # Without passwordless sudo, ansible's become step needs the sudo password.
  [ "$need_become_pass" = "1" ] && extra+=(--ask-become-pass)

  export ARSENALE_HOST="$target_host"
  [ -n "${ARSENALE_DEPLOY_USER:-}" ] && export ARSENALE_DEPLOY_USER

  log "Running the Arsenale production installer..."
  (cd "$install_dir" && ansible-playbook playbooks/install.yml "${extra[@]}")
}

# ── Main ─────────────────────────────────────────────────────────────────────

main() {
  require_cmd curl
  require_cmd awk
  require_cmd grep
  require_cmd head
  require_cmd sed
  require_cmd tar

  local os
  os="$(normalize_os)"
  normalize_arch >/dev/null
  if [ "$os" = "darwin" ]; then
    die "the podman platform install is Linux-only. Run it on a Linux host, or use the development workflow (make dev) on macOS."
  fi

  ensure_prerequisites
  ensure_sudo
  ensure_local_ssh
  download_bundle
  log "Installing Ansible Galaxy collections..."
  # Do not suppress output: on a network/auth failure the operator needs the
  # error detail (set -e aborts the script otherwise with no context).
  (cd "$install_dir" && ansible-galaxy collection install -r requirements.yml)
  prepare_secrets
  run_installer

  # Default client port from vars.yml is 3000.
  local port="${ARSENALE_CLIENT_PORT:-3000}"
  printf '\n'
  log "Arsenale is installed."
  printf '    Open:        https://%s:%s\n' "$domain" "$port"
  printf '    Operations:  cd %s && make status   (or: arsenale ...)\n' "$install_dir"
  printf '    Re-run this command any time to upgrade or reconfigure.\n'
}

main "$@"
