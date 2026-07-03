# How to use the AaraMinds Pack in VS Code + Copilot

This is the day-to-day user guide for working with this pack inside VS Code + GitHub Copilot Chat on Mac.

To set up (or re-set-up) on a Mac, run `copilot/setup-mac.sh` — full guide in [copilot/README.md](copilot/README.md). See **Repurpose on Mac** below for the short version.

For the pack's overall governance, voice, and anti-patterns, see [.claude/CLAUDE.md](.claude/CLAUDE.md).

---

## Repurpose on Mac (setup)

The pack was built for Claude Code, but Claude Code sign-in is blocked by the corporate proxy, so it
runs under **VS Code + GitHub Copilot** instead. One script does the machine-specific work:

```bash
copilot/setup-mac.sh        # re-run after pulls, server edits, or on a new machine; it self-locates, so CWD doesn't matter
```

It checks Go >= 1.25, **rebuilds the MCP server from source** (native Apple-Silicon arm64 — no committed
binary trusted), smoke-tests it, registers the server in your VS Code **user** config, and installs the
four agents into `~/.copilot/agents/`. Two scopes result:

- **This repo** works from committed config (`.vscode/mcp.json`, `.vscode/settings.json`) when you open it.
- **Every repo** works from the user-level install — so the tools and agents are there while you review
  your real work repos, not only here.

Prerequisites: Apple Silicon Mac, VS Code with Copilot + Copilot Chat signed in, and `brew install go`
(1.25+). Full setup, verification, and troubleshooting live in [copilot/README.md](copilot/README.md).

---

## What you have access to in VS Code

After setup, three things are wired up:

