---
category: fixed
audience: users, operators
area: control-deck
action: none
breaking: false
---
The Control Deck now includes the session-bound CSRF token when a browser's first page load is authenticated through HTTP Basic. User and M3U account changes work in that flow without weakening CSRF checks.
