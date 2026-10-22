---
category: security
audience: operators
area: security
action: For remote VOD WebDAV, configure a TLS certificate and key plus username and a password of at least 16 bytes.
breaking: true
---
VOD WebDAV now serves plaintext only on loopback; listeners on other interfaces require TLS and authentication. The Control Deck login form also requires its signed CSRF token, session cookies are marked Secure for direct HTTPS and trusted loopback HTTPS proxies, and proxied active content is returned as inert data.
