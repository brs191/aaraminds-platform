# Module 5 Full Generated Review — Business Analyst Agent

**Date:** 2026-05-20
**Module under test:** `05_AI_Architecture_Diagram_System_v1.2`
**Review mode:** Blueprint Conformance Review (per the v1.2 selector table)
**Baseline:** `Business_Analyst_Agent_Blueprint_Final_2026-05-20.md` (Module 8 blueprint)

This is the full generated review called out as fix 9 on the Module 5 promotion path. It serves two purposes: (1) deliver an actual systems-review verdict on the BA Agent implementation as described in the stress prompt; (2) demonstrate whether v1.2 produces sharper output than the paper-only stress tests did.

## Input under review

Stress Prompt 1 from `StressTest_Module5_SystemsReview.md`:

> Review this deployed Business Analyst Agent against the Module 8 blueprint baseline.
> Blueprint constraint: Traceability-by-Construction.
> Current implementation:
> - Ingests Teams transcripts, SharePoint docs, and Jira tickets.
> - Drafts user stories and acceptance criteria.
> - Creates Jira tickets after product owner approval.
> - Stores project memory and glossary terms.
> - Routes security-sensitive requirements to a security reviewer.
> Known issue: Two unsupported requirements reached sprint planning last week. The team has traces for model calls and tool calls, but no trace that links each requirement back to source evidence.

---

## Review Verdict

**Conditionally ready.**

The Defining Operational Constraint of the BA Agent blueprint — Traceability-by-Construction — is broken. Per the v1.2 DOC gating rule, the verdict cannot be better than Conditionally ready until the source-to-requirement trace is restored. The verdict would drop to **Blocked** if the regulated-enterprise default assumption is confirmed (Azure-first, Microsoft 365, SharePoint with PII), because the implementation as described does not name PII classification, redaction, or audit logging for source access — and SharePoint and Teams transcripts routinely carry PII.

The two unsupported requirements that reached sprint planning are not an incident in isolation. They are evidence that the only control between draft and authoritative artifact — product owner approval — is ceremonial when the PO has no trace artifact to verify against. The system is operating outside its blueprint contract.

## Baseline Used

- Module 8 blueprint: BA Agent Blueprint, Final 2026-05-20.
- DOC under test: **Traceability-by-Construction.** Every requirement output must be evidence-backed, assumption-labeled, open-question-labeled, or conflict-labeled.
- Phase gate under test: blueprint defines Phase 1 as read-only analysis and draft generation with no writes to authoritative systems. The implementation creates Jira tickets, which is Phase 2 behavior.
- Default regulated context: Azure-first enterprise with Microsoft 365, SharePoint, Teams, Entra ID, Azure DevOps. PII and regulated-data handling rules apply.

## Conformance Findings

11 substantive findings, within the 7-12 budget. Severities follow the v1.2 escalation rules.

### 1. [High] Source-to-requirement trace is missing — DOC violation

**Evidence:** Stress prompt states traces exist for model calls and tool calls but there is no trace linking each generated requirement back to source evidence. Two unsupported requirements reached sprint planning as a result.

**Why it matters:** The DOC (Traceability-by-Construction) is the load-bearing claim of the blueprint. The blueprint requires every requirement to be evidence-backed, assumption-labeled, open-question-labeled, or conflict-labeled. None of those labels are enforceable without a per-requirement evidence link. Without it, the PO approval gate cannot distinguish a grounded requirement from a fabricated one, and downstream artifacts inherit the gap.

**Fix:** Emit a `source_evidence_ids` field on every generated requirement record, populated with the IDs returned by the retrieval call that grounded the requirement. Persist the field in the requirement store, surface it in the PO review packet, and gate Jira creation on non-empty `source_evidence_ids` (or an explicit assumption/open-question/conflict label).

**Owner:** BA Agent platform team.

**Re-review trigger:** Any future change to the requirement-generation prompt, the retrieval layer, or the Jira write path.

### 2. [High] PO approval is the only gate, and it has been demonstrated to fail

**Evidence:** Implementation description names PO approval as the gate before Jira creation. Two unsupported requirements reached sprint planning, which means PO approval passed unsupported drafts twice in one week.

**Why it matters:** A single approval gate is acceptable in the blueprint, but only if the PO has the trace artifact required to verify the requirement. They do not (see Finding 1). The gate is therefore ceremonial: it produces approval signatures without producing verified requirements. The blueprint's Phase 2 exit criterion ("Reviewer-routing accuracy and rework reduction meet pilot targets") cannot be claimed.

**Fix:** Combine with Finding 1 — make `source_evidence_ids` mandatory for Jira creation. Add a second-line check: flag any requirement with retrieval confidence below an agreed threshold, or with fewer than N source evidence items, for SME or architect review before PO approval.

