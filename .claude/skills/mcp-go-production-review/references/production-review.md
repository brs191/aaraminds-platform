# Skill — MCP-Go Production Review

## Purpose

Conduct a pre-production review of an MCP-Go server. This is a checklist-driven assessment that confirms a server is ready to handle real traffic, real users, and real failure modes. It pulls together concerns from server-basics, tool design, security, observability, testing, and deployment into one structured pass before the production tag.

## When to run a production review

- Before the first production deploy of a new MCP server.
- Before opening a server to external users (any external user is "production traffic", even if internal infrastructure-wise).
- After major architectural changes (transport switch, new authentication scheme, new tool with state-changing behaviour).
- Quarterly, as a hygiene pass, even when nothing has changed — drift is the silent enemy.

## Review sections

### 1. Server skeleton

- [ ] `main.go` is minimal: configuration parsing, dependency wiring, transport choice, signal handling. No business logic.
- [ ] Transport choice is correct for the deployment target.
- [ ] Structured JSON logging via `slog`, going to `os.Stderr` (always, but mandatory for stdio).
- [ ] Bounded shutdown via `signal.NotifyContext` covering SIGINT and SIGTERM.
- [ ] No `panic` on startup errors that aren't truly unrecoverable; return error and exit non-zero with a logged message.

### 2. Tool design

- [ ] Each tool has a single, clearly named intent. No `manage_*` or `do_*` kitchen-sink tools.
- [ ] Each tool has a typed Go input struct, not `map[string]any` at the handler.
- [ ] Each tool's description tells the agent *when* to use it, not just *what* it does.
- [ ] Each state-changing tool is tagged as high-risk-tier and has a dry-run mode or two-step (plan/apply) shape.
- [ ] Each tool's contract is in `contracts/architecture-tools/implemented/<name>.md`.

### 3. Project structure

- [ ] Service logic in `internal/services/<name>/`; MCP wiring in `internal/tools/<name>/register.go`.
- [ ] No business logic in `register.go` (the handler is parse-validate-call-format).
- [ ] No package-level globals for state (loggers passed via parameter, not `slog.Default()`).
- [ ] Each service package has table-driven tests in `service_test.go`.

### 4. Security

- [ ] Risk tiers assigned per tool; high-risk tools have stronger logging and dry-run.
- [ ] Authentication required at the transport layer (OAuth/JWT for HTTP; process-boundary auth for stdio).
- [ ] Authorisation per tool — not just "is the caller authenticated" but "is this identity allowed to call this tool".
- [ ] No secrets in env vars, config files, or source. Key Vault + Managed Identity for everything.
- [ ] Inputs are size-bounded, depth-bounded, and pattern-validated before reaching the service layer.
- [ ] No prompt-injection defence in the server itself; instead, the client-side prompt frames tool output as data. Documented.
- [ ] Audit log for every state-changing tool call with `{identity, tool, arguments_hash, decision, latency}`.

### 5. Observability

- [ ] `tool_call_started`, `tool_call_completed`, `tool_call_rejected`, `tool_call_errored` events emitted per call.
- [ ] Every log line includes `tool` as a structured field for per-tool dashboards.
- [ ] No full `input_json` or raw output logged; derived counts only.
- [ ] Per-tool latency histograms, success/error counts, derived domain counters.
- [ ] At least one alert: error rate > N% over M minutes, paging on-call.
- [ ] SLOs declared: availability, P99 latency.
- [ ] For HTTP transport: distributed tracing across the request boundary with `traceparent` propagation.

### 6. Testing

- [ ] Unit tests for pure helpers.
- [ ] Handler tests for missing input, invalid JSON, service-error path, valid path.
- [ ] Service-layer table-driven tests covering every named rule (positive + negative cases).
- [ ] Contract test for `tools/list` catalog.
- [ ] Security test suite: oversized input, schema-bypass, path-traversal, auth-bypass.
- [ ] `go test ./...`, `go test -race ./...`, `go vet ./...`, `gofmt -l .` all clean in CI.
- [ ] Code coverage isn't a goal; tests exist for every named rule and every error path.

### 7. Deployment

- [ ] Multi-stage Dockerfile producing a distroless, non-root, stripped, no-CGO image under ~25 MB.
- [ ] Health probes on `/healthz`, not on `/`. Probe exercises actual readiness.
- [ ] Resource limits set to measured P99 usage + 20% margin, not arbitrary defaults.
- [ ] Multi-revision rollout configured (Container Apps Multiple mode, AKS rolling, etc.).
- [ ] Documented rollback procedure: single command flips traffic to previous revision.

### 8. Documentation

- [ ] `README.md` describes purpose, build, run, configuration env vars, transport options.
- [ ] Each implemented tool has a contract under `contracts/architecture-tools/implemented/`.
- [ ] Threat model documented (or referenced from a separate document).
- [ ] Runbook for operational events: alert response, rollback, manual recovery, contact rotation.

### 9. Freshness

- [ ] Ecosystem facts dated within the last quarter.
- [ ] Skill content references match the SDK version actually in use.
- [ ] No `TODO`, `FIXME`, `XXX` in user-facing docs.

### 10. Validation evidence

- [ ] Per-skill evals exist for the patterns this server depends on and pass.
- [ ] Demo or smoke-test artifact runs cleanly against the production build.

## Producing the review report

A production review produces a single document with:

1. **Verdict.** Ready to ship / conditionally ready (with named conditions) / not ready (with blockers).
2. **Findings.** One section per review area above, with `pass / soft-fail / hard-fail` per checklist item and a one-line note where relevant.
3. **Blockers.** Issues that must be fixed before deploy.
4. **Follow-ups.** Issues that can be addressed post-deploy.
5. **Re-review trigger.** When the next review happens (e.g., before next major version, or quarterly).

A reviewer who can't articulate a finding in one line is reviewing too abstractly; require specificity.

## What to read next

- `../../mcp-go-server-building/references/enterprise-security.md` — the security checklist in depth
- `../../mcp-go-server-building/references/observability.md` — observability checklist in depth
- `testing.md` — testing checklist in depth
- `deployment.md` — deployment checklist in depth
- `../../../../validation/governance/release-checklist.md` — the cross-pack release runbook
