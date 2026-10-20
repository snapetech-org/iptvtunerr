#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SOURCE_REF="HEAD"
TAG=""
USE_WORKTREE=0

usage() {
	cat <<'EOF'
Usage: scripts/sync-platform-packages.sh [--working-tree] [--source-ref REF] [--tag vX.Y.Z]

Synchronize packaging/unraid to its public package repository. --working-tree is for the initial publication before these
package files have been committed to the source repository.
EOF
}

while (($#)); do
	case "$1" in
		--working-tree)
			USE_WORKTREE=1
			shift
			;;
		--source-ref)
			[[ $# -ge 2 ]] || { usage >&2; exit 2; }
			SOURCE_REF="$2"
			shift 2
			;;
		--tag)
			[[ $# -ge 2 ]] || { usage >&2; exit 2; }
			TAG="$2"
			shift 2
			;;
		-h|--help)
			usage
			exit 0
			;;
		*)
			usage >&2
			exit 2
			;;
	esac
done

if ((USE_WORKTREE == 0)) && ! git -C "$ROOT_DIR" cat-file -e "${SOURCE_REF}^{commit}" 2>/dev/null; then
	echo "Source reference '$SOURCE_REF' is not a commit in this checkout." >&2
	exit 1
fi

if [[ -n "$TAG" && ! "$TAG" =~ ^v[0-9]+\.[0-9]+\.[0-9]+([.-][0-9A-Za-z]+)*$ ]]; then
	echo "Release tag must look like vX.Y.Z; got '$TAG'." >&2
	exit 2
fi

require() {
	command -v "$1" >/dev/null 2>&1 || {
		echo "Package synchronization requires '$1'." >&2
		exit 1
	}
}

require git
require rsync
require tar

source_description="$(git -C "$ROOT_DIR" rev-parse --short "${SOURCE_REF}^{commit}" 2>/dev/null || git -C "$ROOT_DIR" rev-parse --short HEAD)"
if ((USE_WORKTREE)); then
	source_description="${source_description}-working-tree"
fi

sync_one() {
	local package_path="$1"
	local repository="$2"
	local branch="$3"
	local checkout
	local stage
	local source_dir

	checkout="$(mktemp -d)"
	stage="$(mktemp -d)"
	trap 'rm -rf "$checkout" "$stage"' RETURN

	if ((USE_WORKTREE)); then
		mkdir -p "$stage/$package_path"
		rsync --archive --delete "$ROOT_DIR/$package_path/" "$stage/$package_path/"
	else
		git -C "$ROOT_DIR" archive --format=tar "${SOURCE_REF}^{commit}" "$package_path" | tar -xf - -C "$stage"
	fi
	source_dir="$stage/$package_path"
	if [[ ! -d "$source_dir" ]]; then
		echo "Package tree '$package_path' is missing at source ref '$SOURCE_REF'." >&2
		return 1
	fi

	git -C "$checkout" init --quiet
	git -C "$checkout" remote add origin "git@github.com:${repository}.git"
	if git -C "$checkout" fetch --quiet origin '+refs/heads/*:refs/remotes/origin/*'; then
		:
	else
		# An empty companion repository has no refs to fetch yet.
		if [[ -n "$(git -C "$checkout" ls-remote --heads origin)" ]]; then
			echo "Unable to fetch branches from $repository." >&2
			return 1
		fi
	fi

	if git -C "$checkout" show-ref --verify --quiet "refs/remotes/origin/$branch"; then
		git -C "$checkout" switch --quiet --track --create "$branch" "origin/$branch"
	elif git -C "$checkout" show-ref --verify --quiet refs/remotes/origin/main; then
		git -C "$checkout" switch --quiet --track --create "$branch" origin/main
	else
		git -C "$checkout" switch --quiet --orphan "$branch"
	fi

	# Do not delete .git or hidden package metadata when mirroring the package tree.
	rsync --archive --delete --exclude='/.git/' --exclude='/.github/' "$source_dir/" "$checkout/"
	git -C "$checkout" add --all
	if ! git -C "$checkout" diff --cached --quiet; then
		git -C "$checkout" -c user.name='IPTV Tunerr Package Sync' \
			-c user.email='actions@users.noreply.github.com' \
			commit --quiet -m "chore: sync package from $source_description"
	fi

	git -C "$checkout" push --quiet origin "HEAD:refs/heads/$branch"
	echo "Synced $package_path to $repository ($branch)."

	if [[ -n "$TAG" ]]; then
		if git -C "$checkout" ls-remote --exit-code --refs origin "refs/tags/$TAG" >/dev/null 2>&1; then
			echo "Keeping existing $repository tag $TAG unchanged."
		else
			git -C "$checkout" -c user.name='IPTV Tunerr Package Sync' \
				-c user.email='actions@users.noreply.github.com' \
				tag --annotate "$TAG" --message="IPTV Tunerr package snapshot $TAG"
			git -C "$checkout" push --quiet origin "refs/tags/$TAG:refs/tags/$TAG"
			echo "Published package tag $TAG to $repository."
		fi
	fi

	trap - RETURN
	rm -rf "$checkout" "$stage"
}

sync_one "packaging/unraid" "snapetech/iptvtunerr-unraid" "main"
