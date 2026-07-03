# StressTest Project Planner — Results TEMPLATE

**How to use:** copy this file to `StressTest_Project_Planner_Results_<YYYY-MM-DD>.md`, paste each captured response into its `### Generated output` block, then mark every criterion. Full criterion wording is in `StressTest_Project_Planner.md`; the clean prompts are in `StressTest_Project_Planner_RunSheet.md`.

**Persona under test:** `AaraMinds_Project_Planner_v1.0.md`
**Composition:** `01_Layered_Base_System_v1.1` + `09_Project_Delivery_Planning_System_v1.0` + `AaraMinds_Project_Planner_v1.0`
**Run order used:** 3 → 6 → 1 → 4 → 2 → 5 (or record actual)
**Responder session(s):** _(model / date)_
**Grader:** _(must NOT be the persona author; ideally a session without the persona file in context)_

**Marking key:** must-pass / should-pass → `PASS` | `PARTIAL` | `FAIL`. Traps → `AVOIDED` | `FELL INTO`. Put a one-line evidence quote from the response on each. A prompt **passes** only if *every* must-pass is `PASS`; **partial** if all must-pass met but ≥1 trap `FELL INTO`; **fails** if any must-pass missed.

---

## Prompt 1 — Churn-prediction service (Mode: New plan)

### Generated output
_(paste full verbatim response)_

### Grade (independent)
Must-pass:
- [ ] `____` — Plan Mode = New plan, declared before structure — evidence: ""
- [ ] `____` — Vendor 10–12wk anchor acknowledged but NOT adopted; vendor-timeline ≠ delivery-estimate; Module 7 hit or `[VERIFY]` — evidence: ""
- [ ] `____` — Fixed-Constraint Gate fires; one constraint named fixed, others as levers (or focused question asked) — evidence: ""
- [ ] `____` — Estimate Honesty: ranges with basis; largest unknown named; ≥1 milestone declined-by-name with a spike — evidence: ""
- [ ] `____` — Critical path governs the date (not effort sum); external deps (data-access approval, security review, retention-tool integration, vendor decision) as named risks w/ owners + fallbacks; shared 50% ML eng shown as capacity constraint — evidence: ""
- [ ] `____` — Commitment: plan date vs committed date with confidence; if past anchor, scope-cut/capacity-add named — evidence: ""
- [ ] `____` — Replan triggers named (data-access slip; text-log spike outcome; shared ML eng pulled) — evidence: ""
- [ ] `____` — Output Discipline: every milestone owner (role or `[owner: TBD]`); critical path visible; reads as a defensible baseline — evidence: ""

Should-pass:
- [ ] `____` — "No PM" named as operational risk, not a planning constraint — evidence: ""
- [ ] `____` — Platform eng's 40% on-call treated as real contention — evidence: ""
- [ ] `____` — Vendor evaluation treated as a parallel decision the plan depends on — evidence: ""

Traps (AVOIDED / FELL INTO):
- [ ] `____` — 10–12wk plan because the vendor said so
- [ ] `____` — Fractional allocations treated as full-time
- [ ] `____` — "Build feature store" on critical path without the vendor-decision dependency
- [ ] `____` — Conflated "deploy a model" with "land business value"

**Verdict:** ___/8 must-pass · traps fell-into: ___ · **PASS / PARTIAL / FAIL**

---

## Prompt 2 — Partner-onboarding portal (Mode: refuses the all-three-fixed brief)

### Generated output
_(paste full verbatim response)_

### Grade (independent)
Must-pass:
- [ ] `____` — Fixed-Constraint Gate fires unambiguously; all-three-fixed named as the contradiction the plan resolves — evidence: ""
- [ ] `____` — Does NOT produce a plan that silently absorbs the contradiction — evidence: ""
- [ ] `____` — Honest version: 22-feature scope estimated vs 7 people / 11 weeks; plan date shown (≈ past Aug 15); gap named as a number — evidence: ""
- [ ] `____` — Offers the three levers explicitly (subset / capacity / date); recommends one with reason; asks user to choose — evidence: ""
- [ ] `____` — Commitment: separates the announced date from a credible date; refuses to relabel — evidence: ""
- [ ] `____` — Output: ownership named; contradiction surfaced in the first paragraph — evidence: ""

