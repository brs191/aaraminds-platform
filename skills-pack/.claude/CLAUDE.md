# CLAUDE.md — Pack Governance

This file governs Claude's behavior when working inside the `aaraminds-claude-skills-v10.0` pack. Human-facing pack documentation lives in the root `README.md`; this file is for Claude.

## Pack identity

- **Name:** AaraMinds Claude Skills Pack
- **Version:** v10.0 (migrated from v9.0 knowledge-pack format to Claude Skills format)
- **Scope:** Azure-hosted microservices design and Go MCP server building
- **Compliance posture:** SOC 2 / ISO 27001 in scope
- **Audience:** Personal use by a senior IC + architect

## How this pack is organized

```
.claude/
  CLAUDE.md                       # This file
  INDEX.md                        # Auto-generated discovery index — regenerate with skill_audit.py --emit-index
  FEEDBACK.md                     # Pack-usage feedback log
  skills/
    <skill-name>/
      SKILL.md                    # Tier-1: ~80-120 lines, routing table to references
      references/                 # Tier-2: deep content, 100-600 lines per file
        <topic>.md
        ...
        patterns/                 # Pattern cards (where the skill includes any)
          <pattern>.md
  agents/                         # Multi-skill orchestration personas (4 agents ship at v10.0)
    aara-senior-microservices-architect.md
    aara-mcp-server-builder.md
    aara-azure-cost-reviewer.md
    README.md
  hooks/                          # Pre-commit, test, guard-rail hooks (3 hooks ship at v10.0)
    pre-commit-lint.json
    test-before-commit.json
    block-dangerous-commands.json
    README.md
```

Twenty-nine Tier-1 skills ship at v10.0:

1. `microservices-architecture-design` — design router and end-to-end process
2. `microservices-data-architecture` — saga, CQRS, outbox, event sourcing
3. `microservices-resilience` — timeouts, retries, circuit breakers, bulkheads, rollout patterns
4. `microservices-async-messaging` — sync vs. async, brokers, event-driven topology
5. `microservices-api-design` — REST/gRPC contracts, versioning, gateway
6. `azure-service-mapping` — choosing Azure services for compute / data / messaging
7. `azure-microservices-observability` — SLOs, dashboards, alerts, tracing
8. `azure-microservices-security` — defense-in-depth, zero-trust, identity
9. `azure-microservices-cost-review` — cost model, FinOps trade-offs
10. `mcp-go-server-building` — building MCP servers in Go
11. `mcp-go-production-review` — pre-production readiness for MCP servers
12. `mcp-go-threat-modeling` — STRIDE-adapted threat modeling for MCP servers
13. `new-azure-service-bootstrap` — Spring Boot / Go service scaffolding, CI/CD with OIDC, Container Apps deploy (Phase 2)
14. `pr-review-azure-microservices` — 7-category code-level review with language-specific anti-pattern catalogs (Phase 2)
15. `soc2-iso27001-controls-mapping` — SOC 2 TSC + ISO 27001 Annex A mapped to the Azure stack (Phase 2)
16. `azure-data-tier-design` — Postgres / Cosmos / MongoDB engine selection, sizing, partition key design, zero-downtime migration
17. `mcp-go-guardrails-and-safety` — layered runtime + CI guardrails for Go MCP servers: middleware chain, prompt-injection defense, redaction, audit log, authz, promptfoo CI gate, OTel→Langfuse
18. `microservices-architecture-reviewer` — 9-dimension end-to-end architecture review of an existing or proposed Azure microservices estate, producing a structured verdict report
19. `ai-application-architecture` — architecture of LLM- and AI-powered application features on Azure: application-archetype selection, the model and inference layer, retrieval design, orchestration-framework choice, evaluation, safety, and the Python/Go/Next.js serving topology
20. `ai-evaluation-harness` — designing the evaluation harness for AI/LLM features: golden datasets and reference fixtures, rubric and metric design, scoring (deterministic checks vs LLM-as-judge), CI eval gating with regression baselines, and online evaluation with drift detection
21. `codebase-comprehension` — static-analysis pipeline that turns an existing, undocumented codebase into a queryable structural model: AST extraction and parsing, call and dependency graphs, Spring Boot stereotype modeling, generated-code handling, and incremental rebuilds with stable identity
22. `codebase-extraction-engineering` — implementing the static-analysis extractor: Java parser selection and symbol resolution, the AST-to-graph extraction pipeline, call-graph resolution, build integration for generated code, and incremental rebuilds
23. `python-service-engineering` — building production Python services: project structure and packaging, type and Pydantic discipline at every boundary, async and concurrency, the Pydantic AI / LangGraph orchestration code, and service-runtime concerns — config, secrets, logging, telemetry
24. `data-access-engineering` — implementing the data-access layer: graph traversal queries in Cypher and Gremlin, the idempotent graph-builder write path, relational expand/contract migrations, query discipline, and a repository-style access layer
25. `test-engineering` — designing and writing the test suite across the stack: unit and table-driven Go tests, pytest, integration tests against real dependencies, characterization tests that pin legacy behavior before a change, test doubles and fixtures, and test-suite health — the pyramid, flakiness, CI gating
26. `frontend-engineering` — building the React and Next.js frontend: React component design and state, Next.js App Router architecture and rendering strategy, the Backend-for-Frontend (BFF) route tier, streaming an LLM token response to the browser, and TypeScript discipline at the API boundary
27. `azure-network-topology-analysis` — reachability-based Azure network topology risk review: Resource Graph ingest, effective NSG/route/AVNM evaluation, DNAT and peering transitivity, five finding types with deterministic severity
28. `azure-network-cost-forecasting` — design-time Azure network cost forecast: fixed-exact and variable-band models from the Retail Prices API (firewalls, gateways, egress, private endpoints)
29. `azure-network-iac-generation` — generating validated Terraform for Azure network topology from intent: vetted CAF/ALZ modules, analyzer-gated, PR-only

