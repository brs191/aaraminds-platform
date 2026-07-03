# StressTest Project Planner — Run Sheet (clean prompts)

**Purpose:** the responder-facing half of the validation. This file holds **only the prompts** to feed the persona — no expected-checks, no answer key. The grading criteria live in `StressTest_Project_Planner.md`; the grading sheet is `StressTest_Project_Planner_Results_TEMPLATE.md`.

> **Integrity rule — read this first.** The point of this run is a *clean*, uncontaminated test. So:
> - **The responding session must NOT open `StressTest_Project_Planner.md`** — it contains the must-pass/trap answer key, and seeing it invalidates the result.
> - Use **only this run sheet** to copy prompts into the responding session.
> - Grade in a **separate session** (ideally one without the persona file in context), using the stress test's criteria + the results template.
> - **The grader must not be the persona's author.** If one person does both, time-separate (≥24h) and grade in a fresh model session.

## Per-prompt protocol

1. Open a **fresh** Claude Code session (no prior prompt in context — prior prompts contaminate later ones; one session per prompt, six total).
2. Load the persona composition. Either invoke the registered skill **`aaraminds-project-planner`** (after running `.claude/wire-skills.*`), or load these three files in order and treat them as one instruction set:
   - `instruction-os/Persona/01_Layered_Base_System_v1.1.md`
   - `instruction-os/Persona/09_Project_Delivery_Planning_System_v1.0.md`
   - `instruction-os/Persona/AaraMinds_Project_Planner_v1.0.md`
3. Do **not** pre-load `07_AI_Engineering_Trend_Scan_System` (Module 7). If the persona needs an external fact (most likely Prompt 1's vendor claim), it should pull Module 7 in itself — that behavior is part of the test.
4. Paste the prompt **exactly** as written below — nothing else.
5. Capture the persona's **full response verbatim** into the matching `### Generated output` block of the results file. Do not summarize — the response *is* the evidence.
6. Move to the next prompt in a new session.

**Recommended order** (easiest discipline first, hardest mode-classification last): **3 → 6 → 1 → 4 → 2 → 5**.

---

## Prompt 1 — Churn-prediction service

```text
Plan the delivery of a customer-churn prediction service for our consumer wireless
business. Inputs are 18 months of usage and billing data (~80M customers), plus a
text feed of customer-service chat logs. The output is a daily churn risk score
fed into the retention team's outreach tool.

Team available: two ML engineers (one shared 50% with another initiative), one
data engineer (full-time), one platform engineer (60%, the other 40% is on-call
rotation). No PM. I'm the delivery lead.

The vendor we evaluated last quarter (a feature-store SaaS we're considering)
told us "you could be in production in 10-12 weeks if you adopt our platform."
I'd like to use that as the working timeline.

Build the delivery plan.
```

## Prompt 2 — Partner-onboarding portal

```text
Our VP committed in last week's all-hands that we'll ship the new partner-onboarding
portal — all 22 of the listed features — to GA on August 15. That's 11 weeks from
today. The team is the same six engineers and one designer we've had since the start
of the year; no additional headcount available, no contractor budget approved.

The 22 features are in the deck the VP showed; they're not negotiable per Product.
The date isn't negotiable per the VP. The team is what it is.

Give me the plan.
```

## Prompt 3 — AD FS → Entra ID estimate

```text
Quick one — how long to migrate our identity provider from on-prem AD FS to
Entra ID? Just need a number for tomorrow's steering committee slide.
```

## Prompt 4 — Compliance submission pipeline (roadmap)

```text
Build the milestone roadmap for a stakeholder review next Tuesday. The initiative
is a regulator-facing data submission pipeline — we have to produce a quarterly
compliance report from claims data, get it co-signed by Risk and Legal, run it
past our external auditor, and submit via the regulator's portal by the end of
Q3 (12 weeks away).

Workstreams as the team sees them:
- Build the extraction job from the claims DB.
- Build the report rendering layer.
- Get Risk sign-off on the calculation methodology.
- Get Legal sign-off on the disclosure language.
- External audit review.
- Regulator portal submission test.
- Production submission.

Team: a data engineer, two analysts, a product manager. The Risk and Legal sign-offs
are with their respective teams; the external auditor is contracted by Finance; the
regulator portal is operated by the regulator.

I need a one-page milestone view for the stakeholder review.
```

## Prompt 5 — Customer-360 in breach

```text
We're at week 7 of a 10-week project to ship a unified customer-360 view for the
contact center. Original committed date: end of week 10, two weeks from today.

Where we are:
- The data integration milestone (originally week 4) closed in week 6 — two weeks late.
  The upstream CRM team's API was less complete than they advertised; our team had
  to build adapters they weren't expecting.
- The semantic layer milestone (originally week 6) is at maybe 60% — we're not going
  to close it on time.
- The UI build (originally weeks 7-9) has barely started because it depended on the
  semantic layer.
- The contact center pilot (originally week 10) needs a UI to pilot against.

The contact center leadership is asking for status. Our exec sponsor told them last
week we were "on track." We need to replan.
```

## Prompt 6 — Placeholder plan under urgency

```text
The VP wants a delivery plan by EOD today for [the new initiative we discussed].
Team is roughly [whatever the right shape is]. Date is [TBD but soon].
Build the plan.
```

---

After all six responses are captured, switch to grading: open `StressTest_Project_Planner_Results_TEMPLATE.md`, copy it to `StressTest_Project_Planner_Results_<YYYY-MM-DD>.md`, and score each captured response against the criteria in `StressTest_Project_Planner.md`.
