# AaraMinds Ranking Personas, Agents & Skills

_Consolidated 2026-05-24. This is the **master copy** — the single canonical ranking for the AaraMinds workspace. The two prior ranking files were folded into it and then removed:_

- `skills-pack/ranking.md` — engineering skills, agents, hooks, MCP tools (v8, 2026-05-24) — **consolidated here and deleted**
- `instruction-os/Persona/Rankings.md` — personas and system modules (2026-05-21) — **consolidated here and deleted**

`governance/archive/AaraMind_Factory.md` — the earlier combined inventory (2026-05-22) — is retained as a dated point-in-time snapshot and carries a banner pointing here. For dated audit history, see `instruction-os/Persona/Validation_History.md`.

---

## Rating legend

The workspace rates two different kinds of artifact, and they use two different rubrics. Both are 1–10.

**Engineering rubric** — skills, agents, hooks, MCP tools (from `skills-pack/ranking.md` v8):

| Column | Measures |
|---|---|
| `claude` | Overall artifact quality, rated by Claude as an independent reviewer (did not author the artifact). |
| `codex` | The same quality rating by ChatGPT Codex as a second independent model. Filled by the 2026-05-25 Codex paper-plus-sample review pass; not a substitute for live behavioral validation where `strength` is `n/t`. |
| `depth` | Content substance — SKILL.md length, reference file count, total reference lines. |
| `strength` | Does it actually do its job — **tested only where invocable** (MCP tools, hooks). `n/t` for skills and agents: they need pack registration in a live Claude Code session to test for real. |
| `diff` | Differentiation vs. comparable artifacts elsewhere. |

**Persona rubric** — personas, system modules, communication skills (from `instruction-os/Persona/Rankings.md`):

| Column | Measures |
|---|---|
| `Claude` | Holistic 1–10: purpose clarity, output-contract strength, anti-pattern specificity, composition cleanliness, rot resistance, behavior under pressure, validation evidence. |
| `Codex` | Last rating from a Codex audit pass. Filled for current active personas and modules as of 2026-05-25. |
| `Status` | Stable / Validated / Draft / Needs work. |

Scale: 10 = best-in-class; 7-8 = strong; 5-6 = works / moderate; 1-4 = weak. **Persona-side cap: 9.3** — paper validation, however thorough, caps here; 9.5+ needs production evidence with team feedback. `n/t` = not tested; `unrated` = on disk but added after the last ranking pass.

---

## Summary

| Category | Count | Location | Avg quality (`claude`) | Notes |
|---|---:|---|---:|---|
| Engineering skills | 35 | `skills-pack/.claude/skills/` | 8.3 (26 non-network rated) | +3 unrated skills added 2026-06-15 (`azure-network-topology-visualization`, `azure-iac-policy-as-code`, `azure-defender-signal-ingestion`); +1 each 2026-06-16 (`prompt-engineering`), 2026-06-18 (`agent-engineering`), 2026-06-18 (`copilot-cost-optimization`); 3 earlier network skills separately rated below |
| Communication skills | 6 | `instruction-os/skills/` | 9.1 (3 Claude-rated) | Codex avg 7.9 (5 rated); +1 unrated skill added 2026-06-17 (`aaraminds-leadership-status-deck`, composes the Executive Narrative Advisor + pptx) |
| Personas | 6 | `instruction-os/Persona/` | 9.17 (6 rated) | Codex avg 9.0 across all 6; `Project_Planner` validated 2026-05-30 (6/6, independent subagent run), Codex-rated 2026-05-31 |
| System modules | 9 | `instruction-os/Persona/` | 9.04 (8 rated) | Codex avg 9.0 across all 9; Module 09 still Claude-unrated |
| Agents | 17 | `skills-pack/.claude/agents/` | 8.5 (4 rated) | +8 unrated agents added 2026-06-15 (`aara-topology-visualizer` + the 7-agent project-delivery lifecycle); +1 each 2026-06-16 (`aara-prompt-engineer`), 2026-06-17 (`aara-status-deck`), 2026-06-18 (`aara-agent-engineer`); +2 2026-06-18 built *and run-tested* through the agent-engineering factory: `aara-business-analyst` (design 89, 6/6, prod-candidate PASS) and `aara-copilot-cost-reviewer` (design 85, 6/6, pilot-PASS); Claude Code agents otherwise `n/t` until run in a live session |
| Hooks | 3 | `skills-pack/.claude/hooks/` | 6.7 | Hook templates now fail closed with python3 parsing; not active in workspace `.claude/settings.json` until merged |
| MCP server tools | 13 | `skills-pack/examples/.../internal/tools/` | 8.2 | Strongest behavioral evidence; fresh build/test/demo verified by Codex on 2026-06-04 |

**Total rated artifacts: 61 Claude-side, 65 Codex-side.** Strongest classes: MCP tools (fresh deterministic execution evidence), personas/modules (roughly 9.0, paper-capped at 9.3), and the deepest skills (`azure-data-tier-design`, `mcp-go-guardrails-and-safety`). Weakest areas: release hygiene around docs/adapters, inactive hook wiring, committed build/demo output, and the three thin network skills still carrying audit warnings.

**Verification snapshot — Codex pass, 2026-06-04.** `skill_audit.py` reports 0 FAIL / 7 WARN / 0 doc-consistency failures across 29 skills. The Go MCP server passed `go test ./...`, `go test -race -count=1 ./...`, `go vet ./...`, fresh build to `/tmp`, fresh demo generation to `/tmp`, and validation against committed goldens. `gofmt -l .` reports one formatting-only issue: `internal/services/design/service_test.go`.

