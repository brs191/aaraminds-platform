# AaraMinds Claude Skills Pack — v10.0

**Date:** May 2026
**Format:** Claude Skills (Anthropic native, progressive disclosure)
**SDK target:** `github.com/mark3labs/mcp-go` (community) or `github.com/modelcontextprotocol/go-sdk` (official) — see `versions.md`
**Go version:** 1.25.x minimum, 1.26.x recommended
**Stack:** Azure-primary; Spring Boot / Go backends; Next.js / React frontend; Postgres / Mongo / Cosmos DB; Terraform AzureRM; GitHub Actions OIDC; Grafana + Prometheus + OpenTelemetry

> **Accuracy note:** Counts here are reconciled against `.claude/INDEX.md` and the filesystem — **35 Tier-1 skills** as of 2026-06-18. The pack grew from an original 15 (pre-2026-05-21) through staged additions; `.claude/INDEX.md`, generated from disk by `skill_audit.py`, is the single source of truth and is verified against every living doc on each audit.

## What this pack is

A self-contained Claude Skills pack covering **Azure-hosted microservices design**, **Go MCP server building**, **new-service bootstrap**, **PR review**, **operational data-tier design**, and **SOC 2 / ISO 27001 controls mapping**. Twenty-nine Tier-1 skills under `.claude/skills/` route Claude to the right deep content. Tier-2 references and pattern cards provide the operational depth. A working example MCP server with 13 tools, an MCP-driven demo across three architectures, and a validation pack ship alongside.

The pack was built in phases:

- **Phase 1 (format migration)** — the v9.0 knowledge pack migrated into native Claude Skills format with progressive disclosure (`SKILL.md` routers + `references/` deep content), governance via `.claude/CLAUDE.md`, proper YAML frontmatter on every skill. Twelve Tier-1 skills delivered.
- **Phase 2 (scope expansion)** — three new Tier-1 skills authored from scratch: `new-azure-service-bootstrap`, `pr-review-azure-microservices`, `soc2-iso27001-controls-mapping`.
- **Phase 3 (agents and hooks)** — three multi-skill agent personas under `.claude/agents/` and three Claude Code hook templates under `.claude/hooks/`.
- **Phase 4 (validation cleanup)** — the never-run per-skill evals were removed; the 12 capability prompts under `validation/prompts/` remain as the validation surface; per-skill verification moved into each `SKILL.md`.
- **Post-phase additions** — eleven further Tier-1 skills were authored after the original four phases closed: `azure-data-tier-design` (operational data-tier depth), `mcp-go-guardrails-and-safety` (layered runtime + CI guardrails), `microservices-architecture-reviewer` (9-dimension verdict review), `ai-application-architecture` (architecture of LLM/AI-powered application features on Azure, added 2026-05-22), `ai-evaluation-harness` (evaluation-harness design for AI/LLM features, added 2026-05-24), `codebase-comprehension` (static-analysis codebase modeling, added 2026-05-24), `codebase-extraction-engineering` (implementing the code extractor, added 2026-05-24), `python-service-engineering` (building the Python service tier, added 2026-05-24), `data-access-engineering` (the graph/relational data-access layer, added 2026-05-24), `test-engineering` (the cross-stack test suite, added 2026-05-24), and `frontend-engineering` (the React/Next.js frontend tier, added 2026-05-24). These brought the total to 26; three network-analysis skills (`azure-network-topology-analysis`, `azure-network-cost-forecasting`, `azure-network-iac-generation`) added 2026-06-03 bring the current total to 29. A static-analysis tool (`validation/tools/skill_audit.py`), a generated discovery index (`.claude/INDEX.md`), and the `inherit` model tier on one agent were added in the same period (see `validation/snapshots/inspiration_hobson.md`).

## Pack structure

