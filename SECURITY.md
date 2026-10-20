# Security

## Network boundaries

- The tuner listener (default port `5004`) is intentionally unauthenticated for HDHomeRun, guide, and stream clients. Keep it on a trusted network, firewall it from the public internet, and do not port-forward it directly.
- Operator pages and actions on the tuner listener are localhost-only by default. `IPTV_TUNERR_UI_ALLOW_LAN=1` makes those endpoints reachable from the LAN without application credentials; use it only on a trusted network.
- The Control Deck (default port `48879`) has separate password authentication and defaults to localhost-only. `IPTV_TUNERR_WEBUI_ALLOW_LAN=1` permits LAN access while retaining authentication.
- The Plex entitlement proxy is a sensitive boundary. Keep the owner token in an owner-only file or environment variable, use `-elevate-live-tv`, and block direct public access to PMS and Plex Relay. See [the proxy runbook](docs/runbooks/plex-live-tv-entitlement-proxy.md).

## Secrets and stored data

- Keep secrets in `.env` (gitignored), environment variables, or owner-only files. Never commit `.env` or log provider credentials.
- `catalog.json` can contain provider credentials in stream URLs. Restrict it to the service account (`chmod 0600 catalog.json`) and protect backups and diagnostic bundles the same way.
- Use `.env.example` as a template and keep real values out of examples, logs, release notes, and support captures.

## Dependency and code scanning

- Dependabot and CodeQL are enabled. Run `govulncheck ./...` for Go code and `npm audit` from `web/` for the Control Deck dependencies when reviewing dependency changes.
- Gitleaks runs in CI. GitHub secret scanning is a separate repository setting; keep it enabled where the repository plan supports it.

## Reporting issues

Report security bugs privately (e.g. to the maintainer) rather than in a public issue.
