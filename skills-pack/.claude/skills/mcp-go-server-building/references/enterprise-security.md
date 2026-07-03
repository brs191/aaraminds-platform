# Skill — MCP-Go Enterprise Security

## Purpose

Apply the security controls an enterprise MCP server actually needs in production — not a checklist of "good ideas" but a defense-in-depth framework where each layer has a specific job, the layers compose, and the gaps where attacks land are explicit.

## Threat model alignment

This skill implements the controls named in `threat-model/mcp-server-threat-model.md`. The threat model identifies eight primary risks; this skill walks through how to address each in Go code:

1. Prompt injection from resource/tool output
2. Tool misuse by agent
3. Cross-tenant data exposure
4. Over-privileged connectors
5. Unbounded API cost
6. Lack of auditability
7. Destructive production changes
8. Secret leakage

Address every one of these explicitly. A control that "we'll add later" is a control that ships disabled.

## Defense-in-depth framework — eight layers

Each layer has one job. Each is necessary; none is sufficient. The strength of the framework is composition, not any single control.

### Layer 1 — Authentication

Establish identity. Never run authenticated as an effectively-shared identity (a single service principal that every user inherits).

For Azure-hosted MCP servers (Container Apps, App Service):

- **Inter-service identity:** Managed Identity. Your MCP server uses a system-assigned or user-assigned managed identity to access Azure backends (Key Vault, Cost Management, Storage). This is non-negotiable for production — connection strings stored in config are a leak waiting to happen.
- **Client identity:** OAuth 2.1 with Entra ID for human-fronted clients. API keys via header (`x-api-key` or similar) for service-to-service when OAuth is heavyweight. Both go through Entra ID for issuance, not server-local databases.

Real authentication middleware (mcp-go style with streamable HTTP):

```go
func AuthMiddleware(validator TokenValidator, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := extractBearerToken(r.Header.Get("Authorization"))
		if token == "" {
			http.Error(w, "missing bearer token", http.StatusUnauthorized)
			return
		}
		identity, err := validator.Validate(r.Context(), token)
		if err != nil {
			// Do not leak validation details in the response
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(r.Context(), identityKey, identity)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
```

Validator checks: signature, expiry, issuer (Entra tenant), audience (this server), and required scopes/claims.

### Layer 2 — Authorization (per-tool)

Authentication says who you are. Authorization says what you can do. **Every tool call goes through authorization at the handler boundary, regardless of how trusted the caller seems.**

```go
type AuthzDecision struct {
	Allowed bool
	Reason  string
}

type Authorizer interface {
	Check(ctx context.Context, principal Identity, toolName string, resource string, action string) AuthzDecision
}

// In the tool handler
decision := authz.Check(ctx, identity, "get_cost_summary", subscriptionID, "cost.read")
if !decision.Allowed {
	auditor.Record(ctx, audit.Event{
		Tool:    "get_cost_summary",
		Subject: subscriptionID,
		Status:  "denied",
		Reason:  decision.Reason,
	})
	return mcp.NewToolResultError("not authorized"), nil
}
```

Authorization checks must include:
- **Role** — does this principal have the role required for this tool's risk tier?
- **Tenant scope** — is this principal allowed to act on this resource's tenant?
- **Environment scope** — production tools have stricter checks than staging/dev
- **Risk-tier check** — Critical-tier tools require approval workflow regardless of role

### Layer 3 — Tenant isolation

Cross-tenant data exposure is one of the most expensive MCP failure modes. A tool reads "the user's data," forgets to scope by tenant, and returns data from another tenant to the wrong user.

Pattern: every tenant-scoped tool takes the tenant ID as an explicit parameter, validates it against the authenticated identity's allowed tenants, and uses it as a filter in every downstream backend call.

```go
// In the tool handler
if !identity.HasTenantAccess(input.TenantID) {
	return mcp.NewToolResultError("tenant access denied"), nil
}
// Pass tenant down explicitly; never let it be inferred from ambient state
result, err := svc.GetCostSummary(ctx, input.TenantID, input.FromDate, input.ToDate)
```

Service layer takes tenant ID as a parameter. Connectors take tenant ID as a parameter. There is no global "current tenant" anywhere in the call chain. Tenant leaks happen when one layer assumes another layer scoped the query.

### Layer 4 — Secrets management

Three rules:

1. **Never store secrets in config files, environment variables (for write secrets), or code.** Use Azure Key Vault (or an equivalent managed secrets store). Read secrets at startup via Managed Identity; refresh on rotation.

