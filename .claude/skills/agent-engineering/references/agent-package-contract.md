# The agent package contract

A serious enterprise agent is not a prompt — it is a **package**. Create mode emits three artifacts;
Review mode checks them. Align field names to the stewarded standards (Linux Foundation AGENTS.md/A2A,
the Mitchell model-card → Anthropic system-card lineage) so reviewers recognize them.

For a simple agent, the spec can be **one `AGENT_SPEC.md` with these sections**; split into separate
files only when complexity warrants (progressive disclosure).

## Artifact 1 — `AGENT_SPEC.md` (descriptive; the novel value-add)

Model-card/system-card structured. Sections:

| Section | Content |
|---|---|
| **Role / identity** | One clear job, narrow and opinionated. Owner. |
| **Scope & boundaries** | Explicit in-scope and out-of-scope; "does not drift into adjacent work." |
| **Problem & success criteria** | The business problem; measurable definition of done/good. |
| **Inputs (input contract)** | Required + optional inputs, types, input modes/MIME, missing-input behavior. |
| **Outputs (output contract)** | Structured output schema, output modes, what downstream consumes it. |
| **Tools + permissions** | Each tool: purpose, risk tier (read/low → write/high), scoped permissions, guardrail. |
| **Workflow** | The reasoning flow / orchestration pattern; decision points; stopping conditions. |
| **Guardrails & HITL** | Input/output/tool-level guardrails (typed: block / flag / confirm); approval points. |
| **Failure modes** | Known failure modes (hallucination, wrong action, cost overrun, loop) + escalation path. |
| **Evaluation** | The eval suite: happy/edge/adversarial; functional/behavioral/safety metrics; CI gate. |
| **Deployment / monitoring** | Versioning, rollback, kill switch, canary, tracing/audit, drift alerts, budget caps. |
| **Limitations & risks** | What it can't do; residual risks; OWASP-Agentic exposure noted. |

This section follows the system-card adaptation for *agents*: document tool-permission risks and
failure modes the way a system card documents dangerous capabilities.

## Artifact 2 — `agent-card.json` (machine-readable interop, A2A)

The one stewarded interop standard for "advertise yourself." A2A Agent Card, hosted at
`/.well-known/agent-card.json`. Core fields:

```json
{
  "protocolVersion": "0.3.0",
  "name": "…", "description": "…", "version": "…", "url": "…",
  "preferredTransport": "JSONRPC",
  "provider": { "organization": "AaraMinds", "url": "…" },
  "capabilities": { "streaming": false, "pushNotifications": false },
  "securitySchemes": { "…": { "type": "oauth2 | bearer | mtls" } },
  "defaultInputModes": ["text/plain"], "defaultOutputModes": ["text/plain"],
  "skills": [
    { "id": "…", "name": "…", "description": "…",
      "tags": ["…"], "examples": ["trigger phrase 1", "…"] }
  ]
}
```
`skills[].tags` + `examples` are the machine-readable scope/trigger surface. Never embed secrets; use
authenticated extended cards + secured endpoints.

## Artifact 3 — the runnable agent file (scaffold per target)

Abstract a canonical internal model — `name`, `description`/trigger, system-prompt body, model,
tool-allow/deny, mcp-servers, permission/sandbox, hooks — then render per target. `name` + `description`
+ body is the irreducible core every target shares.

### Claude Code subagent — `.claude/agents/<name>.md` (YAML frontmatter + Markdown body)
Frontmatter: `name` (required, lowercase-hyphen), `description` (required; drives auto-delegation —
include "use proactively" / boundaries), `tools` (allowlist; omit = inherit), `model`
(`sonnet`/`opus`/`haiku`/`inherit`), optional `permissionMode`, `maxTurns`, `mcpServers`, `hooks`
(PreToolUse/PostToolUse/Stop — the enforced guardrail surface). Body = system prompt: role line →
"When invoked:" numbered workflow → checklist → output rules. **(This is the AaraMinds pack's native
format — match the existing `aara-*` agents.)**

### GitHub Copilot custom agent — `.github/agents/<name>.agent.md` (YAML + Markdown, ≤30k chars)
Frontmatter: `description` (required), optional `name`, `target` (`vscode`/`github-copilot`), `tools`
(list; `[]` = none, `["*"]` = all; `server/*` namespacing), `model`, `mcp-servers` (GitHub.com),
`user-invocable`, `disable-model-invocation`. Tool aliases: `execute`/`read`/`edit`/`search`/`agent`/
`web`/`todo`.

### OpenAI Codex agent — `.codex/agents/<name>.toml` (TOML; most volatile target — flag `[VERIFY]`)
Required: `name`, `description`, `developer_instructions` (system prompt). Optional: `model`,
`model_reasoning_effort` (low/medium/high), `sandbox_mode` (read-only/workspace-write), `[mcp_servers.X]`,
`[[skills.config]]`.

### Companion — `AGENTS.md`
When the agent runs in a repo, emit an `AGENTS.md` (plain Markdown: overview, build/test/lint commands,
conventions, security notes) — the project-instruction layer, distinct from the agent definition.

## Review checks this contract

Review mode verifies all three artifacts exist, the runnable file is valid for its target, and the
spec sections are complete (missing sections are findings, not silent gaps).
