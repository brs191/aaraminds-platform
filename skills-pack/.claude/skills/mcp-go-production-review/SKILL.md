---
name: mcp-go-production-review
description: Conducts pre-production readiness review of Go MCP servers, covering layered testing (unit, contract, transport, security, smoke), deployment shape (Dockerfile, Container Apps manifest, blue-green rollout), CI/CD quality gates (PR / post-merge / nightly / pre-release), and a structured 10-section review checklist. Use when a Go MCP server is approaching first deploy, when reviewing readiness before a tagged release, when designing the CI pipeline for an MCP server, or when triaging an MCP server's anti-pattern smells. Do not use for designing or writing the server (use mcp-go-server-building) or for threat modeling (use mcp-go-threat-modeling).
version: 1.0.0
last_updated: 2026-05-18
---

# MCP Go Production Review

## When to use

Trigger this skill when an MCP server is moving from "works locally" to "ready for production": pre-deploy review, CI gate design, deployment manifest review, post-incident readiness audit, or producing a 10-section production-review report. Common triggers: "is this MCP server ready to ship," "design the CI pipeline for this server," "review this Dockerfile," "we hit a production issue — what did we miss in review."

Do **not** use this skill for: server design and tool-shape decisions (`mcp-go-server-building`); threat modeling and security testing (`mcp-go-threat-modeling`); broader microservices production posture outside the MCP layer (`azure-microservices-security`, `microservices-resilience`, `azure-microservices-observability`).

## The critical decision rule — every PR gate runs in under 5 minutes

If the PR gate is slow, developers route around it. The discipline collapses. PR-blocking checks (lint, vet, build, race tests, security scan, contract test) **must** complete in under 5 minutes for a typical service. Demo runs, full integration suites, image scans against fresh CVE databases — those belong post-merge or pre-release, not on the PR.

Once PR gates are fast, then add slower gates on post-merge (Docker build, image scan) and nightly (demo, eval pack) and pre-release (10-section review). This is gate hierarchy; ignoring it is how teams end up with a slow CI that nobody respects.

## The 10-section review framework

| # | Section | What to check | Reference |
|---|---|---|---|
| 1 | Server skeleton | `main.go` minimal, slog-to-stderr, bounded shutdown | `mcp-go-server-building` |
| 2 | Tool design | One intent per tool, typed input struct, action-verb names | `mcp-go-server-building` |
| 3 | Project structure | `internal/services/` vs `internal/tools/` separation enforced | `mcp-go-server-building` |
| 4 | Security | Risk tiers, auth at transport, per-tool authz, no plaintext secrets | `references/production-review.md` |
| 5 | Observability | Tool-call event log, per-tool metrics, alert on error rate | `mcp-go-server-building` |
| 6 | Testing | Unit + handler + service + contract + security + smoke | `references/testing.md` |
| 7 | Deployment | Multi-stage distroless image, health probes, resource limits | `references/deployment.md` |
| 8 | Documentation | README, per-tool contracts, runbook | `references/production-review.md` |
| 9 | Freshness | Ecosystem facts dated within the quarter | `references/production-review.md` |
| 10 | Validation | Per-skill evals or capability prompts have current pass status | `references/production-review.md` |

For the full checklist with per-section pass/soft-fail/hard-fail criteria, see `references/production-review.md`.

## Review-pass logic

1. **Run the gate hierarchy first.** Before opening the production-review document, confirm CI is green: gofmt clean, `go vet` clean, `go build ./...` clean, `go test -race ./...` clean, `govulncheck ./...` clean. If CI is red, the review doesn't start. See `references/cicd-quality-gates.md`.

2. **Walk the 10 sections in order.** Each section gets a pass / soft-fail / hard-fail rating with a one-line note explaining the call. Hard-fails block release; soft-fails get a tracked follow-up but don't block.

3. **For each hard-fail, name the specific defect.** Not "improve security" — "no per-tool authorization; any authenticated caller can invoke `deploy_service`." Specificity is the discipline.

4. **Verify deployment shape.** Image is multi-stage distroless, non-root, stripped (`-ldflags="-s -w"`), `CGO_ENABLED=0`. Container Apps manifest uses `activeRevisionsMode: Multiple` (enables blue-green/canary). Health probes hit a real `/healthz` endpoint that exercises readiness, not a static-200 handler. Resource limits are measured (P95 + 20% headroom), not arbitrary defaults. See `references/deployment.md`.

5. **Verify testing breadth.** Service layer has table-driven tests for every named rule. Tool handlers tested for missing/invalid input. At least one contract test against `tools/list`. Security suite covers oversized input, schema bypass, path traversal, auth bypass. Smoke test runs against the built binary post-deploy. See `references/testing.md`.

