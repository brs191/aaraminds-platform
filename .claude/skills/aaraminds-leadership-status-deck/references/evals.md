# Evals — leadership-status-deck

Per Anthropic's skill-authoring guidance, the bar is behavioral, not cosmetic. These scenarios are run
with the skill loaded; each must satisfy the **60-second success test** and the slide/contract gates.
Run on Sonnet and Opus (what's fine for Opus may underspecify for a smaller model).

## The pass/fail bar (applies to every scenario)

A reviewer (fresh subagent) reading only the generated deck must answer all five in 60 seconds:
(1) on track? (2) what changed? (3) what's at risk? (4) what decision is needed? (5) what should I
care about? Plus: five mandatory slides present; overall RAG = worst load-bearing area (no watermelon);
dimensional health with trend arrows; top 3–5 risks with owners; ask on slide 2 + slide 6; every
metric defined or `[VERIFY]`; visual-QA pass run; all six deliverables emitted.

## Scenario 1 — the "activity-dump rescue" (the real-world failure mode)

```json
{
  "skills": ["aaraminds-leadership-status-deck"],
  "query": "Build my June monthly VP status deck from these notes.",
  "files": ["dense_pmo_notes.md (per-initiative tables: stakeholders, dependencies, code-impacted apps, milestone tables, DORA metrics — no exec summary, no risk roll-up, no ask)", "may_deck.pptx"],
  "expected_behavior": "Produces an exec summary (slide 2) and dimensional health dashboard (slide 3) that did NOT exist in the source; moves the per-initiative detail to the appendix; surfaces a top-3–5 risk slide; computes MoM arrows vs may_deck; flags missing ask as 'no decision needed' or asks. A VP can answer the 5 questions in 60s."
}
```
Fails if: the deck reproduces the dense tables as primary slides, or has no executive summary / health dashboard.

## Scenario 2 — first-deck baseline (no prior month)

```json
{
  "skills": ["aaraminds-leadership-status-deck"],
  "query": "First monthly status deck for the Payments Reliability program, audience VP.",
  "files": ["status_notes.md", "raid_log.csv", "milestones.csv"],
  "expected_behavior": "Builds a clean baseline deck, omits trend arrows (states 'baseline month'), saves the deck as the trend seed, and emits the verification report listing any [VERIFY] gaps. Does not error on the missing previous deck."
}
```
Fails if: it errors on the missing prior deck, or fabricates trend arrows.

## Scenario 3 — integrity under missing data (no fabrication)

```json
{
  "skills": ["aaraminds-leadership-status-deck"],
  "query": "Refresh the monthly deck. Some inputs are incomplete.",
  "files": ["partial_notes.md (a milestone % missing; a risk with no owner; a metric with no baseline; a workstream renamed since last month)", "prior_deck.pptx"],
  "expected_behavior": "Marks the missing % and baseline [VERIFY]; marks the risk owner [VERIFY]; states the rename→trend mapping or marks the arrow [VERIFY]; never invents a number, owner, or status. The verification report lists every gap."
}
```
Fails if: any number, owner, or RAG is fabricated, or the rename is silently treated as continuity.

## Scoring rubric (score every run; enables regression tracking)

A fresh-subagent reviewer scores the generated deck on these dimensions:

| Dimension | Scale | Pass bar |
|---|---|---|
| Narrative clarity (answer-first, message titles) | 0–5 | ≥ 4 |
| Executive narrative strength (answers *why now*, *why it matters*, *why leadership should care*) | 0–5 | ≥ 4 |
| RAG integrity (no watermelon, thresholds applied, trend correct) | 0–5 | ≥ 4 |
| Evidence traceability (claims → source) | 0–5 | ≥ 4 |
| No fabrication (numbers/owners/status sourced or `[VERIFY]`) | pass/fail | pass |
| Visual QA (no overflow/overlap/placeholder; ≥1 fix-verify cycle) | pass/fail | pass |
| Executive usefulness (5 questions answerable in 60s) | 0–5 | ≥ 4 |

A run passes only if both pass/fail gates pass and every 0–5 dimension is ≥ 4. Record the scores per
run so a template change that regresses any dimension is caught.

## How to use

Run baseline **without** the skill first to confirm the gap is real, then with the skill. Score with
the rubric above. Bring any failure back into the SKILL.md or references as the smallest fix. Keep
these green before shipping a template change.
