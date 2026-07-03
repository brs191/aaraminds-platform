# StressTest_Module8

## Purpose

Validation prompts for `08_AI_Agent_Blueprint_System_v1.1.md` and `AaraMinds_AI_Agent_Blueprint_Advisor_v1.0.md`.

Use these to confirm the advisor preserves the original stable Agent Blueprint behavior inside the active Persona system.

## Golden Prompt 1 — FinOps AI Agent

```text
Design a FinOps AI Agent that detects cloud cost anomalies, explains drivers, recommends savings actions, and routes high-risk savings actions for approval.
```

Expected checks:

- Explicitly justifies why an agent is needed beyond dashboarding or scheduled reporting.
- Phrases unsupported numeric improvement targets as targets or marks them `[VERIFY]`.
- Defaults to single-agent unless complexity justifies otherwise.
- Names the concrete failure mode of the rejected multi-agent alternative.
- Gives a default framework/runtime choice with switch conditions if frameworks are recommended.
- Names the environment assumption behind the framework/runtime default.
- Separates deterministic math from LLM reasoning.
- Defines In scope / Out of scope / Human-only.
- Includes approvals, audit logs, cost telemetry, rollback, and evaluation.
- Groups evaluation scorers by intent where useful.
- Includes acceptance criteria for future systems review and re-review triggers.
- Produces Mermaid sequence with anomaly path, approval/error branch, post-approval handoff, and rejection/change-request path.
- Includes a dedicated operational-constraint callout slot in the architecture poster specification.

## Golden Prompt 2 — Incident Triage Agent

```text
Build an Incident Triage agent that handles alert intake, classifies severity, recommends runbooks, and manages on-call handoff for enterprise ops teams.
```

Expected checks:

- Explicitly justifies why an agent is needed beyond static alert routing.
- Optimizes for latency and reliability.
- Names the concrete failure mode of the rejected multi-agent alternative.
- Defines human-only boundaries for incident ownership and high-risk remediation.
- Includes traceability, escalation, feedback loop, and audit sampling.
- Groups evaluation scorers by intent where useful.
- Includes acceptance criteria for future systems review and re-review triggers.
- Produces Mermaid sequence with happy path and escalation/error branch.

## Golden Prompt 3 — TokenOptimizer Agent

```text
Build a TokenOptimizer agent that reduces unnecessary token usage across prompts, context, memory, RAG results, tool outputs, and multi-agent communication while preserving accuracy and intent.
```

Expected checks:

- Explicitly justifies why an agent is needed beyond static prompt compression.
- Tests whether multi-agent is justified.
- If multi-agent, explains distinct specialist roles and rejected single-agent alternative.
- Names the concrete failure mode of the rejected single-agent alternative.
- Establishes self-funding economic discipline or equivalent cost-governed operating constraint.
- Includes cost-per-reliable-outcome, regression evaluation, and rollback.
- Groups evaluation scorers by intent where useful.
- Includes acceptance criteria for future systems review and re-review triggers.

## Pressure Prompt — Agent Justification

```text
Design an AI agent that checks a folder every night, renames files using a fixed naming convention, and emails me a summary.
```

Expected checks:

- Does not force an agent design.
- Recommends scheduled automation or workflow engine unless autonomy requirements are added.
- Explains what additional complexity would justify an agent.
- Does not produce full systems review acceptance criteria unless an agent blueprint is still requested.
