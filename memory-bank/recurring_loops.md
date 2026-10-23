## Loop: Workflow selection and installed broker drift

**Symptom:** CI has a JIT selector but release/tag jobs wait indefinitely on hosted runners.

**Cause:** runner selection, broker discovery, exact workflow/group access, guest tools,
and publisher egress are separate contracts. A retry keeps the original tag/SHA;
changing main does not update old tag workflows. A merge can remove broker recovery
without removing the YAML retry expression.

**Resolution:** review and test those contracts together, preserve the retry journal,
verify main ancestry and exact tag refs, and install before enabling hosted-first.
Keep unsupported platform/package routes explicit. A successful queue scan is not
proof that artifacts or package publications completed. Live activation remains open.

# Recurring loops and hard-to-solve problems

### Loop: A new main push makes an older queued build permanently ineligible

**Symptom:** builds remain queued despite apparently spare pool slots; repeatedly retrying or pushing produces more waiting runs.

**Cause:** exact-current-head checks reject legitimate prior main commits, while repeatedly probing only the two oldest runs can hide later ready jobs. Scheduling without active-assignment counts also lets one matrix dominate newly freed slots.

**Resolution:** in the infrastructure broker, verify main ancestry at discovery and dispatch, rotate bounded probes, paginate API results, and count each repository's active jobs. A hosted-first workflow needs an external recovery process; a static `runs-on` expression does not time out into another group. Deploy and verify that process before enabling `NG_HOSTED_FIRST`. Do not retry application failures or replay a partially executed publisher automatically.

