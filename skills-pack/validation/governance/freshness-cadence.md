# Freshness Cadence and Ownership

The pack's claims age. Ecosystem facts decay (Go versions, SDK versions, MCP spec versions, Azure service tiers). Pattern advice can fall behind new platform features. References become stale. Without a named owner and a written cadence, this happens silently and the pack drifts from useful to misleading.

This document is the freshness contract.

## Ownership

This is a personal pack (per `.claude/CLAUDE.md`: "Personal use by a senior IC + architect"). All ownership rolls up to the pack maintainer; no separate backup chain exists. If the pack is forked to a team setting, replace the rows below with named individuals before relying on the cadence — anonymous ownership is unenforceable ownership.

| Concern | Owner | Cadence |
|---|---|---|
| Ecosystem facts (Go version, MCP SDK versions, MCP spec) | pack maintainer | Quarterly |
| Azure service capabilities & pricing referenced in skills | pack maintainer | Quarterly |
| Per-skill evals (run, score, log result) | pack maintainer | Before each tagged release |
| Validation prompts (refresh reference outputs if drift is real) | pack maintainer | Semi-annually |
| Demo goldens (regenerate against current MCP server build) | pack maintainer | Before each tagged release |
| Pattern card cross-references (ensure links don't rot) | pack maintainer | Semi-annually |
| Threat model (re-evaluate against new attacker capabilities) | pack maintainer | Annually |
| External references (URLs, document IDs) | pack maintainer | Annually |

The maintainer runs the refresh and updates the pack. In a team adoption, split the rows by topic competence (Go ecosystem owner, Azure capabilities owner, security owner, etc.) and add a named backup per row.

## Quarterly refresh checklist

Run on a fixed date each quarter (e.g., the first Wednesday of January, April, July, October). Estimated time: 2–3 hours total.

### 1. Ecosystem facts (`skills/mcp/00-ecosystem-facts.md`)

- [ ] Check current Go stable version (`go.dev/dl/`); update if a new minor has shipped
- [ ] Check `github.com/mark3labs/mcp-go` latest release; note any breaking changes
- [ ] Check `github.com/modelcontextprotocol/go-sdk` latest release; note any breaking changes
- [ ] Check MCP spec version at `modelcontextprotocol.io`; note any new spec version
- [ ] Update the dated header in `00-ecosystem-facts.md` with the new verification date
- [ ] If any version changed materially: write a "what changed" note in the same file
- [ ] If the SDK or spec changes break example code: file a follow-up to update examples in the same release

### 2. Azure service capabilities

- [ ] Re-verify any service tier / pricing claim in `09-azure-mapping.md` and `12-cost-and-tradeoffs.md` (Container Apps pricing model, Service Bus tiers, Azure SQL DTU/vCore costs)
- [ ] Check for new Azure services that obsolete current recommendations (e.g., new managed services that should replace existing patterns)
- [ ] Note any preview features that have GA'd (or deprecated services like SSE-MCP-transport — note status)
- [ ] Update dates and figures in the skill files

### 3. Spot-check three skills

Each quarter, randomly pick three skill files (one MCP, one microservices skill, one pattern card) and run their evals. Log the date and the result. If any fail, file an issue against the skill.

```
Last spot-check:
  Date: [__________]
  Files checked:
    - [__________]
    - [__________]
    - [__________]
  Results:
    - [__________]
```

## Pre-release refresh

Before tagging a release (any version bump on the pack):

- [ ] Run `make demo && make validate` in `demo/architecture-review-demo/`. All architectures pass.
- [ ] Run the 12 capability prompts under `validation/prompts/`; log results in front-matter (`last_run`, `last_result`). At least 80% pass at their declared threshold.
- [ ] Update `ROADMAP.md` and `README.md` with the new quality position and what shipped.
- [ ] Update this document with the release date.

```
Last release: [__________]
Demo goldens regenerated: [yes / no, date]
Skill evals run: [yes / no, date, pass rate]
Validation prompts checked: [yes / no, date]
```

## Annual review

Once per year, an open review of the pack as a whole:

- [ ] Are the gaps in `ROADMAP.md` still the right gaps? Has the field moved?
- [ ] Are the validation prompts still discriminating? Update or replace those that no longer catch failures.
- [ ] Are skill files still the right shape? Should anything be split, merged, or retired?
- [ ] Is the team's threat model (maintained per project; see `skills/mcp/20-mcp-go-threat-modeling.md` for the framework) still complete given new attacker techniques?
- [ ] Are external references in skill files still valid URLs?

## Drift signals (anyone, anytime)

Anyone can file a freshness issue when:

- A new MCP spec version, SDK release, or Go version drops between quarterly refreshes
- An Azure service deprecation or pricing change affects skill content
- A validation prompt produces a passing response that nonetheless looks wrong (the rubric is stale)
- An eval consistently fails after a skill change (the skill drifted or the eval drifted)
- A user reports a pack claim that contradicts current reality

Drift issues take priority over scheduled refreshes. The named owners triage them.

## What this is not

- Not a build pipeline. Refreshes are human-driven.
- Not a substitute for normal code review. Skill changes still go through review.
- Not a release-gate enforcer. The pre-release checklist is a recommendation; the team decides whether to ship without checking every item.

The point of the doc is to make freshness *visible* and *owned*. The discipline is the value.
