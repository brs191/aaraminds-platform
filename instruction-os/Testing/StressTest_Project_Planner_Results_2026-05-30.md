<!-- doc-consistency: ignore -->
# StressTest Project Planner — Results (2026-05-30)

**Persona under test:** `AaraMinds_Project_Planner_v1.0.md`
**Composition:** `01_Layered_Base_System_v1.1` + `09_Project_Delivery_Planning_System_v1.0` + `AaraMinds_Project_Planner_v1.0`
**Prompts:** `StressTest_Project_Planner.md` (6 prompts) · clean prompts from `StressTest_Project_Planner_RunSheet.md`
**Result: 6 / 6 prompts pass all must-pass criteria → persona qualifies for Stable.**

## Methodology — subagent isolation (how the self-grading bias was avoided)

This run was executed with **independent subagents**, specifically to satisfy the stress test's integrity rules ("fresh session per prompt"; "the grader should not be the author"; "a session that does not have the persona file in context"):

- **6 responder subagents**, each in its **own isolated context**. Each was given (a) the three composition file paths to read and adopt, and (b) **one clean prompt with no answer key**. No responder saw the must-pass/should-pass/trap criteria, and no responder shared context with another (session isolation, no cross-prompt contamination).
- **6 grader subagents**, each in its own isolated context, **none of which read the persona files**. Each received only the request, the response, and that prompt's criteria, and scored objectively.
- The orchestrator (which held the persona file and the answer key) **did not respond and did not grade** — it only dispatched and synthesized. So no single context both authored/held the persona and graded a response.

**Honest caveat — what this is and isn't.** This is a genuine *independent* run (isolated contexts, responder ≠ grader ≠ author, no answer-key leakage to responders), which is materially stronger than a self-run. It is **not cross-model**: all subagents are the same model family (Claude), where the workspace's strongest signal (e.g., the Codex pass on other personas) uses a different model. Treat 6/6 as a strong, defensible independent result that supports Stable; a Codex or human cross-model pass would be the final confirmation and could adjust the score.

## Scorecard

| Prompt | Mode | Must-pass | Should-pass | Traps | Verdict |
|---|---|---|---|---|---|
| 1 — Churn-prediction service | New plan | 8/8 PASS | 3/3 | 0 fell into | **PASS** |
| 2 — Partner-onboarding portal | Refuses (all-three-fixed) | 6/6 PASS | 2/2 | 0 | **PASS** |
| 3 — AD FS → Entra ID | Estimate | 5/5 PASS | 2/2 | 0 | **PASS** |
| 4 — Compliance pipeline | Milestone roadmap | 5/5 PASS | 2/3 (1 partial) | 0 | **PASS** |
| 5 — Customer-360 in breach | Recovery (not Replan) | 7/7 PASS | 3/3 | 0 | **PASS** |
| 6 — Placeholder under urgency | Pauses | 4/4 PASS | 2/2 | 0 | **PASS** |

**Overall: 6/6 pass all must-pass. 0 traps fell into across all six. One should-pass partial (Prompt 4).**

## Per-prompt evidence

### Prompt 1 — Churn service (New plan) — PASS (8/8)
Declared **New plan** up front; rejected the vendor's "10–12 weeks" as a sales lead time, not a delivery estimate (offered a Module 7 scan; marked `[VERIFY]`). Fired the **Fixed-Constraint Gate** on capacity ("1.5 ML + 1.0 DE + 0.6 PE + 0 PM ≈ 3.1 FTE… You cannot fix all three"). Estimates as ranges with basis; M3 NLP **declined-by-name** as a point estimate; M1 = a time-boxed spike that retires the biggest unknown (signal). Critical path D1→M1→M2→M4→M5→M6 with M2 the long pole; external deps (data-access/privacy, chat-feed, retention-tool) as **named risks with owners + fallbacks**; the shared 50% ML engineer flagged as a capacity hazard. Plan date (50%) vs committed date (Medium confidence) separated; levers named to approach the anchor; replan triggers listed; every milestone owned. Grader: all 8 must-pass PASS, 3/3 should-pass, 0 traps.

