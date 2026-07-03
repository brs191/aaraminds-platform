# Code Review Checklist — 7 Categories, Per-Item Detection

This is the full checklist that the `pr-review-azure-microservices` skill applies. Each category has named items with detection cues. A reviewer walks the list, noting hard-fails first.

## Category 1 — Correctness

| Item | What to check | Detection cue | Severity |
|---|---|---|---|
| Logic matches PR intent | Does the code do what the PR description says? | Read PR description → diff; mismatch is a hard fail | Hard fail if wrong |
| Edge cases | null / empty input, max-size input, single-element collection, zero-quantity, future/past timestamps | Look for: any code with `if x != null { ... }` without symmetric handling of the null case in tests | Hard fail on missing handling of a realistic edge case |
| Concurrency | Shared mutable state; race conditions; lock ordering | `static` fields in Spring; package-level vars in Go; missing `synchronized` or `sync.Mutex` | Hard fail if a race exists; comment if defensible |
| Error semantics | Errors propagated with context; not swallowed; not panic-on-error-in-Go | `} catch (Exception e) { }` (Java); `_ = something()` (Go) where `_` discards an error | Hard fail on swallowed errors that affect state |
| Idempotency | Mutations that retry safely; idempotency keys on POST | New POST handler without idempotency key parameter | Comment unless retry is implied |
| Resource cleanup | Files, connections, contexts closed | Missing `defer` (Go) or `try-with-resources` (Java) | Hard fail on leak risk |

## Category 2 — Security

| Item | What to check | Detection cue | Severity |
|---|---|---|---|
| No plaintext secrets | Connection strings, tokens, API keys in code | Search diff for: `password=`, `apikey=`, `Bearer `, GUID-shaped strings, base64-shaped strings | Hard fail (always) |
| Auth required | Endpoint annotated with auth requirement | Spring: `@PreAuthorize` or `@Secured`; Go: middleware check; absence = public endpoint | Hard fail if intended-private endpoint is public |
| Authorization (not just auth) | Caller's identity is checked against the resource being accessed | Search for tenant/customer/owner ID checks; missing = anyone authenticated can access anyone's data | Hard fail on cross-tenant or cross-user access without check |
| Input validation | Pattern, length, range, enum checks | Spring: `@Valid` + Bean Validation annotations; Go: explicit `if` checks; absence = trust-the-client | Hard fail on user-controlled input flowing to query/disk/network without validation |
| SQL/NoSQL injection | Parameterized queries or builder API; never string concatenation | Spring Data: `@Query` with `?1` or `:param` (good) vs. string-interpolated SQL (bad). Go `pgx`: parameterized vs. `fmt.Sprintf` | Hard fail on string-built queries with user input |
| Sensitive data in logs | PII, tokens, full request bodies | `log.info("request: {}", request)` where `request` may contain PII | Comment; hard fail if PCI/PHI data |
| Dependency security | New dependencies vetted; no known CVEs | `pom.xml`/`go.mod` diff; check for outdated versions | Comment unless CVE is critical |
| Output redaction | Tool output / response sanitizes secrets if echoing back | Service that returns log lines, configs, env vars unredacted | Hard fail on secret leak |

## Category 3 — Observability

| Item | What to check | Detection cue | Severity |
|---|---|---|---|
| Trace context propagation | Spans created for outbound calls; `traceparent` flowing | New HTTP/gRPC/messaging call without `otelhttp.NewTransport` (Go) or auto-instrumentation gap (Java) | Hard fail if a new external call has no span |
| Structured logs | JSON logs with trace ID / span ID; correlation fields | `System.out.println`, `fmt.Println`, `log.Printf` without structure | Hard fail on unstructured logs in new code |
| Domain-meaningful metrics | New endpoints have Prometheus counters / histograms relevant to the domain | No metrics emitted for the new endpoint | Comment unless this is on a critical path |
| Alerts updated | If SLO touched or new critical path added, alerts updated alongside | New high-volume endpoint without alert config in `infra/observability/` | Comment; hard fail if endpoint is critical-tier |
| No PII in logs | Customer IDs may be PII depending on regime; emails are PII; full request bodies usually are | `log.info("user: {}", user.toString())` | Comment unless GDPR/PHI/PCI, then hard fail |
| Runbook updates | If alert thresholds changed or new alert added, runbook section exists for the new alert | New alert YAML in `infra/observability/` without matching runbook update | Comment |

## Category 4 — Testing

