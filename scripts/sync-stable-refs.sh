#!/usr/bin/env bash
# Keep the stable pointers aligned with the newest stable release tag (vX.Y.Z):
#   - master        fast-forwarded to the stable tag's commit (created if missing)
#   - :latest image must resolve to the same digest as :<stable tag> (when IMAGE is set)
#   - main          is the integration branch; commits past the tag are reported, not moved
#
# Usage: scripts/sync-stable-refs.sh [--check|--apply]   (default: --apply)
# Env:   REMOTE (default origin), IMAGE (for example ghcr.io/owner/iptvtunerr; empty skips the image check)
# Never force-pushes; a diverged master fails loudly.
set -euo pipefail

MODE="${1:---apply}"
REMOTE="${REMOTE:-origin}"
IMAGE="${IMAGE:-}"
err() { echo "[sync-stable-refs] ERROR: $*" >&2; exit 1; }
say() { echo "[sync-stable-refs] $*"; }

case "$MODE" in --check|--apply) ;; *) err "usage: $0 [--check|--apply]" ;; esac

git fetch --quiet "$REMOTE" --tags
git fetch --quiet "$REMOTE" main 2>/dev/null || true
git fetch --quiet "$REMOTE" master 2>/dev/null || true

stable="$(git tag --list 'v[0-9]*.[0-9]*.[0-9]*' --sort=-v:refname | grep -E '^v[0-9]+\.[0-9]+\.[0-9]+$' | head -1)"
[[ -n "$stable" ]] || err "no stable tag found"
commit="$(git rev-list -n 1 "$stable")"
say "newest stable release: $stable ($commit)"

if git rev-parse -q --verify "refs/remotes/$REMOTE/main" >/dev/null; then
  main_commit="$(git rev-parse "$REMOTE/main")"
  if [[ "$main_commit" == "$commit" ]]; then
    say "main == $stable"
  elif git merge-base --is-ancestor "$commit" "$main_commit"; then
    say "main is $(git rev-list --count "$commit..$main_commit") commit(s) ahead of $stable (expected between releases)"
  else
    err "main does not contain $stable"
  fi
fi

rc=0
if git rev-parse -q --verify "refs/remotes/$REMOTE/master" >/dev/null; then
  master_commit="$(git rev-parse "$REMOTE/master")"
  if [[ "$master_commit" == "$commit" ]]; then
    say "master == $stable"
  elif git merge-base --is-ancestor "$master_commit" "$commit"; then
    if [[ "$MODE" == "--apply" ]]; then
      git push "$REMOTE" "$commit:refs/heads/master"
      say "master fast-forwarded to $stable"
    else
      say "master is behind $stable (run with --apply)"; rc=1
    fi
  else
    err "master has diverged from $stable; resolve manually (no force-push)"
  fi
else
  if [[ "$MODE" == "--apply" ]]; then
    git push "$REMOTE" "$commit:refs/heads/master"
    say "master created at $stable"
  else
    say "master does not exist (run with --apply)"; rc=1
  fi
fi

if [[ -n "$IMAGE" ]]; then
  command -v docker >/dev/null || err "docker is required for the image check"
  latest_digest="$(docker buildx imagetools inspect "$IMAGE:latest" --format '{{.Manifest.Digest}}')" || err "cannot inspect $IMAGE:latest"
  stable_digest="$(docker buildx imagetools inspect "$IMAGE:$stable" --format '{{.Manifest.Digest}}')" || err "cannot inspect $IMAGE:$stable"
  [[ "$latest_digest" == "$stable_digest" ]] || err "$IMAGE:latest ($latest_digest) != $IMAGE:$stable ($stable_digest)"
  say ":latest == $stable ($stable_digest)"
fi
exit "$rc"