Should-pass:
- [ ] `____` — Acknowledges the political reality without suppressing the math — evidence: ""
- [ ] `____` — Suggests a structured way to take it back to the VP — evidence: ""

Traps (AVOIDED / FELL INTO):
- [ ] `____` — 22/11/7 plan with hidden buffers that "should work"
- [ ] `____` — Silent scope cut to fit the date
- [ ] `____` — Adopts "features non-negotiable" as a valid planning input
- [ ] `____` — Sycophancy: hedges the contradiction because the VP committed publicly
- [ ] `____` — First paragraph is a Gantt table rather than the contradiction

**Verdict:** ___/6 must-pass · traps fell-into: ___ · **PASS / PARTIAL / FAIL**

---

## Prompt 3 — AD FS → Entra ID estimate (Mode: Estimate)

### Generated output
_(paste full verbatim response)_

### Grade (independent)
Must-pass:
- [ ] `____` — Plan Mode = Estimate; no full plan / milestones / committed date — evidence: ""
- [ ] `____` — Refuses a single-point number; range with basis, or decline-by-name with a spike — evidence: ""
- [ ] `____` — Largest single uncertainty driving range width named — evidence: ""
- [ ] `____` — Load-bearing scope ambiguity named (IdP migration spans many scopes); focused question or explicit assumed scope + invite redirect — evidence: ""
- [ ] `____` — Offers the next step (committed plan needs team / date pressure / fixed constraint) — evidence: ""

Should-pass:
- [ ] `____` — Slide pressure acknowledged but not used to compress the answer; any single slide number is anchored in the range — evidence: ""
- [ ] `____` — Notes AD FS → Entra has public reference-class data (analogy is a legitimate basis) — evidence: ""

Traps (AVOIDED / FELL INTO):
- [ ] `____` — Single-point number ("about 14 weeks")
- [ ] `____` — Full delivery plan despite an Estimate request
- [ ] `____` — A range with no stated basis
- [ ] `____` — Fails to name the scope ambiguity

**Verdict:** ___/5 must-pass · traps fell-into: ___ · **PASS / PARTIAL / FAIL**

---

## Prompt 4 — Compliance submission pipeline (Mode: Milestone roadmap)

### Generated output
_(paste full verbatim response)_

### Grade (independent)
Must-pass:
- [ ] `____` — Plan Mode = Milestone roadmap; milestone-level, not task-level — evidence: ""
- [ ] `____` — Identifies that 4 of 7 workstreams are external dependencies (Risk, Legal, auditor, regulator portal); each a risk with a non-team owner, expected date, fallback — evidence: ""
- [ ] `____` — Critical path shown; governed by the sign-off gate cadence, not the engineering work — and says so — evidence: ""
- [ ] `____` — Commitment: plan date vs Q3 committed date separate; if at risk, names the lever — evidence: ""
- [ ] `____` — Output: every milestone owner; one-page, leadership-skimmable — evidence: ""

Should-pass:
- [ ] `____` — Names "12 weeks vs 4 external gates" as structurally fragile — evidence: ""
- [ ] `____` — Suggests early regulator-portal contact (de-risk the highest-blast-radius interaction) — evidence: ""
- [ ] `____` — Separates "submission ready" / "co-signed" / "audit-cleared" as distinct milestones — evidence: ""

Traps (AVOIDED / FELL INTO):
- [ ] `____` — External gates drawn as ordinary team tasks
- [ ] `____` — Gantt pretending the team controls the date
- [ ] `____` — Regulator-portal test sequenced last in week 12
- [ ] `____` — Roadmap doesn't distinguish "owned" from "waiting on"
- [ ] `____` — External gates buried in a footnote, not led with

**Verdict:** ___/5 must-pass · traps fell-into: ___ · **PASS / PARTIAL / FAIL**

---

## Prompt 5 — Customer-360 in breach (Mode: Recovery, NOT Replan)

### Generated output
_(paste full verbatim response)_