| Item | What to check | Detection cue | Severity |
|---|---|---|---|
| Happy path | Test exercises the success scenario | New service method without a test for the success case | Hard fail |
| Error paths | Test exercises each named error (not found, unauthorized, validation failure, downstream timeout) | New error handling without a test that triggers it | Hard fail |
| Edge cases | Test covers boundary values | Tests for "typical" inputs but no boundary tests | Comment unless boundary is realistic, then hard fail |
| Integration coverage | If the PR adds a service-to-service call, integration test exists | New `RestClient.exchange(...)` (Java) or `http.Client.Do(req)` (Go) call without integration coverage | Comment if mockable; hard fail if external dependency contract is critical |
| Test quality | Tests assert specific behavior, not just "no exception thrown" | `assertNotNull(result)` as the only assertion in a new test | Comment unless trivial; otherwise hard fail |
| Test isolation | Tests don't share mutable state; can run in any order | `static` fields in test classes; `t.Parallel()` (Go) with shared state | Hard fail on tests that depend on order |
| Test names | Names describe behavior, not method names | `testCreate()` vs. `whenInvalidCustomerId_thenReturns400()` | Comment |

## Category 5 — Performance

| Item | What to check | Detection cue | Severity |
|---|---|---|---|
| N+1 queries | Loop over collection that triggers a query per element | `for (Customer c : customers) { c.getOrders(); }` (Spring lazy loading); repeated `findById` in a loop | Hard fail on hot path; comment otherwise |
| Reasonable allocations | No unbounded allocation based on request size | Allocating a slice/array sized by user input without a cap | Comment unless obviously DOS-able, then hard fail |
| Appropriate timeouts | Every outbound call has a timeout | New HTTP/gRPC/DB call without an explicit timeout | Hard fail on production-touching outbound calls |
| Bulkhead isolation | Outbound calls to different dependencies use separate connection pools where it matters | Single shared `RestClient` / `http.Client` used for all downstreams when one is slow | Comment; hard fail in known multi-downstream services |
| Caching | Cache reads where appropriate, with TTL and invalidation considered | New high-volume read endpoint without cache evaluation | Comment |
| Async vs sync | New side-effect call should be async if caller doesn't need response inline | New synchronous call from request path to "send notification" or similar | Comment; hard fail if it adds significant latency to user path |

## Category 6 — Infrastructure (Terraform / IaC)

| Item | What to check | Detection cue | Severity |
|---|---|---|---|
| No plaintext secrets in `.tf` | All secrets sourced from Key Vault or pipeline variables | `password = "Hunter2!"` in any `.tf` file | Hard fail (always) |
| RBAC scoped | Role assignments target the minimum scope | `role_definition_name = "Owner"` scoped to a subscription | Hard fail unless explicitly justified |
| Managed Identity | Service identity is Managed Identity, not a service principal with secret | `azurerm_user_assigned_identity` or `system_assigned` block; no `azurerm_application` with a password | Hard fail on long-lived SP secret |
| Tags applied | All resources have `environment`, `owner`, `cost-center` tags | Resource block without `tags` | Comment |
| Lifecycle blocks reasonable | `ignore_changes` not used to paper over drift | `lifecycle { ignore_changes = all }` | Hard fail |
| Module sources pinned | External modules reference a tag or commit SHA, not `main` | `source = "github.com/.../module"` with no `ref=` | Hard fail |
| State backend | `azurerm` backend with `use_oidc = true` | `local` backend in any non-experimental tf module | Hard fail |
| Naming conventions | Resources follow the team's naming standard | `name = "tempresource123"` or similar | Comment |

## Category 7 — Documentation / Compliance

| Item | What to check | Detection cue | Severity |
|---|---|---|---|
| ADR for architectural change | New service-to-service interaction, new data store, new pattern adoption | Diff introduces a new dependency without ADR | Comment unless the change is large, then hard fail |
| Runbook updated | New alert or changed alert threshold has matching runbook entry | New alert YAML without `docs/runbook.md` update | Comment |
| OpenAPI / schema | API changes update the OpenAPI spec | Endpoint added/changed without `openapi.yaml` update | Hard fail on customer-facing API |
| SOC 2 / ISO 27001 scope | Change affects audit evidence (auth, audit log, access control, encryption) | Touches identity/auth code, encryption config, or audit emission without controls-map note | Hard fail if scope changes; comment if marginal |
| Migration safety | Database/schema migration is forward-compatible during rolling deploy | New schema migration that drops a column referenced by current production code | Hard fail |
| Breaking changes | Versioning is honored; consumers notified | API contract break without `/v2/` URI or header version | Hard fail on external API; comment on internal |

## How to comment

Comments should be **specific** and **actionable**. Compare:

- **Bad:** "Improve error handling here."
- **Good:** "Line 47: the `EntityNotFoundException` is caught and swallowed. The caller expects a 404 for missing preferences, not a 200 with empty body. Suggest: let the exception propagate or convert to a `ResponseStatusException(NOT_FOUND)`."

Use the GitHub PR review's per-line comment feature for line-specific issues; use the summary comment for category-level findings and the verdict.

## Verdict format

Every PR review ends with a structured verdict:

```
## Verdict
[approve | approve-with-comments | request-changes | hard-reject]

## Findings by category
Category 1 (correctness): [pass | comments | fail]
Category 2 (security): [pass | comments | fail]
... (all 7)

## Hard-fails to address before merge
- [specific named issue with line reference]

## Non-blocking comments
- [comment]
```

A reviewer who can't fill this template hasn't done the review yet. The template forces specificity.
