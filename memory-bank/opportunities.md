# Opportunities

## Sports Automation follow-ups (2026-10-01)

- **Team aliases for EPG matching.** Provider XMLTV often uses short names or city-only labels unlike API-Sports. Add an operator-managed alias map or curated per-dataset aliases with explicit ambiguity handling; do not weaken the team-plus-time match based on guesswork.
- **Additional canonical datasets.** Extend beyond MLB/NFL/NCAA/NBA only after implementing and validating sport-specific response adapters, season/date semantics, and status handling. Broader API-Basketball leagues and cups remain unsupported.
- **Live API and provider validation.** The implementation has no production API key in this checkout. Confirm response normalization, real quota headers, representative EPG matches, and end-to-end playback with an operator-controlled test instance before broad rollout.
- **Live refresh policy.** The first version avoids automatic polling to protect provider quotas. If live polling is added, make it visible, cache-aware, bounded to live games, and adjustable by the operator.

## WebUI bundle follow-up (2026-10-01)

- **Split the operator deck into route chunks.** The production JavaScript bundle is about 1.35 MB minified (roughly 400 KB gzipped), above Vite's 500 KB chunk warning. The Sports page adds only a small increment; assess route-level splitting across the existing deck instead of optimizing this feature in isolation.

## Release-channel audit on 2026-10-01

