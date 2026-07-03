# Roadmap — v10.0

> **Accuracy note (2026-05-22):** This roadmap is reconciled against the pack on disk. Earlier revisions stopped at "15 skills, four phases complete." The pack subsequently grew to **32 skills** plus a static-analysis tool and a generated index. That work is recorded below as "post-phase additions" rather than being silently absorbed into the phase history.

## Current release

**v10.0 — May 2026 — Claude Skills format, 29 Tier-1 skills**

v10.0 migrates the v9.0 knowledge pack into Anthropic's Claude Skills format and extends it. Twelve skills were migrated; fourteen were authored fresh. The pack now ships:

| Area | Count | Status |
|---|---|---|
| Tier-1 skills (`.claude/skills/`) | 26 | Shipped |
| Tier-2 reference files | see `.claude/INDEX.md` | Shipped |
| Pattern cards | see `.claude/INDEX.md` | Shipped |
| Agents (`.claude/agents/`) | 3 | Shipped |
| Hooks (`.claude/hooks/`) | 3 | Shipped |
| Example MCP server | 1 (13 tools) | Builds, tests pass under `-race` |
| End-to-end demo | 1 (3 architectures × 5 tools) | `make validate` passes |
| Capability prompts (`validation/prompts/`) | 12 | Shipped — not yet run end-to-end |
| Static-analysis tool (`validation/tools/skill_audit.py`) | 1 | Shipped |
| Discovery index (`.claude/INDEX.md`) | 1 | Generated |
| Governance docs | 2 | Freshness cadence + release checklist |
| Copilot adapter (`copilot/`) | 1 | Shipped |

Reference-file and pattern-card counts are deliberately not pinned in this table: they change whenever a reference is added, and `.claude/INDEX.md` — generated from disk — is their single source of truth.

### Phase 1 — format migration (complete)

The v9.0 knowledge pack migrated into native Claude Skills format with progressive disclosure. No Tier-2 content was rewritten; existing skill files and pattern cards became Tier-2 references under twelve Tier-1 `SKILL.md` wrappers with YAML frontmatter.

| Item | Status |
|---|---|
| `.claude/CLAUDE.md` (pack-wide governance) | Shipped |
| Twelve Tier-1 skill folders with `references/` and pattern subdirs | Shipped |
| Reference files moved from v9.0 layout to per-skill `references/` folders | Shipped |
| Curriculum-style filenames (`01-foo.md`) renamed to capability-style (`foo.md`) | Shipped |
| Cross-references rewritten with correct relative paths | Shipped |
| Pattern cards distributed across relevant `references/patterns/` folders | Shipped |
| `migration-map.md` recording every file move | Shipped |

### Phase 2 — three new skills (complete)

Three Tier-1 skills authored from scratch (not migrated). Same format as Phase 1: SKILL.md router + `references/` deep content, frontmatter, brownfield worked example, named anti-pattern, verification questions.

| Skill | What it covers |
|---|---|
| `new-azure-service-bootstrap` | Scaffolding a new Azure microservice: Spring Boot 21+ or Go 1.25+, Terraform AzureRM, GitHub Actions OIDC, Container Apps deploy, OpenTelemetry from day one, Key Vault + Managed Identity |
| `pr-review-azure-microservices` | 7-category code-review checklist + Spring Boot / Go / Terraform anti-pattern catalogs; review-pass logic; worked example |
| `soc2-iso27001-controls-mapping` | SOC 2 TSC + ISO 27001 Annex A mapped to the Azure stack with KQL evidence queries; SoA template; evidence-pack structure |

### Phase 3 — agents and hooks (complete)

`.claude/agents/` populated with three multi-skill personas; `.claude/hooks/` with three Claude Code hook templates. Agents compose with skills (deciding *when* to invoke); hooks are event-driven shell commands at the tool-call layer.

| Agent | Model | Scope |
|---|---|---|
| `aara-senior-microservices-architect` | opus | End-to-end architecture design and review |
| `aara-mcp-server-builder` | inherit | Designing, building, threat-modeling, reviewing Go MCP servers |
| `aara-azure-cost-reviewer` | sonnet | FinOps — bill review, sizing, RI evaluation, idle detection |

| Hook | Event | Behavior |
|---|---|---|
| `pre-commit-lint.json` | `PreToolUse` on `git commit` | `gofmt -l` / `go vet` (Go) or `mvn spotless:check` (Java); blocks on failure |
| `test-before-commit.json` | `PreToolUse` on `git commit` | `go test -race` (Go) or `mvn test` (Java); blocks on failure; `TEST_BEFORE_COMMIT_SKIP=1` bypass |
| `block-dangerous-commands.json` | `PreToolUse` on `Bash` | Blocks `rm -rf /`, force-push to protected branches, `DROP DATABASE`, `az group delete *prod*`, prod `kubectl delete`, `curl ... \| bash`, fork bombs |

