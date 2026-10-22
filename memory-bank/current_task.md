# Current task: Remediate 227 code-scanning findings and release (2026-10-22)

- **Goal:** review and resolve the 227 open GitHub code-scanning alerts (225 Semgrep OSS, 2 CodeQL), verify the final exact head, and publish the next stable release.
- **Scope:** inventory the live alerts under `/tmp/iptvtunerr-security-quality-20261022/`; trace each unique finding to source and rule; fix confirmed defects with regression coverage; document evidence-based false-positive dispositions; update release notes and required memory; push to the GitHub default branch and verify scanner, CI, release, and stable-ref outcomes.
- **Out of scope:** unrelated product changes, GitLab publication, and weakening scanners or globally suppressing rules to lower the count.
- **Assumptions:** all 227 captured alerts are attached to `refs/heads/main`; there are no open Dependabot alerts. GitHub currently lists `v0.1.93` as latest, and `v0.1.94` is the prepared but unpublished next release. The alert export reflects analysis at `f3b17c2`; reconcile it against the new exact-head analysis after push.
- **Status:** implementation commit `ac15e76`, smoke-fixture commit `ebff303`, and changelog commit `5bc0cf4` are pushed to GitHub `main` through upstream base `afae0b4`. `scripts/verify` and `scripts/release-readiness.sh` pass; the tagged Action SHAs match upstream; the built image runs as UID/GID `10001:10001` with writable state/cache paths; targeted Trivy reports no high/critical findings in the three runtime Dockerfiles. The local report-only Semgrep scan still returns 156 conservative findings for source-by-source reconciliation. The unpublished v0.1.94 changelog matches all 9 fragments. The first exact-head Local Identity Leak Check exposed a history-range bug: it rescanned legacy commit trailers. The scanner now supports push-base and PR-commit ranges, with regression tests; this follow-up is locally verified and awaits commit/push. Hosted exact-head checks, refreshed alert analysis/dispositions, release publication, and stable-ref verification remain.
- **Verification evidence:** first full verify exposed an outdated login smoke fixture (missing the signed form token); updated the fixture to GET and parse the real hidden field. `/tmp/iptvtunerr-security-quality-20261022/verify-final.log`, `release-readiness.log`, and `verify-identity-fix.log` pass. The scanner range tests cover clean ranges, detected leaks, and invalid input failing closed. The initial smoke failure remains in `verify.log`.
- **Plan:** commit and push the scanner-range correction to GitHub `main`; wait for exact-head CI, CodeQL, Semgrep, Trivy, and identity checks; reconcile refreshed alerts by source, closing fixed findings and dismissing only individually substantiated false positives; verify the v0.1.94 changelog against its fragments; tag/publish v0.1.94; verify its assets and automatic `master`/stable-image sync; comment on issue #59 with the fix and release information; then record final outcomes.
- **Scratch outputs:** `/tmp/iptvtunerr-security-quality-20261022/alerts.json`, `alert-inventory.md`, `focused-tests.log`, `verify.log`, `verify-final.log`, `release-readiness.log`, `security-scan.log`, `semgrep-final.log`, `release-preview.md`, `docker-build.log`, `dockerfile-trivy.log`, `dockerfiles/` (temporary Trivy-only copies, to remove after scan), `hls.js-1.5.7.min.js`, and `pycache/` (task-scoped Python cache).
- **Container smoke output:** task-owned image tag `iptv-tunerr:security-quality-20261022` passed non-root/path-write checks; remove this image and any task-owned container after validation, preserving shared Docker caches.

## Completed prior task: Fix Control Deck CSRF bootstrap on Basic Auth page loads (2026-10-22)

- **Status:** complete. Fix commit `a78e599` and closeout commit `d7f1229` are pushed to GitHub `main`; issue #59 has the fix details and verification results.
- **Verification outputs retained:** `/tmp/iptvtunerr-issue59-csrf-20261022/` contains the full verification log and council evidence.

---

## Deferred prior task: Cross-project CVE sweep, repository cleanup, and releases (2026-10-22)

- **Goal:** finish the security sweep across QuestarrNG, ROMarrNG, slskR, slskdN, BookshelfNG, ChaptarrNG, SeerrNG, TorrentNG, and IPTV Tunerr; remediate confirmed exposure; review third-party changes for security; resolve repository work directly on each GitHub default branch; publish the requested releases except ChaptarrNG and SeerrNG.
- **Scope:** inspect dependency/runtime advisories, PRs, issues, branches, stashes, and dirty worktrees for these nine repositories; preserve evidence and secrets; do not publish to GitLab.
- **Status:** in progress. JavaScript and .NET dependency audits are clean. PHP is absent. QuestarrNG's `certifi` pin is already fixed on `main`; its Dependabot alert remains stale. slskR and IPTV Tunerr now pin Node.js 22.23.4; IPTV Tunerr also targets Go 1.26.10. The verifier prunes vendored and nested worktree directories; the full prescribed verification passed before the Node pin update.
- **Destructive-operation rollback:** before deleting any remote/local branch refs, make a repository Git bundle under `/tmp/cve-sweep-20261022/` and retain the branch-name/SHA map until the cleanup is verified; restore a deleted branch by pushing its saved SHA. Closed issues and PRs can be reopened. Do not rewrite published history or delete published releases/tags; correct release mistakes only with a successor release.
- **Next steps:** finish current-advisory checks, review and resolve outstanding PR/issue/worktree state, then validate, commit, push, and release only exact verified heads.

## Prior active release task: Prepare IPTV Tunerr v0.1.94 (2026-10-21)

- **Goal:** release the current GitHub `main` head, including the patched Vite dependency and OIDC sign-in changes, through the repository's normal release and stable-ref workflows.
- **Scope:** add a user-facing security note for the Vite toolchain update, generate the `v0.1.94` changelog section, push release preparation to GitHub `main`, wait for exact-head CI, and publish and verify the GitHub release. Do not push the GitLab mirror.
- **Assumptions:** `v0.1.93` is the latest stable release, so the next version is `v0.1.94`; the package workflow advances `master` and stable images after release publication.
- **Status:** in progress. The current `main` head is `3e0e101f9fa6fcc41acf7331c9fabf44d3d64aab`; hosted verification is still queued.
- **Plan:** commit and push the new release-note fragment; generate and commit the checked-in changelog section; push `main`; require successful checks for the final exact head; tag `v0.1.94`; monitor release and stable-ref workflows; record actual outcomes in task history.
- **Next steps:** add the Vite security fragment, prepare the changelog, then validate and publish only after exact-head checks pass.

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