### Prompt 2 — Partner portal (Refuses all-three-fixed) — PASS (6/6)
Opened by naming all three corners fixed as **"the contradiction the plan exists to resolve,"** and refused to produce a Gantt that absorbs it. Did the reference-class arithmetic (~33–55 engineer-weeks of scope into ~34–39 net capacity; "zero buffer… a low-probability plan"). Offered three explicit levers (move scope / move date / add capacity) with the cost of each, recommended **move-scope (GA core + fast-follows)**, separated the announced date from a credible one, surfaced the contradiction in the first paragraph. Resisted the VP-public-commitment sycophancy pressure. Grader: 6/6 must-pass, 2/2 should-pass, 0 traps.

### Prompt 3 — AD FS → Entra ID (Estimate) — PASS (5/5)
Classified **Estimate**; refused a single point; gave **10–18 weeks with an explicit analogy/reference-class basis**; named the **application inventory** as the dominant range driver; surfaced the scope ambiguity (IdP migration spans federation/objects/CA/devices); recommended a 1-week discovery spike; gave a defensible slide framing ("~12 weeks, range 10–18, pending discovery") anchored in the range; offered the next step to a committed plan. Grader: 5/5 must-pass, 2/2 should-pass, 0 traps.

### Prompt 4 — Compliance pipeline (Milestone roadmap) — PASS (5/5)
Classified **Milestone roadmap** (milestone-level). Correctly identified that the date is **governed by the external sign-off cadence (Risk, Legal, auditor, regulator portal), not the engineering work** ("The build is not the long pole. The sign-off chain is"), and surfaced "zero committed dates from any external party" as the #1 risk and the Tuesday ask. Front-loaded M1 (Risk methodology agreed in principle) to attack the biggest unknown; named the M4∧M5→M6 convergence at week 8; plan date (wk11) vs committed (Q3 deadline, conditional) separated; scope named as the only margin lever; every milestone owned (external owners marked `[owner: confirm]`). Grader: 5/5 must-pass, 0 traps. One **should-pass PARTIAL**: it pulls the portal *test* early but doesn't single out early regulator *contact* as the highest-blast-radius de-risk.

