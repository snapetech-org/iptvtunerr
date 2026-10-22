#!/usr/bin/env bash
set -euo pipefail

if [[ $# -eq 0 ]]; then
  echo "usage: $0 <tool-or-group>..." >&2
  exit 2
fi

have() {
  command -v "$1" >/dev/null 2>&1
}

can_run_as_root() {
  if (( EUID == 0 )); then
    return 0
  fi

  have sudo && sudo -n true >/dev/null 2>&1
}

run_as_root() {
  if (( EUID == 0 )); then
    "$@"
  else
    sudo -n "$@"
  fi
}

install_unprivileged_linux_tool() {
  # Keep the no-root JIT path reproducible: these versions and digests come
  # from the corresponding upstream GitHub release assets.
  local tool="$1"
  local machine_arch
  local asset
  local asset_dir
  local checksum
  local version
  local url
  local command_name

  machine_arch="$(uname -m)"
  case "$tool:$machine_arch" in
    jq:x86_64|jq:amd64)
      version="1.8.2"
      asset="jq-linux-amd64"
      checksum="b1c22172dd303f3be49e935aa56aa48a8b7a46e0bc838b4997d3bb451495870f"
      url="https://github.com/jqlang/jq/releases/download/jq-${version}/${asset}"
      command_name="jq"
      ;;
    jq:aarch64|jq:arm64)
      version="1.8.2"
      asset="jq-linux-arm64"
      checksum="8b85c817833814ddca00a144c33705546355afccf0cf39b188f3cdb48b852309"
      url="https://github.com/jqlang/jq/releases/download/jq-${version}/${asset}"
      command_name="jq"
      ;;
    ripgrep:x86_64|ripgrep:amd64)
      version="15.2.0"
      asset="ripgrep-${version}-x86_64-unknown-linux-musl.tar.gz"
      asset_dir="ripgrep-${version}-x86_64-unknown-linux-musl"
      checksum="33e15bcf1624b25cdd2a55813a47a2f95dbe126268203e76aa6a585d1e7b149c"
      url="https://github.com/BurntSushi/ripgrep/releases/download/${version}/${asset}"
      command_name="rg"
      ;;
    ripgrep:aarch64|ripgrep:arm64)
      version="15.2.0"
      asset="ripgrep-${version}-aarch64-unknown-linux-musl.tar.gz"
      asset_dir="ripgrep-${version}-aarch64-unknown-linux-musl"
      checksum="800b1e7206afe799dfb5a6901f23147cfaabe0e52210538100f61e86e1740915"
      url="https://github.com/BurntSushi/ripgrep/releases/download/${version}/${asset}"
      command_name="rg"
      ;;
    *)
      return 1
      ;;
  esac

  local runner_temp="${RUNNER_TEMP:-${TMPDIR:-/tmp}}"
  local tool_root="${runner_temp%/}/iptvtunerr-ci-tools"
  local tool_bin="${tool_root}/bin"
  local download_dir
  local download_path
  local destination="${tool_bin}/${command_name}"

  mkdir -p "$tool_bin"
  if [[ -L "$tool_root" || -L "$tool_bin" || ! -O "$tool_root" || ! -O "$tool_bin" ]]; then
    echo "RUNNER_TEMP tool directory must be owned by this job and cannot be a symlink." >&2
    return 1
  fi
  chmod 700 "$tool_root" "$tool_bin"
  download_dir="$(mktemp -d "${runner_temp%/}/iptvtunerr-tools.XXXXXX")"
  download_path="${download_dir}/${asset}"

  if ! curl --fail --location --silent --show-error --retry 3 --retry-connrefused --max-time 90 \
    "$url" --output "$download_path"; then
    echo "Could not download pinned ${tool} ${version}." >&2
    rm -rf -- "$download_dir"
    return 1
  fi

  if ! printf '%s  %s\n' "$checksum" "$download_path" | sha256sum --check --status; then
    echo "Checksum validation failed for pinned ${tool} ${version}." >&2
    rm -rf -- "$download_dir"
    return 1
  fi

  if [[ "$tool" == "ripgrep" ]]; then
    if ! tar --extract --gzip --to-stdout --file "$download_path" "${asset_dir}/rg" \
      > "${download_dir}/rg" || [[ ! -s "${download_dir}/rg" ]]; then
      echo "Could not extract pinned ripgrep ${version}." >&2
      rm -rf -- "$download_dir"
      return 1
    fi
    download_path="${download_dir}/rg"
  fi

  if ! install -m 0755 "$download_path" "$destination"; then
    echo "Could not install pinned ${tool} ${version} in RUNNER_TEMP." >&2
    rm -rf -- "$download_dir"
    return 1
  fi

  rm -rf -- "$download_dir"
  echo "Installed pinned ${tool} ${version} in RUNNER_TEMP."
}

