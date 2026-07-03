# Portfolio roll-up mode (opt-in)

The default skill produces **one program → one deck → one month.** This mode handles the AVP/VP who
owns a *portfolio* — many programs, workstreams, and directors — and needs one roll-up, not ten decks.

> Scope note: portfolio mode is a documented extension of this skill today. If it grows its own
> template depth, charts, and drill-down tooling, promote it to a sibling skill
> (`aaraminds-portfolio-status-deck`) rather than bloating this one. For now it reuses every gate,
> threshold, and trend rule below — only the altitude changes.

## When to use

The user owns/oversees multiple programs and asks for a "portfolio update," "all my programs,"
"director/AVP roll-up," or supplies more than one program's inputs.

## The same principle, scaled

**Portfolio overall RAG = the worst load-bearing *program*, never an average.** The no-watermelon
rule applies one level up: a portfolio with one Red program is not Green because seven others are.

## Portfolio template

| # | Slide | The one thing it delivers |
|---|---|---|
| 1 | **Portfolio cover** | Portfolio name · month · owner · overall RAG (= worst load-bearing program) |
| 2 | **Portfolio summary** | Counts — **Green N · Amber N · Red N** across programs · the single portfolio-level headline · the portfolio-level ask · confidence |
| 3 | **Program health matrix** | One row per program: RAG · trend arrow · one-line why · owner (drill-down ref to its full deck) |
| 4 | **Top enterprise risks** | Top 3–5 risks *across* programs (de-duplicated, severity-ranked) with owner + the program(s) they hit |
| 5 | **Top decisions needed** | The cross-portfolio decisions, ranked by what they unblock · owner · due date |
| 6 | **Cross-program dependencies** | Dependencies that span programs (the systemic ones) · blocking which committed milestones · owner |
| — | **Appendix** | Each program's full single-program deck, appended; portfolio-level financials; the full risk/dependency registers |

## How it composes

- Each program contributes its **single-program deck's executive summary + health dashboard** — built
  with the normal skill — which roll up into the matrix (slide 3) and append in full (appendix).
- Risks and dependencies are **de-duplicated and ranked across programs**; the same risk hitting three
  programs becomes one enterprise risk noting all three (this is signal, not triple-counting).
- Trend arrows on the matrix use the same deterministic rules, computed per program vs last month.
- Cross-program dependencies (slide 6) are the portfolio's highest-leverage view — surface the ones
  that block a committed milestone in another program.

## Verify (portfolio)

- Portfolio RAG = worst load-bearing program (no averaging)?
- Green/Amber/Red counts reconcile with the program matrix?
- Top enterprise risks de-duplicated across programs, each with owner + affected programs?
- Cross-program dependencies surfaced, not buried in per-program detail?
- Every program's full deck reachable in the appendix (drill-down preserved)?
