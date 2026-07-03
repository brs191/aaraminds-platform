# Template — runnable agent file

Fill this in for the canonical internal model, then render to the target format (per-platform
frontmatter field tables in `references/agent-package-contract.md`: Claude `.claude/agents/<name>.md`,
Copilot `.github/agents/<name>.agent.md`, Codex `.codex/agents/<name>.toml`). `name` + `description` +
body is the irreducible core every target shares.

```md
---
name: {{agent-name}}
description: {{one-line: what it does + when to use + "use proactively" / boundaries}}
version: 0.1.0
owner: {{owner}}
runtime_target: {{claude | copilot | codex | internal}}
status: draft        # draft | pilot-candidate | production-candidate | production
model: {{inherit | sonnet | opus | haiku | full-id}}
tools: [{{allowlist — least privilege}}]
# optional: permissionMode, maxTurns, mcpServers, hooks (PreToolUse/Stop = enforced guardrail surface)
---

# {{Agent Display Name}}

## Mission
{{one sentence}}

## Business purpose
- Problem: {{}}   - Users: {{}}   - Job-to-be-done: {{}}   - Measurable improvement: {{}}
- Why an agent is justified (not a single call / workflow): {{}}

## Use when / Do not use when
Use when: {{1}} · {{2}} · {{3}}
Do not use when: {{1}} · {{2}}

## In scope / Out of scope / Human-only decisions
In scope: {{}}
Out of scope: {{}}
Human-only (agent must NOT decide alone): {{irreversible / financial / policy / scope changes}}

## Workflow
1. {{step → concrete action/output}}
2. …
Stopping conditions: {{max turns / final-output tool / error}} · Escalation: {{when + to whom}}

## Tools (each: purpose · risk tier · scope · guardrail)
| Tool | Purpose | Risk (read=low / write=high) | Scope (least-privilege) | Guardrail (block/flag/confirm, at the side effect) |
|---|---|---|---|---|

## Guardrails & failure modes
- Input / output / tool-level guardrails: {{}}
- Lethal-trifecta status: {{private data? untrusted content? external comms? — gate one leg if all three}}
- Failure modes + escalation: {{hallucination / wrong action / loop / cost}}

## Output format
{{exact structured shape}}

## Success criteria
{{verifiable "done/good" the agent can self-check}}
```

Keep the body lean; tool/guardrail depth belongs in the spec (`agent-spec-template.md`), not the
runnable file.