Hook caveat: all three parse `$CLAUDE_TOOL_INPUT` with `jq`. On a host without `jq` they fail open (pass everything through). Confirm `jq` is installed before relying on them; the workspace master ranking [`../Ranking.md`](../Ranking.md) records this.

### Phase 4 — validation cleanup (complete)

The 34 per-skill evals shipped under `validation/skill-evals/` were removed. Reasons: they were never run (`last_run: never`); their frontmatter referenced v9.0 paths that no longer existed after the Phase 1 moves; they duplicated coverage from the 12 capability prompts. Per-skill verification now lives in each `SKILL.md`'s "Verification questions" section. The prompt-pass threshold was recalibrated from "≥90% per-skill" to "≥80% capability-prompt."

### Post-phase additions (complete, after the original four phases)

Eleven further Tier-1 skills and supporting tooling were added after the four-phase build was declared closed. They are recorded here rather than retconned into the phase history.

| Addition | What it is |
|---|---|
| `azure-data-tier-design` skill | Operational data-tier depth — engine selection (Postgres / Cosmos / MongoDB / Azure SQL / MySQL / Redis), schema and index design, query execution, partitioning, HA/DR, zero-downtime migration. The deepest skill in the pack by reference count (see `.claude/INDEX.md`). |
| `mcp-go-guardrails-and-safety` skill | Layered runtime + CI guardrails for Go MCP servers — tool-handler middleware chain, argument sanitization, output redaction, prompt-injection defense, structured audit log, tool authorization, CI eval gate. |
| `microservices-architecture-reviewer` skill | 9-dimension end-to-end review of an existing or proposed estate, producing a structured verdict report. Split out from `microservices-architecture-design` so design and review have distinct triggers. |
| `ai-application-architecture` skill | Architecture of LLM- and AI-powered application features on Azure — application-archetype selection, the model and inference layer, retrieval design, orchestration-framework choice, evaluation, safety, and the Python/Go/Next.js serving topology. Added 2026-05-22; completed at skill version 1.0.0 — SKILL.md router plus six Tier-2 references and six archetype pattern cards. |
| `ai-evaluation-harness` skill | Evaluation-harness design for AI/LLM features — golden datasets and reference fixtures, rubric and metric design, scoring (deterministic checks vs LLM-as-judge), CI eval gating with regression baselines, and online evaluation with drift detection. Added 2026-05-24; SKILL.md router plus five Tier-2 references. |
| `codebase-comprehension` skill | Static-analysis pipeline that turns an existing, undocumented codebase into a queryable structural model — AST extraction and parsing, call and dependency graphs, Spring Boot stereotype modeling, generated-code handling, incremental rebuilds with stable identity. Added 2026-05-24; SKILL.md router plus five Tier-2 references. |
| `codebase-extraction-engineering` skill | Implements the static-analysis extractor — Java parser selection and symbol resolution, the AST-to-graph pipeline, call-graph resolution, build integration for generated code, incremental rebuilds. The build-side companion to `codebase-comprehension`. Added 2026-05-24; SKILL.md router plus five Tier-2 references. |
| `python-service-engineering` skill | Builds production Python services — project structure and packaging, type and Pydantic discipline, async, the Pydantic AI / LangGraph orchestration code, runtime concerns (config, secrets, telemetry). The build-side companion to `ai-application-architecture`. Added 2026-05-24; SKILL.md router plus five Tier-2 references. |
| `data-access-engineering` skill | Implements the data-access layer — Cypher and Gremlin traversal queries including blast-radius walks, the idempotent graph-builder write path, relational expand/contract schema migrations, query discipline, and a repository-style access boundary. The build-side companion to `azure-data-tier-design`. Added 2026-05-24; SKILL.md router plus five Tier-2 references. |
| `test-engineering` skill | Designs and writes the test suite across the stack — unit and table-driven Go tests, pytest, integration tests against real dependencies via Testcontainers, characterization tests that pin legacy behavior before a change, test doubles and fixtures, and test-suite health (the pyramid, flakiness, CI gating). The deterministic-test counterpart to `ai-evaluation-harness`. Added 2026-05-24; SKILL.md router plus five Tier-2 references. |
| `frontend-engineering` skill | Builds the React and Next.js frontend — React component design and state, Next.js App Router architecture and rendering strategy, the Backend-for-Frontend route tier, streaming an LLM token response from the BFF to the browser, and TypeScript discipline at the API boundary. The frontend companion to `ai-application-architecture`'s serving topology. Added 2026-05-24; SKILL.md router plus five Tier-2 references. |
| `validation/tools/skill_audit.py` | Stdlib-only static-analysis tool: lints skill structure, checks living docs for count drift against disk, and regenerates `.claude/INDEX.md` with `--emit-index`. Adapted from the wshobson/agents PluginEval pattern (see `validation/snapshots/inspiration_hobson.md`). |
| `.claude/INDEX.md` | Generated flat discovery index — every skill, pattern card, agent, hook with one-line scopes. |
| `inherit` model tier | `aara-mcp-server-builder` switched from `opus` to `inherit` so the user picks the model per session. |

