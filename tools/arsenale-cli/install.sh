#!/usr/bin/env bash
set -euo pipefail

repo="${ARSENALE_REPO:-dnviti/arsenale}"
version="${ARSENALE_VERSION:-latest}"
install_dir="${ARSENALE_INSTALL_DIR:-}"
tmp_dir="$(mktemp -d)"

cleanup() {
  rm -rf "$tmp_dir"
}
trap cleanup EXIT

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    printf 'error: required command not found: %s\n' "$1" >&2
    exit 1
  fi
}

normalize_os() {
  case "$(uname -s)" in
    Linux) printf 'linux' ;;
    Darwin) printf 'darwin' ;;
    *)
      printf 'error: unsupported operating system: %s\n' "$(uname -s)" >&2
      exit 1
      ;;
  esac
}

normalize_arch() {
  case "$(uname -m)" in
    x86_64 | amd64) printf 'amd64' ;;
    arm64 | aarch64) printf 'arm64' ;;
    *)
      printf 'error: unsupported CPU architecture: %s\n' "$(uname -m)" >&2
      exit 1
      ;;
  esac
}

latest_version() {
  curl -fsSL "https://api.github.com/repos/${repo}/releases/latest" |
    sed -nE 's/.*"tag_name"[[:space:]]*:[[:space:]]*"v?([^"]+)".*/\1/p' |
    head -1
}

resolve_version() {
  local requested="$1"
  if [ "$requested" = "latest" ]; then
    local resolved
    resolved="$(latest_version)"
    if [ -z "$resolved" ]; then
      printf 'error: could not resolve latest Arsenale release\n' >&2
      exit 1
    fi
    printf '%s' "$resolved"
    return
  fi
  printf '%s' "${requested#v}"
}

default_install_dir() {
  if [ -w /usr/local/bin ]; then
    printf '/usr/local/bin'
  else
    printf '%s/.local/bin' "$HOME"
  fi
}

verify_checksum() {
  local archive_name="$1"
  local checksums_file="$2"
  local expected
  expected="$(awk -v archive="$archive_name" '$2 == archive { print $1; exit }' "$checksums_file")"
  if [ -z "$expected" ]; then
    printf 'error: checksum for %s not found\n' "$archive_name" >&2
    exit 1
  fi

  local actual
  if command -v sha256sum >/dev/null 2>&1; then
    actual="$(sha256sum "$tmp_dir/$archive_name" | awk '{print $1}')"
  else
    require_cmd shasum
    actual="$(shasum -a 256 "$tmp_dir/$archive_name" | awk '{print $1}')"
  fi

  if [ "$actual" != "$expected" ]; then
    printf 'error: checksum mismatch for %s\n' "$archive_name" >&2
    exit 1
  fi
}

append_profile_block() {
  local profile="$1"
  local marker="$2"
  local body="$3"

  if [ "${ARSENALE_SKIP_SHELL_PROFILE:-0}" = "1" ]; then
    return
  fi
  if [ -z "${HOME:-}" ] || [ -z "$profile" ]; then
    return
  fi
  if [ -f "$profile" ] && grep -Fq "$marker" "$profile"; then
    return
  fi

  local profile_dir
  profile_dir="${profile%/*}"
  if [ "$profile_dir" != "$profile" ]; then
    mkdir -p "$profile_dir"
  fi

  {
    printf '\n# %s\n' "$marker"
    printf '%s\n' "$body"
  } >>"$profile"
}

install_shell_completions() {
  local binary="$1"

  if [ "${ARSENALE_SKIP_COMPLETIONS:-0}" = "1" ]; then
    printf 'Skipped shell completion installation.\n'
    return
  fi
  if [ -z "${HOME:-}" ]; then
    printf 'Skipping shell completion installation because HOME is not set.\n' >&2
    return
  fi

  local data_home config_home bash_dir bash_file bash_profile zsh_dir zsh_file zshrc fish_dir fish_file
  data_home="${XDG_DATA_HOME:-$HOME/.local/share}"
  config_home="${XDG_CONFIG_HOME:-$HOME/.config}"

  bash_dir="$data_home/bash-completion/completions"
  bash_file="$bash_dir/arsenale"
  mkdir -p "$bash_dir"
  "$binary" completion bash >"$bash_file"
  chmod 0644 "$bash_file"
  bash_profile="$HOME/.bashrc"
  if [ "$(uname -s)" = "Darwin" ] && { [ -f "$HOME/.bash_profile" ] || [ ! -f "$HOME/.bashrc" ]; }; then
    bash_profile="$HOME/.bash_profile"
  fi
  append_profile_block "$bash_profile" "Arsenale CLI completion" "[ -r \"$bash_file\" ] && source \"$bash_file\""

  zsh_dir="$data_home/zsh/site-functions"
  zsh_file="$zsh_dir/_arsenale"
  mkdir -p "$zsh_dir"
  "$binary" completion zsh >"$zsh_file"
  chmod 0644 "$zsh_file"
  zshrc="${ZDOTDIR:-$HOME}/.zshrc"
  append_profile_block "$zshrc" "Arsenale CLI zsh completion" "fpath=(\"$zsh_dir\" \$fpath)
autoload -Uz compinit
compinit"

  fish_dir="$config_home/fish/completions"
  fish_file="$fish_dir/arsenale.fish"
  mkdir -p "$fish_dir"
  "$binary" completion fish >"$fish_file"
  chmod 0644 "$fish_file"

  printf 'Installed shell completions for bash, zsh, and fish.\n'
  if [ "${ARSENALE_SKIP_SHELL_PROFILE:-0}" = "1" ]; then
    printf 'Shell profile updates were skipped; source the completion files manually if needed.\n'
  fi
}

require_cmd curl
require_cmd awk
require_cmd grep
require_cmd head
require_cmd install
require_cmd sed
require_cmd tar

os="$(normalize_os)"
arch="$(normalize_arch)"
version="$(resolve_version "$version")"
download_base="https://github.com/${repo}/releases/download/v${version}"
archive_name="arsenale-cli_${version}_${os}_${arch}.tar.gz"

if [ -z "$install_dir" ]; then
  install_dir="$(default_install_dir)"
fi

printf 'Installing Arsenale CLI %s for %s/%s...\n' "$version" "$os" "$arch"
curl -fsSLo "$tmp_dir/$archive_name" "${download_base}/${archive_name}"
curl -fsSLo "$tmp_dir/checksums_sha256.txt" "${download_base}/checksums_sha256.txt"
verify_checksum "$archive_name" "$tmp_dir/checksums_sha256.txt"

tar -xzf "$tmp_dir/$archive_name" -C "$tmp_dir"
mkdir -p "$install_dir"
install -m 0755 "$tmp_dir/arsenale" "$install_dir/arsenale"
install_shell_completions "$install_dir/arsenale"

printf 'Installed: %s/arsenale\n' "$install_dir"
if ! command -v arsenale >/dev/null 2>&1 && ! printf '%s' ":$PATH:" | grep -Fq ":${install_dir}:"; then
  printf 'Add %s to PATH to run arsenale from any shell.\n' "$install_dir"
fi
