# Validation Prompts

Twelve curated prompts that exercise the pack across four capability areas. Each prompt has a rubric, a reference output, and notes on which skills it stresses.

## Format

Every prompt file has the same structure so they can be processed mechanically or skimmed by hand:

```markdown
---
id: <area>/<NN-slug>
area: <mcp-server-building | microservices-design | architecture-review | cross-cutting>
exercises:
  - <pack/path/to/skill.md>
pass_threshold: <N>/<M>   # e.g., 7/9 — N rubric points satisfied out of M total
last_run: <YYYY-MM-DD or "never">
last_result: <pass | fail | "never run">
---

# <Prompt title>

## Context

<What the prompt assumes, what's attached as context, what role the responder is playing.>

## Prompt

<The literal prompt to feed to an LLM. Quote it as-is, no edits.>

## Rubric

A response passes if it covers at least <N> of the following <M> points:

- [ ] **<Short label>** — <Specific thing the response must address>
...

## Reference output

<A hand-curated exemplar of what a quality response looks like. Length should match
the bar set by the rubric — long enough to demonstrate the right depth.>

## Notes

<Optional: why this prompt was chosen, what it stresses about the pack,
what kind of regression it would catch.>
```

## Index

### MCP server building (3)

| ID | Title | Pass | Exercises |
|---|---|---|---|
| [mcp-server-building/01](mcp-server-building/01-design-typed-tool.md) | Design a typed-input MCP tool | 6/8 | `02-mcp-go-tool-design`, `06-mcp-go-project-structure` |
| [mcp-server-building/02](mcp-server-building/02-add-observability-to-tool.md) | Add observability to an existing tool | 6/8 | `08-mcp-go-observability`, `02-mcp-go-tool-design` |
| [mcp-server-building/03](mcp-server-building/03-defend-against-poison-input.md) | Defend a tool against poison input | 6/8 | `07-mcp-go-enterprise-security`, `18-mcp-go-anti-patterns` |

### Microservices design (4)

| ID | Title | Pass | Exercises |
|---|---|---|---|
| [microservices-design/01](microservices-design/01-decompose-monolith.md) | Decompose a checkout monolith | 7/10 | `03-domain-decomposition`, `04-service-boundaries` |
| [microservices-design/02](microservices-design/02-choose-data-pattern.md) | Choose a data pattern for cross-service consistency | 6/8 | `05-data-architecture`, patterns/saga, patterns/transactional-outbox |
| [microservices-design/03](microservices-design/03-event-driven-vs-sync.md) | Event-driven vs. synchronous for a notification flow | 6/8 | `07-async-messaging`, patterns/event-driven-architecture |
| [microservices-design/04](microservices-design/04-cost-vs-resilience-tradeoff.md) | Trade cost against resilience for a small team | 6/8 | `12-cost-and-tradeoffs`, `06-resilience-patterns` |

### Architecture review (3)

| ID | Title | Pass | Exercises |
|---|---|---|---|
| [architecture-review/01](architecture-review/01-saga-design-review.md) | Review a saga design for compensation gaps | 7/9 | patterns/saga, patterns/idempotent-consumer |
| [architecture-review/02](architecture-review/02-event-sourcing-fit.md) | Is event sourcing the right fit here? | 6/8 | patterns/event-sourcing, patterns/cqrs |
| [architecture-review/03](architecture-review/03-zero-trust-gap-review.md) | Find zero-trust gaps in a microservices design | 7/9 | patterns/zero-trust-service-access, `11-security-design` |

### Cross-cutting (2)

| ID | Title | Pass | Exercises |
|---|---|---|---|
| [cross-cutting/01](cross-cutting/01-azure-mapping-tradeoff.md) | Justify a pattern → Azure service mapping | 6/8 | `09-azure-mapping`, multiple pattern cards |
| [cross-cutting/02](cross-cutting/02-pattern-card-cross-reference.md) | Trace a pattern's related-patterns chain | 5/7 | All pattern cards |

Total: 12 prompts, 95 rubric points across them, pass thresholds calibrated per prompt difficulty.