```text
skills-pack/
├── README.md                              # This file
├── ROADMAP.md                             # Release notes and forward path
├── VERIFICATION_CHECKLIST.md              # Step-by-step verification for adopters
├── versions.md                            # Ecosystem version pins (Go, SDKs, MCP spec)
├── migration-map.md                       # v9.0 → v10.0 file moves
├── usage.md                               # Wiring the pack into a repo (Claude Code)
├── how-to-use-in-vscode.md                # Day-to-day use under VS Code + Copilot
│   # Per-artifact ratings live in the workspace master ranking: ../Ranking.md
│
├── .claude/
│   ├── CLAUDE.md                          # Pack-wide Claude behavior governance
│   ├── INDEX.md                           # Auto-generated discovery index
│   ├── FEEDBACK.md                        # Inter-session feedback log
│   ├── skills/                            # 35 Tier-1 skills (progressive disclosure)
│   │   └── <skill-name>/
│   │       ├── SKILL.md                   # Tier-1: ~80–120-line router with frontmatter
│   │       └── references/                # Tier-2: deep content per topic
│   │           ├── <topic>.md
│   │           └── patterns/              # Pattern cards (where relevant)
│   ├── agents/                            # 4 multi-skill personas + README
│   │   ├── aara-senior-microservices-architect.md   # End-to-end architecture (opus)
│   │   ├── aara-mcp-server-builder.md               # Building / reviewing Go MCP servers (inherit)
│   │   ├── aara-azure-cost-reviewer.md              # FinOps bill review & sizing (sonnet)
│   │   └── aara-network-topology-reviewer.md        # Network reachability review (inherit)
│   └── hooks/                             # 3 pre-commit / test / guard hooks + README
│       ├── pre-commit-lint.json
│       ├── test-before-commit.json
│       └── block-dangerous-commands.json
│
├── copilot/                               # VS Code + GitHub Copilot adapter
│   ├── README.md                          # Install / verify
│   ├── install.sh                         # Idempotent installer
│   ├── mcp.json                           # MCP server config for Copilot agent mode
│   └── agents/*.agent.md                  # The 17 agents as VS Code Custom Agents
│
├── examples/microservices-system-design-mcp-server/
│   ├── README.md                          # How to build, run, configure
│   ├── go.mod / go.sum / Makefile / Dockerfile
│   ├── cmd/server/main.go                 # Entry point with transport selection
│   ├── internal/
│   │   ├── mcpserver/server.go            # Composition root; wires all 13 tools
│   │   ├── services/                      # 11 service packages
│   │   └── tools/                         # 11 tool packages
│   ├── contracts/architecture-tools/implemented/   # 10 architecture-tool contracts
│   └── testdata/                          # Input + golden output JSON per tool
│
├── demo/architecture-review-demo/
│   ├── README.md / Makefile
│   ├── demo_runner.py                     # Stdlib-only Python MCP client over stdio
│   ├── validate_outputs.py                # Canonical-JSON comparison vs. goldens
│   ├── input/                             # 3 master architecture descriptions
│   ├── golden/<arch>/<tool>.json          # Captured outputs from the live server
│   └── out/<arch>/<tool>.json             # Last generated run
│
└── validation/
    ├── README.md
    ├── prompts/                           # 12 capability prompts with rubrics + reference outputs
    │   ├── mcp-server-building/           # 3 prompts
    │   ├── microservices-design/          # 4 prompts
    │   ├── architecture-review/           # 3 prompts
    │   └── cross-cutting/                 # 2 prompts
    ├── governance/
    │   ├── freshness-cadence.md           # Quarterly refresh ownership
    │   └── release-checklist.md           # Pre-release runbook
    ├── tools/
    │   ├── skill_audit.py                 # Static-analysis + index generator + doc-consistency check
    │   └── README.md
    ├── snapshots/                         # Frozen point-in-time records (reviews, plans, logs)
    └── skill-audit-*.md                   # Static-analysis reports (regenerate with skill_audit.py)
```

## The 35 Tier-1 skills

Each skill has a `SKILL.md` with frontmatter (`name`, `description`, `version`, `last_updated`) and a `references/` folder with deep content. Progressive disclosure: Claude reads the SKILL.md first to decide if it applies, then drills into references as needed. The tables below list *what each skill is for*; per-skill reference and pattern-card counts are deliberately **not** repeated here — the generated `.claude/INDEX.md` is the single source for those numbers.

### Microservices design (6 skills)

| Skill | When Claude routes here |
|---|---|
| `microservices-architecture-design` | Designing a new microservices system end-to-end; deciding whether microservices are right for the problem; producing an ADR |
| `microservices-architecture-reviewer` | Reviewing an existing or proposed estate end-to-end; producing a 9-dimension structured verdict report |
| `microservices-data-architecture` | Cross-service data consistency: saga, outbox, CQRS, event sourcing, idempotent consumer, database-per-service |
| `microservices-resilience` | Timeouts, retries, circuit breakers, bulkheads, queue-based load leveling, rollout patterns (blue-green, canary, strangler fig) |
| `microservices-async-messaging` | Sync vs. async, broker selection (Service Bus / Event Grid / Event Hubs), ordering, DLQ, tracing across async hops |
| `microservices-api-design` | REST and gRPC contracts, versioning, error model, pagination, idempotency, API Management placement, Backend-for-Frontend |

### Azure platform (5 skills)

