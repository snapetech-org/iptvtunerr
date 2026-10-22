#!/usr/bin/env python3
"""Require immutable, version-documented GitHub Actions references."""

from __future__ import annotations

import re
import sys
from pathlib import Path


SHA_RE = re.compile(r"^[0-9a-f]{40}$")
VERSION_RE = re.compile(r"\bv[0-9]+(?:\.[0-9A-Za-z-]+){0,2}\b")


def check(root: Path) -> list[str]:
    workflows = root / ".github" / "workflows"
    failures: list[str] = []
    checked = 0
    paths = sorted((*workflows.glob("*.yml"), *workflows.glob("*.yaml")))
    for path in paths:
        for number, line in enumerate(path.read_text(encoding="utf-8").splitlines(), 1):
            stripped = line.lstrip()
            if not stripped.startswith("uses:") and not stripped.startswith("- uses:"):
                continue
            value = stripped.split("uses:", 1)[1].split("#", 1)[0].strip().strip("\"'")
            if value.startswith(("./", "docker://")):
                continue
            action, separator, reference = value.rpartition("@")
            location = f"{path.relative_to(root)}:{number}"
            if not separator or not action:
                failures.append(f"{location}: action reference must include an immutable commit SHA: {value}")
                continue
            checked += 1
            if not SHA_RE.fullmatch(reference):
                failures.append(f"{location}: action reference is mutable; pin a full 40-character SHA: {value}")
                continue
            comment = line.split("#", 1)[1] if "#" in line else ""
            if not VERSION_RE.search(comment):
                failures.append(f"{location}: add the pinned upstream version in a trailing comment: {value}")
    if checked == 0:
        failures.append("no remote GitHub Actions references were checked")
    if not failures:
        print(f"Verified {checked} immutable GitHub Actions references across workflow files.")
    return failures


def main() -> int:
    failures = check(Path(__file__).resolve().parent.parent)
    for failure in failures:
        print(f"ERROR: {failure}", file=sys.stderr)
    return 1 if failures else 0


if __name__ == "__main__":
    raise SystemExit(main())
