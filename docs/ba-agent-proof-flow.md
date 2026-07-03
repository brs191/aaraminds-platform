# BA Agent Proof Flow

This is the implementation-facing proof flow for AAP v1.3.

1. User submits BA engagement brief.
2. Runtime loads `examples/ba-agent.manifest.yaml`.
3. Runtime validates manifest shape, status, skill paths, telemetry payload mode, blocked-actions reference, and pinned tool contracts.
4. Runtime initializes `run_id`, `agent_id`, `manifest_version`, `engagement_id`, `user_id`, and `tenant_namespace`.
5. Runtime opens a trace record and writes `agent_started` and `manifest_validated` audit events.
6. Memory access reads only records scoped to the active `engagement_id`.
7. Agent requests MCP tool call through the tool gateway.
8. Tool gateway checks manifest allowlist, contract version, approval boundary, and engagement scope.
9. Tool executes, is denied, or blocks for approval.
10. Runtime records `ToolInvocation` and `AuditEvent`.
11. BA output separates source-backed facts, assumptions, open questions, risks, recommendations, generated draft content, and evidence references.
12. Output is evaluated against the BA Agent package gate.
13. One real failure is converted into a `SkillRevision` with before/after benchmark evidence.

The current implementation proves steps 2 through 10 with a deterministic local harness.