These additions are why the pack reached **26 skills** (29 with the three network skills added 2026-06-03), not the 15 described in pre-2026-05-21 documentation.

## Open work

The pack is feature-complete for its declared scope, but the following items are genuinely outstanding:

1. **Run the 12 capability prompts.** They have never been executed end-to-end. Until they are, the 32 skills are statically linted but not behaviorally tested. Record results in each prompt's `last_run` / `last_result` frontmatter.
2. **Reconcile the pack's home path. — Resolved 2026-05-22.** The canonical copy is now the single pack folder at `C:\aaraminds\skills-pack`. `usage.md`, `how-to-use-in-vscode.md`, `copilot/README.md`, `copilot/install.sh`, `copilot/mcp.json`, and the three `copilot/agents/*.agent.md` files were converted to pack-root-relative paths (no document hard-codes an absolute home). See `.claude/FEEDBACK.md`.
3. **Fix the `Container Apps` false positive** in the example server's `review_microservice_design` (the pattern matcher flags an Azure-native service as non-Azure-native). Tracked in the workspace master ranking [`../Ranking.md`](../Ranking.md) (Notes & caveats).
4. **Update `VERIFICATION_CHECKLIST.md`. — Resolved 2026-05-22.** Its prose and verification commands state 26 skills throughout.
5. **Decide the platform story.** The pack is formatted for Claude Code but is run under Copilot, where auto-routing, hooks, and progressive disclosure do not apply. Either commit to the Copilot framing or stop describing Copilot-inert machinery as live features.
6. **Keep the docs honest automatically. — Resolved 2026-05-22.** `skill_audit.py` now runs a doc-consistency pass that fails when a living document's skill/agent/hook/reference counts disagree with disk; dated point-in-time records live under `validation/snapshots/` and are exempt. This replaces the manual reconciliation that earlier revisions needed.

## Forward path

There is no fixed release calendar — this is a personal, single-maintainer pack. Forward work runs on the freshness cadence in `validation/governance/freshness-cadence.md`:

- **Quarterly** — re-verify ecosystem facts (Go / MCP SDK / Azure / Spring Boot versions); refresh dated claims in `mcp-go-server-building/references/ecosystem-facts.md`. Run `skill_audit.py` and regenerate `.claude/INDEX.md`.
- **Semi-annually** — refresh capability-prompt reference outputs if pattern cards have evolved.
- **Annually** — review the Tier-1 skill list. Should anything be split, merged, or retired? Are skill triggers still firing on the right requests?
- **Per change** — bump `version` per semver, update `last_updated`, regenerate the index.

### Future scope (optional)

| Direction | What it would add | Effort |
|---|---|---|
| Multi-language SDK coverage | Python and TypeScript MCP-server skills and example servers | Substantial |
| Domain repos | Additional example servers for security review, FinOps, infrastructure review | One repo per domain |
| Conformance test suite | Validate the example server against the MCP `conformance` spec repo | One-off |
| Capability-prompt automation | An LLM-driven runner that scores the 12 prompts against their rubrics | Moderate |

---

## Prior release

**v9.0 — Quality knowledge pack (flat-file format)**

v9.0 was the consolidated, standalone knowledge pack before the Claude Skills format migration: 12 microservices-design skills + 22 MCP-server-building skills + 21 pattern cards as flat files, plus the example server, demo, validation prompts, and 34 per-skill evals. v10.0 migrated this content; the Tier-2 bodies are largely unchanged from v9.0.

Note on quality claims: earlier roadmap and README revisions described v9.0 as "8/10 in independent review" and "9.0+ quality." Those figures carried no baseline, rubric, or named reviewer and have been removed. What can be stated honestly is in the README's "Quality position" section: the MCP server is tested and deterministic; the skills are statically linted but not yet behaviorally tested.
