---
name: agent-engineering
description: Design, review, evaluate, and harden enterprise AI agents — the AI Agent Designer & Evaluator quality system. Use when the task is to CREATE an agent from a business problem (emit an agent package + scaffold the runnable file), REVIEW an existing agent against a 100-point rubric, or EVALUATE one with functional/behavioral/safety tests. Composes the blueprint advisor, prompt-engineering, ai-evaluation-harness, and security skills. Enforces the firewall between design score and run-tested behavior — no agent ships production-ready on a paper score. Not for MCP servers or one-off prompts. Never invents capabilities, evals, owners, or risk — gaps are marked [VERIFY].
version: 2.4.0
last_updated: 2026-06-18
---

# Agent Engineering — AI Agent Designer & Evaluator

A quality system for enterprise AI agents. The value is not "an agent that makes agents" — it is
**repeatable agent quality**: every agent designed, reviewed, evaluated, and hardened to the same
standard before production. A **thin composition layer** — it owns the three modes, the agent-package
contract, the 100-point rubric with hard gates, and the design-vs-behavior firewall, and routes the
depth to other skills.

## When to use

Trigger this skill when the deliverable is an **AI agent or a judgment about one**: "design/build an
agent for X," "scope an enterprise agent," "review this agent / is it production-ready," "score this
agent," "evaluate / test this agent," "what's wrong with this agent," "harden this agent before
production." It is the governance layer for the AaraMinds agent fleet.

Do **not** use it for: building MCP servers (`mcp-go-server-building`); one-off prompt or instruction
writing with no agent package (`prompt-engineering`); generic AI/LLM feature architecture with no agent
deliverable (`ai-application-architecture`); or standing up an eval harness for a non-agent feature
(`ai-evaluation-harness` directly).

## The two rules that govern everything

1. **Match the mode first.** Create (problem → built agent), Review (agent → scored verdict), or
   Evaluate (agent → run-tested results). They have different inputs, outputs, and gates.
