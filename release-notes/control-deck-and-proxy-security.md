---
category: security
audience: operators
area: security
action: Keep Control Deck login protected and rely on recognized form fields for Plex Live TV subscription requests.
breaking: false
---
Control Deck login now bounds inputs and failure tracking, rejects cross-origin Basic-auth mutations, and fails closed without secure tokens. The UI uses the server's CSRF header; the old header remains accepted. Plex Live TV classification checks explicit fields and preserves oversized bodies. CI uses trusted PR code without unused Git credentials or access to the private identity denylist.
