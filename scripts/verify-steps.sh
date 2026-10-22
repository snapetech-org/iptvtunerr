#!/usr/bin/env bash
# Verification: format → vet → test → build. Fail fast, fail noisy.
# CI runs this via scripts/verify. Keep memory-bank/commands.yml in sync.

set -e
cd "$(dirname "$0")/.."
ROOT="$PWD"

err() { echo "[iptv-tunerr verify] ERROR: $*" >&2; exit 1; }
step() { echo "[iptv-tunerr verify] ==> $*"; }

# --- Release channel parity ---
step "release-channel wiring (scripts/check-release-channel-wiring.sh)"
if ! bash ./scripts/check-release-channel-wiring.sh; then
  err "release-channel wiring contract failed"
fi

# --- Immutable GitHub Actions references ---
step "workflow action pinning (scripts/check-workflow-action-pins.py)"
if ! python3 ./scripts/check-workflow-action-pins.py; then
  err "workflow actions must use immutable, version-documented commit SHAs"
fi

# --- Format (fail if any file needs formatting; vendor/ is excluded) ---
step "format (gofmt -s -l)"
# Prune vendored trees and nested Git worktrees so local agent checkouts do not
# become part of the repository's formatting gate.
UNFORMATTED=$(find . -type d \( -name vendor -o -name worktrees -o -name .git \) -prune -o \
  -type f -name '*.go' -print0 \
  | xargs -0 gofmt -s -l 2>/dev/null | grep -v '^$' || true)
if [[ -n "$UNFORMATTED" ]]; then
  echo "The following files need 'gofmt -s -w':"
  echo "$UNFORMATTED"
  err "format check failed — run: gofmt -s -w ."
fi

# --- Vet ---
step "vet (go vet ./...)"
if ! go vet ./...; then
  err "vet failed"
fi

# --- Harness / helper script syntax (no network; catches edit typos in CI) ---
step "script syntax (bash -n, py_compile, and node --check)"
shopt -s nullglob
for f in "$ROOT"/scripts/*.sh; do
  bash -n "$f" || err "bash -n failed on $f"
done
for f in "$ROOT"/.githooks/*; do
  [[ -f "$f" ]] || continue
  bash -n "$f" || err "bash -n failed on $f"
done
for f in "$ROOT"/packaging/scripts/*.sh; do
  bash -n "$f" || err "bash -n failed on $f"
done
command -v python3 >/dev/null 2>&1 || err "python3 required for scripts/*.py syntax check (py_compile)"
for f in "$ROOT"/scripts/*.py; do
  python3 -m py_compile "$f" || err "py_compile failed on $f"
done
for f in "$ROOT"/scripts/*.mjs; do
  node --check "$f" || err "node --check failed on $f"
done
step "Python support-script tests"
python3 -m unittest discover -s "$ROOT/scripts/tests" -p 'test_*.py' || err "Python support-script tests failed"
step "Node release-script tests"
node --test "$ROOT/scripts/check-changelog-tags.test.mjs" || err "Node release-script tests failed"

# --- Test ---
step "test (go test ./...)"
# -count=1 avoids cache so CI always runs tests; -short allows skipping slow tests later
if ! go test -count=1 ./...; then
  err "tests failed"
fi

# --- Build ---
step "build (go build ./cmd/iptv-tunerr)"
if ! go build -o /dev/null ./cmd/iptv-tunerr; then
  err "build failed"
fi

# --- Linux ARMv7 compile regression ---
step "linux ARMv7 compile (vendored go-fuse splice)"
if ! GOOS=linux GOARCH=arm GOARM=7 go build -mod=vendor -o /dev/null github.com/hanwen/go-fuse/v2/splice; then
  err "linux ARMv7 vendored go-fuse compile failed"
fi

# --- Binary smoke ---
step "binary smoke (scripts/ci-smoke.sh)"
if ! bash ./scripts/ci-smoke.sh; then
  err "binary smoke failed"
fi

echo "[iptv-tunerr verify] ==> all steps OK"
