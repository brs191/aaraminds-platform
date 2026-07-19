# How Agents Loop: The Agent Loop, the Developer Loop, and the Feedback Loop

**Audience:** engineering teams building or operating agents on the AaraMinds platform.
**Depth:** practitioner → advanced. Assumes you know what an LLM and a tool call are.
**Status:** conceptual reference. The stable parts (why loops exist, how they fail) will hold; the dated parts (tool/spec status) are marked and sourced.

---

## Why this doc exists

Teams use the word "loop" to mean at least three different things, and they collide in the same sentence. Someone says "we need a feedback loop for the agent" and one person hears *the runtime reasoning loop inside a single task*, another hears *the developer iterating in their IDE*, and a third hears *the organization learning from production over weeks*. These are three different systems, on three different clocks, owned by three different parties, with three different failure modes.

This doc gives everyone the same vocabulary and the same mental model. Read it before designing an agent's runtime behavior, before proposing an "agent feedback" feature, or before reviewing someone else's design that uses the word "loop."

The short version: **an agent is a loop, not a pipeline. There is more than one loop, they are nested, and most of the engineering risk lives in where each loop *stops*.**

---

## The core idea: an agent is a loop, not a pipeline

A traditional program is a pipeline: input goes in, fixed steps run in order, output comes out. You can read the code and know every path.

An agent is different. Given a goal, it decides *at runtime* what to do next, does it, looks at the result, and decides again — until it judges the goal met or gives up. The control flow is not in your code; it is a decision the model makes on each pass. That single property — a decision inside the loop — is the entire reason agents are useful (they handle tasks you can't fully script) and the entire reason they are hard to operate (you can't fully predict the path).

Everything in this doc follows from that. If the decision were fixed, you'd have a workflow and none of the loop machinery would matter. Because the decision is learned and probabilistic, you need explicit stopping conditions, bounded permissions, tracing, and evaluation — for every loop below.

---

## The three loops at a glance

