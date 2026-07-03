---
name: copilot-cost-optimization
description: Analyze enterprise GitHub Copilot usage + billing data and recommend cost optimizations under Copilot's usage-based AI-Credit model (token-derived; premium-requests are the legacy mode). Use when reviewing org/enterprise Copilot spend, right-sizing models and seats, setting budgets/ULBs, or turning a Copilot metrics + metered-usage dashboard into a cost-optimization verdict. Covers the AI-Credit cost model, the metrics-vs-billing data-surface split, the optimization levers, and a FinOps recommendation framework. Not for Azure cloud cost (azure-microservices-cost-review) or prompt work (prompt-engineering). Figures are version-sensitive — mark [VERIFY]; never fabricate consumption.
version: 1.1.0
last_updated: 2026-06-18
---

# Copilot Cost Optimization

FinOps for GitHub Copilot at org/enterprise scale: turn usage + billing data into a ranked set of
cost-optimization recommendations with a defensible verdict. Sibling to `azure-microservices-cost-review`
(Azure FinOps) — same discipline, different cost surface.

## When to use

Trigger when the task is to **review org/enterprise GitHub Copilot spend and recommend optimizations**:
"our Copilot bill is too high," "optimize Copilot cost," "right-size Copilot models/seats," "set Copilot
budgets," "turn our Copilot usage/billing dashboard into recommendations." Org/enterprise FinOps for
Copilot.

Do **not** use for: Azure cloud cost (`azure-microservices-cost-review`); prompt-level token reduction
(`prompt-engineering`); designing the agent that *runs* this review (`agent-engineering`); or analyzing
direct Claude/OpenAI **API** token bills (this is Copilot-specific — though the levers rhyme).

## The critical fact that reframes everything (verify at use time)

**Since 2026-06-01, Copilot bills usage in token-derived "AI Credits" (1 credit = $0.01), not "premium
requests."** Premium requests survive only as a **legacy** path for annual Pro/Pro+ holdouts. So the
cost driver is now **per-token model rates × tokens consumed**, converted to credits — i.e. **model
choice and interaction volume**, not a flat per-request count. Build recommendations around credits/$;
treat premium-request multipliers as a sunsetting compatibility mode. (Full model in
`references/cost-model-and-data.md`.) **Code completions + next-edit suggestions stay unlimited/free** —
the metered surface is Chat, CLI, cloud/coding agent, Spaces, Spark, and code review.

## The decision rule — read cost from data, never invent it

Every consumption figure (credits, $, tokens, seat counts) comes from the org's **billing / metered
usage report**, not from the model's head. A number with no source line is marked `[VERIFY]`. And every
*rate* (per-token prices, flex allotments, promo credits) is version-sensitive and must be confirmed
against current GitHub docs at review time — they change frequently. Fabricated savings do not ship.

## The two data surfaces — do not conflate them (`references/cost-model-and-data.md`)

Cost and productivity live in **different systems**; the value is in joining them:

| Surface | Has | Lacks | Access |
|---|---|---|---|
| **Copilot Metrics API / usage dashboard** | active/engaged users, acceptance, lines (user- vs agent-initiated), model usage by chat mode, per-user `used_*` | **no cost / credits / tokens** | org admin / enterprise; ≥5-active-user privacy gate; ~2-day latency |
| **Billing / metered usage report** | AI-credit ($) consumption by **user / model / SKU / cost-center** | productivity/acceptance | **per-user cost attribution generally needs *enterprise* access** — org owners often can't see by-user |
| **Local session telemetry** (e.g. `copilot-token-budget` MCP) | per-developer **raw tokens + credits + $** + instruction-overhead, **zero-network, no enterprise access** | only machines running the reader | the strongest surface for an org's own endpoints; carries the tool's caveats (UTC-boundary drift, unvalidated forecast, Go↔TS drift) — see references |

So: cost-per-model/user from billing; "is that spend producing accepted code?" from metrics. The agent
must state which surface each number came from, and flag when per-user attribution isn't available at
the caller's access level.

## Optimization levers (ranked; full detail in `references/optimization-and-recommendations.md`)

1. **Model right-sizing — the #1 lever.** Cost scales with per-token rate, so frontier models (Opus,
   GPT-5.5-class) cost multiples of lightweight ones (mini/Haiku/Flash-class). Route routine work to
   cheap/included models; reserve frontier models for hard reasoning; set a cheaper **default model**
   and **disable** premium models via the Models policy where appropriate.
2. **Enable auto model selection** — a standing ~10% model-cost discount in Chat/CLI/cloud agent. Free.
3. **Agent-mode discipline** — agentic sessions make many model calls across files and are the dominant
   cost driver; scope tasks tightly, prefer chat for quick questions.
4. **Exploit caching** — cached input ≈10× cheaper than fresh; reuse stable context (watch Anthropic
   cache-*write* cost).