2. **Design score ≠ behavior score — and go/no-go needs both.** Review mode scores the *artifact*
   (static): is it well-designed? Evaluate mode scores *behavior* (dynamic): does it pass real test
   cases? A 95/100 design that has never been run is **not** production-ready. The firewall is
   absolute: a high paper score never substitutes for passing functional/behavioral/safety evals on
   actual runs. (Anthropic: "don't take any eval score at face value until someone reads the
   transcripts"; a 42% score was a grader bug, not a model failure.)

## Modes

| Mode | Input | Output | Reference |
|---|---|---|---|
| **Create** | A business problem / use-case | A complete **agent package** (spec + agent-card + runnable file) | `references/create-mode.md` |
| **Review** | An existing agent (file + spec) | A **100-point score** + Agent Review Summary + prioritized backlog | `references/review-rubric.md` |
| **Evaluate** | An agent + a test environment | **Functional / behavioral / safety** results, trajectory + tool-call scores, go/no-go | `references/evaluation-design.md` |

## Composition — reuse, don't rebuild

This skill orchestrates assets the pack already has. It adds the lifecycle wrapper and the rubric.

| Phase | Routes to (existing) |
|---|---|
| Problem framing, archetype, blueprint | `aaraminds-ai-agent-blueprint-advisor` (+ Module 08), `ai-application-architecture` (agent vs workflow vs single call; agentic-loop archetype) |
| System prompt / instructions / triggering description | `prompt-engineering` (+ `aara-prompt-engineer`) |
| Security & governance review | `azure-microservices-security`, `soc2-iso27001-controls-mapping`, Module 05 (AI Systems Review) |
| Evaluation harness (golden sets, scorers, CI gate, triggering evals) | `ai-evaluation-harness` (+ `aara-ai-evaluation-engineer`) |
| Improvement backlog / replan | `aara-project-planner` / reviewer backlog pattern |

The runnable orchestrator is the `aara-agent-engineer` agent, which walks the phases and delegates the
prompt and eval phases to those specialist agents.

## The critical design calls (encode in Create, check in Review)

1. **Earn the agent** — single LLM call or deterministic workflow unless the task needs open-ended,
   model-directed tool use over an unpredictable step count. Start simple.
2. **Single agent before multi-agent** — split only on proven branching complexity or tool *overlap*;
   keep writing/synthesis single-threaded ("actions carry implicit decisions").
3. **Tools are the agent-computer interface** — few, namespaced, high-signal, poka-yoke'd,
   risk-tiered (read = low; write/irreversible/financial = high).
4. **Guardrails layered and at the side effect** — input/output/tool-level + stopping conditions +
   HITL on high-risk actions.
5. **Prompt injection is architectural** — lethal-trifecta test (private data + untrusted content +
   external comms → remove/gate one leg); reject content-filtering as the fix; least-privilege identity.
6. **Eval-first** — the eval *is* the spec; write it alongside the agent.

Depth in `references/create-mode.md`, `evaluation-design.md`, and `security-governance.md`.

## The 100-point rubric (summary — full rubric + hard gates in `references/review-rubric.md`)

Eleven dimensions (v2 rebalanced — eval weighted highest): Problem Fit (10) · Role Clarity (10) ·
Scope Boundaries (10) · Input Contract (8) · Output Contract (8) · Workflow Design (10) · Tool & Data
Safety (10) · Guardrails & Failure Modes (10) · **Evaluation Coverage (12)** · Production Readiness (7)
· Executive Usability (5, conditional). **Hard gates (override the total):** no eval strategy or no
guardrails → capped at "prototype" (≤79); **never run against test cases → cannot be production-ready**
(the firewall); any fabricated metric/owner/capability → a hard finding. Bands: 90–100 production-ready
*candidate* · 80–89 strong *pilot* candidate · 70–79 prototype · 60–69 weak · <60 redesign. The
**release decision** (PASS / CONDITIONAL PASS / FAIL per requested stage) comes from the staged release
gate, which requires *executed* eval results for a production candidate.

## Worked example — Review mode on `aara-status-deck`

The skill was dogfooded against an existing AaraMinds agent. Review mode scored `aara-status-deck` v1.3
at **77/100 (useful prototype)** on the v2 rubric — clear earned problem and strong contracts, but
specific defect-shaped findings (F-002 unsandboxed `Bash`, no risk-tier/HITL; F-003 failure modes not
enumerated; F-004 no max-turn/monitoring). The staged **release gate** then returned **CONDITIONAL PASS
(pilot) · FAIL (production candidate)** — because the evals are designed but **never run**, and executed
results are required for a production candidate. That is the firewall working: design quality never buys
readiness. Full review in `eval/dogfood-review-aara-status-deck.md`.

## Anti-patterns

- **Paper-score-as-readiness** — shipping on a design score with no run evals (the firewall stops this).
- **Innovation theater** — a multi-agent constellation where one agent + good tools would do.
- **Filter-as-injection-defense** — claiming content filtering solves prompt injection.
- **Thin-wrapper / overlapping tools**, **excessive agency** (broad standing perms, no HITL on
  irreversible/financial actions), **no stopping conditions** (unbounded loops, no kill switch).
- **One-sided evals** — testing only where a behavior should fire, never where it shouldn't.
- **Fabricated readiness** — claiming evals/monitoring/owners that don't exist. Mark `[VERIFY]`.

## Verification — checks before delivering

- Mode identified; outputs match its contract? Create: all 3 artifacts emitted and the runnable file
  valid for its target?
- "Earn the agent" made explicit; single-before-multi justified; tools risk-tiered; guardrails layered
  and at the side effect; HITL on high-risk; stopping conditions?
- Lethal-trifecta checked; no content-filter-as-injection-defense; least-privilege identity?
- Review: 100-pt rubric + hard gates applied; firewall respected; backlog prioritized. Evaluate:
  functional + behavioral + safety run; trajectory/tool-call scored; pass *rates*; transcripts read?
- Every readiness/capability/metric claim sourced or `[VERIFY]`?

## Read next

- `references/create-mode.md` — 7 phases: intake → archetype → design → emit package → scaffold file.
- `references/agent-package-contract.md` — the 3 artifacts: AGENT_SPEC.md, agent-card.json (A2A), runnable file (Claude/Copilot/Codex).
- `references/review-rubric.md` — the 100-point rubric, hard gates, bands, 10-question checklist, Review Summary format.
- `references/evaluation-design.md` — functional/behavioral/safety, grader families, LLM-as-judge pitfalls, trajectory match, CI gates.
- `references/security-governance.md` — OWASP Agentic Top 10, MAESTRO, lethal trifecta, guardrails, scoped identity, tracing, readiness gate.
- `templates/` — copy-ready fill-ins: runnable-agent, agent-spec, review-scorecard (severity + F-001 + P0–P3), **release-gate (staged)**, eval-plan + golden dataset, package-index, **tool-risk-register**, **efficiency-scorecard**, **trace-review**.
- `schemas/` — machine-readable JSON Schemas (eval-case, eval-result, trace-review, release-gate) with **conditional firewall enforcement**.
- `scripts/` — runnable validators (validate-schemas, check-release-gate, check-package-completeness, check-dependencies, run-evals adapter) + a sample CI workflow.
- `references/source-index.md` — sources + last-verified. `references/evaluator-handoff-contract.md` — the exact request/response to `aara-ai-evaluation-engineer`.
- `examples/` — a full worked package (`leadership-status-agent-package/`) + 4 release-gate instances. `eval/` — dogfood + scenarios. `README.md` — overview + limitations.
