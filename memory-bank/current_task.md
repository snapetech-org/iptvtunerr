# Current task: Remediate IPTV Tunerr web dependency advisories (2026-10-21)

- **Goal:** move the web build from vulnerable Vite 8.0.16 to the fixed 8.3.5 release.
- **Scope:** apply the matching `web/package.json` and lockfile security update; preserve Go runtime behavior and the separate in-progress OIDC work.
- **Assumptions:** the affected Vite dependency graph is build-time tooling. Node CI already uses the maintained Node 22 line.
- **Status:** the fixed dependency and lockfile are prepared; `npm --prefix web audit --json` reports zero advisories. Dependabot PR #57 has remained queued, so the exact security patch is being applied to main directly.
- **Verification:** npm audit reports 0 vulnerabilities; `git diff --check` passed. No product tests were run for this dependency-only change.
- **Next steps:** commit and push the staged security update, then check the main branch alert state and clean the task worktree.


---
## Deferred in-flight task: Cross-project OIDC implementation audit and hardening

- **Goal:** find and fix correctness, security, accessibility, configuration-help, and documentation gaps in OIDC across all nine SnapeTech applications.
- **Scope:** audit `seerrng`, `slskR`, `slskdN`, `TorrentNG`, `iptvTunerr`, `ROMarrNG`, `QuestarrNG`, `bookshelfng`, and `chaptarrng`; preserve each app's native session and authorization model; add focused regression coverage and release notes; publish through each repo's default-branch review path.
- **Story IDs:** OIDC-AUDIT-001 through OIDC-AUDIT-009 in [work_breakdown.md](work_breakdown.md).
- **Status:** source-level fixes, provider-help accessibility polish, and operator-doc updates are implemented in isolated worktrees. Formatting, focused/full verification, PR checks, and cross-repo delivery are in progress. Original dirty checkouts remain untouched.
- **Verification:** run the repository-prescribed or focused OIDC suites and builds where available, validate changed release notes and docs, and run `git diff --check`. Live identity-provider credentials are not part of this local audit; no live provider flow will be claimed.
- **Next steps:** finish formatting and regression checks; inspect current remote default-branch heads; commit, push, review, and merge each repository where required checks allow; append actual results to task history; clean only task-owned build outputs and worktrees.

---

## Completed task: OIDC sign-in for the Control Deck

- **Goal:** add server-side OpenID Connect sign-in to the IPTV Tunerr Control Deck while preserving its native session, role, CSRF, password, Basic-authentication, and separate tuner/media boundaries.
- **Scope:** implement the stack-native OIDC adapter, provider presentation for the shared 14-provider catalog, local provider icons and setup links, operator guidance, and focused automated coverage. Do not change authentication on tuner, stream, or media-server endpoints.
- **Success criteria:** the Control Deck uses the shared authorization-code/admission contract; verified-email allow-lists deny by default; provider hints remain presentation-only; Apple uses its documented form-post exception; the existing app session remains authoritative.
- **Status:** implementation and operator documentation are complete. The full Go suite and focused OIDC checks passed in the clean rollout worktree. Cross-project publication is handled by the authorized repository PR workflow.
- **Verification:** `go test ./...`; provider-map parity check for all 14 branded IDs and byte-identical local SVG sprites; `git diff --check`.
- **Next steps:** none for this implementation. No live identity-provider tenant or credentials were used.

---

## Completed task: Finish scanner fixes and final push

## Completed prerequisite: Commit audit fixes and release v0.1.93

- **Goal:** commit the complete reviewed working tree, push it to GitHub `main`, prepare the curated changelog, improve the Discord announcement format, and publish a new stable release.
- **Scope:** include the security audit remediation already present in the working tree; create append-only release fragments and a changelog section; send a concise, categorized Discord summary with a link to assets/checksums; push through the GitHub-only `ghrelease` remote; tag the release commit and monitor the GitHub release and stable-ref workflows.
- **Assumptions:** `v0.1.92` is the latest stable release, so the next version is `v0.1.93`. The repository workflow advances `master` to the stable tag after publication. Do not push the configured GitLab mirror, whose ref service has a documented failure.
- **Plan:** validate the release notes and Discord formatter; commit the implementation and fragment; regenerate and commit the `v0.1.93` changelog section; push `main`; check required CI; tag and push `v0.1.93`; monitor release, publisher, and stable-ref workflows; record final outcomes in task history.
- **Status:** `v0.1.93` is published with 18 assets. Discord and Matrix announcement jobs succeeded. Docker, AUR, COPR, and Chocolatey publishing succeeded; Winget PR [#456399](https://github.com/microsoft/winget-pkgs/pull/456399) was submitted. PPA signing failed because the configured signer has no secret key, so the overall Release workflow ended in failure after publishing the GitHub Release.
- **Stable refs:** sync run `39360483784` advanced `master` to `v0.1.93` and verified `:latest` at digest `sha256:81149fce942dc58183c15936e1f66f006b707cd515ffca27c9c94038efdca718`.
- **Follow-ups:** restore the matching PPA signing key and check Chocolatey moderation and the Winget PR when their external reviews complete. Do not push the GitLab mirror.

## Completed follow-up: Restore Snap publishing

- **Goal:** check the latest Snap Store credential in Proton Pass without invalidating existing keys, restore Snap publishing for the already-published `v0.1.93`, and remove the intentional skip so future releases dispatch Snap automatically.
- **Scope:** inspect the Snap publisher and wiring check; verify the existing repository secret against the Store without displaying it; if it fails, create an additional credential, update only `SNAPCRAFT_STORE_CREDENTIALS`, and publish `v0.1.93`; update release-channel and memory-bank status.
- **Assumptions:** use the existing package and release tag; do not create another GitHub release or revoke old Snap credentials. Keep the pre-existing security-scanning worktree changes intact while handling this focused follow-up.
- **Plan:** inspect Proton Pass and the Snap Store UI on the dedicated agent VM; test the existing workflow secret; if needed, add a new non-revoking credential and replace the GitHub secret; enable automatic dispatch, update its assertion/docs, push to GitHub `main`, and manually dispatch `v0.1.93`; record the workflow and Store result.
- **Completed:** the existing `SNAPCRAFT_STORE_CREDENTIALS` secret passed the `v0.1.93` Snap build and stable publish. Proton Pass's newest Snap Store CI entry was checked with its password masked. No credential was created, replaced, or revoked. The release workflow no longer skips Snap, and the public Store page reports `latest/stable 0.1.93`.
- **Verification:** Snap publisher run [39361186479](https://github.com/snapetech-org/iptvtunerr/actions/runs/39361186479) succeeded; `bash scripts/check-release-channel-wiring.sh`, shell syntax checks, and `git diff --check` passed. The local Semgrep hook now defaults to one worker after its unconstrained run hit an `io_uring` allocation failure.
- **Next steps:** none.

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
