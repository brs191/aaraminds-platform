# Monthly leader status deck — slide-by-slide spec (VP-optimized)

The locked template. Same order, same intent, every month — so the leader pattern-matches instantly.
**Mandatory: slides 1–5.** Slide 6 is mandatory when a decision exists. Slide 7 is recommended.
Optimize for AVP/VP altitude; push working detail to the appendix.

Title rule (non-negotiable): **the slide title states the message, not the topic.** "CPR live since
Dec 9; ADI Enterprise onboarded — 2026 scope on track" — not "CPR Update."

The five-question bar: after slides 1–6 a leader must, in 60 seconds, know — are we on track? what
changed? what's at risk? what decision is needed? what should I care about?

---

## Slide 1 — Cover

- **Carries:** initiative/program · reporting month · owner + reviewers · **overall RAG** (one large
  chip) with a month-over-month arrow.
- **Overall RAG rule:** equals the reality of the **worst load-bearing dimension or workstream**,
  never an average that launders red.

## Slide 2 — Executive summary (the 30-second slide)

The most important slide. If the leader reads only this, they can still answer the five questions.

- **Overall status** — RAG + one-sentence headline (message-led).
- **Key wins** — 2–3, each with its business meaning (not activity).
- **Key risks** — the 1–3 that matter most (full list on slide 5).
- **Leadership ask** — classified (Inform / Align / Decide / Unblock / Sponsor / Accept-risk), or an
  explicit "No decision needed this month."
- **Confidence** — High / Medium / Low + one-line reason. Score it by these criteria:
  - **High:** prior deck available · status notes complete · metrics sourced · risks have
    owner/date/mitigation · no unresolved critical `[VERIFY]`.
  - **Medium:** some inputs missing, but core health and top risks are sourced.
  - **Low:** missing prior deck, missing metric baselines, unclear RAG, or a major unresolved
    `[VERIFY]`.
- **Since last month** — a one-line change summary (drawn from the MoM change-log).
- **Structure (narrative spine / BLUF):** What changed → Why it matters → Confidence → Ask.

## Slide 3 — Program health dashboard

The one "dashboard" slide. Leaders consume color faster than percentages.

- **Dimensional RAG** — one row per dimension with a chip + **trend arrow** + one-line "why":
  **Scope · Schedule · Quality · Cost · Risk · Dependencies.** (RAG each dimension independently;
  keep Risk separate — a project can be Green on delivery and high on Risk.)
- **Per-workstream roll-up** — a compact second table: workstream · RAG · arrow · one-line reason
  (≤7 rows; remainder to appendix).
- **Trend arrows are computed** from last month's deck (see the table below), never asserted.

### Default RAG thresholds (override per your PMO)

There is no universal numeric RAG standard — best practice is that each PMO sets and *publishes* its
own. These are sensible defaults the skill applies unless the user/PMO supplies different ones; show
the threshold key on the dashboard so the reader knows the rules:

| Dimension | Amber | Red |
|---|---|---|
| **Schedule** | committed milestone slips > 5 business days | VP-committed milestone slips > 10 business days |
| **Cost** | forecast exceeds baseline by 5–10% | forecast exceeds baseline by > 10% |
| **Quality** | known Sev3 defects against the gate | Sev1/Sev2 defect blocks the release |
| **Scope** | unapproved scope change in flight | committed scope cannot be delivered in the window |
| **Dependencies** | external dependency at risk | external dependency blocks a committed milestone |
| **Risk** | open risk with mitigation in progress | mitigation needs leadership action and no owner/date exists |

Apply consistently month to month — changing thresholds mid-stream destroys trust. If the user
overrides a threshold, record it so next month matches.

### Deterministic trend rules

| Last month | This month | Arrow |
|---|---|---|
| Red | Amber or Green | ↑ improved |
| Amber | Green | ↑ improved |
| Green | Green (or any unchanged) | → unchanged |
| Amber | Amber (unchanged) | → unchanged |
| Green | Amber or Red | ↓ worsened |
| Amber | Red | ↓ worsened |
| — (didn't exist) | present | NEW |
| present | removed | moved to appendix / noted as closed |

**Rename / merge / split:** if a dimension or workstream is renamed, merged, split, or removed, state
the mapping used to compute the arrow (e.g., "ROME + CDR merged into True North-CDR"). If the mapping
is unclear, mark the arrow `[VERIFY]` rather than inventing continuity.

## Slide 4 — Major accomplishments

Top wins only, framed as outcomes — never an activity log.

- **Top 5 accomplishments** (hard cap), each: the win · its **business impact** · **completion
  evidence** (what proves it's done — a date, a metric, a sign-off).
- **Planned-vs-actual traceability:** tie each to the prior commitment — "committed by this review;
  delivered / slipped to <date> because <reason>." Full milestone table lives in the appendix.
- **Metric rule:** every number carries definition + baseline + window + actual/forecast/target, or
  `[VERIFY]`.

## Slide 5 — Risks & blockers

The honest slide. Leaders fund and unblock from here.

- **Top 3–5 risks only** (not every risk — the long tail goes to the RAID appendix). One row each:
  - **Risk statement** (specific) · **business impact** · **probability** (H/M/L) · **severity**
    (H/M/L) · **mitigation** · **owner** · **open since** (month first raised).
- **De-euphemize:** "Risk," not "challenge." If off track, name the dimension — scope / time / cost /
  adoption / quality / dependency / governance / benefits.
- **Aging:** a risk open ≥3 months is escalation-worthy — flag it.

## Slide 6 — Leadership decisions required (mandatory when any decision exists)

- **Include when** there is a decision; **omit** when there isn't — and say "No decision needed this
  month" on slide 2.
- **Per decision:** **issue** · **impact** (what's blocked / at stake) · **decision needed** (with
  options + your recommendation) · **due date** · **owner**.
- Must agree with the ask stated on slide 2.

## Slide 7 — Next-month outlook (recommended)

- **Upcoming milestones** — the 2–4 that define next month.
- **Forecasted risks** — what could move the RAG.
- **Confidence level** — High / Medium / Low for next month's plan + reason.

## Appendix (optional)

Detailed milestone tables · dependency tracking · architecture diagrams · team/DORA metrics (deploy
frequency, defects, turnaround) · full per-capability detail · the full RAID log · the full MoM
change-log. Nothing here is load-bearing for a decision — if a leader needs it to decide, it belongs
in slides 1–6.

---

## Month-over-month change-log (feeds slide 2 + a deliverable)

Compute and carry every month:

- **New risks** (raised this month) · **Closed risks** (retired) · **Escalated risks** (severity or
  probability up) · **Completed milestones** · **Slipped milestones** (with old vs new date + reason).

## Q&A prep (speaker notes, not a slide)

Anticipate: Why now? What changed since last month? Measurable impact? What's blocked? What decision
do you need? What if we do nothing? What risk are we accepting? Confidence level? Next milestone?
What would make this fail?

## Default deck size

5–7 primary slides + appendix. A healthy month is ~6. A month with a live decision or a red dimension
may reach 7. If primary slides exceed ~7, detail is leaking out of the appendix — pull it back.
