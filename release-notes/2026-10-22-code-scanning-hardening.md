---
category: security
audience: operators
area: security
action: For remote VOD WebDAV, configure a TLS certificate and key plus username and a password of at least 16 bytes; put remote or LAN Control Deck browser access behind an HTTPS reverse proxy.
breaking: true
---
Remote VOD WebDAV now requires TLS and authentication; plaintext remains loopback-only. The Control Deck enforces signed CSRF protection and always marks session and OIDC state cookies Secure. Local browser access uses loopback; remote or LAN login requires an HTTPS reverse proxy. Proxied active content is inert, and generated M3U fields reject control characters to prevent directive injection.
