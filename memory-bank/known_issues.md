# Known issues

## Plex / Deployment

- **The old local split-brain Tunerr/Plex fallback is intentionally removed (2026-05-12).** Do not recreate local production jobs that register the same Plex DVR identity as the systemd-owned host. Active supported deployment paths are binary, Docker, systemd/bare-metal, and k3s when k3s is the single owner for its Plex DVR identity.

- **Plex can report a DVR device as `dead` even when enabled channel mappings are healthy.** The watchdog must not recreate a mapped DVR solely because of that flag; recreate only when mappings are missing or badly under-activated.

## Security

- **OIDC was source-tested but not exercised against a live provider tenant in the 2026-10-21 cross-project audit.** Provider callback behavior can still depend on a specific tenant's claim shape, client registration, or browser cookie policy. Use the provider-specific setup and troubleshooting checklist before relying on OIDC as the only Control Deck sign-in path.

- **Credentials:** Keep secrets in `.env`, environment variables, host-local service environment files, or a dedicated app-managed secret file with owner-only permissions. The Sports page stores only the API-Sports key this way; never expose or commit its state file. `.env` is ignored, and secrets must never be logged.

- **Live TV abuse blocking must not override valid Plex authorization.** A source/IP block can be triggered by missing-token probes from Plex clients or shared networks. The proxy must allow an already-authorized Plex token to bypass the source block while continuing to deny missing or unauthorized tokens.

- **Tuner APIs remain unauthenticated on the trusted network.** The media-server-facing listener on port `5004` has no general authentication. Operator handlers are localhost-only by default, but `IPTV_TUNERR_UI_ALLOW_LAN=1` exposes those operator routes to LAN clients without app credentials. Do not expose port `5004` to the internet; use this LAN option only on a trusted network.

## Sports Automation

- **Event matching depends on provider and guide naming.** API-Sports does not provide channel carriage or stream URLs. Tunerr matches canonical team names/codes in current lineup channel names and merged XMLTV programme text, with programme starts limited to three hours from the canonical start. Provider-specific abbreviations and generic guide labels can leave a real game unmatched. Only current lineup channels with streams can be generated.
- **Canonical schedule coverage is limited to MLB, NFL, NCAA Football, and NBA.** NFL and NCAA Football share one API-Sports product endpoint and are partitioned by league ID. NBA uses API-NBA v2; other Basketball leagues and cups still rely on provider/EPG behavior without canonical schedule matching.

## Release / Packaging

- **Unraid repository license detection needs a canonical root license file.** The app's dual-license preamble made GitHub report `Other`, which the Community Applications portal rejected. Keep the license file the portal scans as standard AGPL-3.0 text for Licensee detection; document the `-or-later` grant and commercial alternative in the package README. Verify with `gh repo view OWNER/REPO --json licenseInfo` after syncing.
- **Winget ZIP manifests must point at the executable inside the archive directory.** The Windows release ZIP contains `iptv-tunerr-vX.Y.Z-windows-amd64/iptv-tunerr.exe`, not a root-level `iptv-tunerr-vX.Y.Z-windows-amd64.exe`. A wrong `NestedInstallerFiles.RelativeFilePath` downloads and hashes fine but fails Microsoft install validation.
- **Docker Hub compatibility image is stale; follow-up deferred.** For `v0.1.87`, the configured Docker Hub identity published `snapetech/iptvtunerr` but received `insufficient_scope` when pushing `keefshape/iptvtunerr`; GHCR and the primary Docker Hub tags are current. Per the user's latest direction, do not rerun Docker Hub publishing or seek additional access for now.
- **Launchpad binary publication is asynchronous.** For `v0.1.88`, the exact Jammy and Noble source packages are `Published`; both amd64 builds report `Successfully built` on the latest check, though Noble's state varied between processing states across earlier API reads. Neither exact amd64 binary is published yet. The first poll stopped on HTTP 502; later polling exposed delayed source visibility and a 30-second timeout that exhausted an equally short retry budget. The publisher now allows 20 minutes for source discovery and 180 seconds of curl retries per request. Report the channel as updated only after both exact amd64 binaries reach `Published`.
- **PPA signing key mismatch blocks automated publication.** The `v0.1.92` and `v0.1.93` Jammy and Noble publisher jobs failed signing with `No secret key` for the configured signer. Restore the matching signing key or update the signer configuration before treating PPA publication as available.
- **Chocolatey package availability still depends on moderation.** The automated `v0.1.92` push returned `403 Forbidden`, while the `v0.1.93` workflow push was accepted. Check the package's moderation state before claiming it is publicly available.

## Repository remotes

- **Pre-release Local Identity Leak Check matched existing dependency-update co-author trailers.** Five commit-message lines matched the private identity denylist before `v0.1.86` was tagged. Post-release main checks pass because the latest-tag scan baseline advanced; the historical lines remain in their original commits. Preserve history as directed and do not weaken the scanner.
- **GitLab mirror ref listing currently fails, and its cached `main` contains an extra donation-links commit.** The GitLab SSH service reports a multi-pack-index signature error during ref listing and rejected a normal push of verified merge candidate `4b5d293` with an index-pack error. The candidate retains the GitLab-only commit; no remote ref changed. Repair the server-side repository index, read current refs, and retry only as a normal fast-forward. Never force-push this branch.
- **The pushed `v0.1.94` tag has no GitHub Release.** Its Release workflow stopped at the changelog-history gate before verification or asset generation because the candidate removed entries from published sections `v0.1.74`, `.75`, and `.77`. Keep the tag immutable; publish the complete fixes from successor `v0.1.95` after restoring those sections exactly from `v0.1.93`. See the release-history recovery loop in [recurring_loops.md](recurring_loops.md).
- **`v0.1.95` release channels need a dispatcher recovery.** The GitHub Release and its 18 assets published, but the first channel-dispatch job exited before running any publisher because `SKIP_RELEASE_CHANNELS` was unset under Bash nounset. The script default and regression tests are now fixed; dispatch `.95` channels from `main` and record each result. The existing PPA signing-key issue remains separate.
