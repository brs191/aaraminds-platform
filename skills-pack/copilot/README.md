# AaraMinds on Mac + GitHub Copilot

The AaraMinds pack was built for Claude Code, but Claude Code's sign-in is blocked here by the
corporate proxy. This adapter runs the pack inside **VS Code + GitHub Copilot** on an Apple
Silicon Mac instead. One script does the machine-specific work; everything else is committed config.

## What you get

- **The MCP server (13 tools)** — the behaviorally-validated core: `review_microservice_design`,
  `detect_architecture_risks`, `generate_architecture_decision_record`, and 10 more. Copilot calls
  these directly in agent mode (deterministic Go, not LLM guesses).
- **17 agents** — `aara-senior-microservices-architect`, `aara-mcp-server-builder`,
  `aara-azure-cost-reviewer`, `aara-network-topology-reviewer`, and 12 more (see `../Ranking.md`). VS Code reads the canonical
  `skills-pack/.claude/agents/*.md` natively — no conversion, no second copy to drift.
- **35 skills** as a **read-on-demand knowledge base** under `skills-pack/.claude/skills/`. Copilot
  does not auto-route skills the way Claude Code does (see "What doesn't carry over").

## Prerequisites

- Apple Silicon Mac (M-series). Intel works too — see Troubleshooting.
- VS Code with the GitHub Copilot + Copilot Chat extensions, signed in.
- Go 1.25+ — `brew install go`. The server is rebuilt from source; no binary is committed.

## Setup (the one repeatable step)

From the repo root:

```bash
bash skills-pack/copilot/setup-mac.sh
```

Re-run it after pulling changes, after editing the server, or on a new machine. It is idempotent
and backs up anything it overwrites.

## What gets installed, and where

| Scope | Mechanism | Active when |
|---|---|---|
| **This repo** | committed `.vscode/mcp.json` + `.vscode/settings.json` | you open the AaraMinds repo in VS Code |
| **Every repo** | `setup-mac.sh` → VS Code user `mcp.json` + `~/.copilot/agents/` | always — including your real work repos |

The script builds `skills-pack/examples/.../mcp-server` (git-ignored), registers it in your VS Code
**user** MCP config with an absolute path, and copies the 17 agents to `~/.copilot/agents/`. That is
what makes the tools and agents available while you review a customer repo — not only here.

## Verify (after a window reload)

1. `Cmd+Shift+P` → **MCP: List Servers** → `aaraminds-microservices` (running).
2. Copilot Chat → **Configure Tools** → ~13 tools listed.
3. Chat → agents dropdown or `/agents` → the four `aara-` agents.
4. `@aara-senior-microservices-architect`, ask a design question → verdict-first, brownfield voice.

## How to use it

- **Agents:** pick one from the dropdown or `@mention` it. They carry the AaraMinds voice and point
  at the skills + MCP tools they rely on.
- **MCP tools:** in agent mode, ask for a review / ADR / topology check — Copilot calls the Go tools
  and reasons over their structured output.
- **Skills:** open the relevant `skills-pack/.claude/skills/<name>/SKILL.md` and add it to context
  when you want that discipline. You are the router.
- **Model:** pick per request in the chat panel (e.g. Opus 4.6) — agents do not pin one.

## What doesn't carry over from Claude Code

- **Skill auto-routing** — Copilot won't fire a skill from its description.
- **Hooks** — the three pre-commit / guard hooks have no Copilot equivalent; adapt them as git
  hooks if you want similar behaviour.

## Single source of truth for agents

Agents live once, in `skills-pack/.claude/agents/`. VS Code and Claude Code both read that folder.
To add or change an agent, edit it there and re-run `setup-mac.sh` — never hand-maintain a second
Copilot copy. That divergence (an old 10-agent copy that disagreed with the canonical 4) is exactly
what this rebuild removed.

## Troubleshooting

- **MCP server shows stopped/errored** — open its output; usually the binary isn't built. Re-run
  `setup-mac.sh`; confirm `go version` is 1.25+.
- **Gatekeeper blocks the binary** — you built it locally so this is rare; if it happens:
  `xattr -d com.apple.quarantine skills-pack/examples/microservices-system-design-mcp-server/mcp-server`.
- **Intel Mac** — the script builds for the native arch, so it just works (amd64 instead of arm64).
- **Agents don't appear** — confirm `~/.copilot/agents/aara-*.md` exist; reload the window;
  `/agents` → Configure Custom Agents shows each agent's source.
- **Where things live** — binary: `skills-pack/examples/.../mcp-server` (git-ignored); user MCP
  config: `~/Library/Application Support/Code/User/mcp.json`; agents: `~/.copilot/agents/`.
- **Uninstall** — remove the `aaraminds-microservices` entry from the user `mcp.json` and the
  `aara-*.md` files from `~/.copilot/agents/`.