**Owner:** BA Agent platform team + product ops.

**Re-review trigger:** Any further unsupported requirement reaching planning; any change that weakens or bypasses the PO approval step.

### 3. [High] Phase 2 behavior is live while Phase 1 controls are incomplete

**Evidence:** Implementation creates Jira tickets (a write to an authoritative system of record). Blueprint Phase 1 explicitly forbids writes to authoritative systems; Phase 2 requires "two clean project cycles with high trace completeness, low unsupported-claim rate, and reviewer acceptance above agreed threshold" before write-capable tools are enabled. Trace completeness and unsupported-claim rate cannot be measured because of Finding 1.

**Why it matters:** The system has been promoted to a phase whose exit criteria from the prior phase have not been met. This is the blueprint's named anti-pattern: "auto-creating backlog items without product-owner approval" — and even with PO approval, the supporting evidence discipline is absent.

**Fix:** Either roll back to Phase 1 (disable Jira write tools; agent emits draft tickets only) until trace completeness can be measured, or accept the live Jira write path under a hard gate on `source_evidence_ids` (per Finding 1) plus an explicit audited Phase 2 exit signoff.

**Owner:** Delivery lead + BA Agent platform team.

**Re-review trigger:** Any re-attempt at expanding the write surface (status changes, scope changes, multi-project rollout).

### 4. [High] PII and regulated-data handling is unspecified for SharePoint and Teams ingestion

**Evidence:** Implementation ingests Teams transcripts and SharePoint documents. The blueprint requires source classification before retrieval, redaction or restriction of sensitive HR / customer / financial / legal / regulated data, and marking of outputs containing sensitive content. The implementation description names none of these.

**Why it matters:** Teams transcripts routinely capture names, customer identifiers, financial figures, and HR-relevant conversation. SharePoint document libraries often hold policy and HR material. The v1.2 escalation rule sets PII findings at minimum High. Under the blueprint's default regulated-enterprise assumption, this is also a likely SOC 2 / ISO 27001 audit finding.

**Fix:** Add a classification step in the evidence intake pipeline that tags every retrieved record with confidentiality level (public, internal, confidential, regulated) before it reaches the requirement-extraction stage. Apply a redaction or restriction policy for confidential and regulated records. Mark any generated requirement or draft that touched confidential or regulated material. Document the policy.

**Owner:** Security / compliance + BA Agent platform team.

**Re-review trigger:** Any expansion of source systems (CRM, HR, finance); any change in tenant or regulated-data scope.

### 5. [High] Audit logging coverage is unstated

**Evidence:** Blueprint requires audit logs covering source access, tool calls, draft generation, review routing, approvals, rejections, and final artifact publication. Implementation description mentions traces for model and tool calls only.

**Why it matters:** Audit logs and traces are different artifacts. Traces support debugging and observability; audit logs support regulatory and incident reconstruction. The blueprint treats them as separate controls. Missing audit logs for source access, draft generation, and approval/rejection is a SOC 2 / ISO 27001 exposure and prevents a credible post-incident timeline of how the two unsupported requirements moved from source to planning.

**Fix:** Add an immutable audit log stream (separate from traces) covering: source access events with classification, draft generation events with `source_evidence_ids`, reviewer routing decisions, approval and rejection events with reviewer identity, and Jira publication events. Retain per regulated-data retention policy.

**Owner:** Security / compliance + BA Agent platform team.

**Re-review trigger:** Any audit or regulatory review; any incident requiring reconstruction of source-to-publication path.

### 6. [Medium] Reviewer routing is described only for security-sensitive items

**Evidence:** Implementation routes "security-sensitive requirements to a security reviewer." Blueprint expects routing to product owner, SME, architect, QA, security, or operations based on content and risk.

**Why it matters:** Narrow routing makes the PO the de facto reviewer of all non-security content, including domain-ambiguous requirements that should reach the SME, system-dependent requirements that should reach an architect, and testability-thin requirements that should reach a QA lead. This concentrates failure in one role and worsens Finding 2.

**Fix:** Implement a content-and-risk classifier that maps a draft requirement to one or more reviewer roles (SME for domain ambiguity, architect for system dependency, QA for testability, security/compliance for regulated, PO for scope). Make the routing decision a logged audit event.

**Owner:** BA Agent platform team.

**Re-review trigger:** Any change to reviewer-routing policy; any incident attributable to wrong-reviewer routing.

### 7. [Medium] Evaluation and CI gate are absent

**Evidence:** Implementation description names no golden set, no scorers, no CI gate, and no regression suite. Blueprint requires all of these before release.

