# AAP Guardrails Checklist

Status: baseline for PRD v1.3.

## Release Blockers

- Agent cannot start without a valid manifest.
- Every allowed tool has a contract and matching pinned version.
- Off-manifest tool calls fail closed and write `tool_denied`.
- Missing tool contracts are treated as `blocked`.
- Unclassified actions default to `hard`.
- Unattended soft approvals escalate to `hard`.
- Active and platform-ready manifests use `payload_mode: hash-and-reference`.
- Memory writes require classification and source citation.
- Cross-engagement memory reads return no records.
- Blocked actions in `governance/aap-blocked-actions.yaml` never execute in v1.

## OWASP Agentic Review Baseline

- Prompt injection cannot mutate manifest, tool contracts, or approval state.
- Tool outputs are treated as data, not instructions.
- Tool call arguments are validated before execution.
- Sensitive payloads are hashed or referenced in telemetry.
- Human approval is outside the model loop for hard actions.
- All denials, approvals, overrides, eval completions, releases, and purges are audited.

