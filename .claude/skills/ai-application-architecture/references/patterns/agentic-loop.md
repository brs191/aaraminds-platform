# Pattern: Agentic Tool-Calling Loop

## Problem

A goal requires steps that cannot be enumerated in advance — the model must inspect intermediate results and decide what to do next, invoking tools until the goal is met. A fixed workflow cannot express "it depends"; this archetype hands control flow to the model. That power is also the danger: the loop can run away on cost, latency, and failure modes a deterministic flow never has.

## Use When

- The task decomposition is genuinely dynamic — the next step depends on what the last step returned.
- A bounded, well-described set of tools exists, each with typed inputs and clear semantics.
- Goal completion is checkable — there is a condition that says "done."
- The latency and cost of multiple model round trips is acceptable.

## Avoid When

- The steps are known ahead of time → `llm-workflow.md`. Most "agents" are workflows in disguise; a fixed DAG is cheaper, faster, and testable.
- The task is one call → `single-shot.md`.
- There is no reliable "done" check — an open-ended loop with no terminator is a cost incident waiting to happen.
- Tool side-effects are irreversible and the loop is unsupervised.

## Shape

A Python orchestration loop — Foundry Agent Service first (managed loop, memory, monitoring), a self-built Pydantic AI or LangGraph loop only when you outgrow it (`orchestration-frameworks.md`). Tools are exposed through the Go MCP tool tier (`mcp-go-server-building`), typed, never untyped JSON. The loop must carry, as first-class design elements: a hard step cap, a token / cost budget that aborts the run, a wall-clock timeout, and one OpenTelemetry trace spanning every tool call. Read-only tools are low-risk; write tools need authorization and ideally a dry-run.

## Trade-offs

| Aspect | Trade-off |
|---|---|
| Flexibility | Handles tasks no fixed flow can express |
| Bounded surface | Cost, latency, and failure are all variable per run — none are knowable ahead of time |
| Observability | Every run is a different path; distributed tracing is non-negotiable, not optional |
| Testability | Hard — behaviour is non-deterministic; evaluate task-success and tool-call correctness, not output equality |

## Common Failure Modes

- **Non-termination** — the loop never satisfies its done-check and runs to the step cap every time. Detection: alert on runs hitting the cap. Prevention: an explicit, checkable goal condition; the cap is a backstop, not the design.
- **Tool thrash** — the model calls the same tool repeatedly with near-identical arguments. Detection: dedupe tool calls in the trace and flag repeats. Prevention: feed prior results back clearly; reconsider whether this is really a workflow.
- **Context explosion** — accumulated tool outputs blow the context window mid-run. Detection: token-count per turn. Prevention: summarize or window the history; do not append blindly.
- **Cost runaway** — a single run costs many times the median. Detection: per-run cost in telemetry against a budget. Prevention: a hard token budget that aborts the run.

## Decision Signals

Use an agentic loop only when the control flow itself must be decided at runtime. If you can draw the steps on a whiteboard before the run, it is a workflow — build that instead.

## Worked signal — Code Intelligence Factory

The CIF brief is explicit: build the BA and QA capabilities as document *generators over the graph* first, and "promote them to fully orchestrated agents only when interactive, iterative behavior is genuinely required." HLD generation is a workflow (`llm-workflow.md`), not an agent — its steps are known. An agentic loop earns its place only in a later, interactive "ask the codebase" feature, and even then under a step cap and a budget.

## References

- `../orchestration-frameworks.md` — Foundry Agent Service vs a self-built loop
- `mcp-go-server-building` — the typed tool tier the loop calls
- Pattern: `llm-workflow.md` — the cheaper, deterministic alternative
- `../evaluation.md` — task-success and tool-correctness scoring
