# Historical intelligence — multi-month memory (opt-in)

Single-month trend (this month vs last) answers "are we better or worse?" Historical intelligence
answers the question that actually moves executives: **"is this a blip or a pattern?"** A risk open
five months is not the same as a new one; a schedule amber for three consecutive months is a chronic
problem, not a wobble. This turns reporting into intelligence.

## The mechanism — a status ledger

Single-month comparison can't see patterns, so the skill maintains a small **persistent ledger** it
updates every month and carries forward as a deliverable. The ledger is the memory; the deck reads
from it.

Ledger record (one row per tracked item — dimension, workstream, risk, dependency, milestone):

```
item_id · type · first_seen_month · status_history[YYYY-MM: RAG/state] ·
current_streak (consecutive months at current status) · times_appeared · last_rebaseline
```

- **First month:** seed the ledger from this month's inputs (no history yet).
- **Each month:** append this month's status to each item's history; recompute streaks; add new items;
  mark closed items closed (keep their history).
- **No ledger but prior decks exist:** reconstruct the ledger from the available prior decks, marking
  any inferred history `[VERIFY]`.

## Derived signals (what the ledger surfaces)

| Signal | Rule | So what |
|---|---|---|
| **Aging risk** | open ≥ 3 months | escalate — mitigation isn't working |
| **Chronic dimension** | Amber-or-worse ≥ 3 consecutive months | structural, not a wobble — needs a different intervention |
| **Systemic dependency** | same dependency in ≥ 3 decks | not a one-off — a standing cross-team issue |
| **Commitment erosion** | milestone re-baselined ≥ 2 times | the plan, not the month, is the problem |
| **Sustained green** | green ≥ 3 months | safe to reduce reporting depth / declare stable |

## Where it shows in the deck

- A compact **"Chronic issues & trends"** callout on the executive summary (slide 2) — the 1–3
  patterns leadership should see (e.g., "Dependency on ICAP has blocked a milestone in 4 of the last 5
  months — needs an executive decision, not another mitigation").
- Aged/chronic items get an **"open since / N months"** marker wherever they appear (risks slide,
  dashboard).
- Full history lives in the appendix and the ledger deliverable.

## Deliverable

The **updated status ledger** is emitted every run alongside the deck, so next month's deck inherits
the memory. Treat it as the trend base together with last month's deck.

## Verify (historical)

- Ledger updated this month (history appended, streaks recomputed)?
- Aging/chronic/systemic signals surfaced on the exec summary where present, not buried?
- Any reconstructed history marked `[VERIFY]`?
- Ledger emitted as a deliverable for next month?
