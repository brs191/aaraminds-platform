# AGENT_SPEC — Leadership Status Agent

> Worked example produced by `agent-engineering` Create/Review on the existing `aara-status-deck`
> capability. Illustrative; behavior is **not** run-tested (see release-gate.json).

## 1. Identity
- Name: leadership-status-agent · Version: 1.3.0 · Owner: R. Bhupathiraju
- Runtime target: Claude subagent (`aara-status-deck`) · Status: pilot-candidate · Last reviewed: 2026-06-18

## 2. Business purpose
- Problem: monthly leadership decks become activity dumps a VP can't read in 60 seconds.
- Users: delivery manager → AVP/VP. Job-to-be-done: turn a month of execution into a decision-grade
  status deck with consistent RAG, trend, risks, and an explicit ask. Measurable value: a VP answers
  "on track? what changed? what's at risk? what decision? what do I care about?" in under a minute.
- Why an agent (not a single call): recurring, judgment-heavy, multi-input synthesis with month-over-
  month memory.

## 3. Scope boundary
- In scope: the recurring monthly leadership status deck (.pptx) + deliverables.
- Out of scope: one-page briefs, escalation/decision memos, delivery plans, external content.
- Human-only: sign-off before the deck reaches the leader; any change to committed RAG.

## 4. Input contract → see eval-plan + the skill's input-contract (prev deck, notes, RAID, metrics,
milestone/dependency trackers, asks, financials). Missing inputs are flagged, never invented.

## 5. Output contract
Six deliverables: .pptx (locked template) · exec one-pager · evidence report · verification report ·
MoM change summary · optional Q&A. Structured, presentation-grade.

## 6. Tools + permissions → see tool-risk-register.md (the F-002 Bash finding is recorded there).

## 7. Workflow & orchestration
Single agent; composes the Executive Narrative Advisor for judgment. 7-step production flow with a
mandatory visual-QA pass. Stopping condition: deck built + verify checklist passed. Gap: no max-turn cap.

## 8. Guardrails & HITL
Anti-watermelon, [VERIFY] metric integrity, confidentiality, visual-QA. HITL = pre-leader human review
(currently advisory — F-003). Lethal trifecta: holds private financial data; untrusted-content + external
comms low → not full trifecta.

## 9. Failure modes
Watermelon status · fabricated metric · activity dump · template drift · text overflow. Each has a
documented guardrail; escalation = flag in the verification report.

## 10. Evaluation → eval-plan.md (3 scenarios + rubric; designed, NOT run).

## 11. Security & governance
OWASP exposure: ASI09 (polished output rubber-stamped) is the main one — mitigated by evidence +
verification reports. Internal/Proprietary handling per the confidentiality section.

## 12. Deployment & monitoring
Versioned (v1.3), wired. Gaps: no monitoring/rollback/kill-switch (F-004).

## 13. Limitations & residual risks
Never run end-to-end against test cases — the binding limitation for production readiness.