---

## Engineering skills — `skills-pack/.claude/skills/` (35)

Native Claude Skills format (Tier-1 `SKILL.md` router + Tier-2 `references/`). Grouped by the pack's capability areas; each area closes with an average. `codex` reflects the 2026-05-25 Codex paper-plus-sample review for the 26 non-network skills. The 12 capability prompts were self-run on 2026-05-30 and passed 12/12, but because they were self-produced/self-graded, `strength` stays `n/t` for the non-network skills until an independent registered-session run confirms them.

### Microservices design (6)

| Skill | Description | claude | codex | depth | strength | diff |
|---|---|---:|---:|---:|---:|---:|
| `microservices-api-design` | REST / gRPC API contracts | 7 | 7.2 | 6 | n/t | 4 |
| `microservices-architecture-design` | End-to-end microservices design | 9 | 8.7 | 7 | n/t | 6 |
| `microservices-architecture-reviewer` | Architecture verdict review report | 9 | 8.8 | 7 | n/t | 8 |
| `microservices-async-messaging` | Sync vs async + broker choice | 8 | 8.0 | 7 | n/t | 6 |
| `microservices-data-architecture` | Saga, outbox, CQRS patterns | 8 | 8.2 | 8 | n/t | 6 |
| `microservices-resilience` | Resilience + rollout patterns | 8 | 7.8 | 7 | n/t | 4 |
| **Area average** | — | **8.2** | **8.1** | **7.0** | **n/t** | **5.7** |

### Azure platform (5)

| Skill | Description | claude | codex | depth | strength | diff |
|---|---|---:|---:|---:|---:|---:|
| `azure-data-tier-design` | Operational data tier design | 9 | 9.0 | 10 | n/t | 8 |
| `azure-microservices-cost-review` | Azure FinOps cost optimization | 8 | 8.0 | 8 | n/t | 7 |
| `azure-microservices-observability` | OpenTelemetry + Grafana observability | 9 | 8.7 | 8 | n/t | 7 |
| `azure-microservices-security` | Defense-in-depth Entra ID security | 6 | 7.4 | 6 | n/t | 6 |
| `azure-service-mapping` | Pattern -> Azure service mapping | 7 | 7.2 | 7 | n/t | 4 |
| **Area average** | — | **7.8** | **8.1** | **7.8** | **n/t** | **6.4** |

### MCP server building (4)

| Skill | Description | claude | codex | depth | strength | diff |
|---|---|---:|---:|---:|---:|---:|
| `mcp-go-guardrails-and-safety` | MCP runtime + CI guardrails | 9 | 8.8 | 9 | n/t | 9 |
| `mcp-go-production-review` | Go MCP pre-production review | 8 | 8.2 | 8 | n/t | 7 |
| `mcp-go-server-building` | Build Go MCP servers | 9 | 8.7 | 9 | n/t | 5 |
| `mcp-go-threat-modeling` | STRIDE threat modeling for MCP | 8 | 8.2 | 6 | n/t | 9 |
| **Area average** | — | **8.5** | **8.5** | **8.0** | **n/t** | **7.5** |

### Engineering workflow + compliance (4)

| Skill | Description | claude | codex | depth | strength | diff |
|---|---|---:|---:|---:|---:|---:|
| `new-azure-service-bootstrap` | Scaffold a new Azure service | 8 | 8.0 | 8 | n/t | 7 |
| `pr-review-azure-microservices` | PR review checklist for services | 9 | 8.7 | 8 | n/t | 6 |
| `soc2-iso27001-controls-mapping` | SOC 2 / ISO 27001 Azure mapping | 8 | 8.2 | 7 | n/t | 9 |
| `azure-iac-policy-as-code` | Gate Terraform on policy via adopted Checkov + OPA/Conftest, alongside the reachability gate (added 2026-06-15) | unrated | unrated | n/t | n/t | n/t |

### AI application design (2)

| Skill | Description | claude | codex | depth | strength | diff |
|---|---|---:|---:|---:|---:|---:|
| `ai-application-architecture` | AI/LLM application architecture on Azure | 9 | 8.8 | 8 | n/t | 8 |
| `ai-evaluation-harness` | Evaluation harness for AI/LLM features | 9 | 8.7 | 6 | n/t | 8 |
| **Area average** | — | **9.0** | **8.8** | **7.0** | **n/t** | **8.0** |

### Network analysis (5)

| Skill | Description | claude | codex | depth | strength | diff |
|---|---|---:|---:|---:|---:|---:|
| `azure-network-topology-analysis` | Reachability-based Azure network topology risk review (graph + NSG/route/AVNM/DNAT + severity) | 8 | unrated | 7 | 7 | 8 |
| `azure-network-cost-forecasting` | Design-time network cost forecast (fixed exact / variable band; Retail Prices API) | 7 | unrated | 6 | 6 | 7 |
| `azure-network-iac-generation` | Generate validated Terraform from intent (vetted CAF/ALZ modules; analyzer-gated, PR-only) | 7 | unrated | 6 | n/t | 7 |
| `azure-network-topology-visualization` | Enterprise risk-annotated topology diagrams: adopt CloudNetDraw/ELK; paint `Analyze()` severity (added 2026-06-15) | unrated | unrated | n/t | n/t | n/t |
| `azure-defender-signal-ingestion` | Consume Defender for Cloud exposure/attack-path signals via ARG; reconcile with the engine; fall back where unlicensed (added 2026-06-15) | unrated | unrated | n/t | n/t | n/t |
| **Area average** | — | **7.3** (3 rated) | — | **6.3** | — | **7.3** |

