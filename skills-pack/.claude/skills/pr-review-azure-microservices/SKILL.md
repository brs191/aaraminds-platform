---
name: pr-review-azure-microservices
description: Reviews pull requests on Azure-hosted microservices (Spring Boot 21+ or Go 1.25+), Terraform AzureRM, and GitHub Actions workflows — a 7-category checklist (correctness, security, observability, testing, performance, IaC, docs/compliance) plus language-specific anti-pattern detection. Use when reviewing a PR, training a teammate on the review bar, producing a structured review comment, or auditing whether past PRs would have caught a production issue. Do not use for designing the service (use new-azure-service-bootstrap or microservices-architecture-design) or end-to-end production readiness (use mcp-go-production-review).
version: 1.0.1
last_updated: 2026-05-30
---

# PR Review — Azure Microservices

## When to use

Trigger this skill when reviewing a pull request on the Azure microservices estate: Spring Boot service, Go service, Terraform infrastructure, or GitHub Actions workflow change. Common triggers: "review this PR," "produce a PR comment template for this change," "teach the team what to look for in PR review," "the last incident slipped through review — what should we add to the checklist."

Do **not** use this skill for: designing a new service (`new-azure-service-bootstrap`); broader architecture review (`microservices-architecture-design`); MCP-server-specific production review (`mcp-go-production-review`); threat-modeling a new attack surface (`mcp-go-threat-modeling`).

## The critical decision rule — review the diff, not the codebase

PR review evaluates the change being proposed, not a re-evaluation of the existing system. If the existing system has problems, file them as separate work; don't hold a small PR hostage to fix systemic issues. Conversely, do not approve a PR that introduces new problems just because the rest of the codebase already has them — "we have this pattern elsewhere" is not justification; it's an admission of accumulated debt.

The review's job is to keep the merge bar high *for new code* while keeping the review cycle fast enough that PRs don't accumulate. A 30-minute review on a 100-line PR is the right size; a 4-hour review on the same PR signals review-scope creep.

## The 7-category review checklist

| # | Category | What to check | Severity rule |
|---|---|---|---|
| 1 | Correctness | Logic does what the PR description claims; edge cases covered; concurrency safe | Hard fail if wrong; comments otherwise |
| 2 | Security | No secrets in code/config; auth and authz correctly applied; input validation | Hard fail on any secret leak |
| 3 | Observability | Logs structured; trace context propagated; metrics emitted; alerts updated if SLO touched | Hard fail if a new endpoint has zero observability |
| 4 | Testing | Unit tests cover happy + edge + failure; integration tests where contracts cross | Hard fail if untested code path has non-trivial logic |
| 5 | Performance | No N+1 queries; reasonable allocations; appropriate timeouts and bulkheads | Comment unless obviously bad (then hard fail) |
| 6 | Infrastructure (Terraform / IaC) | Module structure correct; RBAC scoped; secrets via Key Vault; tags applied | Hard fail on any plaintext secret in tf files |
| 7 | Documentation / compliance | ADR for architectural changes; runbook updates if alert behavior changes; SOC 2 / ISO 27001 evidence trail | Comment for missing ADR; hard fail if compliance scope changes without controls map |

For the full checklist with per-item detection cues, see `references/code-review-checklist.md`.

## Review-pass logic

1. **Read the PR description first.** What is this change supposed to do? If the description is missing, vague, or contradicts the diff, comment on the description before reading code — it'll cost more time to review without context.

2. **Skim the diff once for shape.** What files changed? Is the change scope reasonable for one PR (under ~500 changed lines of meaningful code)? Larger PRs are an anti-pattern; comment on the size and request a split if the diff exceeds ~500 LOC of real change.

3. **Walk the 7 categories in order, one at a time.** For each, look for hard-fails first; if any, stop and request changes. Otherwise, accumulate non-blocking comments.

4. **Apply language-specific anti-pattern detection.** If the diff touches Spring Boot code, scan against `references/spring-boot-anti-patterns.md`. If Go, scan against `references/go-anti-patterns.md`. If Terraform, scan against `references/terraform-anti-patterns.md`. These are the named patterns most likely to bite — fast pattern matching, not deep review.

5. **Verify the verification.** Did the author run the tests? Did CI pass? If CI is red, the review doesn't start. If CI is green but the test diff is suspicious (e.g., a test was *removed* without explanation), block until clarified.

6. **Produce the review.** Structured comment with: verdict (approve / approve with non-blocking comments / request changes / hard reject), category-by-category notes, line-level comments on the diff, suggested commits where applicable.

## Worked example — brownfield: reviewing a Spring Boot PR that adds a new endpoint

