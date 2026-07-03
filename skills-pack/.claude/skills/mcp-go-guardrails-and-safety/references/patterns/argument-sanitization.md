# Pattern — Argument Sanitization

## Problem

Tool args flow into shell commands, SQL queries, file paths, URLs, regex patterns, and other downstream consumers — each with its own injection threat surface. The MCP SDK's JSON schema validation ensures the type and shape; it does **not** ensure the *content* is safe for the consumer. `"file_path": "../../../etc/passwd"` passes JSON schema for `"type": "string"`; it's still a path-traversal attack. Argument sanitization at the boundary, before the value enters the consumer's grammar, prevents the entire category of injection-into-downstream attacks.

## Use when

- Any tool handler that uses string args in shell commands, SQL, file paths, URLs, regex, OS APIs that consume paths, or HTTP outbound calls
- Defense in depth around primary fixes (parameterized SQL, path-confinement libraries, allowlists)
- Brownfield hardening — auditing existing handlers for sanitization gaps

## Avoid when

- The arg is a numeric or boolean that's already type-validated; sanitization is moot
- The handler doesn't actually use string args in downstream consumers — pure compute, no escape

## Implementation steps

### Step 1 — identify the downstream consumer for each arg

For every string arg of every tool, name where it ends up:

| Consumer | Threat | Primary fix | Sanitization (defense in depth) |
|---|---|---|---|
| Shell command | Command injection | **Never use `sh -c "..." + arg`**; use `exec.Command("prog", arg)` with separate argv | Reject metacharacters: `;`, `|`, `&`, `$`, `\``, newline |
| SQL query | SQL injection | **Use parameterized queries** (`$1` placeholders, never string concat) | Length cap; charset allowlist for known-safe contexts |
| File path | Path traversal | Confine to an allowed root via `filepath.Clean` + prefix check; or use `os.Root` (Go 1.24+) | Reject `..`, absolute paths, NUL bytes; canonicalize |
| URL | SSRF, scheme injection | Allowlist scheme (https only); allowlist host (internal services rejected) | Reject userinfo segment; reject `file://`, `gopher://`, etc. |
| Regex pattern | Regex DoS | Length cap; compile timeout (re2 in Go is safe by default) | Reject patterns with nested quantifiers above threshold |
| HTTP header | Header injection | Reject CR/LF | Sanitize same |
| Email / SMTP | Email injection | Reject CR/LF; validate with strict regex | — |
| Command-line for child process | Command injection | Pass argv array, never shell string | — |

### Step 2 — sanitize per consumer

Put each sanitizer in `internal/sanitize` and use at the **use site**, not at the boundary. Sanitization at the boundary risks being skipped or double-applied; sanitization at the use site is local to the consumer that needs it.

### Shell args

```go
package sanitize

import "fmt"

// For exec.Command argv args. Reject metacharacters defensively
// even though argv form doesn't shell-interpret them.
func ShellArg(s string) (string, error) {
    if len(s) > 4096 {
        return "", fmt.Errorf("shell arg too long")
    }
    for _, r := range s {
        switch r {
        case 0, '\n', '\r':
            return "", fmt.Errorf("shell arg contains forbidden character")
        }
    }
    return s, nil
}
```

Usage:

```go
arg, err := sanitize.ShellArg(req.GetString("filename", ""))
if err != nil {
    return mcp.NewToolResultError(err.Error()), nil
}
cmd := exec.Command("git", "show", arg)  // argv form, never shell -c
```

### SQL — parameterized queries (primary defense)

```go
// Bad — concatenation
rows, _ := db.Query("SELECT * FROM orders WHERE customer = '" + arg + "'")

// Good — parameterized
rows, _ := db.Query("SELECT * FROM orders WHERE customer = $1", arg)
```

If you ever can't use parameters (legacy DSL, dynamic table name), the answer is allowlist not sanitize. Validate the value is in a known-good set:

```go
allowedTables := map[string]bool{"orders": true, "customers": true}
if !allowedTables[tableName] {
    return "", fmt.Errorf("table not allowed")
}
query := fmt.Sprintf("SELECT * FROM %s WHERE id = $1", tableName)
```

### File paths

```go
package sanitize

import (
    "fmt"
    "path/filepath"
    "strings"
)

// SafePath confines path to rootDir. Resolves the path and verifies
// it starts with rootDir after canonicalization. Rejects absolute paths,
// paths containing NUL, and unexpected control characters.
func SafePath(rootDir, p string) (string, error) {
    if p == "" {
        return "", fmt.Errorf("empty path")
    }
    if strings.ContainsRune(p, 0) {
        return "", fmt.Errorf("path contains NUL")
    }
    if filepath.IsAbs(p) {
        return "", fmt.Errorf("absolute path not allowed")
    }
    cleaned := filepath.Clean(filepath.Join(rootDir, p))
    rootClean := filepath.Clean(rootDir) + string(filepath.Separator)
    if !strings.HasPrefix(cleaned+string(filepath.Separator), rootClean) {
        return "", fmt.Errorf("path escapes root")
    }
    return cleaned, nil
}
```

For Go 1.24+: prefer `os.Root` for confinement-by-design — it can't be tricked by symlinks the way `filepath.Clean` can.

```go
import "os"

root, err := os.OpenRoot("/data/safe")
if err != nil { /* ... */ }
defer root.Close()

f, err := root.Open(relPath)  // root.Open refuses to escape /data/safe
```

### URLs

