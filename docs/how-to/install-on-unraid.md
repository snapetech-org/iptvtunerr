---
id: install-on-unraid
type: how-to
status: stable
tags: [deployment, unraid, docker]
---

# Install IPTV Tunerr on Unraid

The Unraid package runs Tunerr's existing GHCR container and stores its state
on persistent appdata paths. Unraid runs on 64-bit x86 systems; this template
uses the amd64 image.

## Install

Once the template is approved, install **IPTV Tunerr** from the Unraid Apps
tab. The package source and manual XML are in the
[IPTV Tunerr Unraid repository](https://github.com/snapetech/iptvtunerr-unraid).
Until catalog approval, add `templates/iptvtunerr.xml` as a user template.

Keep the default tuner port `5004` and dashboard port `48879` unless they
conflict with another container. Map `/var/lib/iptvtunerr` and
`/var/cache/iptvtunerr` to persistent appdata storage. Set
`IPTV_TUNERR_BASE_URL` to the host address and mapped tuner port that Plex,
Emby, or Jellyfin can reach.

Enter provider credentials as template variables, or use an M3U URL. M3U URLs
may contain credentials, so the template masks that field. The operator
dashboard requires a password and uses HTTP Basic/session authentication.

## Network and resources

Keep tuner and dashboard ports on a trusted LAN or VPN. Neither port should be
forwarded directly from the public internet. Each simultaneous stream relays
provider traffic, so network load scales with concurrent playback. Optional
ffmpeg remuxing/transcoding, recordings, and large guide refreshes increase
CPU, RAM, and disk use. No fixed minimum hardware requirement has been measured.

## Connect Plex

Use the Base URL as the tuner device URL and append `/guide.xml` for the guide.
If automatic registration is enabled, provide the optional Plex URL and token
in the container template.

For the package README and troubleshooting steps, see the
[Unraid package installation guide](https://github.com/snapetech/iptvtunerr-unraid/blob/main/docs/installation.md).

The [Community Apps submission flow](https://ca.unraid.net/submit/help)
requires a public package repository, root `ca_profile.xml`, license, and
valid template XML. Validate and Scan the repository before submitting it for
review.

See also
--------

- [Deployment overview](deployment.md)
- [Platform suitability](../explanations/platform-suitability.md)

Related ADRs
------------
