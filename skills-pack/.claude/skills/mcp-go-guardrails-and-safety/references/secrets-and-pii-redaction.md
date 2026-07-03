# Secrets and PII Redaction

## Purpose

A tool that reads a file or queries a system can return secrets (API keys, tokens, connection strings) or PII (emails, SSNs, names) by accident. If that output flows back into the LLM client's context, the secret is now in the conversation log, the model provider's training data eligibility, and any observability traces. Redaction is the cheap, defense-in-depth fix: scrub patterns before output goes anywhere. This reference covers regex-based secret detection, structured PII redaction, the redaction middleware, and integration with Azure AI Content Safety's PII detection API for the cases regex can't cover.

## The two redaction paths

```
Tool output
  │
  ├──→ [1] Regex secret redaction → returned to LLM client
  │
  └──→ [1] Regex secret redaction → audit log
```

Output goes two places (the client and the audit log). **Redact before both.** A redactor that runs on the return path but not on the log path leaves the secret in the log.

## Regex patterns for common secrets

```go
package redaction

import "regexp"

type pattern struct {
    name string
    re   *regexp.Regexp
}

var secretPatterns = []pattern{
    {"aws_access_key",  regexp.MustCompile(`AKIA[0-9A-Z]{16}`)},
    {"aws_secret",      regexp.MustCompile(`(?i)aws[_\-\s]*secret[_\-\s]*access[_\-\s]*key[_\-\s]*[=:]\s*['"]?[a-zA-Z0-9/+=]{40}['"]?`)},
    {"github_token",    regexp.MustCompile(`gh[pousr]_[A-Za-z0-9]{36,255}`)},
    {"github_classic",  regexp.MustCompile(`[a-f0-9]{40}`)},  // tighten by context
    {"azure_storage",   regexp.MustCompile(`DefaultEndpointsProtocol=https;AccountName=[^;]+;AccountKey=[^;]+`)},
    {"azure_sas",       regexp.MustCompile(`\?sv=\d{4}-\d{2}-\d{2}&[^"\s]*sig=[A-Za-z0-9%]+`)},
    {"azure_subscription", regexp.MustCompile(`[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`)},  // also matches GUIDs; tighten
    {"jwt",             regexp.MustCompile(`eyJ[A-Za-z0-9_-]+\.eyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+`)},
    {"private_key",     regexp.MustCompile(`-----BEGIN [A-Z ]*PRIVATE KEY-----[\s\S]+?-----END [A-Z ]*PRIVATE KEY-----`)},
    {"slack_token",     regexp.MustCompile(`xox[abprs]-[A-Za-z0-9-]+`)},
    {"openai_key",      regexp.MustCompile(`sk-[A-Za-z0-9]{32,}`)},
    {"anthropic_key",   regexp.MustCompile(`sk-ant-[A-Za-z0-9_-]{20,}`)},
    {"bearer_token",    regexp.MustCompile(`(?i)bearer\s+[A-Za-z0-9._\-+/=]{20,}`)},
    {"connection_str",  regexp.MustCompile(`(?i)(server|host|data\s*source)\s*=\s*[^;]+;\s*(password|pwd)\s*=\s*[^;]+`)},
}

func Redact(s string) (string, []string) {
    var hits []string
    out := s
    for _, p := range secretPatterns {
        if p.re.MatchString(out) {
            hits = append(hits, p.name)
            out = p.re.ReplaceAllString(out, "[REDACTED:"+p.name+"]")
        }
    }
    return out, hits
}
```

The pattern list is illustrative. Real-world maintenance: track every new key format used in your stack; add to the list. Use a published library (`detect-secrets`, gitleaks rules) as a baseline.

**Tighten the broad patterns.** `[a-f0-9]{40}` matches GitHub classic tokens *and* any 40-char hex (commit SHAs, hashes). Add context anchors: `(?:token|key|secret)[\s=:]+[a-f0-9]{40}`. The trade-off: precision vs recall. For high-impact patterns (private keys, connection strings) prefer recall; for low-impact noise patterns, prefer precision.

## Structured PII redaction

For known PII fields, redact by structure, not regex. If a tool returns a user record with `{"email": "...", "ssn": "..."}`, redact the field by name:

```go
type FieldRedactor struct {
    fields map[string]bool
}

