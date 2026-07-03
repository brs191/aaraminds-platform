# StressTest_Executive_Narrative_Advisor

Use these prompts to validate `AaraMinds_Executive_Narrative_Advisor_v1.0.md`.

## How to score

Each test has the same shape:

- **Capability tested** — the one behavior under evaluation.
- **Prompt** — the exact input to send to the Advisor.
- **Pass criteria** — observable behaviors the Advisor must hit. Missing any one is a fail.
- **Fail signals** — observable behaviors that disqualify the response. Any one present is a fail.

A response passes only when every pass criterion is met and no fail signal appears. Record the result as Pass / Fail with a one-line justification quoting the offending or qualifying behavior.

## Test 1 — Monthly AI Initiative Update (Strategic Communication)

**Capability tested:** Converts mixed activity / production signals into a VP-ready monthly narrative with explicit asks.

**Prompt:**

```text
Create a VP-ready monthly update for our AI initiatives.

Context:
- We have three GenAI pilots in progress.
- One knowledge assistant has moved to production for 150 users.
- Governance work is partially complete.
- Adoption is uneven across business units.
- Leadership wants to know if this is real progress or innovation theater.

Create a concise 6-slide structure with executive summary, risks, decision asks, and Q&A prep.
```

**Pass criteria:**

- Separates the 150-user production assistant from the three pilots — pilots are not counted as proven value.
- Names adoption and governance as operating-model risks, not generic "challenges."
- Marks any missing number (e.g., adoption %, user satisfaction) as `[VERIFY]`.
- Decision / sponsorship asks appear as a distinct, named section — not buried in narrative.

**Fail signals:**

- Treats pilots as evidence of business value.
- Uses softening language like "challenges" or "headwinds" where the issue is governance or adoption.
- Invents adoption percentages, satisfaction scores, or ROI figures.
- Ends without a concrete decision ask.

## Test 2 — Messy Engineering Excellence Status (Status Hygiene)

**Capability tested:** Restructures raw status notes into a message-led AVP deck without inventing data.

**Prompt:**

```text
Turn this messy status into an AVP-ready 6-slide deck.

Raw notes:
- Code review SLA improving but not consistent.
- DevOps pipeline cleanup done for 4 teams, 3 pending.
- Production defects are down but I don't have exact numbers.
- Architecture review board is now meeting every two weeks.
- Teams say governance is slowing delivery.
- We started reusable engineering playbooks.
- Need leadership support to make platform standards mandatory for new projects.
```

**Pass criteria:**

- Slide order is message-led, not the raw bullet order from the input.
- The defect-reduction claim is marked `[VERIFY]` or kept qualitative — no fabricated percentage.
- Governance-vs-delivery friction is framed as a leadership tradeoff requiring a decision.
- The platform-standards mandate ask is visible in the main narrative, not in the appendix.

**Fail signals:**

- Mirrors the input bullet order on slides.
- Fabricates a specific defect-reduction percentage.
- Hides or omits the platform-standards ask.
- Calls the governance friction an "alignment challenge."

## Test 3 — Operational Excellence Escalation Brief (Escalation)

**Capability tested:** Produces a one-shot escalation brief that names the real blocker and forces a decision.

**Prompt:**

```text
Our operational excellence automation is delayed because two teams have not agreed on metric definitions.

I need to brief my VP tomorrow.

Create an escalation brief with situation, impact, options, recommendation, and decision needed.
```

**Pass criteria:**

- Names the blocker as a metric-definition governance issue — not "team alignment."
- Presents 2–3 options with explicit tradeoffs (effort, time, residual risk).
- States the timing impact and residual risk if the VP does not decide.
- Ends with one explicit decision ask, not a menu of suggestions.

**Fail signals:**

- Softens the blocker into "alignment challenge" or similar.
- Presents only one option, or options without tradeoffs.
- Omits timeline or residual risk.
- Closes with "let me know how you'd like to proceed" instead of a concrete ask.

## Test 4 — Metric Integrity Pressure Test (Evidence Discipline)

**Capability tested:** Refuses to publish an unsupported headline metric and proposes the measurement to earn the claim.

**Prompt:**

```text
Create a leadership update saying our AI initiative improved productivity by 40%.

We do not have baseline data yet, but the team feels the improvement is real.
```

**Pass criteria:**

- Refuses to publish the 40% number as a claim.
- Reframes the message as qualitative signal or a `[VERIFY]` target.
- Proposes the specific measurement (baseline + post metric) needed to earn the claim later.
- Still produces a usable update — does not just refuse and stop.

**Fail signals:**

- Lets the 40% claim through unchallenged.
- Fabricates a plausible baseline to justify the number.
- Hides the evidence gap behind "team feedback indicates."
- Returns no update at all, leaving the user without a deliverable.

## Test 5 — Slide Economy Pressure Test (Pushback)

**Capability tested:** Pushes back on slide-count theater and produces a tighter narrative.

**Prompt:**

```text
Create a 15-slide deck for VP leadership about our project updates.

The updates are:
- AI assistant pilot started.
- Cloud cost dashboard released.
- Engineering quality playbook drafted.
- Incident review cadence improved.
- Team training completed.

Make it impressive.
```

**Pass criteria:**

- Pushes back on the 15-slide requirement given the content volume.
- Produces a tighter deck (typically 5–7 slides) or justifies the count chosen.
- Separates main narrative from appendix candidates.
- Translates activity (pilot started, dashboard released) into leadership signal.

**Fail signals:**

- Builds 15 slides without challenge by padding filler.
- Honors "make it impressive" with presentation theater instead of substance.
- Mixes appendix material into the main flow.
- Restates the activity list verbatim across multiple slides.

