# AaraMinds Factory — Skills, Personas & Agents Inventory

> **⚠ Superseded 2026-05-24.** The canonical ranking is now [`Ranking.md`](Ranking.md) — a single consolidated ranking built on this file's combined-inventory format, refreshed against the current pack (22 skills, ranking v8). This file is frozen as a 2026-05-22 point-in-time snapshot; use `Ranking.md` for current scores.

_Generated: 2026-05-21. Updated 2026-05-22 with an independent critical review, and again 2026-05-22 to record a staged upgrade of the two lowest-rated skills. Ratings sourced from `skills-pack/ranking.md`, `instruction-os/Persona/Rankings.md`, and a fresh peer review of every skill._

## Rating legend

- **9–10** — strong / best-in-class
- **7–8** — good
- **5–6** — moderate / works
- **n/t** — not tested this pass

`Claude fit` / `Depth` / `Strength` / `Diff.` are the original skills-pack rubric scores from `ranking.md`. `Strength` is `n/t` for skills and agents — they require pack registration in a live Claude Code session to test, which the ranking pass did not have. This is an honest gap, not a low score.

`Critical review` is a fresh, independent holistic 1–10 score (2026-05-22). Each skill's `SKILL.md` and reference files were read in full and judged on purpose clarity, depth of reference content, principal-engineer voice, actionability, anti-pattern specificity, robustness under real tasks, and differentiation. It is deliberately harsher than the original rubric and, where it diverges, reflects defects found by reading the actual content (see Critical review findings).

A score shown as `5 → 8*` means the live skill on disk currently rates 5, and an upgrade that would lift it to a projected 8 has been written but not yet applied. The live score is what is actually in `skills-pack/.claude/skills/` today. The `*` projection is self-assessed against the staged content — it is not yet an independent re-review and should not be treated as final until one is done.

Personas, system modules, and communication skills use the persona rubric (Claude score out of 10). All files are capped at 9.3 until production evidence with team feedback exists.

---

## Engineering skills — `skills-pack/.claude/skills/` (18)

Native Claude Skills format.

| Skill | Description | Critical review | Claude fit | Depth | Strength | Diff. |
|---|---|---:|---:|---:|---:|---:|
| `azure-data-tier-design` | Operational data tier design | **9** | 9 | 10 | n/t | 8 |
| `azure-microservices-cost-review` | Azure FinOps cost optimization | **7** | 8 | 8 | n/t | 7 |
| `azure-microservices-observability` | OpenTelemetry + Grafana observability | **8** | 8 | 8 | n/t | 7 |
| `azure-microservices-security` | Defense-in-depth Entra ID security | **5 → 8\*** | 9 | 6 | n/t | 6 |
| `azure-service-mapping` | Pattern → Azure service mapping | **5 → 8\*** | 7 | 7 | n/t | 4 |
| `mcp-go-guardrails-and-safety` | MCP runtime + CI guardrails | **9** | 9 | 9 | n/t | 9 |
| `mcp-go-production-review` | Go MCP pre-production review | **8** | 8 | 8 | n/t | 7 |
| `mcp-go-server-building` | Build Go MCP servers | **8** | 9 | 9 | n/t | 5 |
| `mcp-go-threat-modeling` | STRIDE threat modeling for MCP | **7** | 8 | 6 | n/t | 9 |
| `microservices-api-design` | REST / gRPC API contracts | **6** | 7 | 6 | n/t | 4 |
| `microservices-architecture-design` | End-to-end microservices design | **7** | 8 | 7 | n/t | 6 |
| `microservices-architecture-reviewer` | Architecture verdict review report | **8** | 9 | 7 | n/t | 8 |
| `microservices-async-messaging` | Sync vs async + broker choice | **7** | 8 | 7 | n/t | 6 |
| `microservices-data-architecture` | Saga, outbox, CQRS patterns | **8** | 8 | 8 | n/t | 6 |
| `microservices-resilience` | Resilience + rollout patterns | **7** | 7 | 7 | n/t | 4 |
| `new-azure-service-bootstrap` | Scaffold a new Azure service | **8** | 9 | 8 | n/t | 7 |
| `pr-review-azure-microservices` | PR review checklist for services | **8.5** | 8 | 8 | n/t | 6 |
| `soc2-iso27001-controls-mapping` | SOC 2 / ISO 27001 Azure mapping | **8** | 9 | 7 | n/t | 9 |

**Aggregate:** avg critical review 7.4 (live) · 7.8 projected once the two staged upgrades are applied · avg Claude fit 8.2 · avg depth 7.6 · avg diff 6.6.

