---
name: aaraminds-leadership-status-deck
description: >-
  Builds the recurring monthly AVP/VP leadership status deck (.pptx) from project inputs.
  Use for monthly status updates, leadership readouts, program-health decks, or refreshing
  last month's status deck. Produces a locked executive template (executive summary, RAG health
  dashboard, accomplishments, risks, leadership decisions, outlook, appendix) with month-over-month
  trend, evidence and verification reports, and a mandatory visual-QA pass. Composes
  aaraminds-executive-narrative-advisor for judgment; not for one-page briefs/memos, delivery plans
  (aaraminds-project-planner), or external content (aaraminds-content-strategist). Never invents
  status, metrics, owners, risks, or decisions — gaps are marked [VERIFY].
version: 1.3.0
last_updated: 2026-06-17
---

# AaraMinds Leadership Status Deck

Turns a month of execution into a **decision-grade monthly status deck** optimized for an AVP/VP
audience. The governing principle, from real-world review: **the deck is not the product —
leadership clarity is the product.** A deck packed with delivery detail can score 8.5/10 for a
delivery manager and 6.5/10 for a VP at the same time; this skill optimizes ruthlessly for the
higher altitude and pushes working detail to the appendix.

This skill is the **production layer**. Narrative judgment lives in
`aaraminds-executive-narrative-advisor` (and its persona/modules), which this skill **loads and
defers to**. What this skill owns: the VP-optimized locked template, deterministic trend, the input
and deliverables contracts, and the `.pptx` build with QA. When narrative judgment and production
scaffolding conflict, the narrative gates win.

## The success test (the bar the deck must clear)

A leader must be able to answer all five questions **within 60 seconds** of opening the deck:

1. **Are we on track?** (overall status)
2. **What changed since last month?** (movement)
3. **What is at risk?** (top risks)
4. **What decision do you need from me?** (the ask)
5. **What should I care about this month?** (the headline)

If a leader cannot answer these five after reviewing the generated deck, the skill has failed. Every
template and gate below exists to serve this test.

## When this skill applies / does not

Applies to a recurring **monthly** leadership status deck for an initiative, program, squad, or
portfolio. Not for: one-page briefs, escalation or decision memos (`aaraminds-executive-narrative-advisor`);
the delivery plan/estimates/roadmap itself (`aaraminds-project-planner`); external content
(`aaraminds-content-strategist`); pure visual polish (Module 2 directly).

## Composition — load before producing

```text
aaraminds-executive-narrative-advisor   (judgment: gates, signal-over-activity, Q&A prep)
+ 02_Visual_Identity_System_v1.1.md      (slide hierarchy, AaraMinds visual identity)
+ the `pptx` skill                       (.pptx generation + the visual-QA discipline)
+ this skill                             (VP template + trend + contracts + build)
```

**Dependency fallback.** If the Executive Narrative Advisor, Module 2, or the `pptx` skill is
unavailable, do **not** stop. Apply the minimal inline core below, build with whatever pptx tooling
is present, and record each missing dependency in the gap log. The inline core (the five gates):
(1) **signal over activity** — every item shows what changed → why it matters → next action; (2)
**metric integrity** — every number gets definition + baseline + window + actual/forecast/target, or
`[VERIFY]`; (3) **risk honesty** — no euphemisms, never soften red/amber to green; (4) **the ask is
explicit** and classified; (5) **one message per slide; the title states the message, not the topic.**

**Pre-flight readiness check.** Confirm before generating and record the answers in the verification
report — a "No" doesn't block the run, it sets confidence and flags gaps: Executive Narrative Advisor
available? (else inline core) · Visual Identity (Module 2)? (else default theme) · `pptx` skill? (else
build with present tooling + note it) · Prior month's deck? (else baseline, no trend arrows) · Input
contract complete? (else flag each missing input and downgrade confidence).

## Audience calibration — optimize for AVP/VP, usable to manager