`azure-network-topology-analysis` (**v1.1.0**) is the only one with a real eval (rare in this pack): five rounds across six fixtures, on a frontier model *and* a verified-pinned Haiku, scored 8/8 recall and every precision trap — but **at parity with the unaided baseline on every run**. The settled finding: capability lives in the deterministic engine (now built and golden-tested), not the prose; the skills are the consistency/spec layer. `strength` 7 = proven-correct-but-not-superior. `azure-network-cost-forecasting` (v0.1.0) has one eval with a known inter-VNet-peering gap; `azure-network-iac-generation` (v0.1.0) is unevaluated (`strength` n/t). All three landed in `skills-pack/.claude/skills/` 2026-06-03; the staged copies and apply scripts have since been swept, leaving only the eval workspace at `skill-staging/eval/`. Evidence: `skill-staging/eval/benchmark.md`; engine: `aaraminds-projects/azure-network-topology-reviewer/engine/`.

---

### Code comprehension (1)

| Skill | Description | claude | codex | depth | strength | diff |
|---|---|---:|---:|---:|---:|---:|
| `codebase-comprehension` | Static-analysis codebase comprehension | 9 | 8.7 | 6 | n/t | 8 |
| **Area average** | — | **9.0** | **8.7** | **6.0** | **n/t** | **8.0** |

### Implementation engineering (5)

| Skill | Description | claude | codex | depth | strength | diff |
|---|---|---:|---:|---:|---:|---:|
| `codebase-extraction-engineering` | Implementing the static-analysis extractor | 9 | 8.8 | 6 | n/t | 8 |
| `data-access-engineering` | Queries, migrations, the data-access layer | 9 | 8.4 | 6 | n/t | 7 |
| `frontend-engineering` | React / Next.js frontend + BFF tier | 8 | 7.8 | 6 | n/t | 6 |
| `python-service-engineering` | Building production Python services | 8 | 7.8 | 6 | n/t | 5 |
| `test-engineering` | Cross-stack test suite design | 8 | 8.0 | 6 | n/t | 5 |
| **Area average** | — | **8.4** | **8.2** | **6.0** | **n/t** | **6.2** |

The five implementation skills were rated by an independent reviewer pass on 2026-05-25 (the reviewer did not author them). They are the pack's build-side half — companions to the design skills (`codebase-comprehension`, `ai-application-architecture`, `azure-data-tier-design`). Their common ceiling is depth: each ships exactly 5 short references, roughly a quarter of the deepest skills in the pack.

**Non-network skills aggregate (26 rated):** avg `claude` 8.3 · avg `codex` 8.3 · avg `depth` 7.2 · avg `diff` 6.7 · `strength` n/t. The three network skills are on disk and separately rated above; audit warnings still flag them as thinner than the pack norm.

### Meta-engineering (3)

Cross-cutting skills (not tied to the Azure-microservices domain).

| Skill | Description | claude | codex | depth | strength | diff |
|---|---|---:|---:|---:|---:|---:|
| `prompt-engineering` | Generate / optimize / teach prompts for AI coding assistants; per-platform tracks for Claude, GitHub Copilot, OpenAI Codex (added 2026-06-16) | unrated | unrated | n/t | n/t | n/t |
| `agent-engineering` | AI Agent Designer & Evaluator: create / review (100-pt rubric) / evaluate enterprise AI agents; design-vs-behavior firewall; staged release gate + templates (v2.4, 2026-06-18) | unrated | unrated | n/t | n/t | n/t |
| `copilot-cost-optimization` | FinOps for GitHub Copilot: AI-Credit cost model, **3 data surfaces** (Metrics API / billing / local `copilot-token-budget` MCP telemetry), optimization levers incl. instruction-overhead, recommendation framework (v1.1, 2026-06-18) | unrated | unrated | n/t | n/t | n/t |

`prompt-engineering`: a router over five references (cross-platform core + one per platform + generate/optimize/teach workflows). Built 2026-06-16 from current official docs (Anthropic best-practices + Opus 4.x; GitHub Copilot + VS Code; OpenAI Codex + GPT-5 + agents.md). Paired with `aara-prompt-engineer`.

