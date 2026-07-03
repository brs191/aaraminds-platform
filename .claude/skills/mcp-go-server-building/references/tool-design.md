# Skill — MCP-Go Tool Design

## Purpose

Design MCP tools that an agent can call safely and a security reviewer can sign off on. This is where most production MCP server defects originate — wrong tool boundaries get baked in during the first sprint and stay broken for the life of the server.

## The principle

An MCP tool exposes an **intent-safe capability**, not a raw backend API. The difference is the load-bearing distinction in this skill.

A raw API wrapper exposes whatever the backend does. The agent has to know the backend's quirks, parameter conventions, and failure semantics. Every backend change ripples out to every agent.

An intent-safe capability exposes what an agent or human user actually wants to accomplish. The tool handles backend quirks internally. The agent sees a stable contract. Backend changes are contained.

**Test for whether a tool is intent-safe:** read the tool name and description aloud. Does it describe what a user wants done, or does it describe a method on an API? If it's the latter, the design is wrong.

## Good and bad tool names

Intent-safe (good):
- `get_cost_summary` — user wants to know what they spent
- `detect_cost_anomalies` — user wants to find unexpected spend
- `query_recent_orders` — user wants order data with bounded scope
- `summarize_failed_pipeline` — user wants a pipeline failure explained
- `recommend_microservice_patterns` — user wants pattern suggestions for their context
- `create_incident_ticket` — user wants a ticket created with structured intent

Raw API wrapper (bad):
- `call_api` — exposes the verb, not the intent
- `run_command` — exposes a runtime, not a capability
- `execute_query` — turns the agent into a database client
- `do_action` — meaningless beyond "something happens"
- `post_to_servicenow` — exposes the backend, not the user goal
- `get_via_rest` — protocol leakage into the tool name

## Risk-tier framework

Every tool falls into one of four risk tiers. The tier determines authentication, authorization, audit, approval, and rate-limiting decisions. Make the tier explicit in the tool's contract.

| Tier | Description | Examples | Required controls |
|---|---|---|---|
| **Low** | Read-only, bounded, non-PII | `list_public_documentation`, `get_service_catalog`, `summarize_recent_logs` (last hour, bounded results) | Authentication, audit log, rate limit |
| **Medium** | Read-only, broader scope or sensitive data | `get_cost_summary` (financial data), `query_user_activity` (PII), `get_incident_details` | Tier-1 + tenant-scoped authorization, output redaction |
| **High** | Write actions, or read with destructive potential | `create_incident_ticket`, `update_service_metadata`, `query_unbounded_logs` | Tier-2 + per-tool authorization, dry-run mode, structured audit |
| **Critical** | Production-impacting writes, destructive actions, financial impact | `restart_service`, `deploy_to_prod`, `delete_resource`, `purchase_reserved_instances` | Tier-3 + human approval workflow, blast-radius assessment, rollback plan |

A tool with risk level Critical that lacks human approval is not a tool design — it's an incident waiting to happen. The framework is enforcement, not paperwork.

## Tool contract template

Every tool gets a contract file (typically `contracts/tools/<tool_name>.md`) with this structure. This is the source of truth — implementation must match the contract, not the other way around.

```markdown
## Tool Name
`tool_name` — intent-based, snake_case, stable

## Purpose
One sentence stating the user intent this tool serves.

## Input Schema
Each field: type, required/optional, allowed values or pattern, max length where bounded.

## Output Schema
Expected response shape, bounded sizes (max items, max payload).

## Risk Tier
Low / Medium / High / Critical (see framework above).

## Authorization
Required role, tenant scope, environment scope (dev/staging/prod).

## Human Approval
Required / Not required / Conditional on parameters (state condition).

## Failure Modes
Expected error cases, recovery semantics, what the agent should retry vs. surface to user.

## Observability
Log event name, metric names, trace attributes, audit event shape.

## Compatibility
Initial version. Breaking-change policy: this contract is versioned.
```

## Implementation patterns

### Official SDK pattern

