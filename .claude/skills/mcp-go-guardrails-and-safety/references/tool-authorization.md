# Tool Authorization

## Purpose

Authorization in MCP servers depends on transport. **Stdio MCP** has implicit auth — the parent process spawned the server and trusts it. **HTTP MCP** has explicit auth — the network is the threat boundary, and every request needs identification + authorization. This reference covers both transports, the per-tool authorization model, Entra ID with managed identity for HTTP MCP, and the audit-log identity propagation that ties calls back to callers.

## The transport decision drives the auth model

```
stdio MCP
  │  Parent process (Claude Desktop, Claude Code) spawned this server.
  │  Implicit trust: anything the parent could do, this server can do.
  │  Auth model: trust the parent; identify "who is the user" via env vars or out-of-band.
  ▼
  Audit log carries: process user, optional CLAUDE_USER env var.

HTTP MCP
  │  Network. Anyone with the URL might try to call.
  │  Explicit trust: every request must authenticate.
  │  Auth model: Entra ID JWT + managed identity for service calls.
  ▼
  Audit log carries: validated JWT subject (oid / upn).
```

Most MCP servers in this pack are stdio. If you're standing up an HTTP MCP server, treat it like any other internet-exposed HTTP service (see `azure-microservices-security`).

## Stdio MCP — what "authorization" means

Stdio MCP servers run as child processes of the agent (Claude Desktop, Claude Code, an internal agent). The OS gives the server the same permissions as the parent. There's no network; there's no untrusted input source other than the LLM client's tool calls.

So "authorization" in stdio context means:

1. **Per-tool capability declaration** — some tools should be off by default and explicitly enabled
2. **Identity carry-over from the parent** — if the parent process has a user identity, propagate it for audit
3. **Capability boundaries enforced at the tool level** — even though the OS allows it, the tool should refuse

### Per-tool enable flag

Some tools (read files, execute commands, network access) should be off by default. Declare a config:

```go
type Config struct {
    AllowedTools map[string]bool `json:"allowed_tools"`
}

func LoadConfig(path string) (*Config, error) {
    b, err := os.ReadFile(path)
    if err != nil {
        return nil, err
    }
    var c Config
    return &c, json.Unmarshal(b, &c)
}
```

Middleware:

```go
func Allowlist(cfg *Config) Middleware {
    return func(next Handler) Handler {
        return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
            if !cfg.AllowedTools[req.Params.Name] {
                return mcp.NewToolResultError(fmt.Sprintf("tool %q is not enabled", req.Params.Name)), nil
            }
            return next(ctx, req)
        }
    }
}
```

Default-deny: the absence of a tool name in `AllowedTools` is rejection. Operators opt in to dangerous tools per deployment.

### Identity carry-over

The parent process can pass identity via env vars or initial handshake metadata:

```go
type Identity struct {
    User  string
    Email string
}

func IdentityFromEnv() Identity {
    return Identity{
        User:  os.Getenv("CLAUDE_USER"),
        Email: os.Getenv("CLAUDE_USER_EMAIL"),
    }
}
```

Audit log carries the identity; every tool call entry includes who triggered it. For internal compliance audits, this maps tool calls to humans.

## HTTP MCP — Entra ID and managed identity

For HTTP MCP, the network is the threat boundary. Every request must authenticate. Stack default: **Microsoft Entra ID** — the same identity surface as the rest of the pack.

### Trust model

```
Client (agent, CI, another service)
  │ obtains JWT from Entra ID
  │   - human → device code flow / browser flow
  │   - service → managed identity → token
  ▼
Bearer JWT in Authorization header
  ▼
MCP HTTP server validates JWT (issuer, audience, signature, expiry)
  │
  ▼
JWT subject (oid) → identity for authorization decision + audit log
```

### JWT validation in Go

Use `github.com/MicahParks/keyfunc` or the official `github.com/Azure/azure-sdk-for-go` token validators. The `keyfunc` constructor API name has changed across major versions (`keyfunc.Get` → `keyfunc.NewDefault` → `keyfunc.NewDefaultCtx` in v3) — verify against your pinned `go.mod` version. Pattern below targets keyfunc v3:

