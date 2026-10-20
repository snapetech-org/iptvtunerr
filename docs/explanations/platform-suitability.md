---
id: platform-suitability
type: explanation
status: stable
tags: [deployment, unraid, resources]
---

# Platform suitability: Unraid

Unraid is a close fit for media-server operators because it already
distributes Docker containers through Community Applications.

## Unraid

Unraid Community Applications already lists IPTV tuner bridges such as
Threadfin, an M3U proxy for Plex, Emby, and Jellyfin based on xTeVe. Tunerr uses
the same container workflow and ships its existing GHCR image, so the Unraid
package is a small template repository rather than a new runtime or container
build.

This is a good fit for a server that already runs Plex or another media server.
The official [Threadfin listing](https://ca.unraid.net/apps/threadfin-0zwbhsx0jsbxj1)
shows this tuner/middleware category is already represented in Community Apps.
Unraid runs on 64-bit x86 hardware; this template uses Tunerr's existing amd64
GHCR image. Use persistent appdata for the catalog and operator state, and keep
the tuner ports inside a trusted LAN or VPN.

## Hardware and workload

The application is a static Go service with optional ffmpeg integration. Idle
load is modest; active relay load is driven mainly by stream bitrate and
concurrency. Transcoding, recordings, large catalogs, and guide refreshes add
CPU, RAM, and storage demand. Unraid NAS hosts can run basic workloads, but low-end
systems are not guaranteed to handle many simultaneous streams or transcoding. Tunerr has no published,
measured CPU/RAM minimum yet.

The practical capacity is therefore determined by the host's network path,
the number and bitrate of concurrent streams, and which optional ffmpeg and
recording features are enabled. A typical modern Unraid NAS should have enough capacity for a basic relay workload; that is an
inference from the service's streaming architecture, not a Tunerr benchmark.
Measure the actual host before relying on multiple transcodes or recordings.

## Catalog gate

Unraid Community Applications requires a public repository with a root
`ca_profile.xml`, an OSI-approved `LICENSE`, and valid Docker template XML.
The submission portal provides Validate and Scan before moderator review. See
the [current Unraid submission requirements](https://ca.unraid.net/submit/help).

See also
--------

- [Unraid installation](../how-to/install-on-unraid.md)
- [Deployment overview](../how-to/deployment.md)

Related ADRs
------------