| Surface | What it is | Where to invoke |
|---|---|---|
| **4 custom agents** | Persona system prompts with scope, voice, and structured-deliverable formats | Copilot Chat → agent dropdown at top of panel, or `@agent-name` inline |
| **1 MCP server (13 tools)** | The `aaraminds-microservices` server exposing deterministic Go tools for ADR generation, architecture review, boundary canvases, etc. | Copilot Chat agent mode — tools are auto-discovered; verify via Cmd+Shift+P → `MCP: List Servers` |
| **32 skills** | Markdown knowledge base under `.claude/skills/` — Tier-1 SKILL.md routers + Tier-2 references + pattern cards (exact counts in `.claude/INDEX.md`) | Read on-demand (Copilot doesn't auto-route — see "Using the skills" below) |

Paths in this guide are relative to the **`skills-pack/` folder** of the repo (the folder that contains `.claude/`). The setup command above self-locates, so it runs from anywhere.

---

## The four agents — when to pick which

| Question shape | Agent | Example trigger |
|---|---|---|
| "Design / review a microservices system end-to-end" | `@aara-senior-microservices-architect` | "I'm extracting a payments service from a Spring Boot monolith. What should I know?" |
| "Build / review / threat-model a Go MCP server" | `@aara-mcp-server-builder` | "Add a tool to my MCP server that generates ADRs from a JSON input" |
| "Review an Azure bill / size infrastructure / evaluate RIs" | `@aara-azure-cost-reviewer` | "Our Azure bill went up 35% last quarter — figure out why" |
| "Review Azure network topology / reachability / attack paths" | `@aara-network-topology-reviewer` | "Audit our hub-and-spoke VNet for unintended internet exposure" |

**Voice you should expect from all four:**
- Lead with the verdict / decision, not context-setting
- Push back on fatal flaws (won't soften into "consider…")
- Brownfield-first thinking ("evolve from here" by default, not "redesign")
- Stack-pinned (Azure, Terraform AzureRM, GitHub Actions OIDC, Spring Boot + Go) — won't translate to AWS/GCP/Bicep "for illustration"
- Specific named risks, not generic ones

If you see hedge-y "you might want to consider" prose, the agent isn't loaded properly — check `~/.copilot/agents/` exists and contains the four `aara-*.md` files (run `copilot/setup-mac.sh` if not).

---

## Using the 13 MCP tools

The tools are **deterministic Go code** — rule-based, fast (~ms), produce structured JSON. They're not LLM calls. The model uses tool output as input to its reasoning, not as the final answer.

### The tool catalog

| Tool | Purpose |
|---|---|
| `review_microservice_design` | Score a design against the 9-dimension framework |
| `recommend_microservice_patterns` | Map architecture shape → pattern set (saga, CQRS, etc.) |
| `score_well_architected_readiness` | Azure WAF-style readiness pillars |
| `generate_service_boundary_canvas` | Bounded-context canvas for a service |
| `generate_api_contract` | OpenAPI / gRPC contract scaffold |
| `detect_architecture_risks` | Find anti-patterns in a proposed design |
| `map_patterns_to_azure_services` | Pattern → Azure service mapping with rationale |
| `generate_observability_plan` | SLOs, dashboards, alerts for a service |
| `generate_architecture_decision_record` | Structured ADR from a context+decision input |
| `generate_deployment_topology` | Container Apps / AKS deployment shape |
| `generate_event_contract` | Event schema with versioning + delivery semantics |
| `generate_resilience_plan` | Timeouts / retries / circuit breakers / bulkheads per dependency |
| `generate_diagram_assets` | Mermaid / C4 diagram source from architecture input |

### When tools fire vs. when you invoke them
- **Tools fire automatically** when the model decides they fit. Watch the chat panel for tool-call indicators.
- **You can invoke explicitly** by mentioning the tool name in chat: "use `generate_architecture_decision_record` for this — context: …, decision: …"
- **You can disable individual tools** via "Configure Tools" in the chat panel if a tool keeps firing when you don't want it.

### Verifying tools are loaded
```
Cmd+Shift+P → "MCP: List Servers"
```
Look for `aaraminds-microservices` with status `running`. If error: click "Show Output" — usually the binary isn't built yet. Re-running `copilot/setup-mac.sh` is safe and idempotent.

---

## Using the 32 skills (the knowledge base layer)

In Claude Code, SKILL.md descriptions auto-routed. **Copilot does not auto-route.** You become the router. Three usable patterns:

### Pattern 1 — Read a SKILL.md before asking
When working on, say, async messaging, open `.claude/skills/microservices-async-messaging/SKILL.md`, read the framework, then ask Copilot with that framework in mind. The agent's voice + your framework knowledge = a focused answer.

### Pattern 2 — Attach the SKILL.md to a chat
Copilot Chat supports file attachment. Drag the relevant SKILL.md (or its referenced Tier-2 file) into the chat. The model reads it as direct context. Most reliable way to get skill-grounded answers.

### Pattern 3 — Tell the agent which skill to use
The agent system prompts reference the skills by name. In chat: "Use the framework from `microservices-data-architecture` for this. Specifically the saga vs. outbox decision tree." The agent will reach for that SKILL.md if you have it open in the workspace, or paraphrase from what's already in the system prompt.

### The skill index
[.claude/INDEX.md](.claude/INDEX.md) is the auto-generated flat index — every skill, pattern card, agent, hook with one-line scopes. Regenerate with:
```bash
python3 validation/tools/skill_audit.py --emit-index
```

### Pattern cards
Each is a self-contained how-to for one pattern (saga, CQRS, circuit-breaker, strangler-fig, etc.). Live at `.claude/skills/<owning-skill>/references/patterns/<pattern>.md`. The full list — every card and its owning skill — is in `.claude/INDEX.md`.

---

## Common workflows — worked examples

### Workflow 1 — Design a new microservices system
1. `@aara-senior-microservices-architect` in chat
2. State the problem in 2-3 sentences (domain, team size, scale, constraints)
3. Expect: verdict on "should this be microservices at all" → bounded-context proposal → service-by-service breakdown → ADR draft
4. The architect may call MCP tools (`generate_service_boundary_canvas`, `recommend_microservice_patterns`) — let it
5. If you want it deeper on one stage (e.g., async messaging), attach `.claude/skills/microservices-async-messaging/SKILL.md` and re-ask

### Workflow 2 — Add a tool to an existing MCP server
1. `@aara-mcp-server-builder` in chat
2. Describe the tool (verb-led name, inputs, outputs, examples)
3. Expect: typed input struct, `internal/services/<name>/service.go` skeleton, `internal/tools/<name>/register.go` skeleton, contract file scaffold, table-driven test stubs
4. The builder follows the layering rule from `.claude/skills/mcp-go-server-building/` — no MCP imports in the service package

### Workflow 3 — Quarterly cost review
1. `@aara-azure-cost-reviewer` in chat
2. Paste your Cost Management export (CSV or summary table)
3. Expect: top spend drivers → cost-lever table → ranked recommendations with savings + effort + risk
4. Verdict-first, dollar-amount-first format

### Workflow 4 — Architecture review of a brownfield estate
1. `@aara-senior-microservices-architect`
2. Describe current state (services, Azure resources, pain points)
3. Architect will likely call `review_microservice_design` and `detect_architecture_risks` MCP tools
4. Expect a 9-dimension verdict report (Decomposition / Data / Resilience / Observability / Security / Cost / API / Async / Service Mapping) — pass / soft-fail / hard-fail per dimension

---

## What's different from Claude Code (don't expect these)

| Claude Code feature | What happens in Copilot |
|---|---|
| SKILL.md descriptions auto-routing skills | Doesn't exist. You attach the SKILL.md or mention it by name. |
| `.claude/hooks/*.json` (pre-commit lint, test-before-commit, block-dangerous-commands) | Doesn't exist. Re-implement as git pre-commit hooks if needed. |
| `model: inherit` per agent | Doesn't exist. You pick the model per chat via the panel. |
| The FEEDBACK.md inter-session memory protocol | Still works — but you're the one feeding it. Copilot doesn't read it automatically; you mention it in chat when relevant. |
| Progressive disclosure (Tier-1 SKILL.md → Tier-2 references) | Was a Claude Code context-budget optimization. In Copilot, you decide what to load; attach Tier-2 directly when needed. |
| The 17 agents delegating to each other ("invokes the X skill") | Copilot custom agents don't auto-delegate. You switch agents manually via the dropdown. |

---

## Maintenance rhythm

### Daily
- Use the pack. The agents and tools get better through use, not through theorizing.

### Per-session
- If an agent gave bad guidance, add a note to [.claude/FEEDBACK.md](.claude/FEEDBACK.md). The protocol from CLAUDE.md applies even though Copilot won't read it automatically — you read it before sessions.

### Quarterly (every ~90 days)
- Run the audit:
  ```bash
  cd <pack root>
  python3 validation/tools/skill_audit.py --emit-index
  ```
- Re-check ecosystem facts (Go versions, Azure tiers, MCP SDK versions) per [.claude/CLAUDE.md](.claude/CLAUDE.md) → "Freshness and governance"
- Synthesize FEEDBACK.md entries into SKILL.md / agent edits; archive absorbed entries

### Annually
- Review the Tier-1 skill list. Should anything split, merge, or retire?
- Re-run [VERIFICATION_CHECKLIST.md](VERIFICATION_CHECKLIST.md) end-to-end as a clean baseline

---

## Troubleshooting

### MCP server doesn't appear in `MCP: List Servers`
- Check `~/Library/Application Support/Code/User/mcp.json` exists and has an `aaraminds-microservices` entry
- Check the `command` path in mcp.json points at the actual binary file
- Re-run `copilot/setup-mac.sh` (idempotent; backs up existing config before changes)

### MCP server shows error indicator
- Click the error indicator in chat → "Show Output" — read the stderr log
- Most common: the binary isn't built. Run `copilot/setup-mac.sh` — it rebuilds from source and smoke-tests.
- Architecture is never an issue: the script builds for your Mac's native arch (arm64 on Apple Silicon, amd64 on Intel).

### Custom agents don't appear in the dropdown
- Check `~/.copilot/agents/` exists and contains the four `aara-*.md` files
- Reload VS Code window: Cmd+Shift+P → "Developer: Reload Window"
- The agents are Claude-format `.md` files — VS Code reads these natively from `~/.copilot/agents/` and `.claude/agents/` (no `.agent.md` rename needed)

### Custom agent fires but ignores the persona
- The agent file may not have loaded — check for YAML parse errors in the frontmatter
- Try a fresh chat with the agent explicitly selected from the dropdown (not via `@mention` inline)
- Confirm you're on a model with enough context (the agent files are 100-200 lines each — well within Opus 4.6's window, but a tiny model may truncate)

### MCP tool returns an error / empty result
- The tool is deterministic Go code — failures are usually input validation
- Click into the tool-call to see the JSON input that was sent — most likely fields are missing or malformed
- The contract for each tool lives at `examples/microservices-system-design-mcp-server/contracts/architecture-tools/implemented/<tool-name>.md`

---

## Where to find what

| What | Where |
|---|---|
| Setup / re-setup on Mac | [copilot/README.md](copilot/README.md), `copilot/setup-mac.sh` |
| Pack governance, voice, anti-patterns | [.claude/CLAUDE.md](.claude/CLAUDE.md) |
| Flat index of every skill, pattern card, agent, and hook | [.claude/INDEX.md](.claude/INDEX.md) |
| Per-skill content | `.claude/skills/<name>/SKILL.md` + `references/` |
| Agent system prompts | `.claude/agents/*.md` (canonical source, read natively by VS Code) or `~/.copilot/agents/` (user-level install) |
| MCP server source + tool contracts | `examples/microservices-system-design-mcp-server/` |
| Per-tool contract | `examples/microservices-system-design-mcp-server/contracts/architecture-tools/implemented/<tool>.md` |
| Audit tool + last audit report | `validation/tools/skill_audit.py`, `validation/skill-audit-*.md` |
| End-to-end verification commands | [VERIFICATION_CHECKLIST.md](VERIFICATION_CHECKLIST.md) |
| Inter-session feedback log | `.claude/FEEDBACK.md` (create if not present) |
