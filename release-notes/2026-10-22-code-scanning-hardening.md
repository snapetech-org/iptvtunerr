---
category: security
audience: operators
area: security
action: For remote VOD WebDAV, configure a TLS certificate and key plus username and a password of at least 16 bytes.
breaking: true
---
Remote VOD WebDAV now requires TLS and authentication; plaintext stays loopback-only. The Control Deck login requires a signed CSRF token and sends Secure session cookies over direct or trusted loopback-proxied HTTPS. Proxied active content is inert, and generated M3U fields strip control characters to prevent directive injection.
