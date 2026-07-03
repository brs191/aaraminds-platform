# agent-engineering — AI Agent Designer & Evaluator

The quality and governance layer for the AaraMinds agent fleet. Use it to **create**, **review**, and
**evaluate** enterprise AI agents to a repeatable standard before they reach production. The governing
idea: *the deck/agent is not the product — repeatable, governed agent quality is.*

## What this is

A thin **composition skill** (it routes to deep skills rather than duplicating them) plus a runnable
orchestrator agent. It owns the lifecycle, the agent-package contract, the 100-point rubric, the staged
release gate, and the **design-vs-behavior firewall** (no agent ships production-ready on a paper
score).

## The three modes

| Mode | Input | Output |
|---|---|---|
| **Create** | A business problem | An agent package (spec + A2A card + runnable file) — built only after "earn the agent" |
| **Review** | An existing agent | A 100-point design score + severity-tagged findings + staged release-gate decision + P0–P3 backlog |
| **Evaluate** | An agent + test env | Functional / behavioral / safety results, trajectory + tool-call + efficiency scoring, go/no-go |

## How to use

1. Invoke the runnable agent: **`aara-agent-engineer`** (it loads this skill and walks the modes), or
   load `SKILL.md` directly.
2. Tell it the mode and target: "design an agent for X", "review this agent", "evaluate this agent."
3. It fills the `templates/` and returns the prescriptive **Agent Engineering Result** (exec summary ·
   files · readiness score · release decision · strengths · risks · P0–P3 fixes · how to use).

## Files

- `SKILL.md` — the router (when to use, two rules, modes, composition, rubric summary, gates).
- `references/` — depth: `create-mode`, `agent-package-contract`, `evaluation-design`,
  `security-governance`, `review-rubric`.
- `templates/` — copy-ready fill-ins: runnable-agent, agent-spec, review-scorecard, **release-gate
  (staged)**, eval-plan + golden dataset, package-index, **tool-risk-register**,
  **agent-efficiency-scorecard**, **agent-trace-review**.
- `schemas/` — machine-readable JSON Schemas for automation: `eval-case`, `eval-result`,
  `trace-review`, `release-gate`.
- `eval/` — the dogfood (this skill scoring `aara-status-deck` → 77/100, CONDITIONAL PASS pilot / FAIL
  production) + eval scenarios.

## Composition (reuse, don't rebuild)

Design → `aaraminds-ai-agent-blueprint-advisor` + `ai-application-architecture`. Prompt →
`prompt-engineering` / `aara-prompt-engineer`. **Evaluation → `ai-evaluation-harness` /
`aara-ai-evaluation-engineer`** (the evaluator specialist — this skill delegates Evaluate mode to it
rather than shipping a duplicate evaluator). Security → `azure-microservices-security` /
`soc2-iso27001-controls-mapping` / Module 05. Backlog → `aara-project-planner`.

**This pack does not ship a duplicate evaluator.** Evaluate mode delegates to
`aara-ai-evaluation-engineer`; if that specialist is unavailable, the pack can still generate an eval
plan but **cannot claim run-tested behavior** (and the release gate stays below production candidate).
Run `scripts/check-dependencies.py <workspace>` to see what's present vs missing.

## Known limitations

- **Unproven by run.** The design is complete and dogfooded once (Review mode), but the three modes
  have not been exercised end-to-end as a registered skill on varied real inputs. By its own firewall it
  is a *strong pilot candidate*, not a proven production capability, until that run happens.
- A sample **validator CI workflow** is included and the schemas are conditionally enforced (firewall).
  Full **behavioral eval execution** (running the agent against the golden set) remains external and is
  wired through `aara-ai-evaluation-engineer` / `ai-evaluation-harness` via the `run-evals.py` adapter +
  `evaluator-handoff-contract.md`.
- Codex `.toml` agent emission is the most volatile target (the format may evolve) — flag `[VERIFY]`.
