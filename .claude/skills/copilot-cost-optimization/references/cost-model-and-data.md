# Copilot cost model + data surfaces (verify all figures at use time)

> Every number here is **version-sensitive** and was current ~2026-06-18. GitHub changes flex
> allotments, per-token rates, the model list, and promo credits frequently. Re-confirm against
> docs.github.com before quoting any figure; mark unconfirmed ones `[VERIFY]`.

## The AI-Credit model (default since 2026-06-01)

- Interactions consume **tokens** (input + output + cached, plus a cache-*write* cost for Anthropic
  models), priced per model, converted to **AI Credits** where **1 credit = $0.01 USD**.
- Cost = (model per-token rate) × (tokens). No flat "request" unit in the new model.
- **Billed:** Chat, CLI, cloud/coding agent, Spaces, Spark, third-party coding agents, code review.
- **NOT billed (unlimited, all paid plans):** code completions + next-edit suggestions.

### Included monthly allowances (base + flex; flex is the most volatile figure)
| Plan | Price | Total monthly credits |
|---|---|---|
| Pro | $10 | 1,500 (1,000 base + 500 flex) |
| Pro+ | $39 | 7,000 (3,900 + 3,100) |
| Max | $100 | 20,000 (10,000 + 10,000) |
| Business | $19/user | 1,900/user (pooled) |
| Enterprise | $39/user | 3,900/user (pooled) |
| Free | $0 | "an allowance" + 2,000 completions/mo — exact credits not published `[VERIFY]` |

- **Promo (2026-06-01 → 2026-09-01 only):** Business 3,000/user, Enterprise 7,000/user, then revert to
  1,900 / 3,900 `[VERIFY]`. **A scheduled cliff** — call it out in any summer review.
- **Pooling:** org/enterprise credits pool at the billing entity (100 Business seats = 190,000 shared
  credits). Adding seats grows the pool immediately; removing shrinks it next cycle. → right-size the
  **pool**, not the seat.

### Per-token rate shape (per 1M tokens; illustrative, re-verify)
- Lightweight/cheap: GPT-5 mini ~$0.25 in / $2 out · Haiku 4.5 ~$1/$5 · Gemini 3 Flash ~$0.50/$3.
- Mid: Claude Sonnet 4.x ~$3/$15 (cached $0.30, cache-write $3.75) · Gemini 3.1 Pro ~$2/$12.
- Frontier (expensive): Claude Opus 4.x ~$5/$25 (cached $0.50, cache-write $6.25) · GPT-5.5 ~$5/$30.
- **Auto model selection → ~10% model-cost discount** in Chat/CLI/cloud agent (free lever).
- Takeaway: frontier ≈ 5–10× lightweight per token → **model choice dominates cost**.

### Overage / blocking
- Beyond included credits: billed at published per-token rates (1 credit = $0.01).
- **No cheaper-model fallback anymore** — when credits/budget exhaust, usage is metered (if the
  "AI-credit paid usage" policy is on) or **blocked**.
- Code review additionally consumes **Actions minutes** with an **undisclosed** model (cost not
  predictable from rate tables). iOS/Android subscribers can't buy extra credits.

## Legacy premium-request mode (sunsetting; annual Pro/Pro+ holdouts only)
1 PRU × model multiplier per interaction. Allowances: Pro 300/mo, Pro+ 1,500/mo; extra $0.04/request;
reset 1st of month UTC; no rollover. Multipliers rose 2026-06-01 (e.g. Opus 4.x 15–27×, Sonnet 4.x
1–9×, code review 13×). Treat as compatibility only — this cohort auto-downgrades to Free at plan
expiry. (`references` only; new orgs are on AI Credits.)

## The two data surfaces (the core architecture)

### A. Copilot Metrics API / usage dashboard — productivity, NO cost
- Fields: daily/weekly/monthly active + engaged users; completion suggested/accepted + acceptance rate;
  lines added/deleted (user- vs agent-initiated, per model, per language); model usage by chat mode
  (`totals_by_model_feature`, `totals_by_language_model`); requests per mode (ask/edit/plan/agent);
  per-user `used_chat`/`used_agent`/`used_cli`/`used_copilot_code_review_active`, `last_activity_at`,
  `ai_adoption_phase`. Exported NDJSON.
- Access: enterprise/org dashboards; API supports enterprise/org/user records. Roles: enterprise owner,
  org admin, billing manager, or a custom role with "View Enterprise Copilot Metrics."
- Constraints: **≥5 active-licensed users/day** privacy gate (no data below it); **~2-day latency**
  (`last_activity_at` faster); ~90-day retention.
- **Has ZERO cost/credit/token fields.**

### B. Billing / metered usage report — cost, NO productivity
- AI-credit ($) consumption grouped by **SKU / plan / user / model / organization / cost-center**;
  filter by `cost_center:`; export. SKUs split (Copilot, Spark, cloud agent separate since 2025-11-01).
- **Access asymmetry:** enterprise owners + billing managers see **per-user** cost; **org owners often
  cannot** (the `user` API param is blocked at org scope) → per-user cost attribution generally needs
  **enterprise** access or a downloaded report. State this limit in any org-scoped review.

### C. Local session telemetry (e.g. `copilot-token-budget`) — per-developer tokens + credits, zero-network
The strongest surface **for an org's own machines**, and the one that fills Surfaces A/B's gaps. Copilot
writes local session files (`~/.copilot/session-state/{uuid}/events.jsonl`; VS Code Chat transcripts);
a reader like `copilot-token-budget` computes per-session cost from the authoritative
`session.shutdown.data.totalNanoAiu` field, broken into all 5 token types (input/output/cache-read/
cache-write/reasoning), and exposes it via an **MCP server** (6 tools — see
`optimization-and-recommendations.md`).
- **Has what A/B lack:** per-developer **raw tokens** *and* credits *and* $, plus **instruction-overhead**
  (always-loaded tokens/msg with a pre-computed reducible amount), with **no enterprise access and no API
  call**. Resolves the per-user-cost access asymmetry for the org's own endpoints.
- **Caveats (from the tool's own critical review — carry as `[VERIFY]`):** local-vs-UTC time drift near
  day/month boundaries; forecast accuracy **unvalidated** (treat forecasts as directional); Go-core ↔
  TS-extension drift (prefer the Go core / MCP surface and cite which one). Coverage is only machines
  running the reader (not the whole org unless deployed widely).

## Confirmed gaps (for Surfaces A/B; Surface C closes most of them locally)
- **No per-developer raw token counts** anywhere developer-facing — the exposed unit is **credits/$**,
  not tokens. (`[VERIFY]` whether any export field exposes underlying tokens.)
- Org owners can't pull per-user cost analytics via API.
- Code-review model undisclosed → per-review cost unpredictable.
- <5-user teams get no Metrics API data at all.