func NewFieldRedactor(fields ...string) *FieldRedactor {
    m := make(map[string]bool, len(fields))
    for _, f := range fields {
        m[strings.ToLower(f)] = true
    }
    return &FieldRedactor{fields: m}
}

func (r *FieldRedactor) RedactJSON(data []byte) ([]byte, error) {
    var v interface{}
    if err := json.Unmarshal(data, &v); err != nil {
        return data, err
    }
    redactRecursive(v, r.fields)
    return json.Marshal(v)
}

func redactRecursive(v interface{}, fields map[string]bool) {
    switch x := v.(type) {
    case map[string]interface{}:
        for k, val := range x {
            if fields[strings.ToLower(k)] {
                x[k] = "[REDACTED]"
            } else {
                redactRecursive(val, fields)
            }
        }
    case []interface{}:
        for _, item := range x {
            redactRecursive(item, fields)
        }
    }
}
```

Common PII fields to redact by default: `email`, `ssn`, `phone`, `address`, `dob`, `passport`, `tax_id`, `credit_card`, `iban`. Make the list configurable per tool — a tool *about* user records may need to keep some PII intact for the LLM to do its job; declare exceptions explicitly.

## Regex for free-text PII

Some PII appears in free-text output (a summarized email contains the email address). Regex:

```go
var piiPatterns = []pattern{
    {"email",       regexp.MustCompile(`\b[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}\b`)},
    {"phone_us",    regexp.MustCompile(`\b(?:\+?1[-.\s]?)?\(?[2-9]\d{2}\)?[-.\s]?\d{3}[-.\s]?\d{4}\b`)},
    {"ssn",         regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`)},
    {"credit_card", regexp.MustCompile(`\b(?:\d[ -]*?){13,16}\b`)},  // tighten with Luhn check
}
```

Be aware of false positives — emails are easy; credit card patterns hit any 13–16 digit sequence. Validate with a Luhn check before redacting credit-card-shaped strings.

## Azure AI Content Safety PII detection

For higher-recall PII detection across many entity types (names, locations, organizations), use Azure AI Content Safety's PII detection API (or Azure AI Language PII service). REST call from Go, same managed-identity pattern as `prompt-injection-defense.md`. Use selectively — it's an API call per redaction, so cache or batch.

Recommendation: regex for the high-impact, low-noise patterns (secrets, structured PII); hosted PII service for free-text PII when regex precision/recall is inadequate.

## The middleware

```go
func RedactOutput(secret *Redactor, pii *FieldRedactor) Middleware {
    return func(next Handler) Handler {
        return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
            res, err := next(ctx, req)
            if err != nil || res == nil {
                return res, err
            }
            for i, c := range res.Content {
                if tc, ok := c.(mcp.TextContent); ok {
                    redacted, hits := secret.Redact(tc.Text)
                    if len(hits) > 0 {
                        slog.Warn("output_redacted",
                            "tool", req.Params.Name,
                            "patterns", hits,
                        )
                    }
                    // structured PII pass — only if content is JSON
                    if json.Valid([]byte(redacted)) {
                        if cleaned, err := pii.RedactJSON([]byte(redacted)); err == nil {
                            redacted = string(cleaned)
                        }
                    }
                    res.Content[i] = mcp.TextContent{Type: "text", Text: redacted}
                }
            }
            return res, nil
        }
    }
}
```

**Place this last in the middleware chain** — after the handler runs, before the result returns. Anything that wants to be logged with full output (audit log of the handler's raw output) must happen before this middleware, which means the audit middleware must also redact independently (or the audit middleware is positioned before redaction and itself applies the redactor).

The simpler pattern: redaction is a service injected into the audit middleware too, so audit-log writes redacted-by-default.

## Redaction in the audit log too

`patterns/structured-audit-log.md` covers the audit middleware. The audit log writes args and output for forensic value, but **never the raw versions** — always run through the redactor first. Test this with a deliberate secret-bearing input; verify the audit log shows `[REDACTED:aws_access_key]`, not the key.

## False positive management

The redactor will sometimes redact things that aren't actually secrets (a `[a-f0-9]{40}` that's a Git commit SHA). Two mitigations:

1. **Allowlist context**: don't redact if the pattern appears in a recognized non-secret context (Git-related output where SHAs are expected).
2. **Tiered patterns**: high-confidence patterns (real key prefixes like `AKIA`, `gh[pousr]_`) always redact; low-confidence (raw hex/UUID) only redact when context suggests secret.

Don't optimize for zero false positives — false positives are recoverable (user notices "[REDACTED:...]" and asks). False negatives are not (secret in the LLM context is leaked).

## Worked example — brownfield: a tool that exfiltrated a token in production

Setup: existing MCP server with a `read_file` tool. A developer asked the agent to "show me the env file", agent called `read_file(/app/.env)`, server returned the file contents including `AZURE_STORAGE_KEY=...`. The token appeared in the agent's response, then in the agent's conversation log shipped to the model provider.

Fix:

1. **Immediate**: rotate the leaked credential. Note the incident.
2. **Add the redaction middleware** to the chain. Test with the same input — verify `AZURE_STORAGE_KEY=...` becomes `AZURE_STORAGE_KEY=[REDACTED:azure_storage]`.
3. **Audit the audit log** — was the raw key also logged? If yes, sanitize the historical log (delete affected entries if possible; rotate aggressively assuming compromise).
4. **Add `read_file` to a tool tier that requires path allowlisting** — the *primary* fix is "this tool shouldn't read `.env` files at all", redaction is defense in depth. See `patterns/argument-sanitization.md` for path allowlisting.
5. **Add a unit test** that calls `read_file(/path/to/synthetic/.env)` with a fake AKIA-style key in the content; assert the redactor catches it.
6. **Backfill redaction across all tools**, not just `read_file`. Any tool can leak; the middleware should apply uniformly.

Total elapsed: 1 day for fix + ongoing for redaction-pattern maintenance. The lesson: don't rely on tool-level discipline ("only read safe files"); enforce at the output boundary.

## Anti-patterns

- **Tool-by-tool redaction.** Inconsistent; one tool gets it wrong. Centralize in middleware.
- **Redacting on the return path but not in the audit log.** Secret survives in the log.
- **Regex with no anchors.** `[a-f0-9]{40}` matches commit SHAs. Add context anchors or tier confidence.
- **No structured PII redaction.** Regex misses field-shaped PII when the value doesn't look PII-shaped (e.g., a username field containing a real name).
- **Failing closed on redaction errors.** A bug in the redactor shouldn't block all tool calls. Log the error, return unredacted with a structured warning, escalate. (Compare with prompt-injection where failing closed *is* the right call for high-risk tools.)
- **Pattern list never updated.** Token formats evolve; new providers add new prefixes. Quarterly review.
- **Treating redaction as the only defense.** Redaction is defense in depth — combined with input validation (don't accept paths to sensitive files), output classification (Prompt Shields), and audit log review. No single layer is sufficient.

## Verification questions

1. Does the redaction middleware run on every tool output before return AND before audit log writes?
2. Is the secret-pattern list current, including provider-specific formats your stack uses (OpenAI, Anthropic, GitHub, and cloud-provider key shapes)?
3. Are broad patterns (raw hex, UUIDs) anchored by context to limit false positives?
4. For structured outputs: is field-name PII redaction running on JSON outputs?
5. Is the audit log verified to contain redacted forms only (test with deliberate secret input)?
6. Is there a path-allowlist for tools that read files (defense in depth — redaction is the last line)?
7. Is the pattern list reviewed at least quarterly and updated based on observed misses?

## What to read next

- `runtime-guardrails-go.md` — the middleware chain
- `patterns/structured-audit-log.md` — where the redactor must also run
- `patterns/argument-sanitization.md` — input-side defense for path traversal etc.
- `prompt-injection-defense.md` — sibling output-safety concern
- `../mcp-go-threat-modeling` — STRIDE Information Disclosure category
- `../soc2-iso27001-controls-mapping` — data classification and confidentiality controls
