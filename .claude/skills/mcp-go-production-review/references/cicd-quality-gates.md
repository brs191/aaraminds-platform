# Skill — MCP-Go CI/CD Quality Gates

## Purpose

Design CI/CD pipelines for MCP-Go servers with quality gates that block regressions early. The aim isn't a maximalist pipeline; it's a focused set of gates that catch the specific failure modes MCP servers have. Each gate has a clear purpose, a clear failure mode it prevents, and a clear cost.

## Gate hierarchy

```
On every PR:
  ├── Lint:        gofmt, go vet
  ├── Build:       go build ./...
  ├── Test:        go test -race ./...
  ├── Security:    govulncheck, dependency scan
  └── Contract:    contract tests pass

On main merge:
  └── Image build: multi-stage, distroless, scan, tag

On manual / nightly:
  ├── Demo run:         make demo && make validate (or equivalent)
  ├── Capability prompts: validation/prompts/* (≥80% pass at declared thresholds)
  └── Smoke test:        deploy to staging, run smoke

Pre-release tag:
  └── Full release-checklist.md walked
```

PR gates are fast (<3 minutes for a small server). Main gates may run images and longer tests (<10 minutes). Nightly gates and pre-release gates can be slower (eval and demo runs).

## Gate 1 — Lint (fast)

```yaml
- name: gofmt
  run: |
    out=$(gofmt -l .)
    if [ -n "$out" ]; then
      echo "Files need gofmt:"
      echo "$out"
      exit 1
    fi
- name: go vet
  run: go vet ./...
```

What it catches: style inconsistency, suspicious constructs (`go vet` finds shadowed variables, format-string mismatches, unreachable code).

Why it's gate 1: cheap, fast, never flaky.

## Gate 2 — Build

```yaml
- name: build
  run: go build ./...
```

What it catches: type errors, dependency drift, import path issues.

Note: `go build ./...` compiles everything but produces no artifacts. If you want a production-shaped binary, also build `./cmd/server`.

## Gate 3 — Test with race detector

```yaml
- name: test
  run: go test -race -count=1 ./...
```

`-race` catches data races. `-count=1` defeats Go's test result cache so the tests actually run.

What it catches: regressions in service logic, handler errors, broken contract tests, races in concurrent code (background workers, request handlers under load).

This is the highest-value gate. Tests should fail noisily and informatively.

## Gate 4 — Security

```yaml
- name: govulncheck
  run: |
    go install golang.org/x/vuln/cmd/govulncheck@latest
    govulncheck ./...
- name: dependency-scan
  uses: github/dependabot@v3   # or equivalent
```

What it catches:
- `govulncheck` flags CVEs in your dependency tree.
- Dependabot raises PRs to update vulnerable packages.

For container images, add an image scanner (Trivy, Grype, or Microsoft Defender for Cloud) on the post-merge build:

```yaml
- name: trivy
  uses: aquasecurity/trivy-action@master
  with:
    image-ref: '<registry>/mcp-server:${{ github.sha }}'
    severity: 'CRITICAL,HIGH'
    exit-code: 1
```

Fail the build on `CRITICAL` and `HIGH`. Lower severity logs but doesn't block.

## Gate 5 — Contract tests

A "contract test" verifies the MCP server adheres to the protocol shape clients depend on:

- `tools/list` returns the expected catalog (no accidentally unregistered tools).
- Each tool's input schema matches its documented contract.
- Error responses follow the structured shape.

```go
func TestContract_ToolCatalog(t *testing.T) {
    expected := []string{
        "generate_service_boundary_canvas",
        "generate_api_contract",
        "detect_architecture_risks",
        // ...
    }
    // Spin up the server, call tools/list, compare.
}
```

What it catches: silent tool registration loss; schema drift breaking external clients.

## Gate 6 — Image build (post-merge or release)

```yaml
- name: docker-build
  run: |
    docker build -t mcp-server:${{ github.sha }} .
    docker run --rm mcp-server:${{ github.sha }} --version  # smoke that the binary runs
```