```go
package cost

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type CostSummaryInput struct {
	SubscriptionID string `json:"subscription_id" jsonschema:"Azure subscription ID, must be valid UUID"`
	FromDate       string `json:"from_date" jsonschema:"start date in YYYY-MM-DD format"`
	ToDate         string `json:"to_date" jsonschema:"end date in YYYY-MM-DD format"`
	Granularity    string `json:"granularity,omitempty" jsonschema:"daily or monthly, default daily"`
}

type CostSummaryOutput struct {
	SubscriptionID string          `json:"subscription_id"`
	TotalCost      float64         `json:"total_cost"`
	Currency       string          `json:"currency"`
	Items          []CostBreakdown `json:"items"`
}

func GetCostSummary(ctx context.Context, req *mcp.CallToolRequest, input CostSummaryInput) (*mcp.CallToolResult, CostSummaryOutput, error) {
	// Validation
	if err := validateUUID(input.SubscriptionID); err != nil {
		return nil, CostSummaryOutput{}, fmt.Errorf("subscription_id must be a valid UUID: %w", err)
	}
	if err := validateISODate(input.FromDate); err != nil {
		return nil, CostSummaryOutput{}, fmt.Errorf("from_date must be YYYY-MM-DD: %w", err)
	}
	// Authorization (tenant-scoped, tier-medium)
	if err := authz.Check(ctx, input.SubscriptionID, "cost.read"); err != nil {
		return nil, CostSummaryOutput{}, err
	}
	// Service-layer call (handler does not contain business logic)
	out, err := svc.GetCostSummary(ctx, input)
	if err != nil {
		return nil, CostSummaryOutput{}, err
	}
	// Audit (every successful call, bounded payload)
	auditor.Record(ctx, audit.Event{Tool: "get_cost_summary", Subject: input.SubscriptionID, Status: "success"})
	return nil, out, nil
}
```

Tool registration:

```go
mcp.AddTool(server,
	&mcp.Tool{Name: "get_cost_summary", Description: "Get Azure cost summary for a subscription and date range"},
	GetCostSummary,
)
```

### mark3labs/mcp-go pattern

```go
package cost

import (
	"context"
	"encoding/json"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerGetCostSummary(s *server.MCPServer, svc *Service, authz *Authorizer, auditor *Auditor) {
	tool := mcp.NewTool("get_cost_summary",
		mcp.WithDescription("Get Azure cost summary for a subscription and date range"),
		mcp.WithString("subscription_id", mcp.Required(), mcp.Description("Azure subscription ID, must be valid UUID")),
		mcp.WithString("from_date", mcp.Required(), mcp.Description("start date YYYY-MM-DD")),
		mcp.WithString("to_date", mcp.Required(), mcp.Description("end date YYYY-MM-DD")),
		mcp.WithString("granularity", mcp.Description("daily or monthly, default daily")),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		subscriptionID, err := req.RequireString("subscription_id")
		if err != nil {
			return mcp.NewToolResultError("subscription_id is required"), nil
		}
		if err := validateUUID(subscriptionID); err != nil {
			return mcp.NewToolResultError("subscription_id must be a valid UUID"), nil
		}
		fromDate, _ := req.RequireString("from_date")
		if _, err := time.Parse("2006-01-02", fromDate); err != nil {
			return mcp.NewToolResultError("from_date must be YYYY-MM-DD"), nil
		}
		toDate, _ := req.RequireString("to_date")
		if _, err := time.Parse("2006-01-02", toDate); err != nil {
			return mcp.NewToolResultError("to_date must be YYYY-MM-DD"), nil
		}
		granularity := req.GetString("granularity", "daily")

		// Authorization (tenant-scoped, tier-medium)
		if err := authz.Check(ctx, subscriptionID, "cost.read"); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		// Service-layer call
		out, err := svc.GetCostSummary(ctx, CostSummaryRequest{
			SubscriptionID: subscriptionID,
			FromDate:       fromDate,
			ToDate:         toDate,
			Granularity:    granularity,
		})
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		// Audit
		auditor.Record(ctx, AuditEvent{Tool: "get_cost_summary", Subject: subscriptionID, Status: "success"})

		// Bounded output
		b, _ := json.MarshalIndent(out, "", "  ")
		return mcp.NewToolResultText(string(b)), nil
	})
}
```

