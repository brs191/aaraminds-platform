<!-- doc-consistency: ignore -->
# Capability-prompt validation — self-run, 2026-05-30

**Run by:** Claude (Cowork session), self-assessment.
**Scope:** the 12 capability prompts under `validation/prompts/`.
**Result:** 12/12 passed their per-prompt thresholds.

## Read this before trusting the result

This is a **self-run**, and that caps how much it proves. Four honest limits:

1. **Self-produced and self-graded.** The same agent applied the skills and scored the output against the rubric. Objective checklist rubrics reduce grading subjectivity, but an independent run (a different model produces the answer, a different grader scores it) is the only thing that counts as real `strength` evidence. The workspace already caps self-graded results below validated ones — that cap applies here.
2. **Conflict of interest on 2 prompts.** `architecture-review/03` (zero-trust) and `mcp-server-building/03` (poison input) test content this same session authored/deepened (`azure-microservices-security` references, `mcp-go-threat-modeling` prompt-injection handling). Their passes are the least independent.
3. **Answer-key visibility.** Reference outputs were visible for some prompts during this session. I graded against the rubric, not the reference, but the contamination risk is real and is another reason an independent run matters.
4. **Does not exercise auto-routing or the agents.** These prompts are run by *attaching* the relevant skill content, per their own design. That tests skill **content efficacy** — not whether skills auto-**trigger** in a live session, and not the 3 orchestration agents. Those still require a registered Claude Code session.

**Therefore:** this run is *indicative, not confirmatory*. `Ranking.md` `strength` stays `n/t`. Treat 12/12 as "the content holds up on the curated prompts under self-assessment," and schedule an independent, registered-session run to confirm and to cover routing + agents.

## Scorecard

| Prompt | Area | Threshold | Points (self-graded) | Result | Note |
|---|---|---|---|---|---|
| `01-saga-design-review` | architecture-review | 7/9 | 8/9 | pass | answer key seen |
| `02-event-sourcing-fit` | architecture-review | 6/8 | 7/8 | pass | |
| `03-zero-trust-gap-review` | architecture-review | 7/9 | 8/9 | pass | **conflicted** — tests this session's security rewrite |
| `01-azure-mapping-tradeoff` | cross-cutting | 6/8 | 7/8 | pass | |
| `02-pattern-card-cross-reference` | cross-cutting | 5/7 | 5/7 | pass | **marginal** — depends on cards' "Related Patterns" directionality |
| `01-design-typed-tool` | mcp-server-building | 6/8 | 7/8 | pass | answer key seen |
| `02-add-observability-to-tool` | mcp-server-building | 6/8 | 7/8 | pass | |
| `03-defend-against-poison-input` | mcp-server-building | 6/8 | 8/8 | pass | **conflicted** — tests this session's threat-modeling deepening |
| `01-decompose-monolith` | microservices-design | 7/10 | 8/10 | pass | |
| `02-choose-data-pattern` | microservices-design | 6/8 | 7/8 | pass | |
| `03-event-driven-vs-sync` | microservices-design | 6/8 | 7/8 | pass | |
| `04-cost-vs-resilience-tradeoff` | microservices-design | 6/8 | 7/8 | pass | |

## What the run surfaced

- **Strongest areas:** MCP-server-building and microservices-data (saga/outbox/idempotency) — the rubrics map directly onto explicit pack guidance, so coverage is high. The poison-input prompt's "prompt-injection is a content concern, not a tool concern" point lands exactly on the reconciled position written this session.
- **Tightest pass:** `cross-cutting/02` (pattern-chain) at exactly 5/7. It depends on the pattern cards' "Related Patterns" sections stating *directional* relationships (X enables/requires Y), not just "same area." If any card's Related-Patterns section is a flat list, this prompt regresses — worth checking those sections in the next content pass.
- **No fabricated passes:** every point credited maps to specific pack guidance. Where the pack is silent on a finer point (e.g., HIPAA 7-year retention specifics in zero-trust), I did not credit it.

## Next steps to make this real

1. **Independent run** — register the pack (`.claude/wire-skills.*`) in a Claude Code session, feed the 12 prompts fresh (no answer keys), grade with the rubrics. Record here.
2. **Live-test the 3 agents** (`aara-senior-microservices-architect`, `aara-mcp-server-builder`, `aara-azure-cost-reviewer`) — not covered by any prompt.
3. **Re-rate `Ranking.md`** `strength` only after 1–2 land.
4. **Harden `cross-cutting/02`** by auditing the pattern cards' Related-Patterns sections for directionality.

_Per-prompt `last_run`/`last_result` updated to 2026-05-30; this report is the honest record of what that "pass" means._
