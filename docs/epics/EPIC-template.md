---
id: epic-template
type: reference
status: stable
tags: [epic, template, planning]
---

# EPIC-0001-<name> (template)

Copy this file to `EPIC-0001-<short-name>.md` when starting a larger or cross-repository initiative. Track work through direct implementation milestones.

**When to use:** cross-repository work, several implementation areas, a non-trivial migration/rollout, or material security/performance work. Smaller tasks use [memory-bank/current_task.md](../../memory-bank/current_task.md).

---

## 1. North star

- **PRD/Epic goal (2–5 sentences):**
- **Non-goals (scope fence):**

---

## 2. Milestones (vertical slices)

| Milestone | Done = verifiable outcomes |
|-----------|----------------------------|
| M1: end-to-end thin slice | |
| M2: hardening (tests/edge cases) | |
| M3: perf/security/operability | |

---

## 3. Story list (granular, scoped)

For each story:

| ID | Acceptance criteria | Files/areas (expected) | Verification | Risk flags |
|----|---------------------|------------------------|--------------|------------|
| STORY-001 | | | `scripts/verify` or … | migration/security/perf |
| STORY-002 | | | | |

---

## 4. Milestone plan

| Milestone | Scope |
|----|--------|
| M1 | first working slice |
| M2 | feature completion |
| M3 | hardening, migration, and docs |

Track completed tasks against the story IDs. Park unrelated ideas in [memory-bank/opportunities.md](../../memory-bank/opportunities.md).

---

## 5. Decision points

Short list of “needs user input” with defaults (ask only when it matters):

- **Decision:** … → **Options:** … → **Default if no answer:** …

---

See also
--------
- [memory-bank/work_breakdown.md](../../memory-bank/work_breakdown.md) (agent-centric mirror)
- [memory-bank/current_task.md](../../memory-bank/current_task.md)

Related ADRs
------------
- *(add as decisions are made)*

Related runbooks
----------------
- *(if applicable)*