5. **Manage Copilot code review** — burns credits **and** Actions minutes with an undisclosed model;
   apply automatic-review policies judiciously.
6. **Reclaim inactive seats; right-size the pool** — credits **pool** at the billing entity, so the
   question is total pool vs total consumption, not per-seat fit. Join metrics `last_activity_at`/`used_*`
   (reclaim idle seats) with billing per-user credits (cap power users).
7. **Budget guardrails** — universal **user-level budget (ULB)** as a hard cap + an enterprise spending
   limit with "stop usage" to prevent runaway agent spend; cost-center budgets for chargeback. (Note:
   exhausting credits no longer falls back to a cheaper model — it meters or **blocks**.)
8. **Instruction-overhead reduction** — always-loaded `.github/instructions`/context adds tokens to every
   message (~12K/msg seen in the field); trim/scope the highest-`reducibleTokens` files. Low-friction
   (no admin policy — dev/repo owner). Surfaced ready-made by `copilot-token-budget`'s `get_instruction_overhead`.

## The review pass

1. **Scope + access.** Org or enterprise? Which surfaces can the caller actually read (per-user cost
   needs enterprise)? Note gaps as findings.
2. **Establish the bill.** Total credits/$ this period, by model · by SKU (Copilot / cloud agent / Spark
   tracked separately) · by user/cost-center where visible. All sourced.
3. **Cross-reference productivity.** For the top spend models/users, pull acceptance + agent-contribution
   from metrics — is the spend producing accepted code?
4. **Apply the levers** in rank order; quantify each recommendation against the sourced baseline (or mark
   `[VERIFY]` if the figure to compute savings isn't available).
5. **Verdict + ranked backlog.** A FinOps verdict (Healthy / Optimizable / Overspending) with the top
   recommendations by $ impact, each with the lever, the owner control (policy/budget/seat), and effort.

## Time-sensitive watch-items (state these in any review)

- **Promo credit cliff: org/enterprise included credits drop on 2026-09-01** (Business 3,000→1,900,
  Enterprise 7,000→3,900 per user `[VERIFY]`). Budgets calibrated in summer under-provision in autumn.
- **Flex allotments + per-token rates + the model list change frequently** — re-verify every review.
- **Legacy premium-request multipliers** apply only to the shrinking annual cohort — treat as sunsetting.

## Worked example (brownfield — an existing, in-production Copilot estate)

A 200-seat Copilot Business org's monthly metered report shows **$4,100 in AI credits**, of which
**$2,600 (63%) is Claude Opus 4.x** usage concentrated in agent mode. The Metrics API shows Opus
agent-initiated lines have an acceptance rate of ~22% vs ~48% for the included/mid models. Cross-
referencing: the org is paying frontier rates for agent work that is largely *not* being accepted.
Verdict: **Overspending.** Top recommendation — set a cheaper default model and restrict Opus via the
Models policy to a named senior cohort (enterprise-owner control), est. **~$1,500/mo** [VERIFY tokens];
#2 — enable auto model selection (~10% on eligible spend); #3 — universal ULB hard cap to stop runaway
agent sessions; plus a watch-item that the org's included pool **drops on 2026-09-01** (promo cliff), so
re-budget before then. Every figure is sourced to the metered report; the Opus token split to compute
exact savings is `[VERIFY]` pending the per-model token export.

## Anti-patterns

- **Fabricated consumption / savings** — quoting credit or $ figures not read from the billing report.
- **Stale rates** — using last quarter's per-token prices or allowances without re-verifying.
- **Surface conflation** — citing the Metrics API for cost (it has none) or billing for acceptance.
- **Per-request thinking** — optimizing "premium requests" when the org is on the AI-Credit model.
- **Ignoring the access asymmetry** — promising per-user cost attribution the caller can't actually pull.
- **Savings without the control** — a recommendation with no owner lever (policy/budget/seat) to enact it.

## Verification questions

1. Is every consumption figure sourced to the billing/metered report, with `[VERIFY]` on anything else?
2. Are rates/allowances confirmed against current docs (not memory), and flagged as version-sensitive?
3. Is the metrics-vs-billing surface stated for each number, and the access level's limits noted?
4. Are recommendations ranked by $ impact, each tied to a concrete admin control to enact it?
5. Are the promo-cliff and rate-volatility watch-items called out?
6. Is model right-sizing (the #1 lever) evaluated against actual acceptance, not assumed?

## What to read next

- `references/cost-model-and-data.md` — the AI-Credit model, per-token rate shape, allowances/pooling,
  the two data surfaces + access matrix, legacy premium-request mode.
- `references/optimization-and-recommendations.md` — each lever in depth, the admin controls (policies,
  the 4 budget levels), the recommendation/verdict output template.