missing=()
for tool in "$@"; do
  case "$tool" in
    ripgrep)
      have rg || missing+=("ripgrep")
      ;;
    jq)
      have jq || missing+=("jq")
      ;;
    zip)
      have zip || missing+=("zip")
      ;;
    curl)
      have curl || missing+=("curl")
      ;;
    gh)
      have gh || missing+=("gh")
      ;;
    copr-cli)
      have copr-cli || missing+=("copr-cli")
      ;;
    gitleaks)
      have gitleaks || missing+=("gitleaks")
      ;;
    rpm)
      have rpmbuild || missing+=("rpm")
      ;;
    python3-pip)
      have pip3 || have pip || missing+=("python3-pip")
      ;;
    debian-packaging)
      have dpkg-source || missing+=("dpkg")
      have dpkg-genchanges || missing+=("dpkg")
      have gpg || missing+=("gnupg")
      have curl || missing+=("curl")
      ;;
    *)
      missing+=("$tool")
      ;;
  esac
done

if [[ ${#missing[@]} -eq 0 ]]; then
  exit 0
fi

dedup=()
for pkg in "${missing[@]}"; do
  found=0
  for existing in "${dedup[@]}"; do
    if [[ "$existing" == "$pkg" ]]; then
      found=1
      break
    fi
  done
  [[ "$found" == "1" ]] || dedup+=("$pkg")
done

if printf '%s\n' "${dedup[@]}" | grep -Fxq gitleaks && have go; then
  go install github.com/zricethezav/gitleaks/v8@v8.24.3
  gopath="$(go env GOPATH)"
  export PATH="${gopath}/bin:${PATH}"
  if [[ -n "${GITHUB_PATH:-}" ]]; then
    echo "${gopath}/bin" >> "$GITHUB_PATH"
  fi
  filtered=()
  for pkg in "${dedup[@]}"; do
    [[ "$pkg" == "gitleaks" ]] || filtered+=("$pkg")
  done
  dedup=("${filtered[@]}")
  [[ ${#dedup[@]} -gt 0 ]] || exit 0
fi

if printf '%s\n' "${dedup[@]}" | grep -Fxq copr-cli && have python3; then
  venv="${RUNNER_TEMP:-/tmp}/iptvtunerr-ci-tools"
  python3 -m venv "$venv"
  "$venv/bin/python" -m pip install --upgrade pip
  "$venv/bin/python" -m pip install copr-cli
  export PATH="${venv}/bin:${PATH}"
  if [[ -n "${GITHUB_PATH:-}" ]]; then
    echo "${venv}/bin" >> "$GITHUB_PATH"
  fi
  filtered=()
  for pkg in "${dedup[@]}"; do
    [[ "$pkg" == "copr-cli" ]] || filtered+=("$pkg")
  done
  dedup=("${filtered[@]}")
  [[ ${#dedup[@]} -gt 0 ]] || exit 0
fi

if ! can_run_as_root && [[ "$(uname -s)" == "Linux" ]]; then
  remaining=()
  for pkg in "${dedup[@]}"; do
    case "$pkg" in
      jq|ripgrep)
        if ! install_unprivileged_linux_tool "$pkg"; then
          remaining+=("$pkg")
        fi
        ;;
      *)
        remaining+=("$pkg")
        ;;
    esac
  done
  dedup=("${remaining[@]}")

  if [[ ${#dedup[@]} -eq 0 ]]; then
    tool_bin="${RUNNER_TEMP:-${TMPDIR:-/tmp}}/iptvtunerr-ci-tools/bin"
    export PATH="${tool_bin}:${PATH}"
    if [[ -n "${GITHUB_PATH:-}" ]]; then
      printf '%s\n' "$tool_bin" >> "$GITHUB_PATH"
    fi
    exit 0
  fi
fi

if have apt-get && can_run_as_root; then
  run_as_root apt-get update
  run_as_root apt-get install -y "${dedup[@]}"
elif have pacman && can_run_as_root; then
  run_as_root pacman -Sy --needed --noconfirm "${dedup[@]}"
elif have dnf && can_run_as_root; then
  run_as_root dnf install -y "${dedup[@]}"
elif have yum && can_run_as_root; then
  run_as_root yum install -y "${dedup[@]}"
elif have apk && can_run_as_root; then
  run_as_root apk add --no-cache "${dedup[@]}"
elif have brew; then
  brew install "${dedup[@]}"
else
  echo "Cannot install missing tools without non-interactive root access or a supported unprivileged download: ${dedup[*]}" >&2
  exit 1
fi
