# Optimization levers, admin controls, and the recommendation output

## The levers, in depth (rank by $ impact against the sourced bill)

1. **Model right-sizing (#1).** Because cost = per-token rate × tokens, the biggest win is routing work
   to the cheapest model that meets the bar. From the billing report, find the top spend models and the
   modes/users driving them; cross-check acceptance/agent-contribution from metrics. If a frontier model
   (Opus/GPT-5.5-class) is doing routine completion-style work, recommend a cheaper default and/or
   disabling the premium model via the **Models policy**. Quantify: (current model $ for that workload) −
   (cheaper model $ at same tokens). Mark `[VERIFY]` if token volume per workload isn't available.
2. **Auto model selection.** Standing ~10% model-cost discount in Chat/CLI/cloud agent — enable
   org-wide; near-zero downside. Quantify as ~10% of eligible spend.
3. **Agent-mode discipline.** Agentic sessions fan out many model calls; they're the dominant driver.
   Recommend scoping agent tasks, preferring chat for quick Qs, and watching agent vs user lines (metrics
   `*_initiated`) against agent spend (billing).
4. **Caching.** Cached input ≈10× cheaper; stable, reused context lowers cost. Watch Anthropic
   cache-*write* surcharge — churny context can cost more.
5. **Code review.** Burns credits + Actions minutes (undisclosed model). Tune automatic-review policies
   to high-value paths rather than every PR.
6. **Seat + pool right-sizing.** Join metrics `last_activity_at`/`used_*` (reclaim seats idle ≥ N days →
   removes fee + pool contribution next cycle) with billing per-user credits. Because credits **pool**,
   evaluate **total pool vs total consumption**; add seats to grow the pool only when consumption is
   healthy *and* producing accepted code.
7. **Budget guardrails.** Set a universal **ULB** as a hard cap (prevents one runaway agent session from
   draining the pool), grant individual ULB exceptions to genuine power users, and set an enterprise
   spending limit with **"stop usage"** to cap the bill. Use **cost-center budgets** for chargeback.

8. **Instruction-overhead reduction (token-level lever — often the fastest win).** `.github/instructions`
   and always-loaded context add tokens to **every** message (observed ~12K tokens/msg in one org). This
   is invisible in Surfaces A/B but exposed by Surface C's `get_instruction_overhead`, which pre-computes
   `reducibleTokens` and `potentialCreditsPerSession` per file. Recommend trimming/scoping the highest-
   `reducibleTokens` files. **Control:** edit/scope the instruction files (dev/repo owner) — no admin
   policy needed, so it's low-friction. Quantify directly from the tool's `potentialCreditsPerSession ×
   sessions`.

## Consuming the `copilot-token-budget` MCP server (the data wiring)

When the org runs `copilot-token-budget`, take inputs from its 6 MCP tools instead of a manual export:

| Tool | Returns (real fields) | Feeds |
|---|---|---|
| `get_budget_status` | credits, pct, allowance, status, daysLeft, forecast, premiumRequests | establish the bill / burn |
| `get_model_costs` | per-model: input/outputRatePer1M, totalCreditsThisMonth, sessionCount, cache/reasoning tokens | model right-sizing (#1) |
| `get_top_consumers` | topSessions/topModels/topProjects: name, credits, in/out tokens, model | seat/pool + power users |
| `get_instruction_overhead` | alwaysLoadedTokens, reducibleTokens, potentialCreditsPerSession, opportunities[] | instruction-overhead lever (#8) |
| `get_usage_timeseries` | buckets: key, credits, sessions, in/out tokens | trend / anomaly / forecast |
| `get_sessions` | sessions: name, model, credits, contextTokens, isActive | drill-down |

**Carry the tool's caveats** in the review: forecasts are directional (unvalidated); period-boundary
figures may slip with local-vs-UTC drift → mark boundary numbers `[VERIFY]`; if the CLI and VS Code
extension disagree, cite the Go-core/MCP surface (better-tested, UTC-correct). Coverage = machines
running the reader only.

## Admin controls (the levers the recommendations must point to)

- **Policies** (org → Copilot → Policies/Models, or enterprise): Feature, Privacy (Allow/Block), and
  **Models** policies. Models policy = the control to **restrict/disable** expensive frontier models or
  set what's available. Enterprise-level policy overrides org-level. Granular per-org enablement exists
  (notably the cloud-agent policy).
- **Default model** management — set a cheaper default for all users; choose per-model whether it's
  auto-enabled or opt-in per org.
- **Budgets (4 levels, check order ULB → pool → cost-center/org/enterprise):**
  - **User-level (ULB)** — always active, hard stop; $0 = block. The per-user cap.
  - **Cost-center** — caps a group's metered charges (post-pool); hard stop only if "stop usage" set.
  - **Organization** — org owners' only lever; can only restrict below enterprise; metered-phase only.
  - **Enterprise** — caps total **metered** charges (overage only, not license fees); metered-phase only.
- **"AI-credit paid usage" policy** — the master switch: off = block at pool exhaustion; on = meter at
  published rates.
- **Seat management** — assign/unassign licenses (largest single lever; effect next cycle).

Every recommendation names the **specific control** that enacts it (which policy / which budget level /
seat action) and the **owner role** (enterprise owner vs org admin vs billing manager) — because access
asymmetry means some controls aren't available at org scope.

## Recommendation / verdict output template

```md
# Copilot Cost Review — <org/enterprise> · <period>

## Verdict
<Healthy | Optimizable | Overspending> — <one-line headline + the single biggest $ lever>

## The bill (sourced from the metered usage report)
- Total: <credits / $>  [source: metered report <date>]
- By model: <top models by $>   · By SKU: <Copilot / cloud agent / Spark>
- By user/cost-center: <if enterprise access; else "per-user unavailable at org scope [VERIFY]">
- Productivity cross-ref (metrics API): acceptance <%>, agent vs user lines, top engaged users

## Recommendations (ranked by $ impact)
| # | Lever | Est. $ impact | Control to enact (owner) | Effort | Confidence |
|---|---|---|---|---|---|
| 1 | Model right-sizing: route X off Opus | $<n>/mo [VERIFY tokens] | Models policy (enterprise owner) | low | med |
| 2 | Enable auto model selection | ~10% of eligible | org Copilot settings (org admin) | low | high |
| … | | | | | |

## Watch-items
- Promo credit cliff 2026-09-01 (Business 3,000→1,900, Enterprise 7,000→3,900/user [VERIFY]) — re-budget.
- Per-token rates / flex allotments / model list change frequently — figures re-verified <date>.

## Data + access notes
- Surfaces read: <metered report> (cost), <Metrics API/dashboard> (productivity).
- Access level: <enterprise | org> — per-user cost attribution <available | NOT available; needs enterprise>.
- [VERIFY] items: <list>
```

## Hard rules
- No $/credit/token figure without a source line; everything else is `[VERIFY]`.
- Re-verify rates/allowances against current docs every review — never quote from memory.
- Recommend savings only where the enacting control exists at the caller's access level; otherwise note
  the access gap as the first recommendation (e.g., "obtain enterprise-level billing access").