What it catches: Dockerfile drift, dependency layering bugs, base-image breakage.

Combine with image scanning (gate 4) and image signing if your environment requires it.

## Gate 7 — Demo run (nightly or pre-release)

The pack ships a demo at `demo/architecture-review-demo/`. Wire it into CI:

```yaml
- name: build-server
  run: go build -o ./mcp-server ./cmd/server
  working-directory: examples/microservices-system-design-mcp-server

- name: demo
  run: |
    MCP_SERVER_BIN=$(pwd)/examples/microservices-system-design-mcp-server/mcp-server \
      make -C demo/architecture-review-demo demo

- name: validate
  run: make -C demo/architecture-review-demo validate
```

What it catches: end-to-end regressions. If demo goldens drift unintentionally, this fails before release.

## Gate 8 — Capability prompts (pre-release)

The pack ships 12 capability prompts at `validation/prompts/`. Run them against your LLM of choice (Claude API, OpenAI, etc.); fail the release if fewer than 80% pass at the prompt's declared threshold. (80%, not 90% — workflow-level prompts are harder than per-skill checks; calibrate accordingly.)

This gate is LLM-driven and slower; it belongs on the pre-release path, not on every PR.

## Branch protection / required checks

```
Required for PR merge:
  ✓ lint
  ✓ build
  ✓ test
  ✓ govulncheck
  ✓ contract

Required for release tag:
  ✓ all PR checks
  ✓ image build + scan
  ✓ demo run + validate
  ✓ capability prompts ≥ 80%
  ✓ release-checklist.md walked
```

PR-blocking checks should complete in <5 minutes. Anything slower runs post-merge or pre-release.

## Cost discipline

Each gate has a cost:
- Lint, build, test: cheap; always.
- Race tests: slightly slower but still cheap; always.
- Security scans: cheap; always.
- Image build: moderate; post-merge.
- Demo: moderate; nightly.
- Per-skill evals: LLM cost (real money); pre-release only.

Resist the urge to gate everything on every PR. Slow PR gates kill developer productivity; the fix becomes "skip CI", which defeats the gates.

## Common failure modes

- **Lint without `-l` check.** `gofmt -d` shows the diff but doesn't fail the build. Detection: gofmt issues land in main. Fix: use `gofmt -l . | tee /dev/stderr | (! grep .)`.
- **`go test` without `-race`.** Concurrency bugs slip through. Detection: race-related production incidents. Fix: always run `-race` in CI.
- **No image scan.** Vulnerable base layers ship. Detection: image scanner in production flags the deployed image. Fix: scan at build time, fail on high/critical.
- **Demo as a PR gate.** Demo takes 5 minutes, runs on every PR, developers grumble and bypass CI. Detection: bypass labels everywhere. Fix: move demo to nightly or pre-release; PR gates stay <5 min.
- **Eval gate without ownership.** Capability prompts fail; release is blocked; nobody knows what to do. Detection: stale failing prompt runs. Fix: capability prompts come with named maintainers via `../../../../validation/governance/freshness-cadence.md`.
- **Required checks not configured in branch protection.** CI passes locally and on PR, but merging-without-CI is possible. Detection: someone bypasses CI; bugs land. Fix: enforce required checks in branch protection settings.

## Verification questions

1. What's the PR-gate runtime? Is it under 5 minutes?
2. Is `go test -race` running, not just `go test`?
3. Is there a dependency vulnerability scan?
4. Is the image scanned for CVEs at build time?
5. Are demo and eval gates on a nightly or pre-release path, not on every PR?
6. Are required checks enforced in branch protection?

## What to read next

- `testing.md` — what tests should look like
- `deployment.md` — what the image must be
- `../../mcp-go-threat-modeling/references/security-test-generation.md` — generating security test cases
- `../../../../validation/governance/release-checklist.md` — the pre-release runbook this gate ties into