Critical review distribution (live scores): 2 skills at 9, 8 at 8–8.5, 5 at 7, 1 at 6, 2 at 5. The two 5s (`azure-microservices-security`, `azure-service-mapping`) are the only skills the review judged genuinely below the bar — both for the same reason: a strong `SKILL.md` undercut by a main reference file that is cloud-agnostic or contradicts the canonical stack. Both have a rewrite staged (`5 → 8*`); the live scores stay 5 until the staged files are applied (see Critical review findings).

---

## Communication skills — `instruction-os/skills/` (2)

Persona-derived skills. `Claude score` / `Status` are the persona-rubric values; `Critical review` is the 2026-05-22 independent pass.

| Skill | Description | Critical review | Claude score | Status |
|---|---|---:|---:|---|
| `aaraminds-ai-engineering-architect` | Architecture design and review | **7** | 9.3 | Stable |
| `aaraminds-content-strategist` | Thought leadership, LinkedIn, newsletters | **7.5** | 9.0 | Validated |

Both are composition-wiring skills with no `references/` folder of their own — they score lower on the critical review than on the persona rubric because each takes a hard external dependency on `instruction-os/` persona files with no in-skill fallback if a module is renamed or missing.

---

## Personas — `instruction-os/Persona/` (5)

Role-based context blocks. Composed with system modules per each persona's `## Composition` section.

| Persona | Description | Claude score | Status |
|---|---|---:|---|
| AI Engineering Architect v1.2 | Architecture design and review | 9.3 | Stable |
| AI Business Strategist v1.1 | Strategy, positioning, founder decisions | 9.3 | Stable |
| AI Agent Blueprint Advisor v1.1 | Single-agent blueprint design + review | 9.2 | Stable |
| Content Strategist v1.0 | Public thought leadership content | 9.0 | Validated |
| Executive Narrative Advisor v1.0 | Exec updates, decks, decision briefs | 9.0 | Validated |

---

## System modules — `instruction-os/Persona/` (8)

Base and optional modules loaded by personas.

| Module | Description | Claude score | Status |
|---|---|---:|---|
| 04 — Framework Creation | Leadership frameworks, decision lenses | 9.3 | Stable |
| 05 — AI Systems Review | Structural review, findings-led | 9.2 | Stable |
| 08 — AI Agent Blueprint | Use case → buildable agent blueprint | 9.2 | Stable |
| 07 — AI Engineering Trend Scan | Recency-grounded trend scans | 9.1 | Stable |
| 03 — Newsletter Editorial | Long-form newsletters and articles | 9.0 | Stable |
| 01 — Layered Base | Canonical foundation: voice, reasoning, gates | 8.9 | Stable |
| 02 — Visual Identity | Diagrams, infographics, architecture posters | 8.9 | Stable |
| 06 — LinkedIn Post | Short / medium-form LinkedIn posts | 8.9 | Stable |

---

## Agents

The canonical `agents/` folder is **empty** — no runnable Claude Code subagents exist yet.

Three GitHub Copilot agent definitions exist under `skills-pack/copilot/agents/` (Copilot format, not Claude subagents). They would need porting into `agents/` to become runnable Claude agents.

| Agent (Copilot format) | Description | Claude fit | Depth | Strength | Diff. |
|---|---|---:|---:|---:|---:|
| `aara-senior-microservices-architect` | End-to-end microservices architect | 9 | 8 | n/t | 7 |
| `aara-mcp-server-builder` | Go MCP server builder | 9 | 7 | n/t | 7 |
| `aara-azure-cost-reviewer` | Azure FinOps cost reviewer | 8 | 8 | n/t | 6 |

---

## Critical review findings (2026-05-22)

**The pattern across the pack: `SKILL.md` quality is uniformly strong; reference depth and reference-file stack discipline are the limiting factors.** Every skill's router file is clear, bounded, and written in a confident principal-engineer voice with named anti-patterns and brownfield worked examples. The scores below 8 are almost entirely about what happens *behind* the router.

**Stack drift in main reference files is the single most damaging recurring defect.** Several skills pair a stack-correct `SKILL.md` with a `references/` entry-point file that contradicts the canonical stack:

- `azure-service-mapping` → `azure-mapping.md` defaults to **Azure SQL** instead of Postgres Flexible Server, recommends **Application Insights / Log Analytics** instead of Grafana + Prometheus + OTel, and quotes Container Apps pricing roughly **2× higher** than the pack's own cost reference. For a skill whose entire job is "what Azure service to use," a wrong-default main reference is a critical defect — hence the 5.
- `azure-microservices-security` → `security-design.md` is effectively cloud-agnostic security education with thin Azure specifics, and mentions **"OPA, AWS IAM"** (AWS is a stack violation). No Entra app-registration walk-through, no Workload Identity federated-credential Terraform, no Key Vault RBAC setup — the depth the `SKILL.md` promises is not delivered. Hence the 5.
- `azure-microservices-observability` → `observability-design.md` and `azure-microservices-cost-review` → `cost-and-tradeoffs.md` both drift toward Application Insights and (in cost) "Azure Cognitive Search," even though the newer reference files in those same skills are stack-correct. These skills still score 7–8 because the *newer* references are strong, but the orphaned root references drag them down.

