# Agents

This directory contains seventeen subagent personas for Claude Code. `aara-copilot-cost-reviewer` (2026-06-18) is a FinOps reviewer for enterprise GitHub Copilot spend, built through the `agent-engineering` factory (design 85/100, 6/6 cases run-tested incl. fabrication/conflation/access-asymmetry refusals). `aara-business-analyst` (2026-06-18) is the requirements front-end of the delivery lifecycle — the first agent produced *and run-tested* through the `agent-engineering` factory (built on the existing BA blueprint; design 90/100; **6/6 golden cases passed, pass^3=1.0** across 3 runs incl. prompt-injection refusal; tested rollback; **production-candidate PASS** — only the live deploy remains). The rest are seven domain agents plus eight added 2026-06-15 (`aara-topology-visualizer` and a 7-agent project-delivery lifecycle). Three newer domain agents: `aara-prompt-engineer` (2026-06-16) engineers, optimizes, and teaches prompts across Claude, GitHub Copilot, and OpenAI Codex; `aara-status-deck` (2026-06-17) produces the recurring monthly leadership status deck; `aara-agent-engineer` (2026-06-18) is the AI Agent Designer & Evaluator — designs, reviews, evaluates, and hardens enterprise AI agents (the fleet's governance layer). Each agent has a focused scope, an opinionated voice, and a set of Tier-1 skills it routes to. The agents compose with the skill system: they decide *when* to invoke a skill and *how* to integrate skills' outputs into a deliverable.

## The agents

| Agent | Model | Scope | Invokes |
|---|---|---|---|
| `aara-senior-microservices-architect` | opus | End-to-end architecture design and review across the microservices estate | 9 skills (architecture-design, data-architecture, resilience, async-messaging, api-design, azure-service-mapping, observability, security, cost-review) |
| `aara-mcp-server-builder` | inherit | Building, reviewing, and threat-modeling Go MCP servers | mcp-go-server-building, mcp-go-production-review, mcp-go-threat-modeling, new-azure-service-bootstrap (Go scaffold half), pr-review-azure-microservices |
| `aara-azure-cost-reviewer` | sonnet | Cost / FinOps work — bill review, sizing, RI evaluation, idle detection | azure-microservices-cost-review (primary), azure-service-mapping + observability (supporting) |
| `aara-network-topology-reviewer` | inherit | Reachability-based Azure network topology review; drift, attack-paths, cost & generation orchestration | azure-network-topology-analysis (primary), -cost-forecasting, -iac-generation, **azure-iac-policy-as-code**, **azure-defender-signal-ingestion**, soc2-iso27001-controls-mapping + the engine's MCP tools |
| `aara-topology-visualizer` | inherit | Produces the risk-annotated topology *diagram* (Phase 4) — consumes the analyzer for severity | azure-network-topology-visualization (primary), azure-network-topology-analysis |
| `aara-project-architect` | inherit | System design, decomposition, ADRs, brownfield evolution → design docs | (design-time; hands to planner/builder) |
| `aara-project-planner` | inherit | Outcome-defined phases, T-shirt estimates, critical path, risk register | (planning) |
| `aara-project-builder` | inherit | Execute a playbook step/ticket: code + tests + green gate + Result log | (implementation; calls mcp-server-builder / python-ai-developer) |
| `aara-project-reviewer` | inherit | Adversarial acceptance review → acceptance memo, gates cited to file:line | (review) |
| `aara-project-debugger` | inherit | Reproduce → root-cause → minimal fix + regression test | (diagnosis) |
| `aara-python-ai-developer` | inherit | Python / LLM-orchestration (explainer, generator intent, reference engines, viz pipeline) | ai-evaluation-harness, python-service-engineering |
| `aara-ai-evaluation-engineer` | inherit | Build/run eval gates (precision/recall, diagram-eval, twin-drift, triggering) | ai-evaluation-harness (primary) |
| `aara-prompt-engineer` | inherit | Generate / optimize / teach prompts for AI coding assistants across Claude, GitHub Copilot, and OpenAI Codex | prompt-engineering (primary) |
| `aara-status-deck` | inherit | Produce the recurring monthly leadership status deck (.pptx), manager-through-VP altitude | aaraminds-leadership-status-deck (primary), composes aaraminds-executive-narrative-advisor + pptx |
| `aara-agent-engineer` | inherit | AI Agent Designer & Evaluator — create / review (100-pt rubric) / evaluate / harden enterprise AI agents | agent-engineering (primary); delegates to aara-prompt-engineer + aara-ai-evaluation-engineer; routes to blueprint-advisor + security skills |
| `aara-business-analyst` | inherit | Trace-first BA: stakeholder inputs → traceable requirements / stories / acceptance criteria; human-gated; hands off to architect + planner | (delivery front-end; built via agent-engineering, pkg in `agent-packages/aara-business-analyst/`) |
| `aara-copilot-cost-reviewer` | inherit | FinOps reviewer for enterprise GitHub Copilot spend (AI-Credit model): usage + billing → ranked, sourced cost-optimization verdict; human-gated (recommends, never enacts) | copilot-cost-optimization (primary); built via agent-engineering, pkg in `agent-packages/aara-copilot-cost-reviewer/` |

## When Claude Code invokes which agent

Claude Code reads the `description` field in each agent's frontmatter and routes invocations based on the user's request. Examples:

- *"Design a new microservices system for an online ordering platform"* → `aara-senior-microservices-architect`
- *"Add a new tool to my MCP server that generates ADRs"* → `aara-mcp-server-builder`
- *"Our Azure bill went up 35% — figure out why"* → `aara-azure-cost-reviewer`
- *"Write an AGENTS.md for this repo"* / *"my Claude system prompt over-triggers its search tool"* → `aara-prompt-engineer`
- *"Build my monthly status deck for my manager and VP"* → `aara-status-deck`
- *"Design an agent for X"* / *"review this agent — is it production-ready?"* / *"evaluate this agent"* → `aara-agent-engineer`

When a request spans agent boundaries (e.g., "design the MCP server architecture for a cost-tracking workflow"), Claude routes to the primary owner and the agent may itself delegate to others or directly invoke the relevant skills.

**On the `inherit` tier:** `aara-mcp-server-builder` is marked `inherit`, meaning it uses the session's default model rather than a hard-pinned one. Most MCP server work (scaffolding, adding tools, embedding guardrails) is well within Sonnet's range; deep architecture decisions (transport choice, threat surface, package layering) benefit from Opus. Override per session with `claude --model opus` when the task warrants it; the default Sonnet is fine for routine work.

## Agent vs. skill — what's the difference?

- A **skill** is a knowledge artifact: SKILL.md + references. It tells Claude *how to think* about a domain. Skills don't have model preferences or tool restrictions.
- An **agent** is a persona: system prompt + tool list + model choice. It tells Claude *how to behave* — voice, escalation rules, delegation patterns, deliverable shape. Agents invoke skills.

A useful analogy: skills are the textbook; agents are the consultant who knows when to open which chapter.

## Two agent families

- **Domain agents** (the first five rows plus `aara-prompt-engineer` and `aara-status-deck`) own a knowledge domain — architecture, MCP building, cost, network review, network visualization, prompt engineering, and leadership status reporting — and route to the engineering or communication skills.
- **Project-delivery agents** (architect → planner → builder → reviewer → debugger, plus python-ai-developer and ai-evaluation-engineer) own the *lifecycle* of executing a playbook/ticket. Authored 2026-06-15 so the antr playbooks' agent references resolve; wired but not yet exercised in a live session.

## Why these domain agents

The first three map to the three loudest workflows; `aara-network-topology-reviewer` (added 2026-06-03) owns the network-topology-review workflow; and `aara-prompt-engineer` (added 2026-06-16) owns the prompt-engineering workflow (generate / optimize / teach across Claude, Copilot, Codex) — each a distinct workflow that fit the "adding a new agent" criteria below:

1. **Designing or reviewing architecture** is the broadest and highest-leverage; needs the architect persona with full skill access.
2. **Building MCP servers** is a specialized engineering task with its own conventions (stdio-stderr rule, package layering, contract files) that don't blend cleanly with general microservices work.
3. **Cost review** has a different voice (FinOps quantification, leadership-facing deliverables) and a different invocation rhythm (monthly / quarterly bill reviews).

Other personas (security auditor, SRE, data engineer, ML engineer) are deliberately not shipped. Reasons:

- The pack's scope (Azure microservices + Go MCP servers) doesn't warrant data / ML personas.
- A separate security-auditor agent would significantly overlap with `aara-senior-microservices-architect` (which invokes `azure-microservices-security` and `soc2-iso27001-controls-mapping`).
- A separate SRE agent would overlap with observability work the architect agent handles.

If a future scope expands the pack (e.g., a data-engineering skill family), the right move is to add a new agent then. Don't pre-create personas with thin scope.

## Voice consistency across agents

All agents share the pack-wide governance from `.claude/CLAUDE.md`:

- Lead with the verdict / decision; justify after.
- No sycophancy — push back on bad designs / bad cost arguments / unsafe code.
- Brownfield-first — most work modifies existing systems.
- Stack-pinned — Azure, Terraform AzureRM, GitHub Actions OIDC, Spring Boot + Go, no cloud drift "for illustration."
- Specific named risks, not generic ones.

Where agents differ is in *scope* and *deliverable shape*: the architect produces ADRs and review documents, the MCP builder produces Go code and contract files, the cost reviewer produces FinOps recommendation tables.

## Adding a new agent later

Should you decide a new persona is warranted (a fourth workflow that doesn't fit the existing three), follow this shape:

1. **One-paragraph definition of scope.** What it owns; what it explicitly does *not* own; which existing agent it would otherwise have stolen work from.
2. **List of skills it invokes.** Cross-reference each invoked skill; ensure the agent's scope doesn't fully overlap any existing agent.
3. **System prompt** with the same shape as the existing three: scope → critical rule → how-you-work → escalation policy → commitments.
4. **Model choice**: `opus` for complex orchestration that always justifies the cost; `sonnet` for narrower or repeatable workflows; `inherit` when the workload range is wide enough that letting the user override per session is genuinely useful.

Avoid the trap of making every skill its own agent. Agents are useful when they orchestrate multiple skills with consistent voice; one agent per skill duplicates the SKILL.md layer for no benefit.

## What's not in scope for agents

- Hooks (pre-commit lint, test-before-commit, dangerous-command blocking) are *event-driven* automation, not personas. They live under `.claude/hooks/` and run shell commands when Claude Code triggers them.
- Skills are not personas. A SKILL.md describes the domain knowledge; an agent decides when to invoke it.

The boundary: agents are who you ask; skills are what they know; hooks are what fires automatically.