Setup: PR titled "Add `GET /v1/customers/{id}/preferences` endpoint." Diff: 3 files changed, 180 lines added, 12 removed. New controller method, new service method, new repository query, 4 new unit tests. No Terraform changes.

Review walk:

1. **PR description.** Says "expose customer preferences for the new mobile UI." That's enough context.
2. **Diff shape.** 180 lines, three files, focused scope. Good shape; proceed.
3. **Category 1 — Correctness.** Controller takes `id` as `@PathVariable Long`. The service tries to load preferences but catches `EntityNotFoundException` and returns `null`. Controller then returns `200 OK` with empty body. **Hard fail.** Should return `404 Not Found`. Comment with the fix.
4. **Category 2 — Security.** Endpoint annotated `@PreAuthorize("hasAuthority('SCOPE_customer.read')")`. Good. Input validation: `id` is path-typed `Long` — Spring handles invalid values with 400. Cross-tenant: customer ID could belong to a different tenant. Does the service check that the caller can see this customer? **Search for tenant check.** Not present. **Hard fail** — request changes: enforce that the authenticated caller's tenant matches the customer's tenant.
5. **Category 3 — Observability.** Controller has `@WithSpan` via Spring Boot OTel starter (auto-instrumented). Log line on the lookup: `log.info("retrieved preferences for customer={}", id)` — works but customer ID is potentially PII. **Comment:** consider hashing the customer ID in logs, or scope the log to debug level.
6. **Category 4 — Testing.** 4 unit tests cover: happy path, customer not found, unauthorized, validation error. Missing: cross-tenant access denial. Tied to the Category 2 finding — add the test alongside the fix.
7. **Category 5 — Performance.** Service method calls `repository.findById(id)` and then `repository.findPreferencesByCustomerId(id)`. **N+1 / two-round-trip.** Single JPA query with `JOIN FETCH` would do both. Comment with the fix; not a hard fail because the query volume is low, but track it.
8. **Category 6 — Infrastructure.** No Terraform changes. Skip.
9. **Category 7 — Documentation.** OpenAPI spec is updated (the diff includes the YAML change). No ADR needed for a single endpoint. No SLO change. No SOC 2 control change. Good.

**Verdict:** Request changes. Two hard fails (404 semantics; tenant scoping). Non-blocking comments on logging PII and N+1 query. Author addresses, re-review takes 15 minutes, merges.

## Anti-pattern — "approve and move on" reviewing

**Bad:** Reviewer reads the PR title, looks at the file list, sees CI green, and approves without reading the diff. The "review" is a rubber stamp.

**Why it fails:** This is how the team accumulates technical debt and security issues. CI catches what tests cover; not all problems are testable. Code review is the human layer of defense, especially for: cross-tenant access, secret handling, observability gaps, IaC scope creep, dependency security. None of these are reliably caught by CI.

**Detection signal:** PRs merging with a single "LGTM" comment in under 60 seconds for a non-trivial diff. Or: the same reviewer approving every PR from a particular author without comment, regardless of size.

**Fix:** Set a minimum-effort bar — at minimum, the 7 categories scanned for hard-fails (5 minutes for a typical PR). If the diff is genuinely trivial (a typo, a comment, a dependency version bump with no breaking change), a one-line "LGTM, trivial" is appropriate and the reviewer should say *why* it was trivial. If the diff is non-trivial, the review must show evidence of having read it: at least one substantive comment, even if positive ("the tenant scoping in `findByCustomerAndTenant` is the right shape").

## Verification questions

1. Did the review walk all 7 categories explicitly, or did it skip some?
2. For each hard-fail, is there a specific named defect (not "improve security" but "no tenant scoping on /v1/customers/{id}/preferences")?
3. Did the review reference the relevant anti-pattern documents (Spring Boot, Go, Terraform) when patterns matched?
4. Is the verdict explicit (approve / approve with comments / request changes / hard reject), not implicit?
5. For large PRs (>500 LOC of meaningful change): was the size flagged, and was a split requested or justified?
6. For PRs that touch observability-relevant code: are the alerts or SLOs updated alongside?

## What to read next

- `references/code-review-checklist.md` — the full 7-category checklist with per-item detection cues
- `references/spring-boot-anti-patterns.md` — Java/Spring patterns to flag in review
- `references/go-anti-patterns.md` — Go patterns to flag in review
- `references/terraform-anti-patterns.md` — Terraform AzureRM patterns to flag
- `new-azure-service-bootstrap` skill — for the standard scaffold every PR should respect
- `azure-microservices-security` skill — for the auth/authz patterns to verify in review
- `azure-microservices-observability` skill — for the observability bar new code must meet
- `soc2-iso27001-controls-mapping` skill — for compliance-relevant changes (any change that affects evidence sources)
