# Install IPTV Tunerr on Unraid

IPTV Tunerr runs as one Docker container. The image is published to GHCR for
amd64, arm64, and arm/v7. Unraid uses the amd64 image on supported hosts.

## Install from Community Applications

After the package is approved, open **Apps**, search for **IPTV Tunerr**, and
install the template. Until catalog approval, add the XML under
`templates/iptvtunerr.xml` as a user template.

Use the following settings:

1. Keep the tuner port mapped to 5004 unless another container already uses it.
2. Keep the operator dashboard port mapped to 48879 unless another service
   uses it.
3. Keep both appdata and cache mapped to persistent storage. Back up the
   appdata path; the cache can be rebuilt after a failure.
4. Set **Base URL** to the URL the media server will actually use, including
   the host port if you changed the mapping, for example
   `http://192.168.1.20:5004`.
5. Set a strong **Dashboard Password** before starting the container.
6. Enter an Xtream-style provider URL and credentials, or provide an M3U URL.
   M3U URLs can contain credentials and are masked in the template.

Plex, Emby, or Jellyfin must be able to reach the Base URL. Keep tuner and
dashboard ports on a trusted LAN or VPN. Do not port-forward either port to
the public internet. The tuner endpoints are designed for trusted media-server
networks; the dashboard uses its own username and password.

## Connect a media server

For Plex, use the Base URL as the tuner device URL and append `/guide.xml` for
the guide URL. The operator dashboard is available on port 48879. Its login is
the username and password configured in the template.

If automatic Plex registration is enabled, set the optional Plex URL and token
variables. Store the token only in Unraid's local container template settings.

## Resource requirements

The service uses little CPU while idle. Each active stream relays provider
traffic through the Unraid host, so upstream and downstream bandwidth scale
with concurrent streams. Optional ffmpeg muxing or transcoding, recording, and
large lineup/guide refreshes can use substantially more CPU, memory, and disk.
No fixed minimum has been measured; size the host for the concurrent stream
count and any transcoding profiles you enable.

Unraid already has Community Applications entries for IPTV tuner bridges such
as xTeVe and Threadfin. Tunerr fits the same media-server workflow and adds a
native stream/guide bridge and operator controls.

## Troubleshooting

- If Plex cannot add the tuner, check that `IPTV_TUNERR_BASE_URL` matches the
  host address and mapped port Plex can reach.
- If the dashboard does not load, verify the 48879 mapping, the
  `IPTV_TUNERR_WEBUI_ALLOW_LAN=1` setting, and the configured password.
- If streams start but stall, inspect the container logs and compare provider
  throughput with the number of simultaneous streams.

## Support

- [Application documentation](https://github.com/snapetech/iptvtunerr/tree/main/docs)
- [Report an issue](https://github.com/snapetech/iptvtunerr/issues)
