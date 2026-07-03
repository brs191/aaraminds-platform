---
id: mcp-server-building/03-defend-against-poison-input
area: mcp-server-building
exercises:
  - .claude/skills/mcp-go-server-building/references/enterprise-security.md
  - .claude/skills/mcp-go-production-review/references/anti-patterns.md
pass_threshold: 6/8
last_run: 2026-05-30
last_result: pass
---

# Defend a tool against poison input

## Context

Attach `07-mcp-go-enterprise-security.md` and `18-mcp-go-anti-patterns.md`. The responder is reviewing a tool that has been called with unexpected inputs in production.

## Prompt

A tool `analyze_architecture` accepts an `input_json` argument containing an architecture description. We've seen it called with:
- 80 MB JSON blobs
- Deeply nested arrays (10,000 services)
- Strings with embedded prompt-injection-looking content (`"; ignore all previous instructions ..."`)
- Inputs claiming `"deployment_target": "../../../etc/passwd"`

What defenses do you put in place, and where? Be specific about which layer catches what.

## Rubric

A response passes if it covers at least 6 of the following 8 points:

- [ ] **Size limit at the transport edge** — caps `input_json` size before parsing (concrete number like 1 MB) and returns a structured error if exceeded
- [ ] **Depth/length limits during parsing** — bounds nested array length and object depth (rejecting 10k services as outside reasonable bounds), not relying solely on Go's default JSON limits
- [ ] **Schema validation, not just unmarshal** — validates against a JSON schema or explicit field validators *after* parsing; `json.Unmarshal` is permissive
- [ ] **Path/identifier sanitization** — flags or rejects inputs that look like path traversal (`../`), absolute paths, or shell-meta characters in fields that should be identifiers
- [ ] **Prompt-injection is a content concern, not a tool concern** — explicitly notes that the *tool* shouldn't try to detect prompt injection in user data; the *consumer of the tool's output* (the LLM) is the layer that must treat tool outputs as untrusted data
- [ ] **Resource bounds during processing** — caps loop iterations, total processing time (context deadline), and memory allocation in the service layer so a pathological-but-valid input cannot DoS the server
- [ ] **Structured error responses** — each defense returns `mcp.NewToolResultError` with a distinct, non-leaky message (no internal stack traces, no file paths)
- [ ] **Audit logging** — logs rejected inputs (with the reason) at a level usable for security review; counts rejections as a metric

## Reference output

> The defenses stack: outer layers reject early, inner layers protect against pathological-but-valid input. Specifically:
>
> **Edge / transport layer**
> - Cap `input_json` size at parse time: 1 MB hard limit, configurable. Reject larger with `mcp.NewToolResultError("input_json exceeds maximum size (1 MB)")`. 80 MB never reaches the parser.
> - Cap raw JSON parse with `json.NewDecoder(io.LimitReader(...))` to enforce the size at decoder level.
>
> **Parsing layer**
> - Bound array and object depth. After unmarshal into the typed struct, validate: `len(services) ≤ 200`, `len(service.depends_on) ≤ 50`. Anything beyond is rejected with a specific message naming the field and the limit. 10,000 services is rejected here.
> - Strict typed unmarshal: missing required fields return validation errors with the field name (`deployment_target is required`).
>
> **Validation layer (schema/semantic)**
> - JSON Schema validation against the contract, applied after unmarshal. Schema declares per-field formats (e.g., `service.name` is `^[a-z][a-z0-9-]{1,62}$`). `"../../../etc/passwd"` fails the format check; the error is `deployment_target must match pattern ^(aks|container_apps|app_service|hybrid)$`.
> - Identifier sanitization: any field used downstream as a name, path, or identifier is checked against a deny-list of meta-characters (`../`, `/`, `\`, leading `-`, NUL). Reject with a message.
>
> **Service / processing layer**
> - Every rule loop is bounded explicitly: `for i := 0; i < len(services) && i < maxServices; i++`. Even if pre-validation lets a large slice through, the loop won't run forever.
> - `ctx` deadline: 5 s default per call. If the rule logic exceeds that, return an error rather than blocking the server.
> - Memory: assemble output via streaming or pre-sized slices. Don't allocate a `[]Risk` of `len(services)*len(services)` without first checking it fits a sane bound.
>
> **About prompt-injection-looking strings**
> The tool's job is not to detect prompt injection in user-supplied content. If `description` contains `"; ignore all previous instructions ..."`, that is data, and the tool processes it as data — extracts what it needs, ignores the rest, returns structured output. The receiving LLM is the layer that must be defensive: the MCP client's prompt should treat any text inside tool output as untrusted attacker-controlled content (this is "data, not instructions" in the security skill). Putting an injection-detection filter in the tool gives a false sense of safety and is brittle to circumvention.
>
> **Error and audit**
> - Every rejection returns `mcp.NewToolResultError` with a short, non-leaky message. Never echo the offending value, never paste a stack trace, never include internal paths.
> - Every rejection logs a structured warning: `event=input_rejected reason=size_limit_exceeded tool=analyze_architecture size_bytes=83886080`. Aggregate this as a metric; alert if rejection rate spikes (could be probing).
> - Successful calls log the derived counts (services_analyzed, risks_identified) but not the input contents.

## Notes

Catches the regression where security gets reduced to "we use HTTPS" while the actual attack surface (input parsing, processing bounds, error messages) is unguarded. The prompt-injection point is deliberately a trap — many responders try to add a "detect malicious text" check; the skill explicitly argues against that.