`agent-engineering`: the governance layer for the agent fleet — three modes (create / review / evaluate) over five references + an `eval/` (dogfood review of `aara-status-deck` + scenarios). Owns the agent-package contract (AGENT_SPEC.md + A2A agent-card.json + runnable file), the 100-point review rubric with hard gates, and the **design-vs-behavior firewall** (no agent is production-ready on a paper score). Composes the blueprint advisor, `ai-application-architecture`, `prompt-engineering`, `ai-evaluation-harness`, and the security skills rather than duplicating them. Built 2026-06-18 from current primary sources (OpenAI *Practical Guide to Building Agents*, Anthropic *Building Effective Agents* + *Demystifying Evals*, OWASP Agentic Top 10 / MAESTRO, A2A Agent Cards, model-card→system-card lineage, NIST AI RMF). **v2.0 (2026-06-18)** merged in the best of an external "Agent Engineer Pack": a `templates/` folder of six copy-ready fill-ins (runnable-agent, agent-spec, review-scorecard, **staged release gate**, eval-plan + golden dataset, package-index), a rebalanced 11-category rubric (Evaluation weighted 12; Scope its own dimension; I/O contracts split), a Blocker/Major/Minor/Observation severity model with `F-001` findings + P0–P3 backlog, and a prescriptive output format — kept ours' OWASP/MAESTRO/trifecta/A2A depth, per-platform scaffolding, audit-compliance, and the dogfood (re-scored 77/100 under v2; release gate: CONDITIONAL PASS pilot / FAIL production). **v2.1 (2026-06-18)** added, from an external review: 4 machine-readable JSON schemas (`eval-case`/`eval-result`/`trace-review`/`release-gate`), 3 templates (tool-risk register, efficiency scorecard, trace-review), a skill README, explicit delegation to `aara-ai-evaluation-engineer` (no duplicate evaluator), and a self-contained distributable pack (skill + bundled agent + README/MANIFEST/INSTALL). **v2.2 (2026-06-18)** addressed an external P0/P1 review: removed `Bash` from the orchestrator's default tools (self-consistency — it had flagged the same on `aara-status-deck`) with `permissionMode: ask` + `maxTurns`; added `agent-card` + `AGENTS.md` templates; added **conditional firewall enforcement** to the release-gate and eval-result schemas (a production-candidate PASS without executed eval results now fails validation); added runnable `scripts/` (validate-schemas, score-release-gate, check-package-completeness, check-dependencies — all tested, 3.8-portable) + a sample CI workflow; added `references/source-index.md`; and shipped a refreshed distributable pack. **v2.3 (2026-06-18)** from a follow-up review: clarified release-gate semantics (Option A — no PASS at production-candidate without executed evals; CONDITIONAL_PASS allowed); fixed CI workflow paths; renamed `score-`→`check-release-gate.py` and added per-stage evidence checks; tightened `check-package-completeness.py` (excludes the pack's own dirs — no template false positives); added a full worked **example package** (`leadership-status-agent-package/` — the first dogfood, scored 77/100, gate CONDITIONAL_PASS pilot), 4 release-gate example instances, `evaluator-handoff-contract.md`, and a `run-evals.py` adapter (honestly exits 3 until a harness is wired). All scripts tested. **v2.4 (2026-06-18)** refined the staged gate to separate a production *candidate* (proven behavior + monitoring plan + tested rollback runbook) from the *production* stage (live monitoring + canary + rollback exercised) — schema + checker + template updated, validated against the BA-agent run. Paired with `aara-agent-engineer`. Router 126 lines, 0 `skill_audit` findings.

Both carry 0 `skill_audit.py` findings on themselves; `unrated` / `n/t` until a ranking-rubric pass and a live registered-session run.

---

## Communication skills — `instruction-os/skills/` (6)

Persona-derived skills. The first three are thin composition wrappers with no `references/` of their own — they score well on the persona rubric but lower on independent review because each hard-depends on the `instruction-os/` persona files. The two added 2026-06-04 are **self-contained**: their operative gates, output contracts, and anti-patterns are inlined so they degrade gracefully without the persona files, with references for full depth.

| Skill | Description | Claude | Codex | Status |
|---|---|---:|---:|---|
| `aaraminds-ai-engineering-architect` | Architecture design and review | 9.3 | 7.0 | Stable |
| `aaraminds-content-strategist` | Thought leadership, LinkedIn, newsletters | 9.0 | 7.5 | Validated |
| `aaraminds-project-planner` | Delivery planning for AI / software projects | 9.0 | 7.8 | Validated |
| `aaraminds-executive-narrative-advisor` | AVP/VP narratives, decision memos, escalation briefs (self-contained core) | unrated | 8.6 | Draft+ (Codex paper-rated 2026-06-04) |
| `aaraminds-ai-agent-blueprint-advisor` | Use-case → buildable enterprise agent blueprint (self-contained core) | unrated | 8.5 | Draft+ (Codex paper-rated 2026-06-04) |

Codex read the two self-contained communication skills on 2026-06-04. Both are materially stronger than the original wiring-only wrappers because they degrade gracefully without loading the full persona files: the Executive Narrative Advisor carries the audience/metric/risk/ask gates inline; the Agent Blueprint Advisor carries boundary-first sequencing, architecture-theatre checks, diagram-completion checks, and the Module 2/5/7 handoff contract. They stay Claude-unrated until a matching persona-rubric pass is run.

**`aaraminds-leadership-status-deck` (added 2026-06-17, v1.3, unrated).** A production-layer skill for the recurring **monthly** leadership status deck (`.pptx`), VP-optimized (usable down to delivery manager). Deliberately thin: it composes `aaraminds-executive-narrative-advisor` for all narrative judgment (signal-over-activity, metric integrity, risk honesty) rather than duplicating it, and owns the repeatable scaffolding. **v1.1 (2026-06-17)** rebuilt the template to the VP-optimized flow (cover · executive summary · dimensional program-health dashboard · top-5 accomplishments · top-3–5 risks · leadership decisions · next-month outlook + confidence · appendix) and added deterministic month-over-month trend rules (incl. rename/merge/split → `[VERIFY]`), a deliverables contract (pptx · exec one-pager · evidence report · verification report · MoM change summary · Q&A), a 60-second five-question success test, a mandatory render-to-image visual-QA pass + overflow guard, a dependency-fallback core, and 3 eval scenarios — incorporating a real-deck review (AT&T STFO status deck) and an open-source benchmark (Anthropic `pptx`/`internal-comms`, `frontend-slides`, Pyramid/BLUF/RAG). **v1.2 (2026-06-17)** added, from a second external review: a tightened triggering description, a pre-flight dependency-readiness check, default PMO-overridable RAG thresholds + threshold key, scored High/Medium/Low confidence criteria, a business-impact translation layer, a 6-dimension eval scoring rubric, and a confidentiality section. **v1.3 (2026-06-17)** added three opt-in modes that keep the single-program core thin — portfolio roll-up (`portfolio-rollup.md`), historical intelligence via a carried status ledger (`historical-intelligence.md`), and role-based audience profiles (`audience-profiles.md`) — plus an "executive narrative strength" eval dimension. Router + 7 reference files. Paired with the `aara-status-deck` agent. Passes `skill_audit.py` with 0 findings on itself; `unrated` / `n/t` until a persona-rubric pass and a live run.

---

## Personas — `instruction-os/Persona/` (6)

Role-based context blocks, composed with system modules per each persona's `## Composition` section.

| Persona | Description | Claude | Codex | Status |
|---|---|---:|---:|---|
| `AaraMinds_AI_Engineering_Architect_v1.2` | Full-lifecycle architect for agent and non-agent AI systems | 9.3 | 9.1 | Stable |
| `AaraMinds_AI_Business_Strategist_v1.1` | Peer strategist for the AI founder - ideas, plans, decisions | 9.3 | 9.0 | Stable |
| `AaraMinds_AI_Agent_Blueprint_Advisor_v1.1` | Single-agent blueprint design and review | 9.2 | 9.0 | Stable |
| `AaraMinds_Content_Strategist_v1.0` | Public thought-leadership content | 9.0 | 8.8 | Validated |
| `AaraMinds_Executive_Narrative_Advisor_v1.0` | Exec updates, leadership decks, decision briefs | 9.0 | 8.7 | Validated |
| `AaraMinds_Project_Planner_v1.0` | Delivery planning for AI / software engineering projects | 9.2 | 9.1 | Stable |

Codex ratings are filled for all six active role personas. `Project_Planner` is now Claude-rated 9.2 / Stable after an independent subagent stress-test run on 2026-05-30 (6/6 prompts pass, plus a v1.1 capability run of 5/5 the same day — resource-and-cost, executive-reporting handoff, agentic-delivery, dependency intelligence, with a no-regression check; see `Testing/StressTest_Project_Planner_Results_2026-05-30.md`) and Codex-rated 9.1 on 2026-05-31 after a paper-plus-evidence review. It is held below the 9.3 paper cap because grading was same-model, not cross-model, and the results file summarizes rather than preserves the full generated responses. Content Strategist and ENA remain below the 9.3 paper cap because their prior stress tests were self-graded.

---

## System modules — `instruction-os/Persona/` (9)

Base and optional modules that personas compose. Module 01 is the canonical foundation; the rest refine it.

| Module | Description | Claude | Codex | Status |
|---|---|---:|---:|---|
| `04_Framework_Creation_System_v1.1` | Leadership frameworks, decision lenses, maturity models | 9.3 | 9.2 | Stable |
| `05_AI_Systems_Review_System_v1.2` | Structural review of AI systems; findings-led | 9.2 | 9.1 | Stable |
| `08_AI_Agent_Blueprint_System_v1.1` | Use case -> buildable AI agent blueprint | 9.2 | 9.2 | Stable |
| `07_AI_Engineering_Trend_Scan_System_v1.1` | Recency-grounded trend scans with sources | 9.1 | 9.1 | Stable |
| `03_Newsletter_Editorial_System_v1.1` | Long-form newsletters and articles | 9.0 | 9.0 | Stable |
| `01_Layered_Base_System_v1.1` | Canonical foundation: identity, voice, reasoning, gates | 8.9 | 9.0 | Stable |
| `02_Visual_Identity_System_v1.1` | Visual identity, diagrams, infographics, architecture posters | 8.9 | 8.9 | Stable |
| `06_LinkedIn_Post_System_v1.1` | Short / medium-form LinkedIn posts | 8.9 | 8.9 | Stable |
| `09_Project_Delivery_Planning_System_v1.0` | Project delivery planning - scoping, estimation, sequencing, risk | unrated | 8.8 | Draft (new 2026-05-24) |

---

## Agents — `skills-pack/.claude/agents/` (17)

Multi-skill orchestration personas (Claude subagent format: `name`, `description`, `model`, restricted `tools`). The original 4 domain agents, plus 8 added 2026-06-15: `aara-topology-visualizer` (Phase-4 diagram orchestration) and a 7-agent project-delivery lifecycle that the antr playbooks reference. The Copilot adapter has its own broader set under `skills-pack/copilot/agents/`, not a 1:1 mirror of this table.

| Agent | Description | Model | claude | codex | depth | strength | diff |
|---|---|---|---:|---:|---:|---:|---:|
| `aara-senior-microservices-architect` | End-to-end microservices architect | opus | 9 | 8.5 | 8 | n/t | 7 |
| `aara-mcp-server-builder` | Go MCP server builder | inherit | 9 | 8.6 | 7 | n/t | 7 |
| `aara-azure-cost-reviewer` | Azure FinOps cost reviewer | sonnet | 8 | 8.0 | 8 | n/t | 6 |
| `aara-network-topology-reviewer` | Reachability-based network topology reviewer; orchestrates the network skills (now incl. policy-as-code + Defender ingestion) + engine MCP tools | inherit | 8 | unrated | 7 | n/t | 8 |
| `aara-topology-visualizer` | Produces the risk-annotated topology *diagram*; consumes the analyzer for severity (added 2026-06-15) | inherit | unrated | unrated | n/t | n/t | n/t |
| `aara-project-architect` | System design, decomposition, ADRs, brownfield evolution (added 2026-06-15) | inherit | unrated | unrated | n/t | n/t | n/t |
| `aara-project-planner` | Outcome-defined phases, T-shirt estimates, critical path, risk register (added 2026-06-15) | inherit | unrated | unrated | n/t | n/t | n/t |
| `aara-project-builder` | Execute a playbook step/ticket: code + tests + green gate + Result log (added 2026-06-15) | inherit | unrated | unrated | n/t | n/t | n/t |
| `aara-project-reviewer` | Adversarial acceptance review → acceptance memo, gates cited to file:line (added 2026-06-15) | inherit | unrated | unrated | n/t | n/t | n/t |
| `aara-project-debugger` | Reproduce → root-cause → minimal fix + regression test (added 2026-06-15) | inherit | unrated | unrated | n/t | n/t | n/t |
| `aara-python-ai-developer` | Python/LLM-orchestration (explainer, generator intent, reference engines, viz pipeline) (added 2026-06-15) | inherit | unrated | unrated | n/t | n/t | n/t |
| `aara-ai-evaluation-engineer` | Build/run eval gates (precision/recall, diagram-eval, twin-drift, triggering) (added 2026-06-15) | inherit | unrated | unrated | n/t | n/t | n/t |
| `aara-prompt-engineer` | Generate/optimize/teach prompts for AI coding assistants across Claude, Copilot, Codex; routes to `prompt-engineering` (added 2026-06-16) | inherit | unrated | unrated | n/t | n/t | n/t |
| `aara-status-deck` | Produce the recurring monthly leadership status deck (.pptx), manager-through-VP; composes Executive Narrative Advisor + `aaraminds-leadership-status-deck` + pptx (added 2026-06-17) | inherit | unrated | unrated | n/t | n/t | n/t |
| `aara-agent-engineer` | AI Agent Designer & Evaluator — create/review/evaluate/harden enterprise AI agents; routes `agent-engineering`, delegates to prompt-engineer + ai-evaluation-engineer (added 2026-06-18) | inherit | unrated | unrated | n/t | n/t | n/t |
| `aara-business-analyst` | Trace-first BA: stakeholder inputs → traceable requirements/stories/AC; human-gated; hands off to architect/planner. First agent built, run-tested, AND taken to a **production-candidate PASS** through the agent-engineering factory (design 90/100; **6/6 golden cases, pass^3=1.0** incl. injection refusal; tested rollback + monitoring plan + MCP-adapter contracts; only the live deploy remains); pkg in `agent-packages/aara-business-analyst/` (added 2026-06-18) | inherit | unrated | **strength: pass^3=1.0** | n/t | n/t | n/t |
| `aara-copilot-cost-reviewer` | FinOps reviewer for enterprise GitHub Copilot spend (AI-Credit model); usage+billing → ranked sourced cost-optimization verdict; human-gated. Built + run-tested via the factory (design 86/100; **9/9 cases** incl. fabrication/conflation/access/stale-rate refusals **and a live integration run against the org's `copilot-token-budget` MCP** — instruction-overhead lever computed, tool caveats carried; pilot-PASS, data source wired); routes to `copilot-cost-optimization`; pkg in `agent-packages/aara-copilot-cost-reviewer/` (added 2026-06-18) | inherit | unrated | **strength: 9/9 run** | n/t | n/t | n/t |

`strength` is `n/t`: agents are dispatched as Claude Code subagents and need the pack registered in a live session to test for real. The 8 agents added 2026-06-15 are authored + wired (`wire-skills.sh`) but unexercised — their real test is running an actual ticket in a live session. Workspace-root `.claude/skills/` and `.claude/agents/` were re-wired via `.claude/wire-skills.sh` on 2026-06-16 — **38 skills, 13 agents now linked** (up from 37/12), including the new `prompt-engineering` skill and `aara-prompt-engineer` agent. Wiring is still `n/t` for behavioral strength: the links make the artifacts discoverable, but a live registered session is needed to confirm invocation.

---

## Hooks — `skills-pack/.claude/hooks/` (3)

| Hook | Description | claude | codex | depth | strength | diff |
|---|---|---:|---:|---:|---:|---:|
| `block-dangerous-commands` | Block destructive Bash commands | 8 | 6.5 | 8 | 8 | 7 |
| `pre-commit-lint` | Lint Go / Java before commit | 6 | 5.8 | 7 | 7 | 5 |
| `test-before-commit` | Run tests before commit | 6 | 5.8 | 7 | 7 | 5 |

**Fix landed 2026-05-27 — fail-open bug retired.** All three hooks now parse `$CLAUDE_TOOL_INPUT` with `python3 -c` (no `jq` dependency) and **fail closed** when `python3` is missing, when input is unparseable, or when `.command` is empty. Verified against 12 cases including malformed input, empty `.command`, and a sandboxed PATH without `python3` — every failure mode exits `2` with a stderr diagnostic. Strength scores no longer depend on a side dependency.

**Activation caveat, verified 2026-06-04.** These are hook templates, not active project settings. Workspace `.claude/settings.json` has no `hooks` block, and instead carries a broad/stale permissions allowlist. Treat the hook ratings as artifact quality, not as current runtime protection for this checkout.

---

## MCP server tools (13)

Bundled Go server at `skills-pack/examples/microservices-system-design-mcp-server/`. These are the **only artifacts with real `strength` evidence** — every tool was invoked over stdio JSON-RPC against the binary with representative inputs.

| Tool | Description | claude | codex | depth | strength | diff |
|---|---|---:|---:|---:|---:|---:|
| `generate_architecture_decision_record` | ADR document generator | 9 | 8.8 | 8 | 9 | 9 |
| `generate_deployment_topology` | Deployment topology generator | 9 | 8.7 | 8 | 9 | 7 |
| `generate_event_contract` | Async event contract generator | 9 | 8.8 | 8 | 9 | 8 |
| `generate_observability_plan` | Observability plan generator | 9 | 8.8 | 8 | 9 | 8 |
| `generate_resilience_plan` | Resilience plan generator | 9 | 8.7 | 8 | 9 | 8 |
| `map_patterns_to_azure_services` | Map patterns to Azure services | 8 | 8.4 | 7 | 9 | 6 |
| `detect_architecture_risks` | Detect architecture risks | 8 | 8.3 | 8 | 8 | 8 |
| `generate_api_contract` | OpenAPI / proto contract | 8 | 8.1 | 7 | 8 | 7 |
| `generate_service_boundary_canvas` | DDD service boundary canvas | 8 | 8.4 | 8 | 8 | 9 |
| `recommend_microservice_patterns` | Recommend microservice patterns | 8 | 8.2 | 8 | 8 | 7 |
| `review_microservice_design` | Review a microservice design | 7 | 7.5 | 8 | 8 | 8 |
| `generate_diagram_assets` | Architecture diagram code | 7 | 7.2 | 6 | 7 | 6 |
| `score_well_architected_readiness` | Azure Well-Architected scoring | 7 | 7.3 | 8 | 7 | 7 |

**MCP tools aggregate:** avg `claude` 8.2 · avg `codex` 8.2 · avg `depth` 7.8 · avg `strength` **8.2** · avg `diff` 7.5.

**Fresh verification, 2026-06-04.** Codex rebuilt the server to `/tmp/aaraminds-mcp-server`, ran the demo against that fresh binary with output directed to `/tmp/aaraminds-demo-out`, and validated the fresh outputs against committed goldens. This is stronger than simply running `make validate`, because it proves the current Go source still reproduces the goldens.

---

## Notes & caveats

- **Strongest artifacts.** The MCP server and demo are the proof anchor. `azure-data-tier-design` (depth 10) and `mcp-go-guardrails-and-safety` (9/9/9) lead the skills. Module 04 (Framework Creation) and the AI Engineering Architect / AI Business Strategist personas top the Claude-side persona scores at 9.3; Codex keeps all paper-only persona/module scores at or below 9.2.
- **Weakest artifacts.** Release hygiene is the binding constraint: current docs still contain stale qualitative claims, the verification checklist has stale expected counts, the Copilot adapter has 10 custom agents while its README/install output says 4/3, and project settings carry stale broad permissions. The three network skills are useful but still thin by the pack's own audit rules.
- **Known open bug.** The example MCP server's `review_microservice_design` can still flag human-entered `Container Apps` as "non-Azure-native" because the rule accepts `container_apps` but not common variants like `Container Apps`, `ACA`, or `Azure Container Apps`. The fix is to normalize deployment-target aliases before scoring. The tool's `strength` score (8) already reflects the defect.
- **Code hygiene.** `go test`, `go test -race`, and `go vet` pass. `gofmt -l .` reports `internal/services/design/service_test.go`; the diff is formatting-only alignment in composite literals.
- **Repository hygiene.** Tracked generated/runtime artifacts remain: the Linux `mcp-server`, `mcp-server-darwin-arm64`, and demo `out/` JSON files. They work today, but source repos age better when binaries and transient demo outputs are rebuilt rather than committed.
- **Validation prompts.** The 12 prompts passed a 2026-05-30 self-run, recorded in `skills-pack/validation/skill-validation-self-run-2026-05-30.md`. Because the run was self-produced/self-graded and some answer keys were visible, it is indicative evidence only; `strength` stays `n/t` for non-network skills.
- **Codex ratings are filled for older classes.** They reflect a 2026-05-25 paper-plus-sample review plus this 2026-06-04 repo analysis, not full behavioral validation. Keep `strength` as the stronger signal wherever it exists.
- **Persona-side 9.3 cap.** No persona or module can exceed 9.3 on paper validation alone; 9.5+ requires production use with team feedback.
- **Pack location.** The canonical pack is `/home/raja/projects/aaraminds-platform/skills-pack`. Older `/home/raja/projects/aaraminds`, OneDrive, and `C:\aaraminds` paths in dated notes are historical.

---

## Maintenance

Regenerate this file whenever `skills-pack/.claude/{skills,agents,hooks}/`, the MCP `internal/tools/`, or `instruction-os/Persona/` changes. Ask Claude: "update `Ranking.md` from the current state of the pack and the persona system." Move dated history into `instruction-os/Persona/Validation_History.md`; keep this file a clean current snapshot.

_Master copy refreshed by Codex on 2026-06-04. `skills-pack/ranking.md` and `instruction-os/Persona/Rankings.md` were consolidated here and deleted; `AaraMind_Factory.md` is retained as a dated snapshot._

**Update — 2026-06-15 (housekeeping).** Added and wired 3 engineering skills (`azure-network-topology-visualization`, `azure-iac-policy-as-code`, `azure-defender-signal-ingestion`) and 8 agents (`aara-topology-visualizer` + the 7-agent project-delivery lifecycle); the `aara-network-topology-reviewer` agent now composes the policy + Defender skills. Counts: 32 skills, 12 agents. The new artifacts are **unrated** (no eval/Codex pass yet) — a ratings pass is the outstanding follow-up. Triggering evals for the 3 new skills live in each skill's `eval/` folder. Stray `iso.js` removed; `skill-staging/` reduced to the original eval workspace.

---

## Codex review pass — 2026-05-25

Historical baseline. Superseded for current-state caveats by the 2026-06-04 pass below, but retained because it explains the original Codex ratings.

Codex ran a critical review of the current skills, personas, agents, hooks, MCP tools, and governance system on 2026-05-25.

**Overall rating:** **8.4 / 10**

| Area | Codex rating | Notes |
|---|---:|---|
| Engineering skills | 8.3 | Strong structure and routing; most still need behavioral validation. |
| Personas | 9.0 | Strongest part of the system; clear composition, gates, anti-patterns, and audience discipline. |
| System modules | 9.1 | Mature, reusable, and pressure-aware; production evidence still caps the top end. |
| Agents | 8.2 | Well-written orchestration personas, but not yet live-tested as registered agents. |
| Communication skills | 7.4 | Useful entry points, but thin wrappers over persona files rather than self-contained skills. |
| Hooks | 6.2 | Lowest-scoring class on depth/differentiation; the jq fail-open flaw was retired 2026-05-27 (now python3, fails closed). Codex re-rate pending. |
| MCP server tools | 8.5 | Best behavioral evidence in the workspace; real stdio execution and golden-output validation. |
| Governance / ranking discipline | 8.8 | Honest status tracking, caveats, and validation history are major strengths. |

**Codex assessment:** The system is already useful for serious work. The next quality jump comes from proof, not more content: run the validation prompts, live-test agents, fix hooks to fail closed, shorten overloaded skill descriptions, and record outcomes.

**Priority improvements for tomorrow:**

1. ~~Fix hooks so missing `jq` fails closed or replace `jq` parsing with a safer built-in parser.~~ **Done 2026-05-27** — hooks now parse with `python3` and fail closed.
2. Shorten overloaded skill descriptions flagged in `skill-audit-2026-05-24.md`.
3. Run the 12 validation prompts end to end and record results.
4. Live-test the 3 agents in the target environment.
5. Re-rate `azure-microservices-security` after its 2026-05-25 reference cleanup.
6. Revisit per-artifact Codex ratings after the validation run and live agent tests; today's ratings are paper-plus-sample baselines.

---

## Codex analysis pass — 2026-06-04

Codex performed a read-only workspace analysis of `/home/raja/projects/aaraminds` and updated this ranking to separate artifact quality from current runtime wiring.

**Overall rating:** **8.5 / 10**

| Area | Codex rating | Current-state notes |
|---|---:|---|
| Engineering skills | 8.3 | 29 on disk; 26 non-network skills remain paper/sample rated, 3 network skills are separately evaluated but thin. `skill_audit.py` is clean on FAILs. |
| Communication skills | 7.9 | Original 3 wrappers remain useful but dependency-heavy; the 2 new self-contained wrappers are stronger and now Codex-rated. |
| Personas | 9.0 | Still one of the strongest parts: clear composition, gates, anti-patterns, and audience discipline. Production evidence still caps the top end. |
| System modules | 9.1 | Mature and reusable; Module 09 remains Claude-unrated. |
| Agents | 8.2 | Four Claude Code agents are well-scoped but not live-tested; root `.claude/agents/` was not wired during this pass. |
| Hooks | 6.8 | Hook templates now fail closed and no longer depend on `jq`; they are not active in workspace settings until merged. |
| MCP server tools | 8.6 | Fresh source-built binary reproduced demo goldens; `go test`, `go test -race`, and `go vet` pass. One formatting-only `gofmt` issue remains. |
| Governance / release hygiene | 8.0 | Static audit catches count drift, but misses stale qualitative claims. Copilot docs/scripts and verification checklist still drift from disk. |

**What changed from the 2026-05-25 pass:**

1. Skills count is now 29, with 3 network skills landed and the generated index current.
2. Communication skills count is now 5; `aaraminds-executive-narrative-advisor` and `aaraminds-ai-agent-blueprint-advisor` are self-contained and Codex-rated.
3. Hooks are safer as artifacts (python3 parsing, fail closed) but not active in this checkout's `.claude/settings.json`.
4. MCP server evidence is stronger: Codex verified a fresh build and fresh demo output against goldens, not just committed `out/` files.
5. The main risk moved from "content quality" to "release hygiene": stale current docs, broad project permissions, committed binaries/output, Copilot adapter drift, and missing CI.

**Priority improvements now:**

1. Fix stale current docs: `skills-pack/README.md`, `ROADMAP.md`, `VERIFICATION_CHECKLIST.md`, `copilot/README.md`, `copilot/install.sh`, and `.claude/wire-skills.ps1`.
2. Remove tracked build/demo output or explicitly reclassify it as release artifact: `mcp-server`, `mcp-server-darwin-arm64`, and `demo/architecture-review-demo/out/`.
3. Tighten `.claude/settings.json`: remove old `brs191` path allowances, broad shell patterns, and stale install permissions; merge hooks only if they should actually protect this workspace.
4. Extend `skill_audit.py` to catch expected-count tables and stale qualitative phrases (`not yet run`, `jq`, `3 agents`, `26 skills`, etc.), not just direct count claims.
5. Fix the one `gofmt` finding in `internal/services/design/service_test.go`.
6. Normalize deployment-target aliases in `review_microservice_design` so `Container Apps`, `ACA`, and `Azure Container Apps` do not false-positive.
7. Run an independent registered-session validation pass for the 12 prompts and all 4 Claude Code agents; only then move non-network skills and agents off `n/t`.
