# Monthly input & deliverables contract

The deck is only as honest as its inputs. This defines what the skill expects in, what it must return
out, and the bar it must clear. Missing inputs are **flagged in the verification report, never
invented.**

## Mandatory inputs

1. **Reporting month** and **initiative/program name + owner**; the audience this month (AVP/VP,
   director, manager).
2. **Previous month's deck** — the .pptx (or its key facts) so trend, slippage, and aged risks are
   computed, not guessed. First month: say "first deck."
3. **Current month's status notes** — what was committed and what shipped; raw notes are fine, the
   skill converts activity to signal.
4. **RAID log** — risks, assumptions, issues, dependencies (feeds slide 5 + the appendix).
5. **Jira / Azure DevOps metrics** — delivery/throughput/defect data (feeds accomplishments + DORA
   appendix).
6. **Milestone tracker** — milestones with planned vs actual/forecast dates and % complete.
7. **Dependency tracker** — cross-team / platform dependencies and their status.
8. **Leadership asks** — any decision/support needed this month (feeds slide 6).
9. **Financial metrics** — budget/spend vs plan (feeds the Cost dimension + appendix).

If a source isn't available as a file, point the skill at notes or connect the system (Jira/Azure
DevOps/RAID tool) and it will pull from there.

## Optional inputs (enable the opt-in modes)

- **Audience profile** — the leader's role (VP Engineering / VP Product / CFO / CIO / Board). Re-weights
  emphasis and ordering only; defaults to generic AVP/VP. See `references/audience-profiles.md`.
- **Status ledger** — the prior multi-month memory file (historical-intelligence mode). If absent, the
  skill reconstructs from available prior decks (`[VERIFY]`) or seeds a new one. See
  `references/historical-intelligence.md`.
- **Multiple programs' inputs** — triggers portfolio roll-up mode. See `references/portfolio-rollup.md`.

## Mandatory deliverables (every run returns all of these)

1. **`.pptx`** — 5–7 primary slides + appendix, AaraMinds visual identity, filename
   `Status_<Initiative>_<YYYY-MM>.pptx`, saved to the user's selected folder.
2. **Executive summary** — a one-page AVP/VP narrative (slide-2 content as prose).
3. **Evidence report** — every load-bearing claim traced to its source input.
4. **Verification report** — every `[VERIFY]` item and gap requiring manual review. Example:
   ```
   [VERIFY] Milestone completion % not found for Contract Builder.
   [VERIFY] Risk owner missing for vendor-onboarding risk.
   [VERIFY] Dependency date inconsistent: tracker says 03-15, notes say 03-22.
   ```
5. **Month-over-month change summary** — new / closed / escalated risks; completed / slipped milestones.
6. **Optional:** speaker notes / Q&A prep.

## The 60-second success test (the skill's pass/fail bar)

After reviewing the generated deck, a leader must answer all five in under a minute:

1. Are we on track? 2. What changed? 3. What is at risk? 4. What decision is required? 5. What should
I care about this month?

If they can't, the deck failed — revise before delivering.

## How the skill handles gaps

- **No previous deck** → clean first deck, saved as the trend baseline; arrows omitted.
- **No RAG set** → proposes RAG per dimension/workstream from progress + risks; asks you to confirm.
- **Undefined metric** → kept but tagged `[VERIFY]`; never fabricates a baseline or percentage.
- **Missing owner** → claim kept, owner marked `[VERIFY]` (owners are mandatory on risks/blockers/
  milestones/asks).
- **No ask** → "No decision needed this month" on slide 2; slide 6 omitted.
- **Vague risk ("some challenges")** → asks for the specific risk + dimension rather than shipping a
  euphemism.
- **Renamed/merged/split workstream with unclear mapping** → trend marked `[VERIFY]`.

## Minimal example (paste-and-go)

```
Month: June 2026
Initiative: Common Capabilities — owner: R. Bhupathiraju — audience: VP (+ AVP)
Previous deck: attached (May 2026)
Status notes: CPR live 9 Dec; ADI Enterprise onboarded; Contract Builder milestone slipped 2 weeks
  (load-test window); integration testing started.
RAID: R1 vendor onboarding delay (impact Q3 release; owner PM; mitigation expedite; open since May).
Milestones: CPR Phase 2 dev 60% (target 06-30); BCLM design done 06-12 (planned 06-10).
Jira/ADO: deploy freq Medium; prod defects 0 since 2026; turnaround <24h.
Dependencies: AQUA, DREAM (green); ICAP (amber — awaiting env).
Leadership ask: Approve expedited vendor onboarding (due 06-15, owner PM).
Financials: 99,012 hrs YTD; 72% enhancements / 15% sustenance / 13% admin (on plan).
```
