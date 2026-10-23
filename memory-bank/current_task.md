# Current task: Standardize release tags and runner PRs (2026-10-23)

- **Goal:** use one pushed `v<version>` tag convention to start new product-release builds across all nine runner-using applications; keep PR/main validation separate and limit manual recovery to already-published outputs.
- **Story IDs:** RELEASE-TAG-001 and RELEASE-TAG-002 in [work_breakdown.md](work_breakdown.md).
- **Scope:** review nine application repositories plus Packer, close manual release-build bypasses, retain existing release assets and channel recovery, and preserve runner trust and platform boundaries.
- **Status:** focused changes are pushed to the matching draft PRs: BookshelfNG #193, SeerrNG #395, QuestarrNG #140, SlskR #194, IPTV Tunerr #61, and TorrentNG #232. All nine application PRs remain drafts. Their latest exact-head checks are pending; Packer #2's CI was rejected by the organization billing/spending cap before a job started. No fix was merged or deployed.
- **Verification:** BookshelfNG 6 workflow-policy tests and release-readiness gate; SeerrNG 36 release-workflow tests plus Prettier; QuestarrNG manual-builder structure check; SlskR release-policy check; TorrentNG 7 runner-policy and 5 release-note tests; IPTV Tunerr 5 dispatcher/release-policy tests, release-channel wiring, 58 action-pin checks, YAML parsing, and embedded Bash syntax. `git diff --check` and YAML parsing pass for changed workflows.
- **Next step:** inspect exact-head checks as they complete; keep PRs draft pending required checks and infrastructure readiness. Broker deployment, runner-group reconciliation, and live release canaries remain separate outstanding work.
- **Assumptions:** IPTV Tunerr’s tester ZIPs are user-facing release assets, so the tester workflow now runs as a reusable job after the `v*` release is published; it has no standalone manual build trigger.
- **Limits:** do not merge PRs, deploy/restart runners, or create a release tag. Preserve existing published tags and native platform requirements.
- **Rollback:** revert the repository-specific workflow changes; preserve shared broker configuration and retry state.

---

## Prior runner deployment session (historical; live state not reverified)