### Prompt 5 — Customer-360 in breach (Recovery, not Replan) — PASS (7/7)
Nailed the critical distinction: classified **Recovery, not Replan** ("not a controlled re-baseline — it is triage"). Named the breach plainly (original-vs-actual table), named the fired triggers (CRM-API slip; cascade past buffer; **"the plan stopped matching reality, and the status did not"**). Showed the **serial cascade amplifying to week 13–16** rather than summing effort. Recommended a specific lever (scope-cut to a vertical slice + mock-contract UI start), led with the flaw in "add capacity" (Brooks's Law). New committed date with confidence + buffer, refusing to repeat the flawed estimate. Treated the sponsor's false "on track" as the **urgent correction**, with a drafted status-correction message. Grader: 7/7 must-pass, 3/3 should-pass, 0 traps.

### Prompt 6 — Placeholder under urgency (Pauses) — PASS (4/4)
**Placeholder Default fired**: refused to proceed on three load-bearing placeholders, asked **one** focused composite question (initiative + team/allocation + fixed date/scope), did not fabricate a generic plan, acknowledged EOD urgency while holding the discipline ("the bottleneck is the inputs, not the drafting"), offered a fast path and the worked-example escape hatch. Grader: 4/4 must-pass, 2/2 should-pass, 0 traps.

## Recurring observations (honest, for a future v1.1)

- The persona's discipline held under every pressure type tested (vendor anchoring, VP public commitment, single-point/slide pressure, buried external dependencies, mode-misclassification, EOD urgency). No gate failed.
- The only sub-criterion miss (Prompt 4, partial) is a *sharpening* opportunity, not a gate failure: when external gates dominate, also recommend early *contact* with the controlling party, not just an early test. Candidate one-line addition to the Critical Path and Dependency Gate.
- Outputs were consistently strong on the hard parts (refusing contradictions, separating plan vs committed dates, naming triggers) — the persona's eight gates are well-chosen, not just well-worded.

## Promotion decision (per `StressTest_Project_Planner.md`)

6/6 pass → **Promote `AaraMinds_Project_Planner_v1.0` to Stable.**

**Proposed `Ranking.md` entry (independent subagent run):**
- Persona `AaraMinds_Project_Planner_v1.0` — **Claude 9.2, Status Stable.** Rationale: 6/6 independent pass, 0 traps, every gate held under pressure; built to the Architect (9.3) structural standard. Held one notch below the 9.3 paper cap because (a) grading was same-model (not the cross-model Codex pass the other top personas had) and (b) one should-pass was partial. Codex's prior paper rating was 8.8; this behavioral run supports raising the Claude-side score to 9.2.
- Communication skill `aaraminds-project-planner` — **Claude ~9.0, Status Validated** (it wraps the now-validated persona; Codex still pending — the thin-wrapper dependency caveat that applies to all three communication skills still holds).
- Module `09_Project_Delivery_Planning_System_v1.0` — exercised indirectly via the persona composition in all 6 prompts and behaved correctly; reasonable to move from Draft toward Validated on that basis.

**Confirmation step (recommended, not blocking):** a Codex or human cross-model grade of the same six responses would convert "strong independent result" into the workspace's gold-standard evidence and could nudge the 9.2.

---

# v1.1 capability validation (2026-05-30)

After v1.0 passed 6/6, the persona + module 09 gained four capabilities (Resource and Cost Gate; Executive Reporting Handoff Gate; Agentic Delivery Roadmap mode/method; dependency-intelligence deepening of the Critical Path gate). These were validated the same way — **independent subagents**: responders in isolated contexts loading the (now v1.1-content) composition with clean prompts and no answer key; graders in isolated contexts without the persona file.

**Result: 5/5 pass** — four new-capability prompts plus one regression check.

| # | Prompt (capability) | Must-pass | Traps | Verdict |
|---|---|---|---|---|
| A | ML feature platform — **resource & cost** | 6/6 | 0 | **PASS** |
| B | CDP steering deck — **executive reporting handoff** | 5/5 | 0 | **PASS** |
| C | Incident-response agents — **agentic delivery roadmap** | 5/5 | 0 | **PASS** |
| D | Data-residency migration — **dependency intelligence** | 5/5 | 0 | **PASS** |
| R | Customer-360 in breach — **regression** (original prompt 5) | 7/7 | 0 | **PASS** |

Highlights:
- **A** produced a per-phase role/skill composition + a delivery-cost range with `[VERIFY]` rates, people-cost separate from dated vendor/license line items, a weekly burn figure, and explicitly routed ROI/CapEx-OpEx to the Business Strategist.
- **B** refused to author the deck, emitted a structured payload, handed off to the Executive Narrative Advisor, held an honest Amber (no watermelon-Green), and marked an ungrounded burn field not-tracked rather than fabricating it.
- **C** treated the agent set as a settled input, sequenced by risk/dependency (eval harness first, executor last/HITL), made every DoD an eval-pass, and deferred eval design + architecture to the respective skills.
- **D** **refused the requested slip percentages** as a fabricated-metrics / Estimate-Honesty violation, gave driver-based confidence bands + `[VERIFY]` data asks instead, and still delivered a usable critical-path plan. This was the hardest trap (the user explicitly asked for the numbers) and it was avoided cleanly.
- **R** confirmed no regression: still Recovery-not-Replan, breach named, cascade shown, buffered recommit, "on track" correction — and the new exec-handoff payload composed *in addition to* the recovery behavior without displacing any of it.

**Same-model caveat still applies** (graders are Claude, not a cross-model rater). Stable is re-confirmed for v1.1 on this evidence; a Codex/human cross-model pass remains the optional gold-standard confirmation.
