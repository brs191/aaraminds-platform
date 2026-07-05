# Phase 0 Validation Plan

Implements BRD v2.1 §21.1 and decision gate DG-001. Duration: 2–3 weeks. Phase 1 funding is contingent on this plan's exit criteria.

## 1. Stakeholder interviews (8–10)

Panel: 2 enterprise architects, 2 engineering leads, 1 security reviewer, 1 compliance/governance reviewer, 1 business owner, 1 platform/product owner, 1 delivery/PMO stakeholder.

### Interview script (45 min)

1. **Current state (10 min).** How do agent ideas become production candidates today? Where do designs die? Who says no, and on what basis?
2. **Artifact walkthrough (15 min).** Show the BA Agent example (`examples/ba-agent.manifest.yaml`, 3 tool contracts, a mock readiness report rendered from `schemas/readiness-report.schema.json`). Ask: what's missing before you would sign off against this?
3. **The core question (10 min).** *"Would you trust an Agent Factory readiness score as a gate before pilot approval — and what would it take for you to?"* Record: yes / yes-with-conditions (name them) / no (why).
4. **Rubric calibration (10 min).** Show rubric v0.1 weights and critical blockers. Which weights are wrong? Which blockers are missing?

### Exit criterion

≥6 of 8 core stakeholders answer yes or yes-with-conditions-the-MVP-can-satisfy [TARGET]. Majority no = stop signal for Phase 1 in current shape; revisit positioning before build.

## 2. Decision gate DG-001: packaging model

Decide before Phase 1 funding: internal enterprise platform / consulting accelerator / public SaaS / open-core. Inputs: interview signal on who would pay or sponsor; current best fit per BRD is internal platform or consulting accelerator. Output: recorded decision with owner and consequences acknowledged (tenancy, identity, pricing, support). SaaS choice triggers a BRD revision, not a silent scope change.

## 3. Other Phase 0 deliverables

| Deliverable | Source |
|---|---|
| Top 3 reference agents confirmed | OQ-002 — proposed: BA, Scrum Master, Migration QA (templates/ in this package) |
| Readiness rubric v0.1 reviewed | readiness-scoring-rubric.md + interview feedback |
| Artifact set and export format confirmed | artifact-schemas.md |
| Source systems for first MCP integrations | OQ-005 — from interviews |
| Runtime decision input | OQ-003 — validate Claude Agent SDK / Foundry items in `docs/runtime-verification-notes.md` |
| Autonomy model confirmed | BRD §17 |

## 4. Phase 0 exit review

One session; attendees = interview panel leads + product owner. Agenda: interview results, DG-001 decision, rubric changes, go/no-go for Phase 1. Output recorded in `governance/`.