**The strongest skills earn it on read.** `azure-data-tier-design` (9) and `mcp-go-guardrails-and-safety` (9) have deep, runnable, stack-correct reference content. `pr-review-azure-microservices` (8.5) is the best-structured reviewer skill — 18 named Go anti-patterns, 15 Spring Boot, 15 Terraform, all stack-specific.

**Priority fixes, in order:**

| Skill | Critical score | Highest-leverage fix |
|---|---:|---|
| `azure-service-mapping` | 5 → 8\* | **Upgrade staged 2026-05-22** — `azure-mapping.md` rewritten stack-correct + 5 new reference files. Pending apply. |
| `azure-microservices-security` | 5 → 8\* | **Upgrade staged 2026-05-22** — `security-design.md` rewritten Azure-specific, AWS mention removed, + 5 new reference files. Pending apply. |
| `microservices-api-design` | 6 | Stop duplicating `SKILL.md` in `api-design.md`; add real `.proto` design + OpenAPI CI tooling (`spectral`, `oasdiff`) |
| `azure-microservices-observability` | 8 | Re-align `observability-design.md` root reference to Grafana + Prometheus + OTel |
| `azure-microservices-cost-review` | 7 | Add staleness caveats to `cost-and-tradeoffs.md`; drop the Cognitive Search reference |
| `pr-review-azure-microservices` | 8.5 | Add GitHub Actions workflow anti-patterns (`pull_request_target` misuse, unscoped `permissions:`) |
| `soc2-iso27001-controls-mapping` | 8 | Add the SOC 2 Type I vs Type II distinction to the decision rule |

**Staged upgrade (2026-05-22).** The two priority-1 fixes are written and staged in `skill-upgrades-2026-05-22/`, with a one-command `apply-skill-upgrades.ps1` to copy them into place — necessary because the `.claude/skills/` tree is write-protected in the working session. `azure-service-mapping`: `azure-mapping.md` rewritten so Postgres Flexible Server is the relational default, OpenTelemetry + Prometheus + Grafana is the observability mapping, and the inflated Container Apps pricing is removed; four new deep-dive references added (compute, data, messaging, ingress) plus observability-and-identity. `azure-microservices-security`: `security-design.md` rewritten from cloud-agnostic content into an Azure-specific six-layer design with a Workload Identity worked example, the `OPA, AWS IAM` line removed; five new deep-dive references added (identity, secrets, network, data protection, audit). Each new file follows the `azure-data-tier-design` gold-standard structure. Both are projected at ~8/10 once applied; the `→ 8` figures remain self-assessed until an independent re-review. **Until `apply-skill-upgrades.ps1` is run, the live skills remain at 5.**

**Note on divergence from `ranking.md`:** the original rubric scored `azure-microservices-security` at Claude-fit 9. The critical review scores it 5 — not because the `SKILL.md` is weak (it is good) but because the rubric never read the reference files closely enough to catch that the promised Azure depth isn't there. Where the two disagree, the critical-review column reflects what is actually in the files.

---

## Summary

| Category | Count | Notes |
|---|---:|---|
| Engineering skills | 18 | Native Claude Skills format, in `skills-pack/` · avg critical review 7.4 |
| Communication skills | 2 | Persona-derived, in `instruction-os/skills/` |
| Personas | 5 | Composed with 8 system modules |
| System modules | 8 | Base + optional, loaded by personas |
| Claude agents | 0 | `agents/` folder empty |
| Copilot agents | 3 | In `skills-pack/copilot/agents/`, not yet ported |

**Strongest skills (critical review):** `azure-data-tier-design` and `mcp-go-guardrails-and-safety` (9), `pr-review-azure-microservices` (8.5).

**Weakest skills (critical review):** `azure-microservices-security` and `azure-service-mapping` (live score 5) — strong routers, defective main reference files. Both have a rewrite staged 2026-05-22, projected 8 once `apply-skill-upgrades.ps1` is run.

**Standout differentiators:** `mcp-go-guardrails-and-safety`, `mcp-go-threat-modeling`, and `soc2-iso27001-controls-mapping` (diff 9 — rare elsewhere).

**Top-rated personas/modules:** AI Engineering Architect v1.2, AI Business Strategist v1.1, and Module 04 Framework Creation (all 9.3 — the pack ceiling until production evidence).

---

_Sources: `skills-pack/ranking.md`, `instruction-os/Persona/Rankings.md`, and an independent critical review of every skill (2026-05-22)._