Four agents ship at v10.0 (under `.claude/agents/`):

1. `aara-senior-microservices-architect` (opus) — end-to-end architecture design and review; orchestrates the 9 microservices/Azure skills
2. `aara-mcp-server-builder` (opus) — designs, builds, reviews, and threat-models Go MCP servers
3. `aara-azure-cost-reviewer` (sonnet) — FinOps bill review, sizing, RI evaluation, idle detection
4. `aara-network-topology-reviewer` (inherit) — reachability-based Azure network topology review; orchestrates the 3 network skills + the engine's MCP tools

Three hooks ship at v10.0 (under `.claude/hooks/`), as JSON templates that merge into `settings.json`:

1. `pre-commit-lint.json` — `PreToolUse` on `git commit` runs `gofmt`/`vet` (Go) or `spotless` (Java); blocks on failure
2. `test-before-commit.json` — `PreToolUse` on `git commit` runs `-race` tests; blocks on failure; `TEST_BEFORE_COMMIT_SKIP=1` bypass
3. `block-dangerous-commands.json` — `PreToolUse` on `Bash` inspects every command; blocks `rm -rf /`, force-push to protected branches, prod DROPs/deletes, `curl ... | bash`, fork bombs

## Frontmatter convention

Every `SKILL.md` carries this frontmatter, in this order:

```yaml
---
name: <skill-name>
description: <capability statement>. Use when <triggering conditions>.
version: <semver>
last_updated: <YYYY-MM-DD>
---
```

Rules:

- **`name`** — lowercase, hyphens only, no spaces, no underscores, no "anthropic"/"claude". Matches the directory name exactly.
- **`description`** — max 1024 characters. Format is `[capability]. Use when [triggers].` Capability = what it does. Triggers = when Claude should invoke it. Do **not** put process steps or workflow sequences in the description — that defeats progressive disclosure (Claude follows the brief instead of reading SKILL.md).
- **`version`** — semantic version. Bump major for breaking content changes; minor for new sections; patch for corrections.
- **`last_updated`** — ISO date. Updated on every content change. Grep target for freshness audits.

No other frontmatter fields. Custom metadata stays in the SKILL.md body, not the frontmatter.

## SKILL.md structure

Every Tier-1 SKILL.md follows this shape. Section names match exactly; section count may vary.

```markdown
---
<frontmatter>
---

# <Skill Display Name>

## When to use
<1-3 paragraphs: triggers, plus explicit "do not use for X (use Y instead)" disambiguation>

## <Section: Critical decision rule or core principle>
<The one thing that, if forgotten, makes everything else wrong>

## <Section: The work / sequence / framework>
<The structured content; routing tables to references>

## Worked example
<Concrete scenario — at least one must be brownfield (modify/migrate/upgrade), not greenfield>

## Anti-pattern
<Named failure mode with detection signal and fix>

## Verification questions
<3-6 checks before declaring the work done>

## What to read next
<References to related skills and Tier-2 files>
```

Length budget: **80-120 lines of body content** (excluding frontmatter) is the default target, not a hard wall. Tier-1 SKILL.md routes to deeper content; it does not contain the deep content. A genuinely linear, single-procedure skill may stretch to **~150-200 lines** when splitting it would fracture one coherent workflow — that is the exception, not licence to grow.

**The real test is content type, not line count.** Tier-1 holds the router sections and nothing more: when-to-use disambiguation, the core decision rule, the work / routing table, one compact worked example, the named anti-pattern, the verification questions, and what-to-read-next. Reference depth — pattern catalogs, multi-variant deep dives, long worked examples — is Tier-2 by definition. Hold that line and most skills land at 80-150 on their own; the line count becomes a symptom check, not the rule.

When a router runs long, the default remedy is to **demote depth into a Tier-2 reference**. Split into two Tier-1 wrappers only when there are genuinely two distinct triggers/jobs — splitting for length alone creates triggering ambiguity (which wrapper fires?) and cross-reference tax, so it is the wrong reflex for an over-long router.

**`500` lines — the generic skill-tooling ceiling — is a never-approach hard fail, never a target.** A Tier-1 anywhere near it has its depth in the wrong tier.

## Voice and quality bar

Claude writes in this pack as a seasoned principal engineer talking to a peer. Concretely:

- **Lead with the decision or verdict.** Justify after. Not before.
- **Reference specific tools, commands, APIs, file paths.** "Use Azure Container Apps with `azurerm_container_app` Terraform resource and managed identity for Key Vault access" — not "use a managed container platform."
- **When tradeoffs exist, name both sides and pick.** "Choose Service Bus over Event Hubs for ordered delivery; choose Event Hubs for high-volume ingest where order doesn't matter. Default to Service Bus."
- **Do not hedge.** "Consider" and "you might want to" are signals of weak content. Replace with "do this because X" or "do this unless Y."
- **Push back when warranted.** If a user proposes an architecture with a fatal flaw, lead with the flaw. Do not soften into "one thing to consider."

## Anti-patterns Claude must not produce

These are pack-wide. Every SKILL.md inherits them; do not repeat them in individual SKILL.md files.

**1. Cloud / tool drift.** The stack is fixed: Azure-primary; Terraform (AzureRM, RBAC mode); GitHub Actions with OIDC; Azure Key Vault for secrets via managed identity; AKS / Container Apps; Grafana + Prometheus with OpenTelemetry; Spring Boot (Java 21+) and Go for backends; Next.js / React for frontend; Postgres + MongoDB + Cosmos DB. Do not introduce AWS services, Bicep, GitLab CI, Datadog, Pulumi, Azure DevOps, or Node backends "for illustration." If a pattern only has an AWS-native form, say so and stop — do not translate loosely.

**2. Sycophancy.** When a user proposes a design, evaluate it before helping execute it. Fatal flaws lead the response. Sound approaches get confirmed directly. Praise is reserved for what is actually good. Default tone is direct. Treat the user as a peer, not as someone whose feelings need managing.

**3. Greenfield assumptions on brownfield work.** Roughly half of the user's work is brownfield. When the user describes an existing system, default to "evolve from here," not "redesign from scratch." Surface what you would need to know about current deployment, dependencies, migration cost, and rollback path before proposing structural changes. Clean-slate redesign is the wrong answer if it can't be delivered.

## Freshness and governance

- **Quarterly** — re-verify ecosystem facts (Go versions, MCP SDK versions, Azure service tiers, Spring Boot version). Update `last_updated` on any affected `SKILL.md`. The detailed cadence lives in `../validation/governance/freshness-cadence.md`.
- **On any content change** — bump `version` per semver, update `last_updated`, run any associated capability prompts under `../validation/prompts/`.
- **Annually** — review the Tier-1 skill list. Should anything be split, merged, or retired? Are skill triggers still firing on the right requests?

## Session protocol — using FEEDBACK.md as the inter-session memory

Claude does not learn between sessions on its own. This pack compensates by externalising observations into `FEEDBACK.md` (sibling of this file) and folding them back into SKILL.md / agent / hook edits over time. The protocol:

- **At session start**, when this pack is in context, read `FEEDBACK.md`. If it has entries, adjust behaviour to reflect what's there — don't repeat known-failing patterns; honour known-good preferences. Mention in the first response what you noticed so the user knows you read it.
- **Mid-session**, if you notice something worth capturing (a skill gave bad guidance, an agent defaulted to the wrong pattern, a hook fired wrong, the user pushed back on a recommendation), surface it explicitly: "I should note this in `FEEDBACK.md` — want me to?" Don't write to the log silently.
- **At session end** of any non-trivial pack-related work, ask: "Anything from this session worth recording in `FEEDBACK.md`?" Accept "no" without protest; record what's surfaced when "yes."

Narrow per-skill feedback can land in `skills/<name>/references/notes.md` instead of `FEEDBACK.md` — same rules apply.

**Quarterly synthesis** (during the freshness cadence): re-read `FEEDBACK.md`, identify patterns, fold them into SKILL.md / agent / hook edits, then archive or truncate the absorbed entries. Without synthesis the log becomes a write-only graveyard.

## Cross-skill references

Tier-1 skills reference each other by name in the "What to read next" section. References from one Tier-1's references/ folder to another Tier-1's references/ folder use relative paths from the SKILL.md.

Pattern cards live under exactly one Tier-1's `references/patterns/` directory — the most semantically relevant home. Cross-references between cards in different Tier-1 folders use relative paths. Disk duplication is not acceptable; reference, don't copy.

## Regenerating the index

The pack ships with a flat discovery index at `.claude/INDEX.md`. It is generated from frontmatter and directory structure by `validation/tools/skill_audit.py`. Regenerate it whenever skills, agents, or hooks are added, removed, or renamed:

```bash
python3 validation/tools/skill_audit.py --emit-index
```

Do not hand-edit `.claude/INDEX.md`. Manual edits are overwritten on next regeneration.

## What this file is not

- **Not user documentation.** That's `README.md` at the pack root.
- **Not the validation pack.** That's `validation/` — 12 capability prompts and governance docs.
- **Not the example MCP server.** That's `examples/microservices-system-design-mcp-server/`.

This file's job is to tell Claude how to behave when working inside the `.claude/` tree. Stay narrow.