| Reader | Wants | Cut to the appendix |
|---|---|---|
| **AVP / VP** | Overall health, what changed, top risks, the decision needed, business meaning | Task-level detail, tool/ticket names, individual-contributor lists, full milestone tables |
| **Director / Senior Manager** | The above + workstream-level health and key milestones | Dependency minutiae, code/test-impacted app lists |
| **Delivery Manager** | Full traceability, every milestone, every dependency | (this is the appendix) |

Default: write the headline, health, risks, and ask at **VP altitude** (business meaning + decision);
let the appendix carry the manager-altitude detail. Same facts, pitched up. If a month is explicitly
for one audience, calibrate to it.

## The locked template (VP-optimized)

The deck is the same every month so the leader pattern-matches instantly. **Mandatory: slides 1–5.**
Slide 6 is mandatory-when-there-is-a-decision (else say "no decision needed" on slide 2). Slide 7 is
recommended. Full slide-by-slide spec in `references/monthly-deck-template.md`.

| # | Slide | Mandatory | The one thing it delivers |
|---|---|:--:|---|
| 1 | **Cover** | ✓ | Initiative · month · owner · **overall RAG** (= worst load-bearing area, never an average) |
| 2 | **Executive summary** | ✓ | Overall status · key wins · key risks · **leadership ask** · **confidence** · what changed since last month — the 30-second slide |
| 3 | **Program health dashboard** | ✓ | RAG by **dimension** (Scope · Schedule · Quality · Cost · Risk · Dependencies) with **trend arrows**, plus a per-workstream RAG roll-up |
| 4 | **Major accomplishments** | ✓ | Top 5 wins · business impact · completion evidence · planned-vs-actual (no activity dump) |
| 5 | **Risks & blockers** | ✓ | Top 3–5 risks: statement · business impact · probability · severity · mitigation · owner · open-since |
| 6 | **Leadership decisions required** | ✓ if any | Per decision: issue · impact · decision needed · due date · owner (else "no decision needed" on slide 2) |
| 7 | **Next-month outlook** | recommended | Upcoming milestones · forecasted risks · **confidence level** |
| — | **Appendix** | optional | Detailed milestones · dependency tracking · architecture diagrams · team/DORA metrics · full per-capability detail · MoM change-log detail |

Slide economy (from the narrative gates): one message per slide; **the title states the message, not
the topic** — "CPR live since Dec 9; ADI Enterprise onboarded — 2026 scope on track," not "CPR
Update." ≤3 proof points per slide except the dashboard. **Architecture context appears in the body
only when a new dependency, scope change, or platform impact occurred this month — otherwise it lives
in the appendix.**

## Three strengths to preserve as standing behaviors

Real leadership decks earn trust through these; bake them in every month:

1. **Always surface accountable owners** — for every risk, blocker, milestone, and leadership ask.
   A claim with no owner is incomplete; mark the owner `[VERIFY]` if unknown.
2. **Maintain commitment traceability** — connect planned commitment → actual accomplishment → missed
   commitment → future commitment. Slide 4 and the appendix carry this; never move a goalpost silently.
3. **Architecture only when it changed** — see the template note above.

## Translate delivery facts into leadership value

Never put a raw engineering/delivery fact on a primary slide — translate it into the business
consequence the leader cares about. The pattern: **<what was done> → <what it unblocks / protects /
saves / risks>.**

- "API integration completed" → "removes the onboarding dependency for downstream teams and protects
  the Q3 launch path."
- "Migrated 4 of 5 services to the new pipeline" → "80% of the estate is now on the supported
  pipeline; the last service is the only remaining single point of failure."
- "Closed 12 defects" → "release-blocking quality gate cleared; no Sev1/Sev2 open against the Q3 gate."

If you cannot name the business consequence, the item is activity, not an accomplishment — drop it to
the appendix or cut it.

## Deterministic month-over-month trend

Trend is **computed from last month's deck**, not asserted. Apply the fixed rules (full table in
`references/monthly-deck-template.md`):