- **Goal:** verify that SeerrNG, SlskR, slskdN, ROMarrNG, QuestarrNG, BookshelfNG, ChaptarrNG, and TorrentNG run their trusted-main checks on the shared JIT pool, fix pool-wide failures, and land every repo change on its GitHub default branch.
- **Scope:** keep PR and credentialed release jobs hosted; preserve broker allowlists and CPU, memory, disk, and job caps; fix common broker/bootstrap behavior; safely deploy tested broker, guest helper, and egress-proxy sources; rerun failed or unverified exact-head jobs and record results.
- **Out of scope:** product behavior changes, weakening scanners/resource gates, canceling active/current-head jobs, or rewriting published history.
- **Status:** all eight route commits are ancestors of their current `main` heads. Packer `main` contains pushed commits `b97cabc`, `80c536d`, `d38c749`, `9254455`, and `488cd14` for release priority, API display-name package mapping, exact Ubuntu ports egress, SlskR tools, and Chaptarr's Python runtime. The 92-test policy suite passes with the admission marker isolated in-process; commit-hook Semgrep scans found zero findings. Packer CI `39897075828` passed at exact head `488cd14`.
- **Root cause and fixes:** GitHub can assign any eligible same-run job to a run-scoped JIT runner, but the broker received Actions API display names while several package rules used YAML job IDs. Matrix jobs also require expanded display names. The broker now unions package requirements for identical runner labels, tests use API-shaped job names, and the SlskR workflow explicitly gets Docker for containerized scans plus `python-is-python3` for its Linux artifact verifier. ChaptarrNG also exposed a blocked exact `ports.ubuntu.com` mirror; Packer now permits that hostname exactly and rejects lookalikes.
- **Application route commits:** SeerrNG `0232db29`; SlskR `2d0588ee`; slskdN `c88ef57b`; ROMarrNG `0d83f802`; QuestarrNG `681f9f61`; BookshelfNG `82a8b503`; ChaptarrNG `2c0142eb`; TorrentNG `e3461660`.
- **Current exact-head checks:** SeerrNG `1b37e0de` CI `39895294513` received its first Vitest shard on the pool; its initial native TypeScript setup failed with missing Docker before the fix, so rerun that failure after the workflow completes. SlskR `c5fe08ef` retry attempt 2 is queued with the new Docker/Python packages. slskdN `605b7094` retry attempt 3 is queued after the earlier `Build and Test` failure. ROMarrNG `85e70f49` retry attempt 2 is queued after the missing-Docker failure. QuestarrNG `a281a0dc` CI, SAST, CodeQL, E2E, and Scorecard passed; its Trivy scan ran on the pool. BookshelfNG `685b31c9` package sync passed, Trivy was dispatched, and the release-note contract failed because the push had no fragment or `release-note: none` marker (unrelated to JIT). ChaptarrNG advanced to `785707bb`; current-head run `39897196030` started on the pool, and its `postgres-migrations` restore step failed; workflow logs are pending run completion. TorrentNG `39554462` passed `msrv-check`; `fuzz-smoke` ran on the pool while the client quality check failed source formatting, unrelated to JIT.
- **Deployment state:** Packer broker SHA-256 `42acf4da58686051bcd2623b449ef609d310f1a6d8db0a20a263bb025040123f`, VM helper `836e88f325e15fe40fd778d9edbde7ecc5ff9bd26b2ca041d70efcc22119d938`, proxy `b7ae407df5faf21d7ba0f3dc105ef2f4c13a1e55d1fe169c37aa534132b85d51`, and common helper `866c5e7b25cad40d2ace0d6af8472c4ebd3bce3f1ab21310e55eb48803e510a6` are installed with root ownership and expected modes on their respective workers. The broker was restarted with markers held, then both exact markers were removed. Primary capacity admitted Packer, ChaptarrNG, TorrentNG, QuestarrNG, SeerrNG, and BookshelfNG jobs; scratch capacity remains closed by its normal memory-reserve gate. Existing resource and PSI limits are unchanged.
- **New pool reliability fix:** slskdN's NuGet vulnerability scan caused QEMU to log `Slirp: fork_exec: Failed to fork child process` followed by `qemu_thread_create: Resource temporarily unavailable`. The guest stayed below its 10 GiB memory cap and the host had no OOM evidence; QEMU's `guestfwd -cmd` spawned a proxy process per TCP connection inside the task-limited runner service. Packer commit `8795a0a` proposed forwarding to a loopback proxy with QEMU's `guestfwd -tcp`, but QEMU uses that backend as one persistent character stream, so it cannot carry independent HTTP sessions. That commit is pushed but not deployed. The corrected implementation uses `guestfwd -cmd` only for a short native helper that transfers its stream descriptor over a private per-VM Unix socket to one proxy service. The service caps active streams at 16, queued handoffs at 32, and request-header reads at 15 seconds; existing allowlist and public-address enforcement stay unchanged. Tests and deployment are in progress. Both admission markers are held; five primary guests were live when drain began. Do not deploy until both hosts and GitHub registrations are idle.
- **Live deployment rollback note:** Before replacing the VM helper/proxy or installing the handoff helper on either worker, keep `/run/gha-jit-admission-paused` on both hosts and retain the current `/usr/local/libexec/gha-jit-vm` and `/usr/local/libexec/gha-egress-proxy` under `/var/backups/gha-jit/20261023-guest-proxy-handoff/`. Roll back by restoring those exact root-owned 0755 copies while admission is paused, removing the new handoff binary, restarting the broker only after both hosts are idle, confirming `operator-drain`, then reopening admission. The deployment does not modify guest workspaces or runner registrations; keep the source backups as recovery evidence.
- **Next steps:** let current-head workflows and the SlskR, slskdN, and ROMarrNG retries progress; inspect the Chaptarr restore failure when the run logs are available; rerun only current-head pool-related failures. Record final exact-head outcomes, update known issues and recurring-loop notes, append task history, ingest the Hindsight correction, and verify repository/disk state before pushing the memory-bank closeout.
- **Completed context:** IPTV Tunerr issue #59 was fixed and commented; all 227 captured scanner findings were dispositioned; release `v0.1.95` was published with 18 assets. Its verification and release evidence remains under the task-owned scratch directory.

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
