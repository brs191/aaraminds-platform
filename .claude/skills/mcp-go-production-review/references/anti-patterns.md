# Skill — MCP-Go Anti-Patterns

## Purpose

Identify and correct the design and implementation mistakes that show up over and over in production MCP servers. This skill is named what to avoid, but the deeper purpose is the failure mode behind each: every anti-pattern listed here has caused a real production incident somewhere.

## Use when

The user asks for a review, design critique, production hardening, or "why is this server behaving badly." Also use proactively when reviewing new MCP server designs before they're built.

## How to use this skill

This is not a checklist to skim. For each anti-pattern, the structure is:

- **Bad:** the pattern as it appears in real code
- **Why it fails:** the underlying failure mode
- **Better:** the fix
- **Detection signal:** how to spot it in an existing server

Match anti-patterns against the design or code in front of you. Where multiple anti-patterns appear together — they often cluster — fix the deepest one first.

## Anti-pattern 1 — Raw API wrapper MCP server

**Bad:**
- One MCP tool per REST endpoint of the backend
- Tool names mirror backend method names (`post_servicenow_incident`, `get_azure_v2024_subscription`)
- Tool inputs are direct passthroughs of backend request shapes

**Why it fails:** The MCP server becomes a thin proxy with no abstraction. Every backend change ripples to every agent. Agents have to know backend quirks. New backends require new tools rather than extending existing intents.

**Better:**
- Tools represent user intents: `create_incident`, `get_cost_summary`, `summarize_failed_pipeline`
- A single tool can call multiple backend endpoints internally
- Backend changes are absorbed in the service layer without changing the tool contract

**Detection signal:** Compare tool names against backend API endpoints. If most tools have a 1:1 correspondence with backend methods, the server is a raw wrapper. See `../../mcp-go-server-building/references/tool-design.md` for the intent-based design framework.

## Anti-pattern 2 — Mega tool

**Bad:**
```go
// One tool that does everything
tool := mcp.NewTool("execute_action",
    mcp.WithString("action", mcp.Required()),
    mcp.WithObject("payload", mcp.Required()),
)
```
Variants: `run_query(query)`, `call_backend(method, path, body)`, `do_thing(thing, options)`.

**Why it fails:** No bounded behavior. Authorization can't be per-action because the action is just a string parameter. Risk tier varies per action but the tool has one risk tier. Audit logging loses meaning. Schema validation is impossible. Every "what does this server do" question becomes "depends on the action parameter."

**Better:** Small explicit tools with named intents, structured inputs, defined risk tiers. If you have 30 distinct actions, you have 30 distinct tools. The cost of more tools is much less than the cost of one mega-tool that erodes every other control.

**Detection signal:** Look at the largest tool by code volume. If its primary input is a string that switches on multiple branches, it's a mega-tool. If you find a `switch input.Action` or `switch req.Method` at the top of a handler, the design is broken.

## Anti-pattern 3 — Shell execution tool

**Bad:**
```go
tool := mcp.NewTool("run_command",
    mcp.WithString("command", mcp.Required()),
)
// Handler does exec.Command(input.Command) or similar
```

**Why it fails:** The agent can run arbitrary code on the server. Every security control elsewhere is irrelevant because the agent can bypass them via shell. This is the worst single decision available to MCP server designers and it keeps appearing because it feels expedient.

**Better:** Predefined tools with explicit parameters and allowlists. If the goal is to "run a deploy script," the tool is `deploy_service(service_name, version)` with a hardcoded mapping to the actual deploy command and the parameters that are allowed. Allowlists are explicit, not pattern-based. Dry-run mode supported. Human approval for any non-trivial command.

**Detection signal:** Search the codebase for `exec.Command`, `exec.LookPath`, `os/exec`, or shell-out patterns in tool handlers. Any match is an incident waiting to happen.

## Anti-pattern 4 — Unbounded query tool

**Bad:**
```go
tool := mcp.NewTool("query_logs",
    mcp.WithString("query", mcp.Required()),
)
// Handler runs the query as-is, returns all results
```

**Why it fails:** Unbounded queries return unbounded data. Context windows blow up. Backend systems get hammered. Sensitive data leaks because nobody scoped the query. Cost runs away. The most expensive log-query bills in cloud history come from agents looping on unbounded query tools.

**Better:** Enforce time windows (default to last hour, max 24 hours). Enforce result limits (default 100, max 1000). Require structured filter parameters rather than free-form queries. Provide summarization for large result sets rather than full dumps. Track cumulative cost per session and rate-limit.

**Detection signal:** Tool accepts a free-form query string. Output size has no documented bound. The tool's contract does not specify max-result-count.

## Anti-pattern 5 — No per-tool authorization

**Bad:** A principal authenticates once (OAuth, API key) and can call every tool. Authorization is "are you authenticated."

**Why it fails:** Authentication says who you are. Authorization says what you can do. Treating them as the same control means every tool runs with maximum privilege of every authenticated caller. The lowest-trust caller gets to use the highest-risk tool.

