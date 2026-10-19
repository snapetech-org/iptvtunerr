---
category: security
audience: operators
area: plex-live-tv-proxy
action: Prefer -elevate-live-tv over -elevate-all, and -owner-token-file over -owner-token.
breaking: false
---
The Plex Live TV proxy now refuses to lend the owner token on non-canonical paths such as `/livetv/../security/token`, and scrubs the owner token from elevated responses. `-elevate-all` is deprecated, logs a warning, and withholds owner-only areas such as `/myplex` and `/security`. Added `-owner-token-file`.