- **Smoketest timeout data loss — fixed and released.** `v0.1.87` retains channels whose probes were interrupted or unsampled when the total smoketest budget expires, while only completed probe failures are removed. The 24,908-channel timeout regression passes; see [the release](https://github.com/snapetech/iptvtunerr/releases/tag/v0.1.87).
- **Current `.87` parity — most publishers confirmed; one archive publication remains active.** AUR, COPR, Chocolatey, Winget, GHCR, and `snapetech/iptvtunerr` are verified at `.87`; Winget PR [#444749](https://github.com/microsoft/winget-pkgs/pull/444749) merged. Launchpad builds succeeded but binary publications are still pending. At the time, Snap stable was `.78` after expired credentials; Docker Hub `keefshape/iptvtunerr` remained stale after authentication/scope failures. Snap stable is restored to `.93`; Docker Hub compatibility publishing remains deferred.
- **Hosted BuildKit DNS — fixed for the primary registries.** Buildx host networking lets the GitHub-hosted multi-architecture build resolve Debian package mirrors. The `.87` image built for amd64, arm64, and arm/v7; its primary GHCR and Docker Hub destinations share digest `sha256:baab1bd9ff3a640ee951c2e994f626f2187e9a1469b204c132433ca6a4831a6f`.

## Implemented or reconciled on 2026-09-30

- **Plex internal-fetcher profile precedence — implemented.** Explicit per-channel profile overrides now take precedence over `IPTV_TUNERR_PLEX_INTERNAL_FETCHER_PROFILE`; the environment setting remains the fallback. See [CLI and environment reference](../docs/reference/cli-and-env-reference.md) and WBS story OPP-001.
- **Public Actions log hygiene — implemented.** Public GitHub Actions workflows use GitHub-hosted Ubuntu or Windows runners, so checkout/tool logs no longer expose paths from the private self-hosted runner. See WBS story OPP-002.
- **Release announcement parity — implemented.** Matrix now sends the curated release body used by GitHub and Discord in bounded messages, escapes note text for HTML, and replaces same-day retry chunks. The first Matrix delivery using this path will be confirmed on the next release. See WBS story OPP-003.
- **First-run package follow-up — registry audit found drift.** AUR and COPR published `v0.1.86`, and the GitHub Release contains `.deb` and `.rpm` assets. The successful [PPA upload run](https://github.com/snapetech/iptvtunerr/actions/runs/36778708581) did not make `v0.1.86` available: Launchpad still serves `0.1.75+ppa202605160302~jammy`. Chocolatey accepted `0.1.86`, and Winget PR [`microsoft/winget-pkgs#444713`](https://github.com/microsoft/winget-pkgs/pull/444713) submits `0.1.86`; both have upstream review gates. Docker Hub `snapetech/iptvtunerr` and GHCR are at `v0.1.86`, while compatibility image `keefshape/iptvtunerr:latest` is at `v0.1.77`. The registered [Snap Store package](https://snapcraft.io/iptvtunerr) has no released channel. See WBS stories OPP-004 and RELCH-001 through RELCH-005.
- **Release package generation — confirmed.** The [`v0.1.86` GitHub Release](https://github.com/snapetech/iptvtunerr/releases/tag/v0.1.86) contains direct `.deb` and `.rpm` assets. See WBS story OPP-004.
- **Single-run CI failure — closed for now.** The unchanged retry passed full verification, and [CI for current `main`](https://github.com/snapetech/iptvtunerr/actions/runs/36787502831) is green. No source-level defect or recurrence was found; reopen only if this failure repeats. See WBS story OPP-005.
- **Windows native proof — completed.** Windows Smoke passed on the GitHub-hosted runner after the runner migration and push trigger: [run 36793770645](https://github.com/snapetech/iptvtunerr/actions/runs/36793770645).

## External follow-up

- **Unraid Community Applications validation.** After replacing the custom dual-license wrapper with canonical AGPL-3.0 text, GitHub now recognizes the package license. Rerun the portal's Validate and Scan actions, submit the repo for moderator review, then complete a real install/update/removal check on an Unraid host.
- **Platform resource measurements.** Measure idle load, direct relay concurrency, guide refresh, recording, and ffmpeg remux/transcode on a representative x86 Unraid machine before publishing CPU/RAM minimums.
- **Release-channel parity follow-up.** Monitor Launchpad binary publication and confirm `.87` appears in the PPA. The `v0.1.92` and `.93` PPA runs failed because their imported GPG key does not match the configured signer fingerprint. Docker Hub compatibility publishing remains deferred. Keep package/index availability distinct from successful workflow submission.
- **Chocolatey v0.1.92 package warnings.** A manual `0.1.92` push was accepted and is pending automated review, but the GitHub publisher run returned `403 Forbidden`. The publisher also reports that `iconUrl` is deprecated and that `tools/chocolateyuninstall.ps1` is not recognized by NuGet. Reconcile feed/API-key status before retrying; then update the package metadata/script naming and verify package contents.
- **Plex Live TV watchdog schedule.** The prior read-only audit found the watchdog service inactive and its newest snapshot dated 2026-06-18. No watchdog timer/service definition is owned by this repository. Verify the deployment's timer/service wiring and collect a fresh snapshot through its host-managed configuration before relying on it during a Live TV failure. See WBS story OPP-007.
- **Plex proxy subscription writes remain compatibility-scoped.** The classifier now checks only recognized `guid`, `key`, `uri`, `ratingKey`, and Plex `hints[...]` fields in URL-encoded forms and queries; it still identifies Live TV values by marker text. A shared user can submit valid XMLTV evidence that Plex accepts as an owner-attributed write. Narrowing further risks the Record-button regressions in `recurring_loops.md`; validate parsed request shapes with a real shared user before changing the compatibility contract.
- **Validate `scrubOwnerToken` against a real PMS.** Unit tests cover headers, XML, gzip JSON and fail-closed sizing; confirm on the live host that Live TV `/livetv/sessions/*` playlists and `/media/providers` still work, and that PMS does or does not resolve `..` (the proxy no longer sends it either way).

## Security audit follow-ups (2026-10-19)

- **Authenticate LAN operator routes.** `IPTV_TUNERR_UI_ALLOW_LAN=1` currently grants unauthenticated LAN access to tuner operator pages, reports, and actions. Add a shared credential or token between the Control Deck and tuner before recommending this setting outside a trusted network; preserve the unauthenticated HDHomeRun, guide, and stream contract.
- **Pin GitHub Actions to immutable revisions.** CI and publisher workflows currently reference actions by major-version tags. The new Semgrep scan flags seven mutable action references in `release.yml`. Dependabot can update full commit-SHA pins to prevent a moved or compromised tag from changing the action code fetched by a workflow.
- **Run the container as non-root.** The primary `Dockerfile` uses Debian's default root user and includes `ffmpeg`; switching the runtime to a dedicated UID needs a volume-permission compatibility plan for Docker and Unraid users.
- **Set explicit reverse-proxy trust for Secure cookies.** The Control Deck sets `Secure` only when Go directly sees TLS. TLS-terminating proxies that speak HTTP to Tunerr need a trusted-proxy configuration before forwarded-proto can safely influence cookie flags.
- **Bound diagnostic guide duration inputs.** Read-only `/guide/highlights.json` and `/guide/capsules.json` already clamp returned row counts, but accept very long caller-provided `soon` and `horizon` durations. Consider a maximum duration so public-LAN requests cannot make guide results span years.

## Workflow lint follow-up (2026-10-19)

- **Clear the existing actionlint baseline warnings.** A repo-wide actionlint pass reports the intentional `if: false` in `package-smoke-disabled.yml` and ShellCheck `SC2012` for `ls` in `tester-bundles.yml`. Keep the disabled job explicit, and replace the artifact-directory `ls` lookup with a filename-safe selection before removing the temporary lint suppressions.

## Local security-scan scope (2026-10-22)

- **Keep Trivy focused on the checkout instead of nested agent worktrees and captured diagnostics.** `scripts/security-scan-local.sh` scanned `.claude/worktrees/`, `.diag/`, and the 440 MB generated `catalog.json` during this task, producing additional findings from stale nested Dockerfiles and Kubernetes manifests. Add safe checkout/worktree exclusions or a tracked-file scan mode, while retaining dependency, IaC, and secret coverage for the active repository.
