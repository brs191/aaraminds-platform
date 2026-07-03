# Skill audit — 2026-06-03

- Skills scanned: **29**
- Skill FAIL findings: **0**
- WARN findings: **7**
- Doc-consistency FAIL findings: **16**

## Failures (must fix)

_None._

## Warnings (consider fixing)

| Skill | Check | Detail |
|---|---|---|
| `azure-network-cost-forecasting` | `DESCRIPTION_OVERLOAD` | 774 chars (recommend <= 700) |
| `azure-network-cost-forecasting` | `ANEMIC_SKILL` | 54 body lines |
| `azure-network-iac-generation` | `DESCRIPTION_OVERLOAD` | 869 chars (recommend <= 700) |
| `azure-network-iac-generation` | `ANEMIC_SKILL` | 54 body lines |
| `azure-network-topology-analysis` | `DESCRIPTION_OVERLOAD` | 792 chars (recommend <= 700) |
| `azure-network-topology-analysis` | `ANEMIC_SKILL` | 57 body lines |
| `azure-network-topology-analysis` | `OFF_STACK_DRIFT` | references/reachability-and-severity.md: 'AWS' (offset 677) |

## Documentation consistency

Living documents must agree with disk on skill / agent / hook / reference counts. Dated records under `validation/snapshots/` are exempt.

| Document | Check | Detail |
|---|---|---|
| `README.md` | `DOC_COUNT_DRIFT` | line 40: states "26 Tier-1 skills", disk has 29 skills |
| `README.md` | `DOC_COUNT_DRIFT` | line 97: states "26 Tier-1 skills", disk has 29 skills |
| `README.md` | `DOC_COUNT_DRIFT` | line 166: states "26 Tier-1 skills", disk has 29 skills |
| `README.md` | `DOC_COUNT_DRIFT` | line 59: states "3 agents", disk has 4 agents |
| `README.md` | `DOC_COUNT_DRIFT` | line 245: states "3 agents", disk has 4 agents |
| `VERIFICATION_CHECKLIST.md` | `DOC_COUNT_DRIFT` | line 196: states "26 Tier-1 skills", disk has 29 skills |
| `copilot/README.md` | `DOC_COUNT_DRIFT` | line 22: states "3 agent", disk has 4 agents |
| `copilot/README.md` | `DOC_COUNT_DRIFT` | line 42: states "3 agent", disk has 4 agents |
| `how-to-use-in-vscode.md` | `DOC_COUNT_DRIFT` | line 141: states "3 agents", disk has 4 agents |
| `migration-map.md` | `DOC_COUNT_DRIFT` | line 106: states "26 Tier-1 skills", disk has 29 skills |
| `migration-map.md` | `DOC_COUNT_DRIFT` | line 111: states "26 Tier-1 skills", disk has 29 skills |
| `migration-map.md` | `DOC_COUNT_DRIFT` | line 106: states "3 agents", disk has 4 agents |
| `usage.md` | `DOC_COUNT_DRIFT` | line 52: states "26 Tier-1 skills", disk has 29 skills |
| `usage.md` | `DOC_COUNT_DRIFT` | line 53: states "3 agents", disk has 4 agents |
| `.claude/INDEX.md` | `INDEX_STALE` | index lists 26 skills, disk has 29 — run: python3 validation/tools/skill_audit.py --emit-index |
| `.claude/INDEX.md` | `INDEX_STALE` | index lists 3 agents, disk has 4 — run: python3 validation/tools/skill_audit.py --emit-index |

## Inventory

| Skill | Body lines | Description chars | References | Version | Last updated |
|---|---|---|---|---|---|
| `ai-application-architecture` | 83 | 695 | 12 | 1.1.0 | 2026-05-30 |
| `ai-evaluation-harness` | 76 | 640 | 6 | 1.1.0 | 2026-05-30 |
| `azure-data-tier-design` | 107 | 630 | 20 | 1.2.1 | 2026-05-30 |
| `azure-microservices-cost-review` | 77 | 563 | 5 | 1.0.0 | 2026-05-18 |
| `azure-microservices-observability` | 82 | 647 | 5 | 1.1.0 | 2026-05-21 |
| `azure-microservices-security` | 90 | 609 | 7 | 1.1.0 | 2026-05-30 |
| `azure-network-cost-forecasting` | 54 | 774 | 3 | 0.1.0 | 2026-06-03 |
| `azure-network-iac-generation` | 54 | 869 | 3 | 0.1.0 | 2026-06-03 |
| `azure-network-topology-analysis` | 57 | 792 | 4 | 1.1.0 | 2026-06-03 |
| `azure-service-mapping` | 96 | 588 | 5 | 1.0.1 | 2026-05-30 |
| `codebase-comprehension` | 76 | 647 | 6 | 1.1.0 | 2026-05-30 |
| `codebase-extraction-engineering` | 78 | 688 | 6 | 1.1.0 | 2026-05-30 |
| `data-access-engineering` | 77 | 689 | 6 | 1.1.0 | 2026-05-30 |
| `frontend-engineering` | 72 | 688 | 5 | 1.0.1 | 2026-05-30 |
| `mcp-go-guardrails-and-safety` | 88 | 649 | 9 | 1.1.0 | 2026-05-30 |
| `mcp-go-production-review` | 92 | 652 | 5 | 1.0.0 | 2026-05-18 |
| `mcp-go-server-building` | 120 | 621 | 15 | 1.0.1 | 2026-05-30 |
| `mcp-go-threat-modeling` | 111 | 605 | 5 | 1.1.0 | 2026-05-30 |
| `microservices-api-design` | 97 | 648 | 3 | 1.0.0 | 2026-05-18 |
| `microservices-architecture-design` | 100 | 645 | 3 | 1.1.0 | 2026-05-19 |
| `microservices-architecture-reviewer` | 93 | 665 | 3 | 1.0.1 | 2026-05-30 |
| `microservices-async-messaging` | 85 | 606 | 4 | 1.0.1 | 2026-05-30 |
| `microservices-data-architecture` | 85 | 591 | 7 | 1.0.0 | 2026-05-18 |
| `microservices-resilience` | 91 | 677 | 6 | 1.0.0 | 2026-05-18 |
| `new-azure-service-bootstrap` | 96 | 671 | 4 | 1.0.0 | 2026-05-18 |
| `pr-review-azure-microservices` | 90 | 630 | 4 | 1.0.1 | 2026-05-30 |
| `python-service-engineering` | 70 | 672 | 5 | 1.0.1 | 2026-05-30 |
| `soc2-iso27001-controls-mapping` | 109 | 599 | 3 | 1.0.1 | 2026-05-30 |
| `test-engineering` | 72 | 626 | 5 | 1.0.1 | 2026-05-30 |

## Agents

| Agent | Model | Description chars |
|---|---|---|
| `aara-azure-cost-reviewer` | sonnet | 718 |
| `aara-mcp-server-builder` | inherit | 790 |
| `aara-network-topology-reviewer` | inherit | 985 |
| `aara-senior-microservices-architect` | opus | 679 |

## Hooks

| Hook | Bytes |
|---|---|
| `block-dangerous-commands.json` | 3597 |
| `pre-commit-lint.json` | 1692 |
| `test-before-commit.json` | 1908 |

## Communication skills (instruction-os)

| Skill | Description chars | Over 1024? |
|---|---|---|
| `aaraminds-ai-engineering-architect` | 824 | no |
| `aaraminds-content-strategist` | 814 | no |
| `aaraminds-project-planner` | 943 | no |

---

_Generated by `validation/tools/skill_audit.py`. Report-only — no source files were modified._