# Current task: Unraid distribution and release follow-ups

- **Goal:** finish distributing IPTV Tunerr on Unraid Community Applications and keep the Unraid package mirror synchronized from this repo.
- **Story IDs:** DIST-001 through DIST-006 in [work_breakdown.md](work_breakdown.md). DIST-003 was withdrawn.
- **Scope:** canonical package tree in `packaging/unraid`, mirrored to `snapetech/iptvtunerr-unraid` by `.githooks/post-commit` and release-tag pushes through `scripts/sync-platform-packages.sh`.
- **Out of scope:** Tunerr runtime features and claiming fixed hardware requirements without measurements.
- **Current status:** v0.1.92 is released (Plex proxy owner-token hardening, SeerrNG recording rules, dependency bumps, Docker Go 1.26 fix); `master` and the `latest` image are kept aligned with the newest stable tag by `.github/workflows/sync-stable-refs.yml`. The Chocolatey publisher fails with 403 until the `CHOCOLATEY_API_KEY` Actions secret is replaced by the operator, and the PPA publisher fails with "No secret key" until its GPG signing secret is restored; the GitLab mirror still rejects pushes (unpacker error).
- **Next steps:** operator replaces the Chocolatey key, then re-dispatch `publish-chocolatey.yml` for the newest tag; run Unraid's portal Validate and Scan, then submit; verify install/update/removal on an Unraid host.

## Completed one-off update (2026-10-02)

- Rotated the Discord release announcement endpoint in the GitHub Actions `DISCORD_RELEASE_WEBHOOK` secret for `snapetech/iptvtunerr`. Verified the secret name is listed and its endpoint ID is absent from tracked files.