**Better:** Every tool call goes through an authorization check at the handler boundary. Authorization considers: role, tenant scope, environment scope, risk tier of this specific tool. Denial is the default for any combination that isn't explicitly allowed.

**Detection signal:** Tool handlers don't import the policy/authorization package. Or they import it but only call it conditionally ("authorize unless it's a read tool"). Read tools also need authorization — read access can still expose data you shouldn't.

## Anti-pattern 6 — No approval boundary

**Bad:** Agent can call `deploy_to_production`, `delete_resource`, `restart_service`, `change_firewall_rule`, `purchase_reserved_instance` directly, without human approval.

**Why it fails:** The first production-impacting action triggered by a hallucinated tool call is the last day of the project. Even with strong guardrails, LLMs make decisions that look reasonable in isolation but are catastrophic in context. The approval gate is the structural protection.

**Better:** Critical-tier tools require explicit human approval. The approval is part of the tool contract, surfaced in metadata. Dry-run mode shows what would happen. The approval workflow integrates with the organization's existing system (ServiceNow, Slack approval bot, PagerDuty), not a custom flow.

**Detection signal:** Search tool contracts for any tool whose action could affect production state but is documented with `Human Approval: Not required`. Each is a potential incident.

## Anti-pattern 7 — Secrets in tool results

**Bad:** Tool result includes a backend response that happens to contain a connection string, API key, OAuth token, password, or similar.

**Why it fails:** The LLM now has the secret. The LLM's response now potentially contains the secret. The user's chat transcript contains the secret. The audit trail contains the secret. The training data of whoever logs LLM interactions contains the secret. Rotating the secret is the only recovery, and you may not know which secrets leaked.

**Better:** Output redaction at the formatter, before any value leaves the tool handler. Test redaction with explicit secret-containing test fixtures. Never assume "this backend wouldn't return a secret" — backends evolve. The redaction layer is the safety net. See `../../mcp-go-server-building/references/enterprise-security.md` for the redaction pattern.

**Detection signal:** Run every tool's response through a secret-shaped-string scanner. Any match is a defect to fix immediately, not a warning to address later.

## Anti-pattern 8 — Business logic inside MCP handler

**Bad:**
```go
s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    // 200 lines of: argument parsing, Azure SDK calls, business logic,
    // formatting, error handling, audit logging, all in one function
})
```

**Why it fails:** The handler is now the only place this logic exists. It's not reusable. It's not testable without the MCP server running. Schema validation, authorization, audit, business logic, backend calls, and formatting are all entangled. Code review becomes "scroll past 200 lines and hope."

**Better:** Handler is thin. Parse arguments → validate → authorize → call service → format result → audit. Each is one or two lines. Business logic lives in `internal/services/`. Backend calls live in `internal/connectors/`. See `../../mcp-go-server-building/references/project-structure.md` for the layering.

**Detection signal:** Tool handlers longer than 60 lines. Tool handlers that import `azure-sdk-for-go` or other backend client packages directly.

## Anti-pattern 9 — No contract discipline

**Bad:** Tool schemas live only in the Go code that registers them. They change silently. There's no separate contract file. Consumers find out about breaking changes when their integration breaks.

**Why it fails:** Tools are public contracts. Public contracts that aren't versioned, reviewed, or documented separately from implementation will drift. Drift discovered by consumers is more expensive than drift caught at code review.

**Better:** Every tool has a contract file in `contracts/tools/<tool_name>.md` (or equivalent). The contract file is the source of truth — implementation must match. Breaking changes to the contract require a version bump and consumer-impact assessment. Contract tests verify that the implementation matches the contract.

**Detection signal:** No `contracts/` directory. Or a `contracts/` directory that's out of sync with what the tools actually do. Or contracts that exist but no automation verifies the implementation matches them.

## Anti-pattern 10 — No observability

**Bad:** Errors are printed to stdout. There's no metrics export. There's no tracing. When something goes wrong in production, the only diagnostic is "the agent's response was weird."

**Why it fails:** Production debugging without instrumentation is archaeological. By the time you figure out what happened, the incident is over but the root cause isn't fixed because you can't reproduce it.

**Better:** Structured logs (slog, JSON, stderr), metrics (OpenTelemetry with named counters and histograms for tool calls, durations, denials, redaction hits), distributed traces (OpenTelemetry, propagated end-to-end). Audit events separate from application logs. See `../../mcp-go-server-building/references/observability.md` for the implementation patterns.

**Detection signal:** Search for `fmt.Println`, `log.Println`, or `os.Stdout` in production code paths. Stdio servers must use stderr for logs (stdout is the protocol wire). Any use of stdout for logging is a defect.

## Anti-pattern 11 — Cross-tenant data leakage via ambient state

**Bad:** Tool reads "the current tenant" from a global variable, thread-local context, or implicit session state. Tenant ID is not an explicit parameter on every function in the call chain.

