# Validation Pack

Evidence that the v10.0 pack works at the quality it claims. Two concerns, two subdirectories:

| Subdirectory | What it contains | When you use it |
|---|---|---|
| [prompts/](prompts/) | Twelve curated capability prompts spanning MCP server building, microservices design, architecture review, and cross-cutting concerns. Each prompt ships with a rubric (objective checklist) and a reference output (exemplar). | Quarterly quality demonstration; before tagging a release; when teaching a new contributor what "good" looks like. |
| [governance/](governance/) | Freshness cadence with named ownership and a pre-release checklist that ties every artifact together. | Quarterly (re-verify Go SDK / Azure / ecosystem facts). Before each tagged release. |

## What changed from v9.0

v9.0 shipped 34 per-skill evals alongside the 12 capability prompts. The per-skill evals were never run (each marked `last_run: never`), drifted from the actual SKILL.md content during the v9.0 → v10.0 migration, and duplicated coverage the capability prompts already provided at the workflow level. Phase 4 of v10.0 trims them to capability prompts only.

If you want per-skill coverage in the future, the right place is each SKILL.md's "Verification questions" section — that's where skill-specific checks live now. Per-skill *evals* (separate files with rubrics) re-introduce drift; verification questions next to the skill content don't.

## Why this exists

This pack makes quality claims provable. Without the validation pack, a change to a load-bearing skill can silently regress without anyone noticing until a user files an issue. With it, every capability area has a small, runnable check that a maintainer can use to catch drift before it ships.

## Validation model: rubric + reference

Goldens cannot be byte-exact for LLM-driven prompts (LLMs are stochastic). Instead, every prompt has two complementary artifacts:

- **Rubric** — a checklist of objective points the response must cover. Each point is yes/no. The pass criterion (e.g., "at least 7 of 9 rubric points") is declared in the prompt.
- **Reference output** — a hand-curated exemplar showing the *shape and depth* of a quality response. This is not a byte-exact target; it is a "good answer looks like this" reference for human reviewers.

A run "passes" when the response satisfies the declared rubric threshold. The reference output is for inspection — does the actual response look like the same quality bar?

This split is deliberate. Rubrics make the check objective and machine-checkable. References keep the quality bar visible and human-meaningful.

## How to run it

This pack provides the **artifacts**. The user provides the **runtime** (Claude Code, an Anthropic SDK script, OpenAI API, anything that takes a prompt and produces a response).

### Manual flow

1. Pick a capability prompt under `prompts/`.
2. Paste the prompt section into your LLM with the relevant skill file(s) attached as context (the prompt's front-matter `exercises:` lists which skills).
3. Score the response against the rubric.
4. Compare against the reference output for shape and depth.
5. If rubric threshold is met, log the run in the prompt's `last_run` field (date + result). Otherwise, file an issue against the skill or the prompt.

### Automated flow (optional)

You can wire a runner script that:

1. Parses a prompt's rubric.
2. Calls your LLM with the linked skill files in context.
3. Scores the response against the rubric (each point: scan the response for the keyphrase or pattern; or use an LLM-as-judge call).
4. Emits PASS / FAIL with per-point breakdown.

This pack does not ship the runner script — the right shape depends on which LLM you use and how you want to integrate it (CI step, pre-commit hook, ad-hoc CLI). The prompt file format is intentionally parseable so a ~100-line script can drive it.

## Coverage at v10.0

| Area | Files | What's covered |
|---|---|---|
| Capability prompts | 12 | 3 MCP-server-building, 4 microservices-design, 3 architecture-review, 2 cross-cutting |
| Governance | 2 | Freshness cadence (filled), pre-release checklist |

Total: 14 files. The Phase 4 trim cut the validation surface roughly in half by dropping the per-skill evals — what remained is the smallest set of artifacts that, if all pass, gives high confidence the pack works at quality.

## What this pack is NOT

- It is not an LLM benchmark. The rubrics are pack-specific, not general capability measurements.
- It is not a substitute for human judgment. The reference outputs are exemplars, not ceilings.
- It is not exhaustive coverage. It is the smallest set of artifacts that proves the workflows the pack claims to support.

## Related

- [governance/freshness-cadence.md](governance/freshness-cadence.md) — who owns each refresh
- [governance/release-checklist.md](governance/release-checklist.md) — pre-release runbook tying every artifact together
- [../ROADMAP.md](../ROADMAP.md) — Phase 4 entry