6. **Surface anti-patterns explicitly.** The known anti-pattern catalog covers stdout logging on stdio, business logic in `register.go`, untyped `map[string]any` inputs, kitchen-sink tools, swallowed errors, global state, missing risk tier on state-changing tools. Check each. See `references/anti-patterns.md`.

7. **Produce a verdict.** "Ready to ship," "Conditionally ready (with named conditions)," or "Not ready (with blockers)." Conditions and blockers are concrete and individually trackable.

## Worked example — brownfield: blocking a release that "passes tests" but has stdout logging

Setup: a Go MCP server is about to tag v1.2.0. CI is green. Service is on Container Apps. PR review didn't catch anything. Production-readiness review is the last gate before tagging.

Decision walk:

1. **Run CI gate verification.** `gofmt -l .` returns empty. `go vet ./...` clean. `go test -race -count=1 ./...` passes. `govulncheck ./...` reports one Medium CVE in a transitive dep — flag as a follow-up, not a blocker. See `references/cicd-quality-gates.md`.
2. **Section 1 (server skeleton).** Open `cmd/server/main.go`. Logger initialized as `slog.New(slog.NewJSONHandler(os.Stdout, ...))`. **Hard-fail.** Under stdio transport, stdout is the protocol wire; writing slog to stdout corrupts every MCP frame. Server appears broken to Claude. See `references/anti-patterns.md`. **Blocker.**
3. **Section 2 (tool design).** Tool list includes `manage_orders` with an `action` string argument that dispatches to create/list/cancel internally. **Soft-fail.** Kitchen-sink tool; should be split into three. Track as follow-up — not a blocker for v1.2.0 because the contract is stable and clients are already calling it.
4. **Section 4 (security).** Authentication: OAuth at API Management for HTTP transport; managed identity for service-to-service. Per-tool authorization: missing on `delete_workspace` — any authenticated caller can invoke it. **Hard-fail. Blocker.**
5. **Section 5 (observability).** Tool-call started/completed events emit with `tool` and `latency_ms` fields. Per-tool error-rate metric exists. Alert at 5% error rate over 5 min. **Pass.**
6. **Section 7 (deployment).** Dockerfile: multi-stage, distroless. Container Apps health probe on `/healthz` exercises a real readiness check (downstream connectivity). Resource limits 0.5 vCPU / 1 GiB — measured. Revisions configured. **Pass.**
7. **Verdict.** "Not ready." Two blockers: (a) stdout logging in `main.go` — fix is 3 lines of code; (b) per-tool authorization on `delete_workspace`. Once both land, re-run review. Soft-fail (kitchen-sink tool) tracked as a Q3 follow-up.

The CI being green was true but unimportant. The 10-section review is what caught the actual issues.

## Anti-pattern — confusing CI green with production-ready

**Bad:** "All tests pass and CI is green; we're good to ship." The team treats CI status as the readiness verdict.

**Why it fails:** Tests verify the code does what the tests describe; they do not verify the deployment shape, the secrets handling, the audit logging, the runbook, the resource limits, the rollback path, or any of the other concerns the production-review framework covers. A green test suite on a stdout-logging server still produces a broken production deployment.

**Detection signal:** in the release ticket / changelog, the only "evidence" listed is "tests pass" or "CI green." No reference to the 10-section review. No verdict. No blockers/follow-ups list.

**Fix:** Run the 10-section review for every release. Include the verdict and the verdict's rationale in the release notes. CI green is necessary, not sufficient.

## Verification questions

1. Is CI green (gofmt, vet, race tests, govulncheck) on the release commit, and is the runtime under 5 minutes?
2. Has the 10-section review been completed with an explicit verdict for this release?
3. For every hard-fail section: is there a specific named defect and an owner for the fix?
4. For every soft-fail: is there a tracked follow-up issue with a target date?
5. Are deployment-shape items verified (distroless image, health probes hit real readiness, resource limits measured)?
6. Is the rollback procedure documented and rehearsed within the last quarter?

## What to read next

- `references/production-review.md` — full 10-section checklist with pass / soft-fail / hard-fail criteria
- `references/testing.md` — the layered test pyramid (unit → handler → service → contract → security → smoke)
- `references/deployment.md` — Dockerfile recipe, Container Apps manifest, blue-green via revisions
- `references/cicd-quality-gates.md` — gate hierarchy (PR / post-merge / nightly / pre-release) and budget
- `references/anti-patterns.md` — named MCP server anti-patterns with detection signals
- `mcp-go-server-building` skill — design-time concerns this review verifies
- `mcp-go-threat-modeling` skill — security-test inputs that feed Section 4