**Why it matters:** Without a hallucinated-requirement-rate scorer running against a golden set, the team has no leading indicator for the kind of failure that produced the two unsupported requirements. Each incident becomes a lagging surprise instead of a regression caught at release time.

**Fix:** Build a small project-scoped golden set from completed projects (historical stakeholder notes, approved requirements, known-ambiguous cases). Add three scorers as a minimum: hallucinated requirement rate, evidence-link correctness, and reviewer-routing correctness. Wire to the release pipeline as a hard gate.

**Owner:** BA Agent platform team + a delivery team partner.

**Re-review trigger:** Before the next prompt, model, or retrieval configuration change.

### 8. [Medium] No low-confidence escalation or open-question routing is described

**Evidence:** Implementation names "stores project memory and glossary terms" but does not describe what happens when a requirement is ambiguous, conflicting, or low-confidence. Blueprint requires escalation for conflicts, low confidence, compliance-sensitive requirements, or missing source evidence.

**Why it matters:** Without an explicit low-confidence path, the agent's choices collapse to "produce a requirement" or "fail silently." Both reach the PO as confident drafts, which is part of how Finding 2 happened.

**Fix:** Add a confidence score to every generated requirement. Below a threshold, route to BA or SME for clarification before PO approval. Above threshold but with detected ambiguity or conflict signal, label the requirement appropriately and route accordingly. Generate open questions as a first-class output, not as commentary.

**Owner:** BA Agent platform team.

**Re-review trigger:** Any change to the requirement-extraction prompt or confidence-scoring logic.

### 9. [Medium] Project memory scope discipline is unstated

**Evidence:** Implementation "stores project memory and glossary terms." Blueprint requires project-scoped memory only, with no cross-client or cross-tenant memory.

**Why it matters:** Memory that crosses project boundaries is the blueprint's named anti-pattern ("Using memory as an uncontrolled source of truth"). If the same agent serves multiple projects or clients, glossary terms or approved-pattern memories from project A can leak terminology into project B's requirements without traceability.

**Fix:** Make the memory store explicitly scoped by `project_id` (and `tenant_id` where applicable). Add a retrieval filter that rejects memory items outside the active project scope. Audit the memory store for any pre-existing cross-project entries.

**Owner:** BA Agent platform team.

**Re-review trigger:** Onboarding of a second project or tenant; any change to memory retrieval logic.

### 10. [Medium] Rollback and kill switch are not described

**Evidence:** Implementation description names no rollback path or kill switch. Blueprint requires the ability to disable write tools, freeze draft publication, roll back prompt/model/retrieval configuration, and restore last approved artifact versions.

**Why it matters:** When the next incident occurs (and Finding 1 makes it likely), the team needs a single action to disable Jira writes without taking the entire agent offline. Without a documented kill switch, the operational response is ad hoc and slower than the incident.

**Fix:** Add a feature flag for the Jira write tool that can be toggled off without redeploy. Document the rollback runbook for prompt, model, and retrieval-config reverts. Test the kill switch quarterly.

**Owner:** BA Agent platform team + operations.

**Re-review trigger:** Any production incident; any change to write-capable tool surface.

### 11. [Low] Feedback loop from reviewer corrections to memory is not described

**Evidence:** Blueprint specifies that "reviewer corrections feed into project memory after approval" and "rejected outputs become negative examples." Implementation description does not name this loop.

**Why it matters:** Without the loop, the same ambiguity or extraction error reappears across drafts. Reviewer effort produces only a one-off fix rather than a permanent improvement.

**Fix:** When the PO or SME edits or rejects a generated requirement, capture the delta and the rationale, then update project memory after the PO approves the memory update. Rejected outputs become labeled negative examples in the golden set.

**Owner:** BA Agent platform team.

**Re-review trigger:** When evaluation scorers are added (see Finding 7).

## Structural Risks

- **Single-point gate ceremoniality.** The blueprint's gate-and-trace pair is a single design — gate without trace becomes ceremony, trace without gate is reporting. The current implementation has the gate and not the trace, which is the worse half of the failure mode. Findings 1 and 2 must be fixed together.
- **Premature phase progression.** The blueprint's phased rollout exists to compound trust before expanding the write surface. The current implementation skipped that compounding. Either the rollout was deliberately accelerated (in which case the decision should be documented) or it happened by drift (in which case Finding 3 is a governance issue, not just a controls issue).
- **Compliance debt accumulating in an Azure-first regulated default.** Findings 4 and 5 together describe a system whose deployment context is regulated by default but whose controls are not. The longer this runs, the more SOC 2 / ISO 27001 evidence the team owes retroactively.

## Control Gaps