- RAG arrow per dimension and per workstream: Red→Amber/Green = **↑ improved**; Amber→Green = **↑**;
  Green→Amber/Red = **↓ worsened**; Amber→Red = **↓**; unchanged = **→**; new = **NEW**; removed =
  **moved to appendix / noted as closed**.
- **Rename / merge / split:** if a workstream is renamed, merged, split, or removed, state the mapping
  used to compute trend. If the mapping is unclear, mark the trend `[VERIFY]` — never fake precision.
- **MoM change-log** (feeds slide 2 and a deliverable): new risks · closed risks · escalated risks ·
  completed milestones · slipped milestones.
- **First month:** build a clean baseline deck, omit arrows, and save it as next month's trend base.
  (This path is common — handle it explicitly, don't error.)

## Mandatory inputs (collect against `references/input-contract.md`)

Previous month's deck · current month's status notes · RAID log · Jira/Azure DevOps metrics ·
milestone tracker · dependency tracker · leadership asks · financial metrics. **Missing inputs are
flagged in the verification report, never invented.** Optional: an **audience profile** (role — e.g.
VP Engineering / VP Product / CFO / CIO; see modes) and the prior **status ledger** (historical mode).

## Deliverables (every run returns all of these)

1. **The `.pptx`** — 5–7 primary slides + appendix, AaraMinds visual identity, dated filename.
2. **Executive summary** — a one-page AVP/VP narrative (the slide-2 content as prose).
3. **Evidence report** — every load-bearing claim traced to its source (which input it came from).
4. **Verification report** — every `[VERIFY]` item and every gap requiring manual review.
5. **Month-over-month change summary** — the change-log above.
6. **Optional:** speaker notes / Q&A prep (the likely leadership questions + answers).
7. **Status ledger** (when historical-intelligence mode is on) — the updated multi-month memory, for
   next month.

## Modes & extensions (opt-in — the default is one program, one deck, one month)

The single-program monthly deck is the default and stays thin. These layer on when the situation calls
for them; each reuses every gate, threshold, and trend rule:

- **Portfolio roll-up** (`references/portfolio-rollup.md`) — for an AVP/VP who owns many programs: a
  portfolio summary (Green/Amber/Red counts), program health matrix, top enterprise risks, top
  decisions, and cross-program dependencies, with each program's full deck in the appendix. Portfolio
  RAG = the worst load-bearing *program*, never an average.
- **Historical intelligence** (`references/historical-intelligence.md`) — a persistent status ledger
  carried month to month, surfacing aging risks, chronic dimensions, systemic dependencies, and
  commitment erosion. Turns reporting into intelligence ("blip vs pattern").
- **Audience profiles** (`references/audience-profiles.md`) — re-weights emphasis/ordering for a role
  (VP Eng / VP Product / CFO / CIO / Board). Changes the lens, never the facts or the RAG.

## Production flow

1. **Collect inputs** against the input contract; flag gaps (don't invent).
2. **Run the narrative gates** (Executive Narrative Advisor, or the inline core): activity → signal;
   classify the ask; validate or `[VERIFY]` every metric; de-euphemize risks; attach owners.
3. **Compute the trend** from last month's deck using the deterministic rules; build the change-log.
4. **Map content into the locked template** (slides 1–5 always; 6 if a decision exists; 7 recommended).
5. **Build the `.pptx`** via `references/pptx-build-recipe.md`.
6. **Run the mandatory visual-QA pass** — render slides to images and have a **fresh subagent**
   inspect for overflow, overlap, off-canvas shapes, low contrast, and leftover placeholder text;
   fix and re-verify at least once before declaring done. (python-pptx autofit does **not** guarantee
   text fits — never trust it silently.)
7. **Emit all deliverables** and run the verify checklist.

## Anti-patterns (deck-specific; narrative anti-patterns also apply)

- **Activity dump** — dense per-initiative tables with no business meaning, pitched at working
  altitude. The #1 real-world failure: a thorough PMO deck that a VP can't read in 60 seconds. Fix:
  top-5 wins with impact; detail to appendix.
- **No executive narrative** — opening with title → programs → architecture → detail, with no
  answer-first summary. Slide 2 exists to prevent this.
- **No health-at-a-glance** — percentages everywhere but no dimensional RAG. Leaders consume color
  faster than numbers.
- **Risks buried in delivery detail** — risks scattered across slides instead of a top-3–5 risk slide.
- **Buried or missing ask** — no clear "what I need from you." It goes on slide 2 and slide 6.
- **Watermelon status** — green cover hiding amber/red. Cover RAG = the worst load-bearing area.
- **Snapshot amnesia** — ignoring last month, so slippage and aged risks vanish.
- **Template drift** — reordering/renaming sections month to month.
- **Topic titles** — labeling the slide instead of stating its message.

## Confidentiality

Leadership decks carry financials, org names, internal system names, and risk detail — often
"Internal Use Only / Proprietary." Apply:

- Do not move or expose source files outside the approved workspace; build only in the user's selected
  folder.
- Redact or pseudonymize sensitive names (people, vendors, customers) on request.
- Never reproduce confidential project detail in generated *examples*, evals, or documentation — use
  generic placeholders.
- Carry a confidentiality marker on the cover/footer when the inputs are marked internal/confidential.

## Verify before delivering

- Can a leader answer all **five success questions in 60 seconds**?
- All **five mandatory slides** present (cover, executive summary, health dashboard, accomplishments,
  risks)? Decisions slide present if any decision exists?
- Overall RAG = the worst load-bearing area (no watermelon)?
- Dimensional health (Scope/Schedule/Quality/Cost/Risk/Dependencies) shown with **trend arrows**
  computed from last month?
- Top 3–5 risks only, each with statement/impact/probability/severity/mitigation/owner/open-since?
- Every accomplishment tied to planned-vs-actual; top 5 only; business impact stated?
- Ask classified and on slide 2 (or "no ask" stated there) and detailed on slide 6?
- Every metric defined + sourced or `[VERIFY]`; every risk/blocker/milestone has an **owner**?
- One message per slide; titles state the message; ≤3 proof points (except dashboard)?
- Architecture in the body only if it changed this month?
- RAG applied with the published thresholds (default or PMO-overridden) and the threshold key shown on
  the dashboard? Confidence scored against the High/Medium/Low criteria?
- **Visual-QA pass run** (render-to-image, fresh-subagent inspection, ≥1 fix-verify cycle)?
- All **six deliverables** emitted (pptx, exec summary, evidence report, verification report, MoM
  change summary, optional Q&A)?

## Read next

Paths relative to the workspace root (the folder containing `instruction-os/`).

- `references/monthly-deck-template.md` — slide-by-slide spec, the deterministic trend table, risk and
  decision field contracts, the dimensional health dashboard.
- `references/input-contract.md` — the monthly intake form, deliverables, and the 60-second success test.
- `references/pptx-build-recipe.md` — the build, the mandatory visual-QA pass, overflow guard, default theme.
- `references/evals.md` — the eval scenarios and the pass/fail bar.
- `references/portfolio-rollup.md` · `references/historical-intelligence.md` ·
  `references/audience-profiles.md` — the opt-in modes (portfolio, multi-month memory, role profiles).
- Judgment engine: `instruction-os/skills/aaraminds-executive-narrative-advisor/SKILL.md` and its source
  `instruction-os/Persona/AaraMinds_Executive_Narrative_Advisor_v1.0.md`.
- Visual identity: `instruction-os/Persona/02_Visual_Identity_System_v1.1.md`.

## Maintenance

Narrative gates mirror `aaraminds-executive-narrative-advisor` as of 2026-06-17 (canonical for
judgment). The VP-optimized template, deterministic trend rules, deliverables contract, and visual-QA
pass were added in v1.1 (2026-06-17) from a real-deck review + open-source benchmark
(Anthropic `pptx`/`internal-comms`, `frontend-slides`, Pyramid/BLUF/RAG best practice). Only the
template, trend, and build mechanics need independent upkeep.