| Skill | When Claude routes here |
|---|---|
| `azure-service-mapping` | Picking the Azure service for compute / data / messaging / cache / gateway / discovery / mesh |
| `azure-data-tier-design` | Operational data-tier design: engine selection, schema and index design, query execution, partitioning, HA/DR, zero-downtime migration |
| `azure-microservices-observability` | OpenTelemetry instrumentation, Grafana + Prometheus dashboards, SLO design, alert configuration |
| `azure-microservices-security` | OAuth 2.1 with Entra ID, Managed Identity, Key Vault, network segmentation, audit logging, zero-trust |
| `azure-microservices-cost-review` | Monthly bill review, infrastructure sizing, reserved instances, scale-to-zero, FinOps recommendations |

### MCP server building (4 skills)

| Skill | When Claude routes here |
|---|---|
| `mcp-go-server-building` | Designing or extending a Go MCP server: SDK choice, server skeleton, tool design, transport, resources, prompts, code generation |
| `mcp-go-production-review` | Pre-production readiness review; CI/CD gate design; deployment manifest; 10-section production review |
| `mcp-go-threat-modeling` | STRIDE for MCP servers; MCP-specific threats (prompt injection, output-as-instructions, tool composition abuse); security test generation |
| `mcp-go-guardrails-and-safety` | Layered runtime + CI guardrails: tool-handler middleware chain, argument sanitization, output redaction, prompt-injection defense, audit logging, authorization, CI eval gate |

### Engineering workflow + compliance (3 skills)

| Skill | When Claude routes here |
|---|---|
| `new-azure-service-bootstrap` | Scaffolding a new Spring Boot or Go service; choosing Java vs. Go; standard `.github/workflows/` with OIDC; Container Apps + Terraform AzureRM |
| `pr-review-azure-microservices` | Reviewing a PR on Spring Boot, Go, or Terraform changes; producing a structured review with hard-fails called out per category |
| `soc2-iso27001-controls-mapping` | Producing SOC 2 / ISO 27001 audit evidence; mapping controls to Azure-native sources (Entra ID logs, Defender, Activity Log, Sentinel); writing the Statement of Applicability |

### AI application design (2 skills)

| Skill | When Claude routes here |
|---|---|
| `ai-application-architecture` | Designing or reviewing an LLM/AI-powered application feature: application-archetype selection, the model and inference layer, RAG design, orchestration-framework choice, evaluation, safety, and the Python/Go/Next.js serving topology. |
| `ai-evaluation-harness` | Building the evaluation harness for an AI/LLM feature: golden datasets and reference fixtures, rubric and metric design, scoring (deterministic vs LLM-as-judge), CI eval gating with regression baselines, and online evaluation with drift detection. |

### Code comprehension (1 skill)

| Skill | When Claude routes here |
|---|---|
| `codebase-comprehension` | Extracting a trustworthy structural model from an existing codebase: AST extraction and parsing, call and dependency graphs, Spring Boot stereotype modeling, generated-code handling, and incremental rebuilds with stable identity |

### Implementation engineering (5 skills)

| Skill | When Claude routes here |
|---|---|
| `codebase-extraction-engineering` | Implementing the static-analysis extractor: Java parser selection and symbol resolution, the AST-to-graph extraction pipeline, call-graph resolution, build integration for generated code, and incremental rebuilds |
| `python-service-engineering` | Building production Python services: project structure, type and Pydantic discipline, async, the Pydantic AI / LangGraph orchestration code, and runtime concerns — config, secrets, telemetry |
| `data-access-engineering` | Implementing the data-access layer: graph traversal queries in Cypher and Gremlin, the idempotent graph-builder write path, relational expand/contract migrations, query discipline, and a repository-style access boundary |
| `test-engineering` | Designing and writing the test suite across the stack: unit and table-driven Go tests, pytest, integration tests against real dependencies, characterization tests that pin legacy behavior before a change, test doubles and fixtures, and test-suite health — the pyramid, flakiness, CI gating |
| `frontend-engineering` | Building the React and Next.js frontend: React component design and state, Next.js App Router architecture and rendering strategy, the Backend-for-Frontend route tier, streaming an LLM token response to the browser, and TypeScript discipline at the API boundary |

### Network analysis (3 skills)

| Skill | When Claude routes here |
|---|---|
| `azure-network-topology-analysis` | Reachability-based Azure network topology risk review: Resource Graph ingest, effective NSG/route/AVNM evaluation, DNAT and peering transitivity, the five finding types with deterministic severity |
| `azure-network-cost-forecasting` | Design-time Azure network cost forecast: fixed-exact and variable-band models from the Retail Prices API for firewalls, gateways, egress, and private endpoints |
| `azure-network-iac-generation` | Generating validated Terraform for Azure network topology from intent: vetted CAF/ALZ modules, analyzer-gated, PR-only output |