```go
package authz

import (
    "context"
    "fmt"
    "net/http"
    "strings"
    "time"

    "github.com/MicahParks/keyfunc/v3"
    "github.com/golang-jwt/jwt/v5"
)

type Validator struct {
    issuer   string
    audience string
    jwks     keyfunc.Keyfunc
}

func NewValidator(tenantID, audience string) (*Validator, error) {
    issuer := fmt.Sprintf("https://login.microsoftonline.com/%s/v2.0", tenantID)
    jwksURL := fmt.Sprintf("https://login.microsoftonline.com/%s/discovery/v2.0/keys", tenantID)

    jwks, err := keyfunc.NewDefaultCtx(context.Background(), []string{jwksURL})
    if err != nil {
        return nil, fmt.Errorf("jwks: %w", err)
    }

    return &Validator{issuer: issuer, audience: audience, jwks: jwks}, nil
}

type Principal struct {
    OID   string
    UPN   string
    Roles []string
}

func (v *Validator) Validate(tokenString string) (*Principal, error) {
    token, err := jwt.Parse(tokenString, v.jwks.Keyfunc,
        jwt.WithIssuer(v.issuer),
        jwt.WithAudience(v.audience),
        jwt.WithExpirationRequired(),
        jwt.WithLeeway(30*time.Second),
    )
    if err != nil {
        return nil, fmt.Errorf("token parse: %w", err)
    }
    if !token.Valid {
        return nil, fmt.Errorf("token invalid")
    }
    claims, ok := token.Claims.(jwt.MapClaims)
    if !ok {
        return nil, fmt.Errorf("claims type")
    }

    p := &Principal{}
    if v, ok := claims["oid"].(string); ok {
        p.OID = v
    }
    if v, ok := claims["upn"].(string); ok {
        p.UPN = v
    } else if v, ok := claims["preferred_username"].(string); ok {
        p.UPN = v
    }
    if rs, ok := claims["roles"].([]interface{}); ok {
        for _, r := range rs {
            if s, ok := r.(string); ok {
                p.Roles = append(p.Roles, s)
            }
        }
    }
    return p, nil
}
```

### The HTTP middleware

```go
type ctxKey int

const principalKey ctxKey = 1

func PrincipalFromContext(ctx context.Context) *Principal {
    p, _ := ctx.Value(principalKey).(*Principal)
    return p
}

func AuthMiddleware(v *Validator) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            auth := r.Header.Get("Authorization")
            if !strings.HasPrefix(auth, "Bearer ") {
                http.Error(w, "missing bearer token", http.StatusUnauthorized)
                return
            }
            tok := strings.TrimPrefix(auth, "Bearer ")
            p, err := v.Validate(tok)
            if err != nil {
                http.Error(w, "invalid token", http.StatusUnauthorized)
                return
            }
            ctx := context.WithValue(r.Context(), principalKey, p)
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}
```

Wire this into the HTTP transport's request pipeline before the MCP dispatcher.

### Per-tool authorization

Even with a valid JWT, some tools require additional authorization (admin role, specific app role, etc.). Declare per tool:

```go
type ToolAuthz struct {
    requiredRoles map[string][]string
}

func (a *ToolAuthz) Allowed(toolName string, p *Principal) bool {
    required, ok := a.requiredRoles[toolName]
    if !ok {
        return true  // no requirement → allow any authenticated principal
    }
    for _, req := range required {
        for _, has := range p.Roles {
            if req == has {
                return true
            }
        }
    }
    return false
}

func Authorize(a *ToolAuthz) Middleware {
    return func(next Handler) Handler {
        return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
            p := PrincipalFromContext(ctx)
            if p == nil {
                return mcp.NewToolResultError("unauthenticated"), nil
            }
            if !a.Allowed(req.Params.Name, p) {
                return mcp.NewToolResultError("not authorized for this tool"), nil
            }
            return next(ctx, req)
        }
    }
}
```

Tools with no requirement allow any authenticated principal; tools with declared roles enforce role membership. Role definitions live in Entra ID app registrations.

### Service-to-service: managed identity

When an Azure service (Container Apps, Functions, AKS pod) calls the MCP server, use managed identity:

```go
import "github.com/Azure/azure-sdk-for-go/sdk/azidentity"

cred, err := azidentity.NewManagedIdentityCredential(nil)
// ...
token, err := cred.GetToken(ctx, policy.TokenRequestOptions{
    Scopes: []string{"api://your-mcp-server-app-id/.default"},
})
// Use token.Token as Bearer in HTTP request to MCP server
```

No stored secrets; the platform issues tokens from the assigned identity. SOC 2 / ISO 27001 friendly — see `soc2-iso27001-controls-mapping`.

## Identity propagation to audit log

The audit middleware reads the principal from context and includes it in every log entry:

```go
slog.Info("tool_call",
    "tool", req.Params.Name,
    "principal_oid", p.OID,
    "principal_upn", p.UPN,
    "duration_ms", durationMs,
    "outcome", outcome,
)
```

For stdio MCP, the equivalent is the env-var identity. Both paths land in the same audit log shape — `patterns/structured-audit-log.md` covers the schema.

## Worked example — brownfield: adding Entra auth to an existing HTTP MCP server

Setup: existing Go MCP server exposed over HTTP behind a custom domain on Azure Container Apps. Currently uses a static API key in a header. Internal team only, but the bare API key is a SOC 2 finding waiting to happen.

Steps:

1. **Register an Entra ID app** for the MCP server. Define app roles (e.g., `Tool.User`, `Tool.Admin`). Set the audience.
2. **Register a second app for the client** if the client is a service (not a person). Grant the client app the `Tool.User` role on the server app.
3. **Wire the JWT validator** in the server. Env vars: `AAD_TENANT_ID`, `AAD_AUDIENCE`.
4. **Add the auth middleware** to the HTTP pipeline before the MCP dispatcher.
5. **Add per-tool role requirements** for high-privilege tools (`Tool.Admin` required for destructive operations).
6. **Migrate clients**:
   - Service clients switch from API key to managed identity token acquisition.
   - Human clients use device code flow during dev; for CI, use workload identity federation (GitHub OIDC → Entra ID).
7. **Run both auth methods in parallel** for a 2-week migration window (validator accepts both legacy API key and JWT). Log which auth method was used.
8. **Disable the API key path** once all clients are on JWT. Verify zero requests using the legacy auth.
9. **Rotate (delete) the legacy API key**.

Total elapsed: 2–4 weeks. Downtime: zero. Compliance posture: improved.

## Anti-patterns

- **Stdio MCP with no per-tool enable flag.** Dangerous tools (file write, command execution) are on by default for every deployment. Operators have no per-deploy control.
- **HTTP MCP with API keys in env.** Rotating keys is a manual ops task; revocation is slow; compromised key is a real incident. Use Entra.
- **Validating JWT without checking audience.** A token valid for another app gets accepted. Always validate `aud`.
- **Validating JWT without checking expiry.** Token replay attacks become trivial. Always require `exp`.
- **Role check on the client side.** Easy to bypass; trust nothing the client tells you. Roles are claims in the validated JWT.
- **Hand-rolling JWT validation.** Use a library; the spec is full of edge cases (clock skew, key rollover, alg confusion).
- **No identity in audit log.** When an incident happens, you can't tell who called the tool. Always propagate principal to audit.

## Verification questions

1. For stdio: is there a per-tool enable flag (default-deny for dangerous tools)?
2. For stdio: is the parent-process identity propagated for audit log entries?
3. For HTTP: is auth Entra ID (Managed Identity for service, browser/device flow for humans), not API key?
4. For HTTP: is JWT validation checking issuer, audience, signature, expiry — and using a library, not hand-rolled?
5. Are per-tool role requirements declared for high-privilege tools, not just blanket authentication?
6. Is the principal propagated from auth middleware to audit middleware via context?
7. For service clients: is the token-acquisition path managed identity, not stored credentials?

## What to read next

- `runtime-guardrails-go.md` — the middleware chain this slots into
- `patterns/structured-audit-log.md` — principal field in the audit log schema
- `../azure-microservices-security` — broader Entra ID, Managed Identity, Private Link surface
- `../mcp-go-server-building` — stdio vs HTTP transport choice at design time
- `../mcp-go-threat-modeling` — STRIDE Spoofing / Elevation of Privilege categories
- `../soc2-iso27001-controls-mapping` — access control and identity controls