2. **Never return secrets in tool outputs.** Even when the agent "needs" the value. Especially when the agent "needs" the value. If a tool's correctness genuinely depends on returning a secret, redesign the tool — the agent should call a backend that uses the secret, not receive the secret.

3. **Never log secrets.** Audit events, application logs, request traces — all must redact before emitting.

Implementation:

```go
type SecretLoader interface {
	Get(ctx context.Context, name string) (string, error)
}

// KeyVaultLoader uses azidentity + azkeys to fetch with Managed Identity
type KeyVaultLoader struct {
	client *azkeys.Client
}

func (l *KeyVaultLoader) Get(ctx context.Context, name string) (string, error) {
	resp, err := l.client.GetSecret(ctx, name, "", nil)
	if err != nil {
		return "", fmt.Errorf("get secret %s: %w", name, err)
	}
	return *resp.Value, nil
}
```

Secrets are loaded into the connectors that need them. Not into a global config struct that everything reads from.

### Layer 5 — Output redaction

Even with best intentions, secrets sometimes slip into output paths. Output redaction is the safety net.

```go
package security

import (
	"regexp"
	"strings"
)

var (
	// Patterns that indicate secrets in unstructured output
	secretPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)(password|secret|token|api[_-]?key|connection[_-]?string)\s*[=:]\s*\S+`),
		regexp.MustCompile(`Bearer\s+[A-Za-z0-9._-]+`),
		regexp.MustCompile(`-----BEGIN [A-Z ]+-----[\s\S]+?-----END [A-Z ]+-----`),
	}
	// Field names that should never appear in tool output
	forbiddenFields = []string{"password", "secret", "token", "api_key", "connection_string", "private_key"}
)

func RedactString(s string) string {
	for _, p := range secretPatterns {
		s = p.ReplaceAllString(s, "[REDACTED]")
	}
	return s
}

func RedactStruct(v any) any {
	// Walk the struct, redact any field whose JSON tag matches forbiddenFields,
	// and apply RedactString to all string field values.
	// Implementation uses reflect; details omitted for brevity.
	return walkAndRedact(v, forbiddenFields)
}
```

Apply redaction at the formatter — the last step before returning the `CallToolResult`. Not earlier; you want internal logs to see the unredacted values for debugging, but external output never does.

The pack's existing redaction implementation (14 lines, substring match against a fixed list) is illustrative, not production. The version above is closer to production-grade but still simpler than what a regulated environment would require.

### Layer 6 — Audit logging

Every tool call produces an audit event. Every. Single. One. Success, failure, denial, timeout. Audit is the answer to "what did this server do at 3am on a Tuesday."

```go
type AuditEvent struct {
	Timestamp     time.Time              `json:"timestamp"`
	Tool          string                 `json:"tool"`
	Subject       string                 `json:"subject"`               // tenant or resource ID
	Identity      string                 `json:"identity"`              // principal ID
	IdentityType  string                 `json:"identity_type"`         // user / service_principal / managed_identity
	Status        string                 `json:"status"`                // success / failure / denied / timeout
	Reason        string                 `json:"reason,omitempty"`
	RiskTier      string                 `json:"risk_tier"`             // low / medium / high / critical
	RequestID     string                 `json:"request_id"`
	DurationMs    int64                  `json:"duration_ms"`
	InputSummary  map[string]any         `json:"input_summary"`         // redacted, key parameters only
	OutputSummary map[string]any         `json:"output_summary,omitempty"` // size, type, no content
}
```

Where to send audit events:

- **Production:** dedicated audit sink (Azure Monitor Log Analytics, dedicated event hub, SIEM). Append-only, retention per compliance requirement.
- **Development:** structured logs to stderr.
- **Never:** the same logger as application logs. Audit events are tamper-evident records; mixing them with debug logs makes both worse.

### Layer 7 — Approval workflow for high-risk tools

Critical-tier tools (production-impacting writes, destructive actions, financial impact) require human approval before execution. The approval is part of the tool's contract, not an afterthought.

Pattern:

```go
type ApprovalRequest struct {
	Tool       string
	Subject    string
	Inputs     map[string]any
	Requestor  string
	ExpiresAt  time.Time
}

type ApprovalDecision struct {
	Approved   bool
	Approver   string
	Reason     string
	ApprovedAt time.Time
}