## What you get, by area

### Skills

The 35 Tier-1 skills are backed by a body of Tier-2 reference files and pattern cards. The exact counts — total references, pattern cards, and the per-skill breakdown — are generated into `.claude/INDEX.md` and are not restated here, so there is one place to look and nothing to keep in sync by hand. Every Tier-1 `SKILL.md` follows the same shape: frontmatter; "When to use" with explicit "do not use for X (use Y instead)" disambiguation; critical decision rule; framework or selector table; brownfield worked example; named anti-pattern with detection signal; verification questions; "What to read next."

Tier-2 references are the operational depth: decision tables, worked code, Azure-service specifics, pattern-by-pattern detail. Pattern cards live under the most relevant skill's `references/patterns/` directory and cover the standard microservices patterns (Saga, CQRS, Circuit Breaker, Event Sourcing, Zero-Trust, etc.) with pattern-specific Problem, Use When, Avoid When, Trade-offs, Failure Modes, and Decision Signals. The full card-to-skill map is in `.claude/INDEX.md`.

### Agents (3) and hooks (3)

Three multi-skill agent personas under `.claude/agents/`: `aara-senior-microservices-architect` (opus), `aara-mcp-server-builder` (inherit), `aara-azure-cost-reviewer` (sonnet). Agents decide *when* to invoke skills; they compose with them rather than duplicating them.

Three Claude Code hook templates under `.claude/hooks/`: `pre-commit-lint`, `test-before-commit`, `block-dangerous-commands`. Hooks are event-driven shell commands that enforce guardrails at the tool-call layer. They are JSON templates — symlinking `hooks/` does not make them fire; merge them into a `settings.json` to activate (see `.claude/hooks/README.md`). Note: the hooks parse tool input with `jq`; on a host without `jq` they fail open. Confirm `jq` is installed before relying on them.

### Example MCP server (13 tools)

A complete Go MCP server demonstrating the patterns in the skills. Builds clean, tests pass under `-race`, fits a multi-stage distroless Dockerfile. Tools include service-boundary canvas generation, architecture risk detection, API contract generation, Azure-service mapping, observability plan generation, ADR generation, deployment topology, event contracts, resilience plans, diagram-asset generation, and the three design-scoring tools (`recommend_microservice_patterns`, `review_microservice_design`, `score_well_architected_readiness`).

Each tool follows the package-layering rule from `mcp-go-server-building`: rule logic in `internal/services/<name>/`, MCP wiring in `internal/tools/<name>/register.go`, contract document in `contracts/architecture-tools/implemented/<name>.md`, table-driven tests next to the service.

### Demo (3 architectures × 5 tools)

The demo at `demo/architecture-review-demo/` is authentic: a stdlib-only Python MCP client over stdio that spawns the Go server, performs the JSON-RPC handshake, and calls five tools per architecture across three deliberately distinct shapes (e-commerce / PCI-DSS, retail banking / SOX, HIPAA patient care / PHI). Outputs are captured as goldens; `make validate` does canonical-JSON comparison so drift is precise. The demo exercises 5 of the server's 13 tools.

### Validation pack

- **12 capability prompts** with rubrics and reference outputs spanning MCP-server-building (3), microservices-design (4), architecture-review (3), and cross-cutting concerns (2).
- **`skill_audit.py`** — a stdlib-only static-analysis tool that lints skill structure (description length, required sections, dead cross-references, stale dates, off-stack drift) and regenerates `.claude/INDEX.md` with `--emit-index`.
- **Governance**: freshness cadence with ownership, and a pre-release runbook.
- Per-skill verification lives in each `SKILL.md`'s "Verification questions" section.

## Quick start

### Verify the example server builds and tests pass

```bash
cd examples/microservices-system-design-mcp-server
go mod tidy && go build ./...
go test -race -count=1 ./...
```

If your local Go is older than the `go.mod` declares, see the Docker recipe in [examples/microservices-system-design-mcp-server/README.md](examples/microservices-system-design-mcp-server/README.md).

### Run the demo end-to-end

```bash
cd examples/microservices-system-design-mcp-server
go build -o ./mcp-server ./cmd/server

cd ../../demo/architecture-review-demo
export MCP_SERVER_BIN=../../examples/microservices-system-design-mcp-server/mcp-server
make demo && make validate
# Expect: "Validation passed: 3 architecture(s) × 5 tool(s) all match golden fixtures."
```

### Re-generate the discovery index after any structural change

```bash
python3 validation/tools/skill_audit.py --emit-index
```

