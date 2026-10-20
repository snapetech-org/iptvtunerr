# Current task: Commit audit fixes and release v0.1.93 (in progress)

- **Goal:** commit the complete reviewed working tree, push it to GitHub `main`, prepare the curated changelog, and publish a new stable release.
- **Scope:** include the security audit remediation already present in the working tree; create an append-only release fragment and changelog section; push through the GitHub-only `ghrelease` remote; tag the release commit and monitor the GitHub release and stable-ref workflows.
- **Assumptions:** `v0.1.92` is the latest stable release, so the next version is `v0.1.93`. The repository workflow advances `master` to the stable tag after publication. Do not push the configured GitLab mirror, whose ref service has a documented failure.
- **Plan:** validate the release note and repo diff; commit the reviewed working tree; prepare and commit the `v0.1.93` changelog section; push `main`; check required CI; tag and push `v0.1.93`; monitor release, publisher, and stable-ref workflows; record final outcomes in task history.
- **Status:** latest GitHub release `v0.1.92` published its assets and announcements. Its channel dispatcher reported Chocolatey `403 Forbidden` and PPA signing failure because the imported key does not match the configured signing fingerprint. Continue the release workflow and report channel results accurately.
- **Next steps:** validate, commit, prepare changelog, push, then tag after the required checks pass.

---

## Completed prerequisite: Security audit and remediation (2026-10-19)

- **Goal:** review the project’s externally reachable surfaces, authentication and authorization boundaries, parsing and resource limits, filesystem/process use, and dependencies; fix concrete security defects and record remaining hardening work.
- **Scope:** one audit/remediation pass, prioritizing verified exploitable defects and preserving existing Plex and operator workflows.
- **Out of scope:** broad refactors, live production changes, and speculative compatibility changes without evidence.
- **Assumptions:** LAN-facing tuner features remain unauthenticated by design; operator endpoints should follow their existing localhost/LAN policy. Avoid compatibility changes to Plex workflows without evidence.
- **Plan:** trace each HTTP boundary and proxy classifier; inspect dangerous file/process/outbound-call paths; check dependency advisories; remediate confirmed issues; update security docs, known issues/opportunities, and task history.
- **Completed:** hardened Control Deck login/session handling, CSRF and Basic-auth mutations; bounded login state and request parsing; narrowed Plex subscription classification while preserving full upstream request bodies; isolated the private identity denylist from pull-request code; tightened CI token permissions; updated security guidance and filed remaining risks.
- **Verification:** `gofmt`, `go vet`, Go build, dependency scans, workflow/shell checks, and `git diff --check`; tests were not run during this pass.
- **Remaining work:** see the Security audit follow-ups in [opportunities.md](opportunities.md), including LAN operator authentication and live Plex compatibility validation.

---

# Previous task: Unraid distribution and release follow-ups

- **Goal:** finish distributing IPTV Tunerr on Unraid Community Applications and submit the root template through the Community Applications portal.
- **Story IDs:** DIST-001 through DIST-006 in [work_breakdown.md](work_breakdown.md). DIST-003 was withdrawn.
- **Scope:** `iptvtunerr.xml` and `ca_profile.xml` live in the repository root; there is no separate Unraid package repository.
- **Out of scope:** Tunerr runtime features and claiming fixed hardware requirements without measurements.
- **Current status:** v0.1.92 is released (Plex proxy owner-token hardening, SeerrNG recording rules, dependency bumps, Docker Go 1.26 fix); `master` and the `latest` image are kept aligned with the newest stable tag by `.github/workflows/sync-stable-refs.yml`. Replaced the repository's `CHOCO_API_KEY` from Proton Pass and republished `v0.1.92`; Chocolatey accepted the push and the package is pending automated review. The PPA publisher fails with "No secret key" until its GPG signing secret is restored; the GitLab mirror still rejects pushes (unpacker error).
- **Next steps:** monitor Chocolatey moderation; run Unraid's portal Validate and Scan, then submit; verify install/update/removal on an Unraid host.

## Completed one-off update (2026-10-02)

- Rotated the Discord release announcement endpoint in the GitHub Actions `DISCORD_RELEASE_WEBHOOK` secret for `snapetech/iptvtunerr`. Verified the secret name is listed and its endpoint ID is absent from tracked files.
