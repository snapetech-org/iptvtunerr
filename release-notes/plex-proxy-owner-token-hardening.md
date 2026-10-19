---
category: security
audience: operators
area: plex-live-tv-proxy
action: Review -elevate-all deployments; add -unsafe-elevate-all or move to -elevate-live-tv. Prefer -owner-token-file over -owner-token.
breaking: true
---
The Plex Live TV proxy now refuses to lend the owner token on non-canonical paths, limits elevation to known Live TV routes, and scrubs the owner token from elevated responses. `-elevate-all` requires `-unsafe-elevate-all` and withholds owner-only areas such as `/myplex` and `/security`. Added `-owner-token-file`.