This runs the static-analysis lint and rewrites `.claude/INDEX.md` from the filesystem. Run it whenever a skill, reference, agent, or hook is added, removed, or renamed — `.claude/INDEX.md` drifts silently otherwise.

### Use the skills with Claude Code

The `.claude/` directory is the entry point. Claude Code reads `.claude/skills/` natively when the pack is opened as a workspace; the pack-wide governance in `.claude/CLAUDE.md` loads automatically. To wire the pack into another repo, see [usage.md](usage.md). To run it under VS Code + GitHub Copilot, see [how-to-use-in-vscode.md](how-to-use-in-vscode.md) and [copilot/README.md](copilot/README.md) — note that on Copilot, skill auto-routing, hooks, and progressive disclosure do not apply; the skills become a manually-attached knowledge base.

## Format choices and trade-offs

**Progressive disclosure (Tier-1 SKILL.md → Tier-2 references)** is the central format choice. Claude reads only what it needs: a SKILL.md to decide whether the skill applies (80–120 lines) and then the specific reference for the depth it needs. This keeps the context window tight while preserving depth. The payoff is realized on Claude Code, which auto-routes on SKILL.md descriptions; under Copilot it does not, and loading becomes manual.

**Twenty-nine Tier-1 skills, grouped by trigger**: skills are grouped by *when Claude should invoke them*, not by file taxonomy. The v9.0 architecture skills 01–04 (router, process, decomposition, boundaries) collapse into `microservices-architecture-design` because they answer the same question.

**Frontmatter governance is strict**: name, description with "Use when X. Do not use for Y" disambiguation, version, last_updated. No custom fields. The description is what Claude reads to decide invocation.

**Stack is pinned**: Azure-primary, Terraform AzureRM, GitHub Actions OIDC, Spring Boot / Go, Next.js / React, Postgres / Mongo / Cosmos DB. `.claude/CLAUDE.md` enforces this; Claude does not propose AWS, Bicep, Pulumi, GitLab CI, or Datadog "for illustration."

## Quality position — what is and is not validated

Be precise about what evidence backs this pack:

- **Verified, tested, deterministic** — the example MCP server. It builds, tests pass under `-race`, and the demo's `make validate` confirms 3 architectures × 5 tools reproduce captured goldens byte-for-byte. This is the strongest evidence in the pack.
- **Statically linted, not behaviorally tested** — the 32 skills (29 audited 2026-06-04; the 3 added 2026-06-15 are not yet behaviorally audited). `skill_audit.py` checks structure (sections present, descriptions sized, cross-references resolve). It does not test whether a skill produces a good answer. The 12 capability prompts under `validation/prompts/` exist to test that, but have not been run end-to-end; running them is open work.
- **Not yet tested** — the 17 agents. Per the workspace master ranking [`../Ranking.md`](../Ranking.md), agent behavior requires a registered Claude Code session and is marked `n/t`.
- **Functional but environment-dependent** — the 3 hooks. They block correctly when `jq` is present and fail open when it is not.

Earlier revisions of this README cited an "8/10 independent pre-ship review" and "9.0+ quality" for v9.0. Those figures had no attached baseline, rubric, or reviewer and have been removed; the pack's own `.claude/CLAUDE.md` forbids unsourced metrics.

| Evidence | Where to find it |
|---|---|
| Deterministic MCP server outputs | `demo/architecture-review-demo/golden/` + `make validate` |
| Static-analysis + doc-consistency report | `validation/skill-audit-*.md` (regenerate with `skill_audit.py`) |
| Capability-level prompts with rubrics | `validation/prompts/` (12 prompts — not yet run) |
| Per-skill verification questions | "Verification questions" section in each `SKILL.md` |
| Pre-release runbook | `validation/governance/release-checklist.md` |
| Per-artifact rating + honest test status | [`../Ranking.md`](../Ranking.md) — workspace master ranking |

See [VERIFICATION_CHECKLIST.md](VERIFICATION_CHECKLIST.md) for the step-by-step verification recipe.

## Known gaps

- `review_microservice_design` in the example server flags `Container Apps` as "non-Azure-native" — a known false positive tracked in [`../Ranking.md`](../Ranking.md) (Notes & caveats).

> **Resolved 2026-07-03:** the canonical copy is now the single pack folder at `/home/raja/projects/aaraminds-platform/skills-pack`. Older paths in dated snapshots are historical. `usage.md`, `how-to-use-in-vscode.md`, the Copilot adapter, and agent files should stay pack-root-relative where possible.

## License and authorship

This pack is a personal, reusable artifact for a senior IC + architect. Bring it into your own project as a snapshot; adapt the patterns and skills to your context rather than depending on them unchanged.