**Why it fails:** Ambient state is shared state. A request from tenant A sets the global, then a concurrent request from tenant B reads the global, then tenant B's tool call returns tenant A's data. The variant: middleware sets context, but a helper function bypasses context and reads from a default tenant.

**Better:** Tenant ID is an explicit parameter on every tool handler, every service method, every connector call. There is no "current tenant" anywhere in the codebase. Tenant validation against the authenticated identity happens at the handler boundary. Tools that forget to pass tenant down would fail to compile because the lower layers require it.

**Detection signal:** Search for "current_tenant", "default_tenant", "context.WithValue" patterns that set tenant in middleware. Any of these is a leakage risk waiting to happen.

## Anti-pattern 12 — Approval bypass without audit differentiation

**Bad:** Critical-tier tools have an approval gate, but the gate accepts a "bypass" parameter or recognizes a special "break-glass" role that skips approval. The bypass is logged at the same level as a normal approval.

**Why it fails:** Break-glass exists because emergencies happen. But if a break-glass usage is logged identically to a normal approval, weekly review can't distinguish "we used break-glass three times this week because of incidents" from "we used break-glass three times this week because someone wanted to skip the workflow."

**Better:** Break-glass usage produces a distinct audit event (`approval_bypass: true`). Alerts fire on every bypass. Weekly review of bypass usage is mandatory. Break-glass requires a paired post-incident review documenting the reason.

**Detection signal:** Search audit events for any field that might indicate bypass. If you can't find one, either there's no bypass path (good) or there's a bypass path that isn't audited differentially (bad).

## Anti-pattern 13 — Unbounded tool result size

**Bad:** Tool returns whatever the backend returned. No max-result limit. No payload size limit. No truncation indicator.

**Why it fails:** A tool that "occasionally returns a 5MB response" destroys the agent's context window, masks data within the response (the model only attends to part of it), and can leak data outside the intended scope. This was implicitly mentioned in anti-pattern 4 (unbounded queries) but also applies to tools whose query is bounded but whose result still has no size cap.

**Better:** Every tool has a documented max payload size (e.g., 100KB) and max result count (e.g., 100 items). Implementations enforce both with explicit truncation. Truncation indicators tell the agent "more results available, use pagination." Pagination tokens enable follow-up calls when more data is genuinely needed.

**Detection signal:** Tool documentation that does not state max-result-count or max-payload-size. Implementation that does not check size before returning.

## Anti-pattern 14 — Stale dependency pins

**Bad:** `go.mod` pins `github.com/mark3labs/mcp-go v0.12.0` from 18 months ago. Or `go 1.21`. Or transitive dependencies that haven't been updated.

**Why it fails:** MCP spec versions evolve. Security patches accumulate. The further behind you fall, the more expensive catching up becomes. At some point you discover you're vulnerable to a known CVE that was fixed 14 months ago in a version you haven't picked up.

**Better:** Dependency-update discipline. Renovate or Dependabot in CI. Quarterly review of major dependencies even if no advisory has surfaced. Pin Go version to a currently-supported major (1.25 or 1.26 as of May 2026). Pin MCP SDK version with explicit upgrade path documented. Re-verify pinned versions against `../../mcp-go-server-building/references/ecosystem-facts.md` quarterly.

**Detection signal:** Compare `go.mod` Go version against the currently-supported Go versions. Compare pinned MCP SDK version against the current release. Any gap of more than two minor versions is a defect to fix.

## Meta anti-pattern — Adding all the rules at once

**Bad:** Reading this skill and trying to apply all 14 anti-patterns to an existing server in one pass.

**Why it fails:** Refactor everything at once and you'll break working code. The team gets demoralized. Half-refactored servers are worse than not-refactored servers because the inconsistency hides defects.

**Better:** Apply anti-pattern fixes one at a time, with tests, in priority order. The priority order in production-impact terms:

1. Anti-pattern 3 (shell execution) — fix immediately if present, it's a critical vulnerability
2. Anti-pattern 6 (no approval boundary) for critical-tier tools — fix before any production deployment
3. Anti-pattern 11 (cross-tenant leakage via ambient state) — fix when found, exposure compounds with traffic
4. Anti-pattern 7 (secrets in results) — automated scan, fix any match
5. Anti-pattern 5 (no per-tool authorization) — add the framework, then expand coverage
6. Anti-pattern 4, 13 (unbounded queries, unbounded results) — add limits, alert on truncation
7. Anti-pattern 1, 2 (raw wrappers, mega tools) — redesign as tools come up for change
8. Anti-pattern 8 (business logic in handlers) — refactor when touching the handler
9. Everything else — opportunistic improvement

## What to read next

- For the tool-design framework that avoids most of these: `../../mcp-go-server-building/references/tool-design.md`
- For the security framework that addresses anti-patterns 5, 6, 7, 11, 12: `../../mcp-go-server-building/references/enterprise-security.md`
- For the project structure that avoids anti-pattern 8: `../../mcp-go-server-building/references/project-structure.md`
- For the observability that surfaces detection signals: `../../mcp-go-server-building/references/observability.md`
