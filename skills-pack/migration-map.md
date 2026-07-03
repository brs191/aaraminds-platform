# v9.0 → v10.0 Migration Map

This document records the file moves performed during the v10.0 Phase-1 format migration, and accounts for the skills authored fresh afterward.

> **Accuracy note (2026-05-22):** The migration tables below cover **Phase 1 only** — the 34 v9.0 skill files and 21 pattern cards that were *migrated* into 12 Tier-1 skills. The pack has since grown to **35 Tier-1 skills** (the others were authored fresh, not migrated, and do not appear in the move tables); see `../Ranking.md` / `.claude/INDEX.md` for the current inventory.

## Phase 1 — migrated content (34 skill files + 21 pattern cards → 12 Tier-1 skills)

Every v9.0 file below has exactly one v10.0 destination.

### Microservices skills

| v9.0 path | v10.0 path |
|---|---|
| `skills/microservices/01-design-router.md` | `.claude/skills/microservices-architecture-design/references/design-router.md` |
| `skills/microservices/02-system-design-process.md` | `.claude/skills/microservices-architecture-design/references/system-design-process.md` |
| `skills/microservices/03-domain-decomposition.md` | `.claude/skills/microservices-architecture-design/references/domain-decomposition.md` |
| `skills/microservices/04-service-boundaries.md` | `.claude/skills/microservices-architecture-design/references/service-boundaries.md` |
| `skills/microservices/05-data-architecture.md` | `.claude/skills/microservices-data-architecture/references/data-architecture.md` |
| `skills/microservices/06-resilience-patterns.md` | `.claude/skills/microservices-resilience/references/resilience-patterns.md` |
| `skills/microservices/07-async-messaging.md` | `.claude/skills/microservices-async-messaging/references/async-messaging.md` |
| `skills/microservices/08-api-design.md` | `.claude/skills/microservices-api-design/references/api-design.md` |
| `skills/microservices/09-azure-mapping.md` | `.claude/skills/azure-service-mapping/references/azure-mapping.md` |
| `skills/microservices/10-observability-design.md` | `.claude/skills/azure-microservices-observability/references/observability-design.md` |
| `skills/microservices/11-security-design.md` | `.claude/skills/azure-microservices-security/references/security-design.md` |
| `skills/microservices/12-cost-and-tradeoffs.md` | `.claude/skills/azure-microservices-cost-review/references/cost-and-tradeoffs.md` |

### MCP skills

| v9.0 path | v10.0 path |
|---|---|
| `skills/mcp/00-ecosystem-facts.md` | `.claude/skills/mcp-go-server-building/references/ecosystem-facts.md` |
| `skills/mcp/01-mcp-go-server-basics.md` | `.claude/skills/mcp-go-server-building/references/server-basics.md` |
| `skills/mcp/02-mcp-go-tool-design.md` | `.claude/skills/mcp-go-server-building/references/tool-design.md` |
| `skills/mcp/03-mcp-go-resources.md` | `.claude/skills/mcp-go-server-building/references/resources.md` |
| `skills/mcp/04-mcp-go-prompts.md` | `.claude/skills/mcp-go-server-building/references/prompts.md` |
| `skills/mcp/05-mcp-go-transport-selection.md` | `.claude/skills/mcp-go-server-building/references/transport-selection.md` |
| `skills/mcp/06-mcp-go-project-structure.md` | `.claude/skills/mcp-go-server-building/references/project-structure.md` |
| `skills/mcp/07-mcp-go-enterprise-security.md` | `.claude/skills/mcp-go-server-building/references/enterprise-security.md` |
| `skills/mcp/08-mcp-go-observability.md` | `.claude/skills/mcp-go-server-building/references/observability.md` |
| `skills/mcp/09-mcp-go-testing.md` | `.claude/skills/mcp-go-production-review/references/testing.md` |
| `skills/mcp/10-mcp-go-deployment.md` | `.claude/skills/mcp-go-production-review/references/deployment.md` |
| `skills/mcp/11-mcp-go-agent-integration.md` | `.claude/skills/mcp-go-server-building/references/agent-integration.md` |
| `skills/mcp/16-mcp-go-production-review.md` | `.claude/skills/mcp-go-production-review/references/production-review.md` |
| `skills/mcp/17-mcp-go-code-generation.md` | `.claude/skills/mcp-go-server-building/references/code-generation.md` |
| `skills/mcp/18-mcp-go-anti-patterns.md` | `.claude/skills/mcp-go-production-review/references/anti-patterns.md` |
| `skills/mcp/19-mcp-go-reference-implementation.md` | `.claude/skills/mcp-go-server-building/references/reference-implementation.md` |
| `skills/mcp/20-mcp-go-threat-modeling.md` | `.claude/skills/mcp-go-threat-modeling/references/threat-modeling.md` |
| `skills/mcp/21-mcp-go-client-integration.md` | `.claude/skills/mcp-go-server-building/references/client-integration.md` |
| `skills/mcp/22-mcp-go-cicd-quality-gates.md` | `.claude/skills/mcp-go-production-review/references/cicd-quality-gates.md` |
| `skills/mcp/24-mcp-go-e2e-agent-demo.md` | `.claude/skills/mcp-go-server-building/references/e2e-agent-demo.md` |
| `skills/mcp/25-mcp-go-security-test-generation.md` | `.claude/skills/mcp-go-threat-modeling/references/security-test-generation.md` |
| `skills/mcp/26-mcp-go-runnable-domain-repos.md` | `.claude/skills/mcp-go-server-building/references/runnable-domain-repos.md` |

### Pattern cards