| Control | Blueprint expectation | Implementation evidence | Severity |
| --- | --- | --- | --- |
| Source-to-requirement trace | Required by DOC | Absent | High |
| `source_evidence_ids` gate on Jira write | Implied by DOC + write-tool allowlist | Absent | High |
| PII classification at intake | Required | Not described | High |
| Audit log distinct from traces | Required | Not described | High |
| Reviewer routing by content + risk | Required | Narrowed to security only | Medium |
| Low-confidence and conflict escalation | Required | Not described | Medium |
| Project-scoped memory | Required | Not enforced in description | Medium |
| Kill switch / write-tool disable | Required | Not described | Medium |

## Observability and Evaluation Gaps

- Traces capture model and tool calls but not retrieval decisions, selected evidence, source-to-requirement linkage, reviewer routing, model version, prompt version, or validation outcomes. Blueprint specifies all of these.
- No golden set. No hallucinated-requirement-rate scorer. No evidence-link-correctness scorer. No CI gate. The team is operating without a leading indicator for the failure mode that already happened twice.
- No mention of rework rate, open-question closure rate, or reviewer override rate metrics. Without these, "reduce rework, not create more review burden" cannot be measured.

## Required Fixes

In priority order. The first three are gating; the rest can run in parallel.

1. Restore the DOC: implement `source_evidence_ids` on every generated requirement; gate Jira creation on it. (Findings 1, 2, 3.)
2. Close the regulated-data gap: PII classification at intake + redaction/marking + audit log distinct from traces. (Findings 4, 5.)
3. Confirm or roll back the phase progression: either re-disable the Jira write tool until trace completeness is measurable, or document an explicit Phase 2 exit signoff. (Finding 3.)
4. Add a minimum evaluation harness: golden set + three scorers + CI gate. (Finding 7.)
5. Broaden reviewer routing beyond security-only. (Finding 6.)
6. Add low-confidence and conflict escalation paths. (Finding 8.)
7. Enforce project-scoped memory. (Finding 9.)
8. Add kill switch and feature flag for the Jira write tool. (Finding 10.)
9. Wire the reviewer-correction feedback loop. (Finding 11.)

## Re-Review Triggers

Per the blueprint's own re-review trigger list, plus what this review uncovered:

- Any further unsupported requirement reaching sprint planning.
- Any change to the requirement-generation prompt, retrieval layer, or Jira write path.
- Any expansion of source systems, tenants, or regulated-data scope.
- Any change to the reviewer-routing policy.
- Any production incident, audit finding, or regulatory review.
- Before any move toward Phase 3 (controlled system-of-record updates).
- Material change in unsupported-claim rate, reviewer override rate, rework rate, or cost-per-approved-requirement.

---

## Module 5 v1.2 self-check on this output

This is the validation step for Module 5 itself. Did v1.2 produce a sharper review than v1.1 would have?

| v1.2 contract item | Honored in this output? |
| --- | ---: |
| Verdict stated explicitly with operating stance | Yes — Conditionally ready, with conditions named |
| DOC gating rule applied | Yes — Traceability-by-Construction broken → verdict capped |
| Findings lead, summary does not bury risks | Yes — verdict + 11 findings before structural risks and control table |
| Each major finding has Severity / Evidence / Why / Fix / Owner / Re-review trigger | Yes — all 11 findings carry the full shape |
| Severity escalation rules applied | Yes — PII finding at High per rule; DOC violation at minimum High per rule |
| Borderline High/Critical distinction used | Yes — Finding 2 (unsupported requirements reach planning) held at High because human PO approval is in the loop; would escalate to Critical if Jira creation were automatic |
| Findings within 7-12 budget | Yes — 11 findings, grouped by theme in the control-gap table |
| Review-mode selector applied | Yes — Blueprint Conformance Review chosen because input includes Module 8 blueprint + implementation |
| Output template matches selector | Yes — Blueprint Conformance structure used |
| Must-check Quality Checklist items satisfied | Yes — verdict, baseline, prioritized findings, finding shape, DOC preserved-or-gated, decision/tool/approval visibility, mode selection |
| Diagram guidance subordinate / absent | Yes — no diagram produced; would only emerge from a finding |

**Verdict on Module 5 v1.2:** the contract did its job. The DOC gating rule and the severity escalation rules in particular shaped the output in ways that would not have happened under v1.1 (the prior version would likely have produced a flatter list without the DOC-first framing). The findings-count budget kept the review from sprawling into 20+ items.

**Recommendation:** Module 5 can move to **9.0 / 10**, **Stable candidate**. To confirm Stable, run one more review on a different system shape — recommend the Prompt 2 (Agentic RAG Production Readiness) scenario — and check whether the same discipline holds under a different mode and template.
