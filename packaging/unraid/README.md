# IPTV Tunerr for Unraid

This repository contains the Unraid Community Applications template for
[IPTV Tunerr](https://github.com/snapetech/iptvtunerr), a tuner and guide
bridge for Plex, Emby, and Jellyfin. The template runs the existing public
multi-architecture image from GHCR.

## Install

After Community Applications approves the submission, search for **IPTV
Tunerr** in the Unraid Apps tab. Until then, install the template XML from
`templates/iptvtunerr.xml` as a user template.

Map the appdata and cache paths to persistent storage. Set
`IPTV_TUNERR_BASE_URL` to the address and host port your media server can reach,
for example `http://192.168.1.20:5004`. Configure an IPTV source using the
template variables or the authenticated operator dashboard on port 48879.

Keep the tuner and dashboard ports on a trusted LAN or VPN. Set a strong
`IPTV_TUNERR_WEBUI_PASS` before starting the container. Do not expose either
port directly to the public internet.

## Resource notes

The Go service is small when idle. Each active stream uses upstream and
downstream bandwidth; simultaneous streams multiply that traffic. Optional
ffmpeg remuxing/transcoding, large guide refreshes, and recording add CPU, RAM,
and disk load. No fixed CPU or memory minimum has been measured for Tunerr.

## Support

- Application source and issues: <https://github.com/snapetech/iptvtunerr>
- Package questions: <https://github.com/snapetech/iptvtunerr-unraid/issues>

## License

IPTV Tunerr is available under AGPL-3.0-or-later or a separate commercial
agreement. The root `LICENSE` contains the standard AGPL version 3 text so
GitHub and Community Applications can recognize the OSI-approved license;
the separate commercial option is described in the
[upstream license notice](https://github.com/snapetech/iptvtunerr/blob/main/LICENSE).