| v9.0 path | v10.0 path |
|---|---|
| `patterns/microservices/saga.md` | `.claude/skills/microservices-data-architecture/references/patterns/saga.md` |
| `patterns/microservices/cqrs.md` | `.claude/skills/microservices-data-architecture/references/patterns/cqrs.md` |
| `patterns/microservices/event-sourcing.md` | `.claude/skills/microservices-data-architecture/references/patterns/event-sourcing.md` |
| `patterns/microservices/transactional-outbox.md` | `.claude/skills/microservices-data-architecture/references/patterns/transactional-outbox.md` |
| `patterns/microservices/database-per-service.md` | `.claude/skills/microservices-data-architecture/references/patterns/database-per-service.md` |
| `patterns/microservices/idempotent-consumer.md` | `.claude/skills/microservices-data-architecture/references/patterns/idempotent-consumer.md` |
| `patterns/microservices/circuit-breaker.md` | `.claude/skills/microservices-resilience/references/patterns/circuit-breaker.md` |
| `patterns/microservices/retry-timeout.md` | `.claude/skills/microservices-resilience/references/patterns/retry-timeout.md` |
| `patterns/microservices/bulkhead.md` | `.claude/skills/microservices-resilience/references/patterns/bulkhead.md` |
| `patterns/microservices/blue-green-canary.md` | `.claude/skills/microservices-resilience/references/patterns/blue-green-canary.md` |
| `patterns/microservices/strangler-fig.md` | `.claude/skills/microservices-resilience/references/patterns/strangler-fig.md` |
| `patterns/microservices/async-messaging.md` | `.claude/skills/microservices-async-messaging/references/patterns/async-messaging.md` |
| `patterns/microservices/event-driven-architecture.md` | `.claude/skills/microservices-async-messaging/references/patterns/event-driven-architecture.md` |
| `patterns/microservices/distributed-tracing.md` | `.claude/skills/microservices-async-messaging/references/patterns/distributed-tracing.md` |
| `patterns/microservices/api-gateway.md` | `.claude/skills/microservices-api-design/references/patterns/api-gateway.md` |
| `patterns/microservices/backend-for-frontend.md` | `.claude/skills/microservices-api-design/references/patterns/backend-for-frontend.md` |
| `patterns/microservices/service-discovery.md` | `.claude/skills/azure-service-mapping/references/patterns/service-discovery.md` |
| `patterns/microservices/service-mesh.md` | `.claude/skills/azure-service-mapping/references/patterns/service-mesh.md` |
| `patterns/microservices/sidecar.md` | `.claude/skills/azure-service-mapping/references/patterns/sidecar.md` |
| `patterns/microservices/cache-aside.md` | `.claude/skills/azure-service-mapping/references/patterns/cache-aside.md` |
| `patterns/microservices/zero-trust-service-access.md` | `.claude/skills/azure-microservices-security/references/patterns/zero-trust-service-access.md` |

Phase-1 total: 34 skill files + 21 pattern cards = **55 migrated files** distributed into 12 Tier-1 skills.

## Skills authored fresh (not migrated)

Twelve Tier-1 skills were written from scratch in or after v10.0 and have no v9.0 source. They are not part of the migration above.

| Skill | When authored | Pattern cards |
|---|---|---|
| `new-azure-service-bootstrap` | Phase 2 | — |
| `pr-review-azure-microservices` | Phase 2 | — |
| `soc2-iso27001-controls-mapping` | Phase 2 | — |
| `azure-data-tier-design` | Post-phase | `partition-key-design`, `connection-pool-sizing`, `caching-patterns` |
| `mcp-go-guardrails-and-safety` | Post-phase | `tool-handler-middleware-chain`, `structured-audit-log`, `argument-sanitization` |
| `microservices-architecture-reviewer` | Post-phase | — |
| `ai-application-architecture` | Post-phase (2026-05-22) | `single-shot`, `rag`, `agentic-loop`, `llm-workflow`, `conversational`, `batch-llm` |
| `ai-evaluation-harness` | Post-phase (2026-05-24) | — |
| `codebase-comprehension` | Post-phase (2026-05-24) | — |
| `codebase-extraction-engineering` | Post-phase (2026-05-24) | — |
| `python-service-engineering` | Post-phase (2026-05-24) | — |
| `data-access-engineering` | Post-phase (2026-05-24) | — |

These seven skills were authored fresh on top of the migrated content; their per-skill reference counts are generated into `.claude/INDEX.md` rather than pinned here.

## Current totals

The pack is **35 Tier-1 skills**, **17 agents**, and **3 hooks**. Reference-file and pattern-card counts change whenever a reference is added, so they are not pinned here: `.claude/INDEX.md` — generated from disk by `validation/tools/skill_audit.py --emit-index` — is their single source of truth, and the same tool's doc-consistency pass verifies the skill/agent/hook counts above against every living document.

## Verification

```bash
# 35 Tier-1 skills, each with a SKILL.md
find .claude/skills -mindepth 1 -maxdepth 1 -type d | wc -l        # expect 29
find .claude/skills -mindepth 2 -maxdepth 2 -name SKILL.md | wc -l # expect 29

# Reference and pattern-card totals — cross-check against the generated index
find .claude/skills -path '*/references/*' -name '*.md' | wc -l           # vs .claude/INDEX.md
find .claude/skills -path '*/references/patterns/*' -name '*.md' | wc -l  # vs .claude/INDEX.md

# v9.0 layout is gone
find skills patterns 2>/dev/null                                   # should return empty

# Regenerate the index and run the audit (lint + doc-consistency check)
python3 validation/tools/skill_audit.py --emit-index
```
