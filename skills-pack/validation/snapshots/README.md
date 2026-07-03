<!-- doc-consistency: ignore — index of frozen snapshots. -->

# Snapshots — frozen point-in-time records

This directory holds documents that captured the pack's state, an assessment, or an
executed plan **at a specific date**. They are deliberately *not* maintained: their
counts, paths, and findings are correct as of the date each one carries, and are
allowed to disagree with the current pack.

Because they are frozen, every file here is exempt from the doc-consistency check in
`validation/tools/skill_audit.py` — the whole directory is skipped, and each file also
carries a `<!-- doc-consistency: ignore -->` marker. Living documentation (`README.md`,
`ROADMAP.md`, `usage.md`, and so on) is *not* exempt and must agree with disk.

## Contents

| File | What it is | Frozen as of |
|---|---|---|
| `CRITICAL-REVIEW-2026-05-21.md` | Full critical review of the pack — findings on documentation drift, identity, validation, paths. | 2026-05-21 |
| `SoftwareDevAgent_TestPlan_2026-05-21.md` | Test plan for the pack and the agents that consume it. | 2026-05-21 |
| `inspiration_hobson.md` | Executed action plan — the wshobson-inspired adoptions (`inherit` model tier, `skill_audit.py`, the discovery index). All three steps are done. | 2026-05-19 |
| `workspace-session-log-2026-05-21.md` | Running log of work done in the workspace through 2026-05-21 (rescued from the former `overview.md`). | 2026-05-21 |

Relative paths inside these files point at the pack as it was laid out when the file
was frozen (most were authored at the pack root). Do not fix them — that would defeat
the purpose of a snapshot. To find the current location of anything, start from the
root `README.md` or `.claude/INDEX.md`.

## When to add a file here

Move a document into this directory when it has become a record rather than a
reference — a dated review, a completed plan, a session log. Living documentation that
should track the pack stays at the pack root or under `validation/`.