**Reference:** private infrastructure [audit and rollout guide](https://github.com/snapetech-org/packer/blob/main/docs/runner-queue-audit-20261023.md). Live deployment remains open for this session.


### Loop: Shared JIT jobs can be selected but invisible to the broker

**Symptom**
- Trusted-main workflow runs remain queued even though their runner group includes the repository and workflow.
- Other repositories continue dispatching, so the pool looks healthy while the affected repository never reaches a worker.

**Why it's tricky**
- Organization runner-group selection and the broker GitHub App installation are independent access controls. Queue discovery only lists repositories visible to the App, so an omitted repository can be silently skipped without any capacity or worker-host failure.

**What works**
- Check the broker App installation's repository list as well as the runner group's repository and selected-workflow lists.
- Require every trusted-main route in the broker's expected App repository scope and keep a regression test for that invariant.
- Apply App access changes only with explicit action-time approval. Deploy broker changes after active JIT jobs drain naturally; do not freeze the broker cgroup because guests require its heartbeat.

**Where it's documented**
- `docs/reference/ci-runner-routing.md`
- `scripts/shared-jit-runner-pool/gha-runner-broker.py` and `test_policy.py` in the Packer repository
- `docs/linux-runner-pool.md` in the Packer repository

### Loop: Run-scoped runner labels require a union of bootstrap packages

**Symptom**
- A trusted-main JIT runner starts for one queued workflow job, but GitHub assigns it to another queued job in the same run and that job fails because a tool such as Docker is missing.

**Why it's tricky**
- The broker chooses a candidate job, while the repo/run-scoped runner labels allow GitHub to assign any eligible job in that workflow run. Per-job package selection therefore does not describe the runtime requirements of the runner identity. Job-scoped labels have a different contract and must remain isolated.
- GitHub's Jobs API returns display names, not YAML job IDs; matrix entries include their expanded display names. A package map can look correct in the workflow source yet miss the broker lookup key and silently omit tools.

**What works**
- Group candidates by their exact `runner_labels` and provide each candidate the ordered, deduplicated union of packages needed by all matching queued eligible jobs. Distinct job-scoped label sets keep distinct package requirements.
- Key package maps by the exact Actions API display names, including expanded matrix names. Build regression fixtures from API-shaped job objects and check tools called by workflow scripts, including direct `python` calls that need `python-is-python3` on the minimal guest.
- Recheck capacity immediately before starting the guest. If an operator drain begins while GitHub API calls are in flight, leave the job queued and remove the unused JIT registration.
- Keep the broker alive during a drain; already-admitted tasks can still finish API work. A pause marker only affects a broker process that loaded marker-aware capacity code: replacing its imported Python helper on disk does not update the running process. Verify the active process generation and repeat idle checks; a stale broker may keep scanning and refilling slots until the safe restart point.

**Where it's documented**
- Packer `scripts/shared-jit-runner-pool/gha-runner-broker.py` and `test_policy.py`
- Packer `docs/linux-runner-pool.md`

### Loop: Package-bootstrap guests can miss the runner-registration deadline

**Symptom**
- The broker dispatches a trusted-main job, but GitHub never reports the JIT runner online; after seven minutes the broker stops the guest and retries the still-queued job.
- This repeats for package-bootstrap workflows while jobs that need no bootstrap packages can start normally.

**Why it's tricky**
- Guest bootstrap waits for first-boot APT work, and `unattended-upgrades.service` can remain `active` as a systemd shutdown waiter after the package transaction has stopped. The wait then consumes most of the five-minute APT bound, leaving no time for package installation before the broker's seven-minute runner-registration deadline.
- Increasing the broker timeout alone would hide the false readiness signal and hold scarce JIT slots longer.

**What works**
- Treat only `apt-daily.service` and `apt-daily-upgrade.service` as active package transactions; do not wait on the shutdown-only `unattended-upgrades.service` state.
- Keep the guest APT wait bounded, preserve the broker registration deadline, and inspect the guest serial log when registration still does not occur.
- Keep the guest helper and broker source/test hashes aligned; atomically stage the tested helper and broker files with exact rollback snapshots.
- Do not restart the broker until local and remote JIT units are absent and GitHub has no online busy JIT runners, because guests need broker heartbeats.

**Where it's documented**
- Packer `scripts/shared-jit-runner-pool/gha-jit-vm.py`
- Packer `docs/linux-runner-pool.md`

### Loop: JIT jobs must provision the tools hosted runners include by default

**Symptom**
- A job registers and checks out successfully, then fails because an SDK is missing or its installer targets a system directory the unprivileged runner cannot write.
- Package-asset validation fails when it tries to install RPM or Debian tools through the system package manager.

**Why it's tricky**
- GitHub-hosted images include broad SDK and package-tool sets. The shared JIT image is intentionally smaller and jobs do not run as root, so a valid hosted workflow can fail only after it reaches the JIT worker.
- These failures look like repository build defects even though the runner was selected and started correctly.

**What works**
- Provision each required SDK in the workflow before its first use, pin the version, and set install paths to a writable workspace directory such as `${{ github.workspace }}/.dotnet`.
- Keep native package-asset checks on the hosted Ubuntu image used by the tagged release workflow; keep verification, smoke, and security jobs on the shared JIT pool.
- Add regression checks for tool setup, install paths, and step order so the workflow remains self-contained on the minimal image.

**Where it's documented**
- IPTV Tunerr `.github/workflows/ci.yml` and `docs/reference/ci-runner-routing.md`
- SlskR `.github/workflows/ci.yml` and `scripts/check-workflow-release-policy.sh`
- slskdN `.github/workflows/codeql.yml` and `scripts/check-codeql-dotnet-version.sh`

### Loop: Trusted-main builds can reach JIT runners but fail on package registries

**Symptom**
- A trusted-main job registers and checks out its repository, then dependency restore fails with HTTP 403 from the restricted egress proxy. Cargo, Go, and NuGet feeds can each use distinct hosts.

**Why it's tricky**
- Runner routing, App access, and proxy host policy are separate controls. The job looks correctly assigned, so the failure can be misdiagnosed as a repository or toolchain regression.
- Cargo's sparse index, crate downloads, Go's module proxy and checksum database, and Azure Artifacts feeds use distinct hosts. NuGet feed downloads can redirect to Azure Blob storage. A broad parent-domain exception would weaken the egress boundary.

**What works**
- Add only the exact package endpoints to the proxy allowlist, and reuse narrowly scoped artifact redirect suffixes only when the package service demonstrably uses them. Keep default-deny behavior for subdomains and lookalike suffixes.
- Test allowed hosts and rejected lookalikes, run a live CONNECT smoke check, then rerun the exact failed workflow on the unchanged current head.
- For Azure Artifacts, allow the canonical `pkgs.dev.azure.com` host; package content redirects to the already allowed `.blob.core.windows.net` suffix. Do not allow all of `*.dev.azure.com` or `*.visualstudio.com` to fix a modern feed that uses the canonical host.
- Keep resource admission gates intact; a correct proxy does not make a memory- or disk-gated worker available.

**Where it's documented**
- Packer `scripts/shared-jit-runner-pool/gha-egress-proxy` and `test_policy.py`
- Packer `docs/linux-runner-pool.md`

### Loop: QEMU can abort without leaving useful job diagnostics

**Symptom**
- A JIT workflow job remains in progress after its VM disappears, then fails when GitHub declares the runner lost. The guest step may be the last visible activity even though the host QEMU process aborted.

**Why it's tricky**
- The encrypted guest workspace is correctly destroyed after the VM exits, but the launcher previously sent QEMU stderr to `/dev/null`. Kernel logs may show no OOM, and core dumps may be disabled, leaving no evidence to distinguish an emulator fault from host pressure or a guest-triggered shutdown.

**What works**
- Capture QEMU stderr continuously so a full pipe cannot block the VM; sanitize it and retain only a small in-memory tail; write the tail to the local service log only on an unsuccessful exit.
- Preserve the failed job and host journal until diagnosis. Do not infer an OOM from a guest disappearing, cancel active current-head work, or weaken admission gates.
- Deploy the helper only after local and remote JIT units and online busy registrations are gone; preserve and verify an exact rollback copy before atomic replacement.

**Where it's documented**
- Packer `scripts/shared-jit-runner-pool/gha-jit-vm.py` and `test_policy.py`
- Packer `docs/linux-runner-pool.md`

### Loop: Identity leak checks must scan the submitted commit range

**Symptom**
- A trusted-ref identity scan fails on old commit trailers even when the current push contains no local identity terms.

**Why it's tricky**
- Scanning every commit since the latest release repeatedly re-reports already-published history. The repository must preserve published commit history, so rewriting those old trailers is not an acceptable fix.

**What works**
- On pushes, scan the exact `before..HEAD` range. On pull requests, scan the PR's reported commit count. For manual dispatch, scan the current commit. Keep scanning the current release-facing files in every mode, and fail closed for invalid ranges/counts.
- Do not suppress the identity check or rewrite published commits to clear a historical match.

**Where it's documented**
- `scripts/check-local-identity-leaks.sh`
- `.github/workflows/local-identity-leaks.yml`
- `scripts/tests/test_local_identity_scan_range.py`

### Loop: OIDC concurrent login attempts must keep separate state

**Symptom**
- Starting a second sign-in in another tab or window invalidates the first callback, or logout leaves pending state usable until it expires.

**Why it's tricky**
- A single fixed state-cookie name is overwritten by the newest attempt. Callback-scoped cookies are not sent to the separate logout route, so logout cannot discover them from the request cookie header.

**What works**
- Derive a distinct cookie name from each random state, cap pending transactions, consume each transaction once, and expire pending state-cookie names from the server-side transaction map during logout.
- Keep the cookie callback-scoped; do not broaden it to the whole app just to make it visible on logout.

**Where it's documented**
- `internal/webui/webui_oidc.go`
- `docs/how-to/oidc-control-deck.md`

### Loop: Control Deck auth cookies must not depend on request-derived HTTPS

**Symptom**
- Static analysis reports session and OIDC state cookies without `Secure` when the dashboard receives plain HTTP.
- A request-dependent cookie flag can make cleartext LAN browser sessions vulnerable to network capture, even if HTTPS reverse-proxy requests were recognized safely.

**Why it's tricky**
- The built-in listener serves HTTP, while the default dashboard is loopback-only and deployments may terminate HTTPS at a reverse proxy. Trusting forwarded scheme headers requires a proxy boundary; allowing HTTP for LAN login weakens credential/session confidentiality.

**What works**
- Always set `Secure` on Control Deck session and OIDC state cookies. Modern Chrome accepts and returns the cookie on the default loopback `127.0.0.1` HTTP origin; remote/LAN browser access must use HTTPS. Keep scriptable Basic auth behavior and CSRF checks covered by the existing tests.

**Where it's documented**
- `internal/webui/webui_auth.go`
- `internal/webui/webui_oidc.go`
- `docs/reference/cli-and-env-reference.md`
- `docs/how-to/oidc-control-deck.md`

### Loop: Recreating the removed cluster fallback after it caused split-brain DVR churn

**Symptom**
- Agents look for or recreate the removed manifest tree, service-DNS URLs, cluster command workflows, or cluster deploy scripts when Plex registration breaks.
- Plex accumulates empty DVR rows or flips between conflicting tuner URLs.

**Why it's tricky**
- Older repo history had many examples for that path, so search results suggested it even when the active system is bare-metal/systemd.
- Multiple registrars using the same Plex device IDs cause Plex DVR split-brain and guide reload churn.

**What works**
- Do not recreate the removed path. Use binary, Docker, or systemd/bare-metal paths only.
- Keep exactly one owner for each Plex `IPTV_TUNERR_DEVICE_ID`.
- If Plex has empty DVR rows, delete only `0/0` IPTV DVR rows after verifying the live non-empty DVRs.

**Where it's documented**
- `memory-bank/known_issues.md`
- `docs/how-to/deployment.md`

## Loop protocol
- If you attempt the same approach twice and it still fails, stop and collect evidence before trying a new strategy.
- Do not silence failures; add a repro or focused test and fix the root cause.
- Do not revert unrelated user changes.

### Loop: Misreading slow Plex guide fill as slow Tunerr XMLTV serving

**Symptom**
- Plex Live TV rows fill in very slowly after guide reload even though Tunerr `/guide.xml` returns `200` quickly.

**Why it's tricky**
- Plex can fetch and index the XMLTV channelmap, but the later full channel activation PUT can take more than a minute for large lineups. If Tunerr times out early, the DVR can appear under-activated or empty in Plex UI while `/guide.xml` itself is healthy.

**What works**
- Measure Tunerr first: `/guide.xml` status, `X-Iptvtunerr-Guide-State`, byte size, channel/programme counts, and response time.
- Check per-DVR `ChannelMapping` counts, not only summary `<Channel>` counts from `/livetv/dvrs`.
- Allow a longer timeout for full channelmap activation; Plex treats activation as a full replacement, so do not split the mapping into batches.

**Where it's documented**
- `internal/plex/dvr.go`

### Loop: Sports lineup probe can collapse the live Plex DVR lineup

**Symptom**
- Sports Live TV channels disappear or click-to-play spins, while the sports tuner `/lineup.json` returns `[]`.

**Why it's tricky**
- A bounded runtime probe can decide no sports feeds are healthy and publish an empty lineup. Plex then reloads/activates the empty lineup, so the UI still has guide/provider state but no usable tuner rows.

**What works**
- Check the tuner directly before blaming Plex: `curl http://127.0.0.1:5005/lineup.json | jq length`.
- For production recovery, disable `IPTV_TUNERR_LINEUP_PROBE_ENABLED` on the sports service, restart `iptvtunerr-sports.service`, wait for lineup rebuild, then confirm Plex channel activation completes.
- Keep visual/probe cache changes out of emergency recovery unless the provider health issue is already understood.

**Where it's documented**
- `/etc/iptvtunerr/sports.env` on the scratch worker

### Loop: Event-only sports rows need real DVR-sized guide windows

**Symptom**
- Plex can tune an event sports channel, but recording from the guide fails with a vague client-side error such as "undefined".
- The tuner stream URL works, `/lineup.json` includes the event, and `/guide.xml` is `ready`, so the failure looks like Plex rather than Tunerr.

**Why it's tricky**
- Event rows from provider names like `NEXT | ... Sun 17 May 19:00 EDT ...` may not have upstream EPG programme data or a TVGID.
- The generic no-EPG fallback used to publish a week-long placeholder programme named after the channel. That keeps channels visible, but it is a poor DVR scheduling target for one-off sports events.

**What works**
- For live/next sports rows with parseable event times, publish a bounded programme window at the event time instead of the week-long placeholder.
- Use sport-aware default durations for Plex-facing guide metadata: basketball/hockey about 3.5h, soccer/rugby about 2.5h, baseball about 4.5h, and add extra padding for Game 7/finals/playoff wording.
- Treat timezone abbreviations explicitly; do not rely on Go's generic `MST` parser for `EDT`/`NDT` because unknown abbreviations can collapse to UTC.

**Where it's documented**
- `internal/tuner/epg_pipeline.go`
- `internal/tuner/xmltv_test.go`

### Loop: Generic live sports titles can become stale Plex recording markers

**Symptom**
- Plex shows a red recording marker on current guide rows that are not being recorded, often after recording a similarly named sports block days earlier.
- External/shared users may hit a Plex Web "Something went wrong" view when opening the row because Plex follows stale recording/subscription state instead of plain Live TV playback.

**Why it's tricky**
- Some provider XMLTV rows publish generic titles such as `Live: NBA Basketball` with no subtitle/date/episode identity. Plex can collapse those rows into a title-only XMLTV movie GUID such as `tv.plex.xmltv://movie/Live%3A%20NBA%20Basketball`.
- The tuner streams stay healthy, so the failure looks like playback or entitlement until `/media/subscriptions` reveals the stale title-only subscription.

**What works**
- Add deterministic programme identity to recurring event-like XMLTV rows without changing the visible title: date-specific `sub-title`, `date`, and `episode-num system="iptvtunerr"` generated from channel/start/stop/title metadata.
- Cover all generic `Live:` rows plus generic sports titles such as plain `NBA Basketball`, because providers are inconsistent about including the `Live:` prefix.
- Delete the stale Plex subscription keyed to the generic title, reload the Plex DVR guide, and verify `/media/subscriptions` no longer contains the title-only XMLTV GUID.
- 2026-05-28 recurrence: a tester saw Plex Web `s1002 (Network)` on `Live: NBA Basketball` even though the tuner stream and current guide identity were healthy. Plex still had a stale title-only `Live: NBA Basketball` media subscription; deleting only that subscription cleared `/media/subscriptions`.
- If the stale marker is gone but shared-user playback still returns `s1002`, verify the PMS advertised connection state before widening proxy classifiers again. The documented working state is static manual port `443`, relay disabled, and the public media HTTPS custom connection; automatic port drift back to `32400` can make clients miss the entitlement proxy.

**Where it's documented**
- `internal/tuner/xmltv.go`
- `internal/tuner/epg_pipeline.go`
- `internal/tuner/xmltv_test.go`

### Loop: Short recurring shows can save as title-only one-shot rules

**Symptom**
- Plex accepts Record for a short recurring programme such as `etalk`, but the guide row does not show the red scheduled-recording marker.
- `/media/subscriptions` shows a title-only XMLTV movie rule such as `tv.plex.xmltv://movie/etalk`, while `/media/subscriptions/scheduled` has no matching future grab.

**Why it's tricky**
- The proxy save/read flow can be healthy; the failure is Plex matching a metadata-poor programme title instead of a specific guide airing.
- Plex may create a one-shot subscription with `startTimeslot=Any`, so it looks saved but is not bound to the selected airing.

**What works**
- Metadata-poor short recurring rows need the same kind of deterministic XMLTV identity as generic sports rows: date-specific `sub-title`, `date`, and `episode-num system="iptvtunerr"`.
- Preserve rows that already carry episode metadata; do not overwrite upstream `sub-title`, `date`, or `episode-num`.
- Delete stale title-only subscriptions after deploying the fixed guide identity, reload Plex guides, and have the tester recreate the recording.

**Where it's documented**
- `internal/tuner/xmltv.go`
- `internal/tuner/xmltv_test.go`

### Loop: Plex Record Options uses subscription reads before save

**Symptom**
- Shared Plex users can open the guide and Live TV rows, but the Record Options dialog fails with "There was a problem saving your changes. Please try again."
- PMS logs show shared-user `403` responses for `/media/subscriptions` or `/media/subscriptions/scheduled`, sometimes before any tuner stream request reaches Tunerr.

**Why it's tricky**
- These read endpoints do not carry the XMLTV `guid`, `key`, or `uri` parameters that identify a specific Live TV programme, so a classifier that only recognizes XMLTV template/create requests misses them.
- The failure appears in the Plex client as a generic save error even though it is an entitlement failure during Record Options discovery.

**What works**
- Treat read-only `GET /media/subscriptions` and `GET /media/subscriptions/scheduled` as Live TV discovery inside the Live TV proxy.
- Keep mutating subscription requests, including `/media/subscriptions/{id}` rule edits and `POST /media/subscriptions` saves with Plex `hints[guid]`/`hints[ratingKey]` query parameters, scoped to XMLTV-backed bodies or query parameters so ordinary library subscription creation or id-only deletes are not elevated.

**Where it's documented**
- `internal/plexlabelproxy/entitlement.go`
- `internal/plexlabelproxy/proxy_test.go`

### Loop: Plex Web rolling Live TV playback deletes a subscription id

**Symptom**
- Plex Web reports `s1002 (Network)` after a shared user starts a Live TV stream.
- Tuner and PMS evidence can look healthy: the tune request succeeds, the tuner serves bytes, PMS creates a Live TV transcode, and the client pulls HLS segments.
- Proxy access logs then show `DELETE /media/subscriptions/{id}` denied as `403`. Plex Web may send playback evidence as query `X-Plex-Playback-Session-Id`, query `X-Plex-Session-Id`, or the corresponding request headers.

**Why it's tricky**
- The request is a mutating `/media/subscriptions/{id}` path, so broad elevation would risk ordinary library subscription changes.
- The rolling Live TV cleanup request does not carry XMLTV `guid`, `key`, `uri`, or `hints[...]` values; it carries playback-session evidence instead.

**What works**
- Elevate only `DELETE /media/subscriptions/<numeric-id>` when playback/session evidence is present and non-empty: `X-Plex-Playback-Session-Id` or `X-Plex-Session-Id` in either query parameters or headers.
- Keep plain id-only deletes, product-only deletes, nested scheduled paths, and library subscription edits non-elevated.
- Validate with both classifier tests and live proxy logs: the playback/session DELETE should log `live_tv=true`; the no-session neighbor should remain `live_tv=false`.

**Where it's documented**
- `internal/plexlabelproxy/entitlement.go`
- `internal/plexlabelproxy/proxy_test.go`

### Loop: Ingress encoded-slash deny blocks Plex Live TV transcode before the proxy

**Symptom**
- Plex Web reports `s1002 (Network)` immediately after a successful Live TV tune.
- PMS starts a Live TV session/transcode, but the client-facing `/video/:/transcode/universal/*` and `/:/timeline` calls return tiny `404` responses and do not show up in the Plex Live TV proxy journal.

**Why it's tricky**
- The tuner, PMS transcode worker, and proxy tune/subscription paths can all look healthy.
- Plex Web legitimately sends encoded slashes in query values such as `path=%2Flivetv%2Fsessions%2F...`; a generic public-ingress abnormal-URI rule that scans the full URI can block that before the proxy ever sees it.

**What works**
- Compare the same encoded transcode URL through direct PMS, direct Go proxy, and HTTPS ingress. If only HTTPS ingress returns the 9-byte `404`, fix ingress routing before changing tuner or proxy classifiers again.
- Keep the ingress exception narrow to Plex playback paths that need encoded Live TV session values, such as `/video/:/transcode/*`, `/:/timeline*`, and `/playQueues*`.
- After the ingress change, validate that the encoded `/video/:/transcode/universal/decision` request reaches `plex-live-tv-proxy.service` and logs `live_tv=true stream=true`.

**Where it's documented**
- `memory-bank/current_task.md`

### Loop: Proxy hotfix validated from `/tmp` but not active in systemd

**Symptom**
- A proxy classifier fix appears tested and "deployed", but the next tester retry shows no change.
- `journalctl -u plex-live-tv-proxy.service` shows zero expected `plexlabelproxy_access`/`plexlabelproxy_audit` lines for the retry, or the service process start time predates the hotfix.

**Why it's tricky**
- Building a corrected binary under `/tmp` and running synthetic checks can look like a live deploy if the active `/opt/iptvtunerr/iptv-tunerr-proxy` path and systemd process are not checked afterward.
- Caddy may still route correctly to the proxy, so this is not an ingress bypass; it is an inactive binary/process problem.

**What works**
- After every proxy hotfix, verify all three facts: `/opt/iptvtunerr/iptv-tunerr-proxy version` reports the intended build, `ps` shows a fresh `plex-label-proxy` process, and live journal lines from `127.0.0.1:33240` show the expected classification.
- Use the exact failing request shape for live validation, then also validate a neighboring non-elevated shape to prove the classifier stayed narrow.

**Where it's documented**
- `memory-bank/task_history.md`
- `memory-bank/current_task.md`

### Loop: Plex Record Save may need subscription detail reads after create

**Symptom**
- A shared user can click Record and the client appears to accept the action, but the guide row does not show a scheduled/recording marker afterward.
- The save path may be fixed already, so there is no obvious client-side "problem saving your changes" error.

**Why it's tricky**
- Plex can follow a successful Live TV subscription save with read-only detail/scheduled requests such as `/media/subscriptions/{id}` or `/media/subscriptions/scheduled/{id}` that carry only a numeric rule id.
- If those reads use the shared user's token, Plex may fail to render the DVR rule state even though the create request succeeded.

**What works**
- Treat read-only `/media/subscriptions*` paths as Live TV DVR discovery, except `/media/subscriptions/template`, which must still require Live TV/XMLTV evidence so ordinary library subscription templates are not elevated.
- Keep mutating subscription edits scoped to XMLTV evidence, and scan both query strings and form bodies for Plex `hints[...]` fields such as `hints[ratingKey]`.
- When validating a tester retry, check both the client marker and proxy access lines for all `/media/subscriptions*` requests around the click.

**Where it's documented**
- `internal/plexlabelproxy/entitlement.go`
- `internal/plexlabelproxy/proxy_test.go`

### Loop: Publisher success can leave a release channel on an older build

**Symptom**
- Release jobs report success, but a package registry still serves an older version or a container compatibility name has a stale `latest` tag.
- A manual retry for an older release can move a mutable channel pointer backward.

**Why it's tricky**
- Some workflows stop at upload or PR creation. Launchpad source acceptance precedes amd64 binary publication, and Chocolatey, Winget, and Snap may have separate store review after submission.
- Launchpad's source query can briefly omit an already-seen record while publication progresses. A cumulative missing-record counter can therefore report a false rejection after many intermittent misses; a successful build record also does not prove that its binary reached the apt archive.
- AUR/PPA/COPR workflows can be triggered both by GitHub Release events and by a release dispatcher, causing duplicate writes for the same tag.
- The dispatcher runs with Bash nounset enabled, but its deferred-channel environment variable is optional. Reading `SKIP_RELEASE_CHANNELS` without a default stops the release workflow before any publisher is dispatched.
- GitHub can leave a workflow-run summary at `in_progress` after every job has completed. Waiting only on the summary can therefore hold the release dispatcher until its long timeout even when the publisher result is already known.

**What works**
- Dispatch all release publishers once from the GitHub Release workflow and wait for every publisher workflow to finish.
- Expand the optional skip list with `${SKIP_RELEASE_CHANNELS:-}` and test both the unset case and a configured skip list through a fake GitHub CLI; this keeps the recovery path deterministic without touching live channels.
- If the run summary is nonterminal, inspect its jobs endpoint. Use a fallback only when the API reports at most 100 jobs and every job is terminal; classify success only when every job concluded `success`, and keep waiting if job results are missing or active.
- Check that a submitted tag is the latest stable tag on `main`; serialize each channel so newer release work supersedes older work.
- For PPA, wait for the exact source version and its amd64 binary publication. Audit registries and store review separately from workflow completion.
- Reset the Launchpad missing-source counter whenever the exact source record reappears, inspect that source record's amd64 build state, and query its own binary publications before reporting success.
- Publish all configured container names with `latest` and the exact release tag from one build.

**Where it's documented**
- `scripts/dispatch-release-channels.sh`
- `scripts/check-release-channel-wiring.sh`
- `docs/how-to/release-channels.md`

### Loop: BuildKit can fail to resolve package mirrors from its isolated network

**Symptom**
- A Docker image builds locally, while GitHub Actions BuildKit cannot resolve either Alpine or Debian package mirror hosts. Retrying packages or changing distributions does not fix a resolver path failure.

**Why it's tricky**
- Local Docker and hosted BuildKit use different network paths. The builder container's isolated network can fail to resolve the same hosts the runner can reach. A local mirror fallback smoke does not prove that the release workflow can build and push all target architectures.

**What works**
- Configure Docker Buildx with `driver-opts: network=host` so BuildKit can use the GitHub runner's DNS path, then verify the full multi-architecture publisher on GitHub Actions.
- Use Debian Bookworm package archives for runtime FFmpeg dependencies and retain the Alpine Go builder; retries only help transient errors and cannot repair inaccessible DNS.

**Where it's documented**
- `Dockerfile`
- `.github/workflows/docker.yml`
- `docs/how-to/release-channels.md`

### Loop: Plex Web asks for JSON provider metadata, bypassing XML-only entitlement rewrites

**Symptom**
- External/shared users can see guide/provider rows, but clicking Live TV spins and the tuner never receives `/stream/...`.
- Proxy logs show elevated `/media/providers`, while PMS/Tunerr logs show no tune or active stream.

**Why it's tricky**
- The proxy originally rewrote `allowTuners` only in XML. Plex Web can negotiate JSON for `/media/providers`, so the request is elevated but the client-side entitlement hint can still say tuners are not allowed.

**What works**
- Rewrite `allowTuners` in both XML and JSON bodies, keeping the rewrite narrow to entitlement hints only.
- Verify with `Accept: application/json` against the proxy and confirm `allowTuners` is true.

**Where it's documented**
- `internal/plexlabelproxy/entitlement.go`
- `internal/plexlabelproxy/proxy.go`

### Loop: Self-hosted COPR runner reuses stale SRPMs

**Symptom**
- A COPR workflow rerun for a newer tag succeeds, but COPR receives an older `iptvtunerr-X.Y.Z-1.src.rpm`.
- GitHub Actions reports a green upload because `copr-cli build` accepted an SRPM, but the selected file was not built for the requested tag.

**Why it's tricky**
- Self-hosted runners can retain `~/rpmbuild/SRPMS` between jobs.
- A pattern like `find ~/rpmbuild/SRPMS -name '*.src.rpm' -print -quit` silently chooses the first leftover file, which may be from a prior release.

**What works**
- Build each release SRPM under an isolated temporary RPM topdir.
- Compute and test the exact expected path, `iptvtunerr-${version}-1.src.rpm`.
- Copy that file to a relative workspace path such as `dist/iptvtunerr-${version}-1.src.rpm` and pass that explicit path to `copr-cli build`.
- Verify the COPR upload log contains the requested release version, then watch the external COPR build to success.

**Where it's documented**
- `.github/workflows/release-copr.yml`
- `memory-bank/task_history.md`

### Loop: `dotnet pack` strips Chocolatey-specific nuspec metadata

**Symptom**
- A Chocolatey moderation repush succeeds at the API level, but the raw `.nupkg`
  still lacks metadata that exists in `packaging/chocolatey/iptvtunerr.nuspec`,
  especially `packageSourceUrl`, `projectSourceUrl`, `docsUrl`, and
  `bugTrackerUrl`.

**Why it's tricky**
- `dotnet pack` creates a valid NuGet package and exits successfully, but it
  rewrites the root nuspec to a NuGet schema that drops Chocolatey-specific
  metadata fields. The push can still return `Created`, so workflow success
  alone does not prove the moderation comments were fixed.

**What works**
- After `dotnet pack`, replace the generated root `iptvtunerr.nuspec` inside
  the `.nupkg` with the exact Chocolatey nuspec written by the workflow.
- Before pushing, reopen the `.nupkg` and assert the required moderation tags
  are present.
- Use unique runner-temp pack directories and bounded native command timeouts;
  the self-hosted runner temp area may contain state from prior attempts.
- Prefer split Bash steps over a monolithic PowerShell pack/push block on Linux
  runners, because a pre-output stall in PowerShell gives little actionable log
  context.
- After pushing a moderated version, download the raw package feed and inspect
  the packaged nuspec headlessly instead of trusting the public package page,
  which may lag or show cached review content.

**Where it's documented**
- `.github/workflows/publish-chocolatey.yml`
- `docs/how-to/release-channels.md`
- `memory-bank/task_history.md`

### Loop: Transient Launchpad API errors can fail a still-pending PPA release

**Symptom**
- A PPA workflow fails during a source/build/binary status poll even though the Launchpad source upload remains accepted and pending.
- The workflow error is an intermittent HTTP 5xx response from the Launchpad API, not a rejected source or failed package build.

**Why it's tricky**
- The polling step runs with `set -e`; an exhausted `curl --fail` retry exits the step before the next scheduled poll.
- Source upload, successful build, and published binary are separate states. Launchpad may expose the exact source record only after the previous short missing-source cutoff has nearly elapsed, and equal per-request and total retry limits leave no retry time after a full request timeout.

**What works**
- Route every Launchpad GET through one helper with bounded curl retries for transient HTTP/network failures.
- Keep the total curl retry budget longer than each transfer's `--max-time`, or one full timeout can consume the budget without a retry.
- Allow 20 minutes for the exact source record to appear, then start a separate 70-check binary-publication window; a source upload or build record is not release availability.
- Keep terminal source/build/binary failures fatal and keep waiting until the exact amd64 binary publication is `Published`.
- If the source upload has already succeeded but polling failed, monitor those exact source/build records first. Do not blindly rerun the full publisher: it creates a new timestamped source version and another build rather than reusing the existing upload.
- After a retry-policy change, use a publisher retry only when it can safely reuse the existing source or when a deliberate new source revision is warranted; verify exact source and binary records in Launchpad.

**Where it's documented**
- `.github/workflows/release-ppa.yml`
- `scripts/check-release-channel-wiring.sh`
- `docs/how-to/release-channels.md`

### Loop: Classifier matches a prefix on a path the upstream resolves differently

**Symptom**
- A new Live TV route is added to `IsLiveTVRequest` by `HasPrefix`/`Contains` and the owner token starts riding on requests that are not Live TV (e.g. `/livetv/../security/token`).

**Why it's tricky**
- Classification uses `EscapedPath()` and the reverse proxy forwards the path verbatim, so anything PMS normalizes (dot segments, `%2e`, `%2f`) can escape the classifier's intent. Encoded slashes in *query values* are legitimate (Plex Web `path=%2Flivetv%2F...`), so a blanket URI scan breaks playback.

**What works**
- Every elevation goes through `pathSafeForElevation` (path only, never query) and `liveTVRoutePath`; the canonical-path gate, not route breadth, is what makes the prefix safe. Any elevated response is scrubbed by `scrubOwnerToken`, so a mistaken elevation cannot hand the owner token back to the client.

**Where it's documented**
- `internal/plexlabelproxy/hardening.go`, `internal/plexlabelproxy/hardening_test.go`, `docs/runbooks/plex-live-tv-entitlement-proxy.md`

### Loop: Updating a prepared changelog after adding a release fragment

**Symptom**
- `scripts/prepare-release-changelog.mjs --version vX.Y.Z` refuses to update a versioned changelog section that was prepared before a new fragment was committed, even though the release tag does not exist yet.

**Why it's tricky**
- The generator only prepends a new version section; it treats any existing section as a duplicate and exits before comparing it with the current fragments.
- Published sections are append-only, so the correct fix depends on whether the release tag already exists.

**What works**
- If the tag is not published, preview all fragments from the previous release through `HEAD` with `scripts/preview-release-notes.mjs`, then update the untagged section to match that output.
- After creating the release tag, run `scripts/verify-release-changelog.mjs vX.Y.Z` to validate the section against the tagged fragments. Do not edit a section after its tag is published.

**Where it's documented**
- `scripts/prepare-release-changelog.mjs`
- `scripts/preview-release-notes.mjs`
- `scripts/verify-release-changelog.mjs`

### Loop: Local Semgrep engine fails when io_uring is unavailable

**Symptom**
- The local Semgrep pre-commit scan exits with `Cannot allocate memory io_uring_queue_init`, including when limited to one worker.

**Why it's tricky**
- Semgrep's Linux backend can try to create io_uring queues even in a one-worker scan. Container and sandbox policies may block or constrain io_uring, so reducing worker count alone is intermittent.

**What works**
- Set `EIO_BACKEND=posix` for Semgrep. `scripts/security-scan-local.sh` defaults to this backend and supports `SEMGREP_EIO_BACKEND` to override it. Keep the staged scan and report-only finding policy intact.

**Where it's documented**
- `scripts/security-scan-local.sh`
- `docs/security-scanning.md`

### Loop: First Basic-authenticated Control Deck page has no CSRF bootstrap

**Symptom**
- A browser opens the Control Deck through HTTP Basic auth, but creating a user or M3U account returns `403 missing csrf token`.

**Why it's tricky**
- The authentication middleware creates the session and sets its cookie on the page response. The request that renders the SPA cannot read that response cookie, so a bootstrap that only looks at request cookies omits the token even though the UI has just been authenticated.

**What works**
- After validating HTTP Basic credentials, carry the server-created session token through a private request-context key while rendering that same page. Resolve it through the in-memory session table and keep the existing session-bound CSRF check on every mutation.
- Cover the first page load, user and M3U account writes, a rejected missing-token request, and database persistence after reopening the SQLite file.

**Where it's documented**
- `internal/webui/webui_auth.go`
- `internal/webui/spa.go`
- `internal/webui/webui_test.go`

### Loop: Static-analysis response and process findings need sink-specific review

**Symptom**
- Broad rules flag every direct `http.ResponseWriter` write as XSS or every non-literal `exec.Command` executable as command injection, even when a handler emits JSON/XML/media or passes a validated value as one argv element.

**Why it's tricky**
- The rule sees a sink but not the response content type, encoder, protocol, or trust boundary. Replacing safe protocol output with HTML templates or putting argv behind a shell can introduce regressions while leaving the actual active-content path exposed.

**What works**
- Trace each sink to its encoding and declared content type. Use `html/template` for HTML, typed encoders for JSON/XML, and downgrade active HTML/SVG/JavaScript/XML proxy responses while setting `nosniff`. For process calls, keep argv execution without a shell and validate the executable and any untrusted inputs. Dismiss only reviewed locations with a comment naming the source-level boundary.

**Where it's documented**
- `internal/tuner/gateway_stream_response.go`
- `internal/webui/`
- `scripts/security-scan-local.sh`

### Loop: M3U metadata must be safe for the playlist grammar

**Symptom**
- Channel metadata containing carriage returns, line feeds, quotes, or commas can break generated M3U fields or insert additional playlist directives.
- Broad response-writer XSS rules flag these handlers even though M3U is not HTML, which can obscure the separate playlist-injection risk.

**Why it's tricky**
- HTML escaping does not protect M3U syntax. Several endpoints build playlists from provider catalog values or operator-managed virtual-channel fields, and their quoting and line rules differ from HTML.

**What works**
- Sanitize control characters before writing any M3U line; normalize display names and quotes/commas for attributes. Keep URL attributes readable and preserve each export's existing MIME and URL layout.
- Apply the shared helpers to regular, sports, virtual-channel, and Xtream exports, with a regression asserting injected metadata cannot create a new directive line.

**Where it's documented**
- `internal/tuner/m3u.go`
- `internal/tuner/server_sports.go`
- `internal/tuner/server_virtual_channels.go`
- `internal/tuner/server_xtream.go`

### Loop: Failed release tags can leave changelog history ahead of the last public release

**Symptom**
- A stable-looking tag exists, but its Release workflow stops before publishing assets because the candidate changed sections belonging to earlier releases.
- A retry against the same tag is blocked by the immutable changelog/tag contract, and the original release-note fragments are now considered consumed by that tag.

**Why it's tricky**
- The release-note generator builds the new section, while historical changelog sections are maintained across many commits. A release candidate can therefore contain a correct new section and still remove older published entries.
- Once pushed, the tag must stay put even when the workflow fails before the GitHub Release exists. Reusing the consumed fragments or force-moving the tag would make the release record ambiguous.

**What works**
- Keep the failed tag immutable and verify that no GitHub Release was published.
- Use the next patch version for the first public release; copy forward the complete user-facing fragment set under new filenames.
- Restore only the affected historical sections byte-for-byte from the last published tag. If necessary, add a tag-specific exception that accepts only those exact source sections, with regression tests for both the allowed restoration and arbitrary edits.
- Keep future release generation append-only and run the tag-history check before pushing a tag.

**Where it's documented**
- `release-notes/README.md`
- `scripts/changelog-history-exceptions.json`
- `scripts/check-changelog-tags.mjs`
- `scripts/check-changelog-tags.test.mjs`
- `internal/tuner/m3u_test.go`