The framing of building with agents as three nested loops running at different speeds was set out by Andrew Ng in June 2026, and the same frame emerged independently across the tooling community that month ([Ng, via explainX, 2026](https://explainx.ai/blog/andrew-ng-three-loops-0-to-1-products-2026); [ADTmag, "Loop Engineering Emerges," July 2026](https://adtmag.com/articles/2026/07/01/loop-engineering-emerges-as-developers-put-ai-coding-agents-on-repeat.aspx)). We use it here because it maps cleanly onto how AaraMinds already governs and certifies agents.

| | **Loop 1 — Agent Loop** | **Loop 2 — Developer Loop** | **Loop 3 — Feedback Loop** |
|---|---|---|---|
| Also called | Agentic loop, execution loop, ReAct loop | Inner-dev loop, steering loop | External loop, org loop, continuous certification |
| Clock | Seconds → minutes | Minutes → hours | Hours → weeks |
| Who runs it | The **agent**, inside the runtime | The **developer**, in the IDE | The **organization** |
| One iteration | reason → act (tool) → observe → decide | review output → steer → update spec/evals | ship → observe production → mine signal → recertify |
| Optimizes | Completing *this* task | Building the *right* agent | Keeping the agent *right over time* |
| Ends when | Goal met, budget spent, or blocked | The agent's behavior matches intent | Never (it's a standing capability) |
| Primary risk | Runaway cost, wrong action | Steering on vibes, spec drift | Learning nothing, or violating privacy |

The loops are **nested and one-directional in what they feed**: Loop 3's learnings update the spec, evals, and rubric that constrain Loop 2's decisions, which update the prompt, tools, and policy that constrain Loop 1's behavior. Signal flows outward (production → org); constraints flow inward (org → runtime). When people say "close the loop," they almost always mean *make Loop 3 actually change Loop 1* — which is the hard part, because those two loops are weeks and a dozen artifacts apart.

AaraMinds adds a fourth loop to this frame — the **Memory Loop (Loop 1.5)**, covered after Loop 1 below. It runs across runs, between Loop 1's seconds and Loop 3's weeks, and is operated by the agent plus the platform rather than by a person or the org. It is our extension, not part of the Ng framing.

---

## Loop 1 — The Agent Loop

**Simple meaning.** The agent, working on one task, repeatedly thinks, uses a tool, looks at the result, and thinks again, until it's done.

**Technical meaning.** A control loop where the model is the controller. Each iteration: the model receives the goal plus accumulated context (prior steps, tool outputs, memory), emits either a tool call or a final answer, the runtime executes any tool call and appends the result to context, and the loop repeats. This is the lineage of the ReAct pattern (reason + act interleaved); in 2026 the industry started calling the discipline of designing this loop well "loop engineering" ([explainX, "What Is Loop Engineering," 2026](https://explainx.ai/blog/what-is-loop-engineering-ai-agents-2026)). The important nuance the folk version drops: **the loop's continuation is itself a model decision.** Nothing external decides "keep going" unless you build it.

**Where it sits.** Inside the agent *runtime* — the adopted execution environment, not something AaraMinds builds. AaraMinds' position is to govern this loop at design time and adopt the runtime that executes it, not to reimplement the runtime.

**Example walkthrough** (a certified SQL-expert agent answering "why is this query slow?"):

1. *Reason:* "I need the execution plan before I can diagnose." → *Act:* calls the read-only `get_query_plan` tool.
2. *Observe:* plan shows a key lookup on a non-covering index. *Reason:* "Likely a missing covering index; confirm row counts."
3. *Act:* calls `get_index_stats`. *Observe:* high logical reads.
4. *Decide:* enough evidence. *Act:* emits final answer — a recommended covering index and the reasoning — **without** executing any DDL, because its manifest grants it no write tool.

Note what governed the path: the available tools, their read-only scope, and the stopping decision. Change any of those and the loop behaves differently.

**What ends the loop — the crux.** A production agent loop must have explicit, external stopping conditions, because the model's own judgment of "done" is unreliable. At minimum: a max-iteration cap, a token/cost budget, a wall-clock timeout, and a "no-progress" detector (same tool, same args, twice → stop). The model's "I'm done" is one signal, not the only one.

**Failure modes** (this is where authority lives):

- **No stopping condition → runaway loop.** The agent retries a failing tool forever, or ping-pongs between two steps. Cost and latency blow up with nothing to show. The single most common production incident for agents.
- **Context growth → degradation and cost.** Every iteration appends to context. Long loops fill the window with stale tool output, quality drops, and per-iteration cost rises. Past a certain length, more looping makes the answer *worse*.
- **Error propagation.** A wrong observation early (bad tool result, misread output) poisons every later decision. The loop confidently builds on a false premise.
- **Unsafe action.** A tool with side effects, called with the wrong arguments, is not undoable by "the agent noticing." "Can call the tool" and "calls the tool safely" are different engineering problems — every tool needs input validation, permissioning, idempotency where it mutates state, timeouts, and errors the agent can actually reason about.
- **Reflection is not validation.** Having the agent critique its own output ("reflection") improves some workflows but is not a safety net: a model that is wrong can be confidently wrong about its self-review too. External checks — tests, tool-verified results, human approval — are what actually catch errors.

**Governance — where AaraMinds binds Loop 1.** Every iteration of this loop runs inside constraints set *before* the loop starts:

- **Manifest tool allowlist** — the loop can only call tools the manifest grants. An action the agent can't name, it can't take.
- **Tool contracts** — each tool's inputs/outputs/permissions/idempotency are specified, so "the loop called a tool" has a defined, validated shape.
- **Approval boundaries** — high-risk actions pause the loop for a human. The loop does not cross a risk threshold autonomously.
- **Scoped memory** — what the loop can read/write across iterations is bounded, with retention and provenance, so a stale or poisoned memory can't silently steer it.
- **The readiness certificate** is the gate that says this loop is allowed to run in production at all. An agent whose manifest is `active` must hold a current passing readiness verdict.

The persona or system prompt shapes how the loop behaves, but **does not enforce** limits — a prompt is not a security boundary. The manifest, contracts, and approval gates are.

---

## Loop 1.5 — The Memory Loop (AaraMinds extension)

The three-loop frame above is Ng's; this loop is ours. It is not in the June 2026 framing, but it passes the same test the other loops pass: it has its own clock, its own operator, and failure modes none of the other loops produce. Within a single run, memory is just context accumulation — Loop 1 covers that. Across runs, memory forms a loop of its own, and pretending it's a Loop 1 property is how memory incidents get misdiagnosed as prompt problems.

**Simple meaning.** What the agent stores in one run changes what it does in later runs, and what it does in later runs changes what it stores.

**Technical meaning.** A cross-run cycle: run N writes a memory record → the platform extracts/consolidates it → run N+1 retrieves it → the retrieval steers reasoning → that reasoning writes new records. One full iteration spans at least two runs. The operator is the **agent plus the platform** — the agent decides what to write, the platform decides what survives, consolidates, and gets retrieved. Neither the developer nor the organization drives an iteration, which is exactly why it is not Loop 2 or Loop 3.

| | **Loop 1.5 — Memory Loop** |
|---|---|
| Also called | Memory cycle, cross-run state, agent memory |
| Clock | Across runs, within an engagement (minutes → days) |
| Who runs it | The **agent + platform**, between runtimes |
| One iteration | write record → extract/consolidate → retrieve in later run → steer behavior → write |
| Optimizes | Continuity across tasks without re-deriving context |
| Ends when | Never (standing state; records end via retention/expiry) |
| Primary risk | Poisoning, staleness, cross-engagement leakage |

**Failure modes** (none of these exist in the other three loops):

- **Memory poisoning.** A bad write in run N silently steers run N+50. Unlike Loop 1 error propagation, it survives the run boundary, outlives the context window, and carries no visible trace in the run it corrupts. A poisoned memory is a poisoned *future*, not a poisoned task.
- **Staleness compounding.** A fact true when written is retrieved as true months later. The record doesn't decay; the world does. Retrieval confidence and record accuracy drift apart silently.
- **Retrieval feedback bias.** Frequently retrieved memories get re-cited and reinforced; rarely retrieved ones effectively vanish. The loop narrows its own worldview without any single step being wrong.
- **Cross-engagement leakage.** A memory written in one engagement surfaces in another. This is a confidentiality incident, not a quality incident — and it is the failure the AAP harness already tests for directly.

**Governance — where AaraMinds binds Loop 1.5.** More of this loop is already governed than the three-loop frame suggests:

- **Memory-record contract** — every write is validated against `schemas/memory-record.schema.json`, including classification and retention fields.
- **Citation gate** — uncited writes are denied, not stored, and audited as `memory_denied` events in the tamper-evident chain (proof fields `UncitedMemoryWriteDenied`, `UncitedMemoryDenialAudited`). Provenance is enforced at write time, which is the cheapest place to fight poisoning.
- **Engagement scoping** — reads are scoped to the active engagement/agent policy; cross-engagement isolation is proven, not assumed (`platform/internal/runtime/memory.go`, `memory_leakage_returned`).
- **Retention/expiry** — expired records are excluded from retrieval (`expired_memory_returned` proof field).

- **Consolidation gate** (closed 2026-07-19) — at most one active record per `(engagement_id, claim_key)`; a conflicting write is denied unless it supersedes the record holding the claim, and a valid supersession retires the old record with a `memory_superseded` audit event. Contradictory cited memories can no longer silently coexist (proof fields `ConflictingClaimWriteDenied`, `SupersededRecordExcluded`, `SupersessionAudited`).
- **Retrieval provenance** (closed 2026-07-19) — every in-scope read emits a `memory_retrieved` audit event listing the returned record ids in the tamper-evident chain; cross-engagement queries are audited as `memory_query_denied`; unauditable reads fail closed. A run can now show which memories steered it (proof fields `MemoryRetrievalAudited`, `CrossEngagementQueryAudited`).

What is **not yet governed** — the remaining open item:

- **Extraction quality.** What the platform distills from a run into a record is exactly the Mem0 OSS + Azure OpenAI spike still open in `docs/runtime-verification-notes.md`. A citation gate on a badly extracted fact is a well-audited wrong memory. The spike now has contracts to measure against: extracted records must produce valid citations and claim keys that survive the consolidation gate.

Both the write side and the retrieval side of this loop are now certified in the harness; extraction — the step that turns run content into records — is the loop's one remaining ungoverned edge.

---

## Loop 2 — The Developer Loop

**Simple meaning.** A person builds the agent by running it, watching what it does, correcting it, and updating the instructions and tests.

**Technical meaning.** The human-in-the-loop iteration that produces and refines the agent's *specification*: its goal, prompt, tool set, policies, and — critically — its evaluation cases. One turn: review the agent's output, decide what's wrong at the *product* level (missing capability, wrong tone, unsafe latitude), and update the spec and evals so the next Loop 1 run behaves better ([Ng three loops, 2026](https://explainx.ai/blog/andrew-ng-three-loops-0-to-1-products-2026)).

**Where it sits.** In the developer's environment — IDE, notebook, eval harness — at design and build time. This is also where the **accept / reject / edit** signal originates when an engineer uses an agent: those keystrokes are the highest-quality feedback that exists about whether an agent's output was right, because they come from the person who had to use it.

**Failure modes.**

- **Steering on vibes.** Correcting behavior by re-reading outputs and tweaking the prompt, with no evals, means you can't tell whether a change helped or just moved the failure somewhere else. For non-deterministic systems, "I ran it and it looked better" is not evidence. Evals — task-level, regression, sampled human review — are the discipline that makes Loop 2 converge instead of oscillate.
- **Spec drift.** The prompt, the tool set, and the eval cases fall out of sync with each other and with what the org actually decided. Six weeks later nobody can say what "correct" means for this agent.
- **Over-fitting to the demo.** Tuning until the three examples the developer keeps testing pass, while the real distribution of inputs stays uncovered.

---

## Loop 3 — The Feedback Loop

**Simple meaning.** The organization watches how the agent does in real use over time and feeds what it learns back into the spec, the evals, and the certification.

**Technical meaning.** A cross-run, organization-owned loop: production behavior is observed, real failures and edge cases are mined into new evaluation cases, the readiness rubric is recalibrated against them, and the agent is re-certified. This is the **data flywheel** — production experience becomes the next round's test suite ([Ng three loops, 2026](https://explainx.ai/blog/andrew-ng-three-loops-0-to-1-products-2026)). It is what the proposed AaraMinds *Continuous Certification & Agent Feedback Loop* implements.

**Why it must exist.** A single passing certification says little about the distribution of behavior in production. Three forces erode a once-certified agent:

- **Distribution shift** — real inputs drift away from what you tested.
- **Dependency drift** — the underlying model, tools, and data sources change under the agent.
- **Synthetic-eval blindness** — the failures that matter most are usually the ones nobody thought to write a test for. Only production surfaces them.

Without Loop 3, an agent's real reliability decays silently after the day it was certified. Evaluation for non-deterministic systems is an ongoing discipline, not a one-time gate.

**Failure modes.**

- **Open loop.** You collect traces and dashboards, but nothing ever changes the spec, evals, or rubric. This is the default failure — telemetry that informs no decision is cost, not feedback. The test of Loop 3 is whether a production finding can be traced to a rubric change and a re-certification.
- **Alert fatigue.** Everything is flagged, so nothing is acted on.
- **Privacy and governance violation.** Loop 3 is the loop that touches real user data and real production traffic, so it is where a well-intentioned design becomes a liability. Centralizing raw prompts, raw completions, or source code to "learn from production" creates exactly the surveillance-and-exfiltration risk AaraMinds exists to prevent. The design constraint is non-negotiable: **observe via standardized telemetry, hash-and-reference payloads rather than centralizing them, keep data tenant-local by default, get consent before first collection, redact at the source, and never let the loop autonomously modify or deploy an agent.** A feedback loop that violates governance is a governance incident with extra steps.

**Governance — how AaraMinds keeps Loop 3 honest.**

- **Standardized observation.** Instrument on the OpenTelemetry GenAI semantic conventions so agent, tool, and model spans are captured in a portable shape. Current status matters here: as of April 2026 the GenAI convention pages are still labeled *Development* and most `gen_ai.*` attributes can still change without a major version bump, though they are stabilizing through 2026 ([OpenTelemetry GenAI spans spec](https://opentelemetry.io/docs/specs/semconv/gen-ai/gen-ai-spans/); [Greptime, May 2026](https://greptime.com/blogs/2026-05-09-opentelemetry-genai-semantic-conventions)). Adopt them, but pin versions and expect churn — don't treat the attribute names as stable contracts yet.
- **Hash-and-reference, tenant-local.** Store a hash and a pointer, keep the payload in the tenant boundary. You get the signal without becoming a data-exfiltration path.
- **Mine, don't just monitor.** The output of Loop 3 is *new eval cases and rubric updates*, not just charts. That is what connects it back to Loop 2 and Loop 1.

---

## How the loops connect

```
        Loop 3 — Feedback (org, hours–weeks)
   production traces ─► mined eval cases ─► rubric recalibration ─► re-certification
        │                                                                  │
        ▼ updates spec + evals + rubric                                    │
        Loop 2 — Developer (person, minutes–hours)                         │
   review ─► steer ─► update spec/evals ◄── accept/reject/edit signal      │
        │                                                                  │
        ▼ updates prompt + tools + policy                                  │
        Loop 1 — Agent (runtime, seconds–minutes)                          │
   reason ─► act(tool) ─► observe ─► decide ──► (governed by manifest,     │
        └────────── repeat until stop ──────┘    contracts, approvals) ────┘
        │                    ▲
        ▼ writes             │ retrieves (scoped, cited, retained)
        Loop 1.5 — Memory (agent + platform, across runs)
   write record ─► extract/consolidate ─► retrieve in later run ─► steer
```

Signal flows up and out; constraints flow down and in. The memory loop sits *beside* Loop 1 rather than above it: each run feeds it on the way out and is fed by it on the way in, which is why a poisoned memory bypasses every per-run control. The value of the whole system is proportional to how tightly the outer loops actually reach the inner one — and that reach is engineering work, not a diagram.

---

## How this maps to AaraMinds today

| Loop | What AaraMinds already does | What's proposed / next |
|---|---|---|
| **Loop 1 — Agent** | Governed at design time: manifest tool allowlist, tool contracts, approval boundaries, scoped memory, autonomy classification, and a readiness certificate that gates production. The runtime is *adopted*, not built. | Keep the governance surface current as runtimes evolve. |
| **Loop 1.5 — Memory** | Write and read sides certified: memory-record contract, citation gate, engagement-scoped reads, retention/expiry, claim-key consolidation with audited supersession, and retrieval provenance in the tamper-evident chain — all proven in the harness. | Close extraction: the Mem0 + Azure OpenAI extraction-quality spike (Phase 2), measured against the citation + consolidation contracts. |
| **Loop 2 — Developer** | Readiness rubric + golden eval cases make Loop 2 converge on evidence, not vibes. Scaffold generator produces the spec artifacts. | The IDE accept/reject/edit signal (the proposed extension) is a Loop-2 source that feeds Loop 3. |
| **Loop 3 — Feedback** | Readiness reports and pack scorecards are point-in-time certification — the honest baseline Loop 3 recalibrates against. | The **Continuous Certification & Agent Feedback Loop** (see its BRD): OTel GenAI instrumentation → hash-and-reference traces → mined eval cases → rubric recalibration → re-certification. |

The one-line takeaway for the team: **we already govern Loop 1 at design time and certify it once. The feedback-loop initiative is about making Loop 3 real — turning production experience into re-certification — without becoming the surveillance system we tell customers not to build. The memory loop's write and retrieval sides are certified in the harness; extraction quality is its one remaining open front.**

---

## Decision guide — designing each loop responsibly

When you design or review an agent, confirm each loop has its non-negotiables:

**Loop 1 (Agent) — before it runs in production:**
- [ ] Explicit stopping conditions: max iterations, cost/token budget, wall-clock timeout, no-progress detector.
- [ ] Every tool is on the manifest allowlist, has a contract, validates inputs, and is idempotent if it mutates state.
- [ ] High-risk actions pause for approval; the loop cannot cross the risk threshold alone.
- [ ] Memory is scoped, retained with limits, and provenance-tracked.
- [ ] A current passing readiness verdict exists.

**Loop 1.5 (Memory) — before memory persists across runs:**
- [ ] Every write validates against the memory-record contract, with classification, retention, and source citation.
- [ ] Reads are engagement-scoped; cross-engagement isolation has a passing proof, not a design intention.
- [ ] Expired records are excluded from retrieval.
- [ ] A consolidation rule exists for conflicting or superseding records (in AAP: one active record per claim key; conflicts fail closed unless superseded).
- [ ] Retrieval is auditable: a run can show which memories steered it (in AAP: `memory_retrieved` events in the audit chain; unauditable reads fail closed).

**Loop 2 (Developer) — while building:**
- [ ] Changes are validated against eval cases, not just re-read.
- [ ] Prompt, tool set, policy, and eval cases stay in sync (one spec).
- [ ] Eval set covers the real input distribution, not just the demo.

**Loop 3 (Feedback) — before you ship the loop itself:**
- [ ] Observation uses standardized telemetry (OTel GenAI), versions pinned.
- [ ] Payloads are hashed-and-referenced and tenant-local; consent precedes collection; redaction fails closed.
- [ ] The loop's output is new eval cases + rubric updates, not just dashboards.
- [ ] No autonomous modification or deployment of an agent.
- [ ] There is a named path from a production finding to a re-certification.

---

## Common misconceptions

- **"The agent loop and the feedback loop are the same thing."** No. Loop 1 runs in seconds inside one task and is run by the agent; Loop 3 runs over weeks across many tasks and is run by the organization. Conflating them is the single most common source of confused agent-feedback designs.
- **"Reflection lets the agent fix its own mistakes."** Reflection helps some workflows but is not validation. A wrong model can be confidently wrong about its self-critique. Use external checks.
- **"More agents in the loop is better."** Multi-agent adds coordination cost, latency, and harder debugging. Reach for it when the work genuinely decomposes into specialized roles, not by default.
- **"Guardrails make the loop safe."** Guardrails catch specific known failure classes. Safety is a property of the whole system — stopping conditions, permissions, evaluation, observability — not a component you bolt on.
- **"We tested it, so it works."** A single passing run says nothing about the distribution. That is the entire reason Loop 3 exists.
- **"Memory is just long context."** No. Context dies with the run; memory survives it. A long context window changes how much one Loop 1 iteration can see — it does nothing about poisoning, staleness, or leakage across runs, which are memory-loop failures. Teams that treat memory as "more context" ship the write path and forget they also shipped a steering mechanism for every future run.

---

## Sources

- Andrew Ng's three loops for 0-to-1 AI products (June 2026): https://explainx.ai/blog/andrew-ng-three-loops-0-to-1-products-2026
- "Loop Engineering Emerges as Developers Put AI Coding Agents on Repeat," ADTmag (July 1, 2026): https://adtmag.com/articles/2026/07/01/loop-engineering-emerges-as-developers-put-ai-coding-agents-on-repeat.aspx
- "What Is Loop Engineering?" explainX (2026): https://explainx.ai/blog/what-is-loop-engineering-ai-agents-2026
- OpenTelemetry GenAI semantic conventions — generative AI client spans (status: Development as of April 2026): https://opentelemetry.io/docs/specs/semconv/gen-ai/gen-ai-spans/
- "How OpenTelemetry Traces LLM Calls, Agent Reasoning, and MCP Tools," Greptime (May 9, 2026): https://greptime.com/blogs/2026-05-09-opentelemetry-genai-semantic-conventions

*Internal references: AaraMinds Continuous Certification & Agent Feedback Loop BRD; readiness rubric (`governance/readiness-rubric.yaml`); agent manifest and tool-contract schemas (`schemas/`).*