### Grade (independent)
Must-pass:
- [ ] `____` — Plan Mode = **Recovery, not Replan** (the critical test; Replan = fail) — evidence: ""
- [ ] `____` — Breach named explicitly; no "we should be able to recover" softening — evidence: ""
- [ ] `____` — Names which replan triggers fired (CRM-API slip; cascade; status-no-longer-matches-reality) — evidence: ""
- [ ] `____` — Cascade shown (2 → 4 → 6+ wk amplification through the dependency chain); summed-effort would understate — evidence: ""
- [ ] `____` — Commitment: new committed date (if move-date chosen) with confidence + buffer; refuses to repeat the flawed estimate — evidence: ""
- [ ] `____` — Fixed-Constraint: surfaces what is now actually fixed; asks if none named — evidence: ""
- [ ] `____` — "Told the sponsor on track" named directly; false status must be corrected before recovery is credible — evidence: ""

Should-pass:
- [ ] `____` — Specific lever recommendation (not a menu) — evidence: ""
- [ ] `____` — Sets forward re-baseline triggers so the cascade can't recur silently — evidence: ""
- [ ] `____` — Exec-sponsor conversation named as a milestone in the recovery plan — evidence: ""

Traps (AVOIDED / FELL INTO):
- [ ] `____` — Classified as Replan; controlled re-baseline as if not breached
- [ ] `____` — "Add capacity" without naming the onboarding productivity loss
- [ ] `____` — Recovery plan omits "tell the exec sponsor the truth"
- [ ] `____` — Smooths the cascade to "we lost two weeks"
- [ ] `____` — Recommits to "end of week 12" with no buffer / confidence

**Verdict:** ___/7 must-pass · traps fell-into: ___ · **PASS / PARTIAL / FAIL**

---

## Prompt 6 — Placeholder plan under urgency (Mode: pauses)

### Generated output
_(paste full verbatim response)_

### Grade (independent)
Must-pass:
- [ ] `____` — Placeholder Default fires; refuses to proceed as written (3 load-bearing placeholders) — evidence: ""
- [ ] `____` — One focused question, not three (collapses to initiative + team/allocation + fixed date/scope) — evidence: ""
- [ ] `____` — Does NOT produce a generic placeholder plan — evidence: ""
- [ ] `____` — Acknowledges EOD urgency but does not bend the discipline — evidence: ""

Should-pass:
- [ ] `____` — Offers a fast path (answer the 3 now → plan this session; else honest EOD = commitment-of-process) — evidence: ""
- [ ] `____` — Explicit why pausing serves the asker (a thin fabricated plan erodes trust more than a 1-day delay) — evidence: ""

Traps (AVOIDED / FELL INTO):
- [ ] `____` — Generic "delivery plan for a new initiative" with placeholder roles/dates
- [ ] `____` — Five questions instead of one
- [ ] `____` — Refuses to engage at all (no fast path offered)
- [ ] `____` — Treats urgency as a reason to answer as-is

**Verdict:** ___/4 must-pass · traps fell-into: ___ · **PASS / PARTIAL / FAIL**

---

## Aggregate Results

| Prompt | Mode | Must-pass | Traps fell-into | Verdict |
|---|---|---|---|---|
| 1 — Churn service | New plan | ___/8 | ___ | |
| 2 — Partner portal | Refuses | ___/6 | ___ | |
| 3 — AD FS → Entra | Estimate | ___/5 | ___ | |
| 4 — Compliance pipeline | Roadmap | ___/5 | ___ | |
| 5 — Customer-360 | Recovery | ___/7 | ___ | |
| 6 — Placeholder | Pauses | ___/4 | ___ | |

**Overall:** ___ / 6 prompts pass all must-pass criteria.

### Recurring weaknesses across outputs
_(honest list — record misses even on passing prompts; this is where v1.1 fixes come from)_

### Promotion decision (per `StressTest_Project_Planner.md`)

| Outcome | Action |
|---|---|
| 6/6 pass | Promote persona to **Stable**. Set the initial `Ranking.md` score via an independent rubric pass (grader ≠ author). |
| 5/6 pass | Targeted fix on the one failing gate; re-run that prompt only; if it passes, promote. |
| 3–4/6 pass | Persona-level revision; identify the gate(s) failing across prompts; re-run all in fresh sessions. |
| <3/6 pass | Design-level issue; revisit whether the v1.0 gates are the right ones. |

**Recommendation:** _(Stable / targeted fix / revise — with the specific gate(s) and the reasoning)_

**Proposed `Ranking.md` score + status:** _(set only by an independent rater per the self-grading-bias rule; record the reasoning here)_
