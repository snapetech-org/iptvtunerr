#!/usr/bin/env python3
"""Scan one release-facing text file without external search/JSON tools."""

from __future__ import annotations

import argparse
import sys
from pathlib import Path

MAX_SCAN_FILE_BYTES = 16 * 1024 * 1024


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser()
    parser.add_argument("--tokens-file", required=True)
    parser.add_argument("--label", required=True)
    parser.add_argument("--display-path", required=True)
    parser.add_argument("path")
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    try:
        tokens = {
            token.casefold()
            for token in Path(args.tokens_file).read_text(encoding="utf-8").splitlines()
            if token
        }
        path = Path(args.path)
        if path.stat().st_size > MAX_SCAN_FILE_BYTES:
            print("identity scanner input exceeds its 16 MiB safety limit.", file=sys.stderr)
            return 2
        content = path.read_bytes()
    except (OSError, UnicodeError):
        print("identity scanner could not read input.", file=sys.stderr)
        return 2

    # Do not report text matches from files containing NUL bytes.
    if b"\0" in content or not tokens:
        return 0

    folded_tokens = tuple(tokens)
    found = False
    for line_number, line in enumerate(
        content.decode("utf-8", errors="replace").split("\n"), start=1
    ):
        folded_line = line.rstrip("\r").casefold()
        if any(token in folded_line for token in folded_tokens):
            print(
                f"{args.label}: {args.display_path}:{line_number}",
                file=sys.stderr,
            )
            found = True

    return 1 if found else 0


if __name__ == "__main__":
    raise SystemExit(main())
