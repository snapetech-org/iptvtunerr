---
category: fixed
audience: users, operators
area: docker-compose
action: none
breaking: false
---
The Docker Compose example now exposes the Control Deck on host loopback and persists its SQLite database. Missing database configuration and failed database opens also produce distinct messages to help diagnose channel-management errors.