type ApprovalGate interface {
	// Request opens an approval workflow and returns either an immediate decision
	// (if pre-approved by policy) or a pending state with a callback channel.
	Request(ctx context.Context, req ApprovalRequest) (ApprovalDecision, error)
}
```

In the tool handler:

```go
if riskTier == "critical" {
	decision, err := approval.Request(ctx, ApprovalRequest{...})
	if err != nil || !decision.Approved {
		return mcp.NewToolResultError("approval required and not granted"), nil
	}
	auditor.Record(ctx, audit.Event{..., Approver: decision.Approver, ApprovalReason: decision.Reason})
}
```

Approval workflows in practice often route through ServiceNow, PagerDuty, a Slack approval bot, or a custom approval service. The MCP server doesn't implement the workflow — it integrates with the one the organization already runs.

### Layer 8 — Dry-run for mutating tools

Critical-tier tools should support dry-run mode. The dry-run executes the validation and authorization paths, reports what would happen, but does not perform the mutation. This is what lets agents (and humans) confirm intent before destructive actions.

```go
type RestartServiceInput struct {
	ServiceID string `json:"service_id"`
	DryRun    bool   `json:"dry_run,omitempty"`
}

func RestartService(ctx context.Context, input RestartServiceInput) (*Result, error) {
	// Validation and authorization always run
	if err := validate(input); err != nil {
		return nil, err
	}
	if err := authorize(ctx, input.ServiceID); err != nil {
		return nil, err
	}
	if input.DryRun {
		// Compute what would happen, return preview, do not mutate
		preview := previewRestart(ctx, input.ServiceID)
		return &Result{DryRun: true, Preview: preview}, nil
	}
	// Real mutation path (with approval gate, audit, rollback plan)
	return doRestart(ctx, input.ServiceID)
}
```

## Production security checklist

Before deploying any MCP server to production, every box must be checked:

- [ ] Authentication uses Managed Identity for backends, OAuth/API key for clients (no shared secrets in config)
- [ ] Every tool has explicit authorization at the handler boundary
- [ ] Tenant scope is an explicit parameter throughout the call chain, never inferred
- [ ] Secrets are loaded from Key Vault via Managed Identity at startup; rotated on schedule
- [ ] Output redaction is applied at the formatter; tested explicitly with secret-containing test fixtures
- [ ] Audit events emitted for every tool call to a dedicated audit sink
- [ ] Critical-tier tools have approval workflow integration
- [ ] Mutating tools support dry-run mode
- [ ] Rate limits enforced per identity and per tool
- [ ] CORS policy explicit for streamable HTTP deployments (allowed origins, methods, headers)
- [ ] TLS terminates at ingress; container does not serve plain HTTP outside the cluster
- [ ] No tool returns unbounded result sets (see `tool-design.md`)
- [ ] No tool executes arbitrary shell commands or unbounded queries (see `../../mcp-go-production-review/references/anti-patterns.md`)

## Attack scenarios with detection signals

**Scenario:** Agent is prompted via a malicious tool result to call a destructive tool with bad parameters.
**Detection:** Audit events show a sequence of tool calls where a read tool's output is followed immediately by a high-risk write tool with parameters that mirror the read tool's output. Pattern-match on this in monitoring.

**Scenario:** Compromised service principal calls every tool the principal has access to, scraping data.
**Detection:** Rate-limit alerts trigger before serious damage. Audit anomaly detection flags principals whose tool-call distribution shifts dramatically.

**Scenario:** Cross-tenant data exposure through a tool that accepts a tenant ID parameter but does not validate the caller's tenant access.
**Detection:** Penetration test the tool by calling it with a tenant ID the test identity does not own. The tool should return "access denied," not data. If it returns data, the scope check is missing.

**Scenario:** Secrets leak through a tool that proxies a backend response containing sensitive headers.
**Detection:** Automated test scans every tool's responses against a corpus of secret-shaped strings. Any match is a defect, not a warning.

**Scenario:** Approval bypass — a critical-tier tool is invoked by a service principal that has permission to skip approval (intended for break-glass) but the bypass is not audited differently from normal approvals.
**Detection:** Audit events include an `approval_bypass` flag; alerts fire on any bypass; weekly review of bypass usage.

## What to read next

- For the threat model these controls address: `threat-model/mcp-server-threat-model.md`
- For tool design patterns that align with risk-tier framework: `tool-design.md`
- For observability that captures security events: `observability.md`
- For the anti-patterns this skill helps you avoid: `../../mcp-go-production-review/references/anti-patterns.md`
