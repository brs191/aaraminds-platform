# Create mode — problem → built, packaged agent

Takes a business problem and produces a complete, reviewable agent package plus the runnable file.
Walks seven phases (the reviewer's role model, run as *phases of one workflow*, delegating two phases
to specialist agents — not seven spawned agents).

## Phase 0 — Earn the agent (the gate before design)

Before designing anything, decide the architecture honestly (route to `ai-application-architecture`):

- **Single LLM call** (with retrieval + in-context examples) if one call satisfies the need.
- **Deterministic workflow** (LLM on fixed code paths) if the steps are known and you want
  predictability.
- **Agent** (model directs its own tool use in a loop) only for open-ended tasks where you can't
  predict the step count or hardcode the path. OpenAI's positive triggers: complex judgment/exceptions,
  brittle/unmaintainable rulesets, or heavy reliance on unstructured data.

Record the decision and why. If it isn't an agent, say so and stop — don't build agency the task
doesn't need.

## Phase 1 — Intake (understand the problem)

Capture: the business problem and who has it; users; constraints; success criteria (what "done/good"
means, measurably); data sensitivity; the environment it runs in. Missing items are flagged, not
invented. Route framing to `aaraminds-ai-agent-blueprint-advisor` (+ Module 08) for boundary-first
scoping.

## Phase 2 — Architect the agent

Design the three components (OpenAI/Google triad): **model · tools · instructions**, plus guardrails
and orchestration.

- **Model:** baseline with the most capable model to set the ceiling, then down-size per sub-task to
  hit the accuracy target at lower cost/latency. Pin the version.
- **Tools (the ACI):** few, high-leverage, **namespaced**, docstring-grade descriptions; consolidate
  chained operations; return high-signal context (no raw UUIDs); poka-yoke; **risk-tier each** (read =
  low; write/irreversible/financial = high). Taxonomy: data / action / orchestration.
- **Single-before-multi:** maximize one agent; split only on proven branching complexity or tool
  *overlap*. If multi-agent is justified (breadth-first parallel reads, context overflow, many tools),
  keep synthesis/writing single-threaded and share full traces.
- **Orchestration:** pick the simplest of the five patterns that works — prompt chaining, routing,
  parallelization, orchestrator-workers, evaluator-optimizer.

## Phase 3 — Instruction / prompt design (delegate)

Route to `prompt-engineering` / `aara-prompt-engineer` to write the system prompt, behavior rules,
input contract, and the triggering description, in the target platform's idiom. Instructions derive
from SOPs; every step maps to a concrete action/output; edge cases are explicit conditional branches.

## Phase 4 — Security & governance design (delegate)

Route to `references/security-governance.md` (+ `azure-microservices-security`, `soc2-iso27001-controls-mapping`,
Module 05). Apply the lethal-trifecta test; scope tools to least privilege; define HITL approval points
for high-risk actions; define audit/tracing; set stopping conditions and a kill switch.

## Phase 5 — Evaluation design (delegate)

Route to `references/evaluation-design.md` / `ai-evaluation-harness` / `aara-ai-evaluation-engineer`.
Produce the golden set (20–50 cases from real/expected failures, balanced where a behavior should AND
should not fire), the functional/behavioral/safety metrics, and the CI regression gate. Eval-first:
write these alongside the agent, not after.

## Phase 6 — Emit the package + scaffold the runnable file

Produce the three-artifact package (see `references/agent-package-contract.md`):
1. `AGENT_SPEC.md` — the descriptive spec (model-card/system-card lineage).
2. `agent-card.json` — A2A-compliant machine-readable interop card.
3. The **runnable agent file**, scaffolded for the target: `.claude/agents/<name>.md` (YAML
   frontmatter + system-prompt body), `.github/agents/<name>.agent.md`, or `.codex/agents/<name>.toml`.
   All share `name` + `description` + body; differ on model/tools/mcp/permission syntax. Plus a
   companion `AGENTS.md` when the agent operates in a repo.

## Phase 7 — Self-review before handoff

Run Review mode (`references/review-rubric.md`) on the agent you just built. Do not hand off a
Create-mode output that scores below the band the use-case requires, or that trips a hard gate. Then
hand the improvement backlog to the user.

## What Create does NOT do

It does not declare the agent production-ready — only Evaluate mode (run against real test cases) can
clear that bar. Create emits a buildable, reviewed package; readiness is earned by behavior.