```go
package sanitize

import (
    "fmt"
    "net/url"
    "strings"
)

func SafeURL(rawURL string, allowedSchemes, allowedHosts []string) (*url.URL, error) {
    if len(rawURL) > 2048 {
        return nil, fmt.Errorf("url too long")
    }
    u, err := url.Parse(rawURL)
    if err != nil {
        return nil, fmt.Errorf("parse: %w", err)
    }
    // scheme allowlist
    schemeOK := false
    for _, s := range allowedSchemes {
        if u.Scheme == s {
            schemeOK = true
            break
        }
    }
    if !schemeOK {
        return nil, fmt.Errorf("scheme %q not allowed", u.Scheme)
    }
    // reject userinfo
    if u.User != nil {
        return nil, fmt.Errorf("userinfo not allowed in url")
    }
    // host allowlist (or denylist for internal addresses)
    if len(allowedHosts) > 0 {
        hostOK := false
        for _, h := range allowedHosts {
            if u.Hostname() == h {
                hostOK = true
                break
            }
        }
        if !hostOK {
            return nil, fmt.Errorf("host %q not allowed", u.Hostname())
        }
    }
    // reject obvious SSRF targets unless explicitly allowed
    host := strings.ToLower(u.Hostname())
    if host == "169.254.169.254" || host == "metadata.google.internal" {
        return nil, fmt.Errorf("cloud metadata endpoint blocked")
    }
    return u, nil
}
```

For server-side fetching, also reject by resolved IP — DNS rebinding can bypass hostname allowlists. Use a `net.Resolver` to resolve before fetching, reject private/loopback IPs, then dial directly to the resolved IP.

### Regex patterns

If a tool accepts a regex from the user, Go's `regexp` package uses RE2 which doesn't have catastrophic backtracking — but length and complexity caps are still useful:

```go
func SafeRegex(pattern string) (*regexp.Regexp, error) {
    if len(pattern) > 1024 {
        return nil, fmt.Errorf("regex too long")
    }
    return regexp.Compile(pattern)
}
```

If you're using PCRE (cgo wrappers), the threat is real — apply much stricter limits or don't accept user-supplied regex.

### Step 3 — log rejections to audit

Sanitization rejections are security signals. The audit middleware (`structured-audit-log.md`) captures them, but the handler should also emit a specific event:

```go
if err := sanitize.SafePath(rootDir, userPath); err != nil {
    slog.Warn("sanitize_rejected",
        "tool", req.Params.Name,
        "consumer", "file_path",
        "reason", err.Error(),
    )
    return mcp.NewToolResultError("invalid path"), nil
}
```

A spike in `sanitize_rejected` events for one tool is either a real attacker, a buggy client, or a usability problem. Triage in observability.

## Trade-offs

| Choice | Gain | Cost |
|---|---|---|
| Sanitize at boundary (middleware) | Single chokepoint | Loses consumer-specific context; can be too strict |
| Sanitize at use site (handler) | Consumer-aware; cheap | Risk of being skipped if handler is added without sanitization |
| Allowlist | Strong safety | Brittle as legitimate inputs evolve |
| Denylist | Easier to maintain | Easier to bypass |

Default: **sanitize at use site with consumer-aware functions; allowlist where the value space is small.** Primary defense (parameterized SQL, argv exec, `os.Root` paths) is always in addition, never alone.

## Common failure modes

### Path traversal via symlink
**Detection**: `filepath.Clean` returns a path inside `rootDir`, but the path resolves to a symlink pointing outside.
**Fix**: use `os.Root` (Go 1.24+) for symlink-safe confinement; or `filepath.EvalSymlinks` + prefix check.

### URL allowlist bypass via DNS rebinding
**Detection**: attacker controls DNS for an allowlisted hostname; first lookup resolves to a public IP (allowlist passes), second lookup (when the HTTP client connects) resolves to internal IP.
**Fix**: resolve once, fetch via the resolved IP; reject private/loopback IPs in the result.

### SQL injection via second-order
**Detection**: input was sanitized when stored, then later used in a query that interpolates it (e.g., admin tools that build a query from a stored search term).
**Fix**: parameterize at every query; never interpolate.

### Shell injection via `sh -c "command " + arg`
**Detection**: code review surfaces `sh -c` with concatenated user input.
**Fix**: `exec.Command("prog", arg)` argv form; the shell isn't involved.

### Regex DoS in non-Go components
**Detection**: a downstream service (Python with `re`, JavaScript with PCRE-style) handles user regex; complexity causes timeout.
**Fix**: length caps; reject suspect patterns (nested quantifiers); time-bound execution at the consumer.

### Sanitization missing for one tool
**Detection**: code review or static analysis finds a handler using `exec.Command` / `db.Query` / `filepath.Join` without going through sanitize package.
**Fix**: lint rule banning these primitives outside the sanitize package's allowed call sites.

## MCP tool opportunities

- **`audit_arg_usage`** — scan handler functions for primitives (`exec.Command`, `db.Query`, `filepath.Join`, `http.Get`, `regexp.Compile`) and verify sanitization is applied to each user-controlled arg.
- **`generate_sanitization_skeleton`** — given a tool's input schema, output the sanitization function skeleton for each arg based on declared consumer.
- **`detect_ssrf_risk`** — find HTTP outbound calls in handler code and flag those that accept user-controlled URLs without `SafeURL`.

## What to read next

- `../runtime-guardrails-go.md` — input validation upstream of sanitization
- `tool-handler-middleware-chain.md` — where the middleware chain handles cross-cutting concerns
- `structured-audit-log.md` — audit log captures sanitization rejections
- `../tool-authorization.md` — auth complements sanitization (auth restricts who; sanitization restricts what)
- `../../mcp-go-threat-modeling` — STRIDE Tampering category
- `../../pr-review-azure-microservices` — code review checklist for unsafe primitives