## Output bounding — non-negotiable

Every tool must bound its output. Unbounded output is one of the most expensive failure modes in MCP servers — it produces context-window-blowing responses that destroy the agent's reasoning quality and can leak data that should never have been in the response.

Bounding rules:
- **Maximum items returned** — every list-returning tool has a max-results parameter with a server-enforced ceiling (e.g., 100 items maximum even if the client asks for 10,000)
- **Maximum payload size** — total response size capped (e.g., 100KB), with truncation indicators when the cap is hit
- **Field-level pruning** — return only what the agent needs; don't proxy raw backend responses verbatim
- **Pagination tokens** — for genuinely large result sets, return a cursor for follow-up calls rather than dumping everything

A tool that "occasionally returns a 5MB response" is a defect, not an edge case.

## Validation discipline

Every input field gets validated against its documented constraint at the tool boundary. Three layers of validation:

1. **Type validation** — JSON Schema or SDK-level type checking (free if you use the official SDK's typed handlers)
2. **Format validation** — UUIDs are UUIDs, dates are dates, emails are emails. Not "non-empty strings."
3. **Semantic validation** — does this subscription ID exist in this tenant? Is this date range allowed for this user?

Layer 1 happens automatically with the official SDK and explicitly with mark3labs/mcp-go. Layers 2 and 3 are always your code.

## Safety checklist before shipping a tool

Before marking a tool as production-ready, every box must be checked:

- [ ] Tool name describes user intent, not backend method
- [ ] Description is one sentence, action-oriented, names the data scope
- [ ] Input schema is explicit (every field has a type and constraint)
- [ ] Output is bounded (max items, max payload, field-pruned)
- [ ] Risk tier is documented in the contract
- [ ] Authorization check happens at the handler boundary, not deep in the service layer
- [ ] Audit event recorded for every call (success and failure)
- [ ] No secrets leak in any output path (tested explicitly — see `enterprise-security.md`)
- [ ] No business logic in the handler — handler parses, validates, authorizes, calls service, formats result
- [ ] Failure modes are differentiated (input error vs. auth error vs. backend error vs. timeout)
- [ ] High/Critical tools have a dry-run mode and approval workflow

## Common failure modes with detection signals

**Symptom:** Agent calls a tool, gets a response, but the response is wrong or incomplete.
**Likely cause:** Schema drift — the tool's documented contract differs from what it actually returns. **Detection:** Contract tests against golden response shapes.

**Symptom:** Tool calls succeed in dev but fail in production with auth errors.
**Likely cause:** Authorization check uses a tenant scope that exists in dev but not in production tenant config. **Detection:** Cross-environment integration tests that exercise the auth path explicitly.

**Symptom:** Response includes a field that looks like a secret (long random string, "key" in field name).
**Likely cause:** Output redaction was added late, missed this field. **Detection:** Automated scan of every tool's response shape for secret-like patterns before deployment.

**Symptom:** Tool occasionally takes 30 seconds, sometimes times out.
**Likely cause:** Backend call has no timeout, gets blocked on a slow query. **Detection:** Every backend call wrapped in `context.WithTimeout` with explicit bound; alert when P99 latency exceeds the bound.

**Symptom:** Different agents get different results for the same tool call.
**Likely cause:** Tool is reading user-scoped state without scoping consistently (e.g., timezone-dependent date logic). **Detection:** Tool inputs should be sufficient to reproduce the output deterministically; if not, the tool is leaking ambient state.

## What to read next

- For project structure that supports this tool design: `project-structure.md`
- For the security controls that gate tool authorization: `enterprise-security.md`
- For the anti-patterns this skill helps you avoid: `../../mcp-go-production-review/references/anti-patterns.md`
