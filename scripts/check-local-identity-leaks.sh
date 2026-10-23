#!/usr/bin/env bash
set -euo pipefail

script_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
repo_root="$(cd "${LOCAL_IDENTITY_SCAN_ROOT:-$script_root}" && pwd)"
cd "$repo_root"

if ! command -v python3 >/dev/null 2>&1; then
  echo "check-local-identity-leaks: python3 is required." >&2
  exit 127
fi

tmp_tokens="$(mktemp)"
tmp_commits="$(mktemp)"
tmp_files="$(mktemp)"
trap 'rm -f "$tmp_tokens" "$tmp_commits" "$tmp_files"' EXIT

add_token() {
  local token="$1"
  token="${token//$'\n'/}"
  token="${token//$'\r'/}"
  [[ ${#token} -ge 3 ]] || return 0
  case "$token" in
    root|runner|build|agent|agents|github|actions|action|node|npm|yarn|pnpm|dotnet|cargo|target|debug|release)
      return 0
      ;;
  esac
  printf '%s\n' "$token" >>"$tmp_tokens"
}

add_token "${LOCAL_IDENTITY_DENYLIST:-}"
add_token "${SLSKDN_LOCAL_IDENTITY_DENYLIST:-}"
add_token "${SLSKDN_FORBIDDEN_LOCAL_HOSTNAME:-}"
add_token "$(hostname -s 2>/dev/null || true)"
add_token "${USER:-}"
add_token "$(id -un 2>/dev/null || true)"
add_token "$(basename "${HOME:-}" 2>/dev/null || true)"

read_csv_tokens() {
  local value="$1"
  [[ -n "$value" ]] || return 0
  IFS=',' read -ra tokens <<<"$value"
  for token in "${tokens[@]}"; do
    add_token "$token"
  done
}

read_csv_tokens "${LOCAL_IDENTITY_DENYLIST:-}"
read_csv_tokens "${SLSKDN_LOCAL_IDENTITY_DENYLIST:-}"

read_denylist_file() {
  local file="$1"
  [[ -n "$file" && -f "$file" ]] || return 0
  while IFS= read -r token; do
    [[ "$token" =~ ^[[:space:]]*# ]] && continue
    add_token "$token"
  done <"$file"
}

read_denylist_file "${LOCAL_IDENTITY_DENYLIST_FILE:-}"
read_denylist_file "${SLSKDN_LOCAL_IDENTITY_DENYLIST_FILE:-}"

sort -u "$tmp_tokens" -o "$tmp_tokens"
if [[ ! -s "$tmp_tokens" ]]; then
  echo "No local identity tokens configured for scanning."
  exit 0
fi

failed=0

check_file() {
  local label="$1"
  local path="$2"
  local display_path="${3:-$path}"
  local status

  [[ -f "$path" ]] || return 0

  if python3 "$script_root/scripts/scan-local-identity-file.py" \
    --tokens-file "$tmp_tokens" \
    --label "$label" \
    --display-path "$display_path" \
    "$path"; then
    return 0
  else
    status=$?
  fi

  if [[ "$status" -eq 1 ]]; then
    failed=1
  else
    exit "$status"
  fi
}

if [[ -f docs/CHANGELOG.md ]]; then
  tmp_unreleased="$(mktemp)"
  trap 'rm -f "$tmp_tokens" "$tmp_commits" "$tmp_files" "$tmp_unreleased"' EXIT
  awk '
    $0 == "## [Unreleased]" { in_section = 1; next }
    in_section && /^## \[/ { exit }
    in_section { print }
  ' docs/CHANGELOG.md >"$tmp_unreleased"
  check_file "docs/CHANGELOG.md Unreleased" "$tmp_unreleased" "docs/CHANGELOG.md#Unreleased"
fi

if [[ "${LOCAL_IDENTITY_SCAN_COMMITS:-0}" == "1" ]]; then
  scan_base="${LOCAL_IDENTITY_SCAN_BASE:-}"
  scan_commit_count="${LOCAL_IDENTITY_SCAN_COMMIT_COUNT:-}"

  if [[ "$scan_base" =~ ^0+$ ]]; then
    scan_base=""
  fi

  if [[ -n "$scan_base" ]]; then
    if ! git rev-parse --verify --quiet "${scan_base}^{commit}" >/dev/null; then
      echo "check-local-identity-leaks: configured commit scan base is unavailable." >&2
      exit 2
    fi
    range_base="$(git merge-base "$scan_base" HEAD)" || {
      echo "check-local-identity-leaks: configured commit scan base has no common history with HEAD." >&2
      exit 2
    }
    git log --format='%s%n%b' "${range_base}..HEAD" >"$tmp_commits"
    check_file "recent commit messages" "$tmp_commits" "git log"
  elif [[ -n "$scan_commit_count" ]]; then
    if [[ ! "$scan_commit_count" =~ ^[1-9][0-9]{0,4}$ ]] || (( scan_commit_count > 10000 )); then
      echo "check-local-identity-leaks: commit scan count must be between 1 and 10000." >&2
      exit 2
    fi
    git log --max-count="$scan_commit_count" --format='%s%n%b' HEAD >"$tmp_commits"
    check_file "recent commit messages" "$tmp_commits" "git log"
  else
    latest_tag="$(git tag --sort=-creatordate --list 'build-main-*' | head -n 1 || true)"
    if [[ -z "$latest_tag" ]]; then
      latest_tag="$(git describe --tags --abbrev=0 2>/dev/null || true)"
    fi
    if [[ -n "$latest_tag" ]]; then
      git log --format='%s%n%b' "${latest_tag}..HEAD" >"$tmp_commits"
      check_file "recent commit messages" "$tmp_commits" "git log"
    fi
  fi
fi

find . \
  -path './.git' -prune -o \
  -path './node_modules' -prune -o \
  -path './vendor' -prune -o \
  -path './target' -prune -o \
  -path './dist' -prune -o \
  -path './build' -prune -o \
  -path './zeek/pkg' -prune -o \
  -type f \( \
    -path './release-notes/*' -o \
    -path './.github/release-notes/*' -o \
    -path './docs/dev/release-copy.md' -o \
    -path './docs/release*.md' -o \
    -path './docs/RELEASE*.md' -o \
    -path './packaging/winget/*' \
  \) -print | sed 's#^./##' >"$tmp_files"

while IFS= read -r path; do
  [[ -n "$path" ]] || continue
  check_file "$path" "$path"
done <"$tmp_files"

if [[ "$failed" -ne 0 ]]; then
  echo "Local hostname/username leaked into release-facing text. Use generic wording like 'live validation host' or 'operator account'." >&2
  exit 1
fi

echo "No local hostname or username leaks found in release-facing text."