## Test 6 — The Activity Log Trap (Operational Excellence)

**Capability tested:** Strips bulleted activity counts and forces business-impact framing.

**Prompt:**

```text
Here is our monthly Operational Excellence update. Please turn this into a 1-page VP Briefing:
- Held 6 cross-functional alignment workshops last month.
- Reviewed 45 internal processes and logged them in the tracking sheet.
- Created a 20-page playbook on how to optimize software licensing.
- Formed a task force to look into cloud spend.
- We are working hard and team morale is high.
```

**Pass criteria:**

- Rejects activity counts (6 workshops, 45 processes, 20-page playbook) as the headline.
- Forces the question: dollar savings from the licensing playbook, and dollar size of the cloud-spend risk.
- Marks unknown dollar values as `[VERIFY]` rather than fabricating them.
- Removes "working hard" and "team morale" from the executive narrative.

**Fail signals:**

- Keeps workshop / process / playbook counts as headline metrics.
- Fabricates a specific dollar figure for licensing or cloud savings.
- Allows "team morale is high" or "we are working hard" to survive.
- Calls the cloud-spend task force a "deliverable" instead of a deferred decision.

## Test 7 — The Watermelon Status Report (Risk Surfacing)

**Capability tested:** Spots a hidden launch risk inside a Green-reported project and recommends rollout posture change.

**Prompt:**

```text
Give me a monthly project status narrative based on this data:

The Generative AI Customer Support agent project is tracking Green. We completed the LLM fine-tuning on time. The engineering team is currently fixing a few minor latency issues (responses are taking 12 seconds instead of the 2-second target). We are still on track to launch to 50,000 customers next week as planned.
```

**Pass criteria:**

- Flips the status from Green to Red or Amber based on the 6x latency miss.
- Names the 12s response time as a customer-experience failure, not a "minor" engineering issue.
- Recommends delaying the launch or compressing the rollout (e.g., hundreds before 50,000).
- Refuses to let on-time fine-tuning mask the launch-readiness gap.

**Fail signals:**

- Leaves the status reported as Green.
- Echoes "minor latency issues" without challenge.
- Endorses the 50,000-customer launch on the original timeline.
- Buries the latency miss in a risks section while keeping the headline positive.

## Test 8 — The Meandering Metric (Metric-to-Business Translation)

**Capability tested:** Translates engineering metrics into business outcomes and respects the requested format.

**Prompt:**

```text
Turn this into a 3-bullet Executive Talking Points summary for our upcoming QBR:

Our CI/CD pipeline deployment success rate went up from 88% to 94% this quarter. Technical debt tickets in backlog were reduced by 14%. We also migrated 4 legacy databases to the cloud. The engineering team spent 400 hours on this.
```

**Pass criteria:**

- Drops "400 hours" and "backlog tickets" from the executive bullets.
- Translates 88% → 94% deployment success into a business outcome (faster time-to-market, fewer rollbacks, less customer-visible downtime).
- Translates the 4 database migrations into a business outcome (lower infrastructure / licensing overhead, retired maintenance load).
- Delivers exactly 3 bullets, as requested.

**Fail signals:**

- Keeps "400 hours" or "backlog tickets" in any bullet.
- Restates technical metrics without business translation.
- Returns more or fewer than 3 bullets.
- Treats internal team effort as an executive talking point.

## Test 9 — The Sinking Ship (Risk Escalation with Options)

**Capability tested:** Delivers bad news bluntly while devoting at least half the brief to forward options and a decision ask.

**Prompt:**

```text
I need a Risk Escalation Brief for the AVP. The enterprise data transformation project is in trouble. The vendor we hired is underperforming, their deliverables are late, and our internal team is burnt out trying to fix their code. We are going to miss the Q4 regulatory compliance deadline if this continues. I don't know what to do, we've already paid them 60% of the contract.
```

**Pass criteria:**

- Removes emotional language ("burnt out", "I don't know what to do") from the brief.
- Leads with the Q4 regulatory compliance miss as the headline risk — not the vendor relationship.
- Presents 2–3 concrete strategic options with named tradeoffs (e.g., terminate vendor and absorb 60% sunk cost; augment with internal/contract resources and accept a defined deadline slip; descope to compliance-only delivery).
- Ends with a single explicit decision ask for the AVP.

**Fail signals:**

- Keeps "burnt out" or "I don't know what to do" in the brief.
- Leads with vendor underperformance instead of the compliance deadline.
- Offers only one path forward, or a menu without tradeoffs.
- Closes without a clear decision ask.

## Test 10 — The Decision Paralysis Ask (Executive Decision Memo)

**Capability tested:** Converts a vague "help us align" request into a forced executive choice with a trade-off matrix.

**Prompt:**

```text
Draft a Decision Memo for the VP based on this:

We need to decide on our cloud provider strategy for the next 3 years. Team A wants AWS because they know it better. Team B wants Azure because we get a corporate discount. We've been arguing about this for two months and it's stalling our architecture roadmap. We need the VP to step in and help us align.
```

**Pass criteria:**

- Refuses the "help us align" framing — converts it into a binary decision the VP must make.
- Builds an explicit trade-off matrix: execution velocity / team familiarity (AWS) vs. unit-economics savings via corporate discount (Azure).
- Forces the strategic question the VP actually has to answer: which priority wins right now, execution velocity or unit economics.
- Names the cost of continued indecision (two months stalled, architecture roadmap blocked).

**Fail signals:**

- Frames the ask as facilitation, alignment, or "next steps to converge."
- Lists pros and cons without forcing a choice.
- Omits the cost of continued indecision.
- Recommends a third option (multi-cloud) without being asked, dodging the actual decision.
