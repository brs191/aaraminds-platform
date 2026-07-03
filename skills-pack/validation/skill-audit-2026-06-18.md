# Skill audit — 2026-06-18

- Skills scanned: **34**
- Skill FAIL findings: **0**
- WARN findings: **15**
- Doc-consistency FAIL findings: **0**

## Failures (must fix)

_None._

## Warnings (consider fixing)

| Skill | Check | Detail |
|---|---|---|
| `azure-defender-signal-ingestion` | `DESCRIPTION_OVERLOAD` | 986 chars (recommend <= 700) |
| `azure-defender-signal-ingestion` | `ANEMIC_SKILL` | 61 body lines |
| `azure-defender-signal-ingestion` | `NO_BROWNFIELD_EXAMPLE` | worked example does not appear to be brownfield |
| `azure-iac-policy-as-code` | `DESCRIPTION_OVERLOAD` | 958 chars (recommend <= 700) |
| `azure-iac-policy-as-code` | `ANEMIC_SKILL` | 63 body lines |
| `azure-iac-policy-as-code` | `NO_BROWNFIELD_EXAMPLE` | worked example does not appear to be brownfield |
| `azure-network-cost-forecasting` | `DESCRIPTION_OVERLOAD` | 774 chars (recommend <= 700) |
| `azure-network-cost-forecasting` | `ANEMIC_SKILL` | 54 body lines |
| `azure-network-iac-generation` | `DESCRIPTION_OVERLOAD` | 869 chars (recommend <= 700) |
| `azure-network-iac-generation` | `ANEMIC_SKILL` | 54 body lines |
| `azure-network-topology-analysis` | `DESCRIPTION_OVERLOAD` | 792 chars (recommend <= 700) |
| `azure-network-topology-analysis` | `ANEMIC_SKILL` | 57 body lines |
| `azure-network-topology-analysis` | `OFF_STACK_DRIFT` | references/reachability-and-severity.md: 'AWS' (offset 677) |
| `azure-network-topology-visualization` | `DESCRIPTION_OVERLOAD` | 880 chars (recommend <= 700) |
| `azure-network-topology-visualization` | `ANEMIC_SKILL` | 66 body lines |

## Documentation consistency

Living documents must agree with disk on skill / agent / hook / reference counts. Dated records under `validation/snapshots/` are exempt.

_No drift — every living document agrees with the filesystem._

## Inventory

| Skill | Body lines | Description chars | References | Version | Last updated |
|---|---|---|---|---|---|
| `agent-engineering` | 126 | 671 | 7 | 2.4.0 | 2026-06-18 |
| `ai-application-architecture` | 83 | 695 | 12 | 1.1.0 | 2026-05-30 |
| `ai-evaluation-harness` | 76 | 640 | 6 | 1.1.0 | 2026-05-30 |
| `azure-data-tier-design` | 107 | 630 | 20 | 1.2.1 | 2026-05-30 |
| `azure-defender-signal-ingestion` | 61 | 986 | 4 | 1.0.0 | 2026-06-15 |
| `azure-iac-policy-as-code` | 63 | 958 | 4 | 1.0.0 | 2026-06-15 |
| `azure-microservices-cost-review` | 77 | 563 | 5 | 1.0.0 | 2026-05-18 |
| `azure-microservices-observability` | 82 | 647 | 5 | 1.1.0 | 2026-05-21 |
| `azure-microservices-security` | 90 | 609 | 7 | 1.1.0 | 2026-05-30 |
| `azure-network-cost-forecasting` | 54 | 774 | 3 | 0.1.0 | 2026-06-03 |
| `azure-network-iac-generation` | 54 | 869 | 3 | 0.1.0 | 2026-06-03 |
| `azure-network-topology-analysis` | 57 | 792 | 4 | 1.1.0 | 2026-06-03 |
| `azure-network-topology-visualization` | 66 | 880 | 4 | 1.0.0 | 2026-06-15 |
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
| `prompt-engineering` | 103 | 681 | 5 | 1.0.0 | 2026-06-16 |
| `python-service-engineering` | 70 | 672 | 5 | 1.0.1 | 2026-05-30 |
| `soc2-iso27001-controls-mapping` | 109 | 599 | 3 | 1.0.1 | 2026-05-30 |
| `test-engineering` | 72 | 626 | 5 | 1.0.1 | 2026-05-30 |

## Agents

| Agent | Model | Description chars |
|---|---|---|
| `aara-agent-engineer` | inherit | 868 |
| `aara-ai-evaluation-engineer` | inherit | 589 |
| `aara-azure-cost-reviewer` | sonnet | 718 |
| `aara-business-analyst` | inherit | 810 |
| `aara-mcp-server-builder` | inherit | 790 |
| `aara-network-topology-reviewer` | inherit | 1044 |
| `aara-project-architect` | inherit | 674 |
| `aara-project-builder` | inherit | 658 |
| `aara-project-debugger` | inherit | 485 |
| `aara-project-planner` | inherit | 492 |
| `aara-project-reviewer` | inherit | 566 |
| `aara-prompt-engineer` | inherit | 754 |
| `aara-python-ai-developer` | inherit | 640 |
| `aara-senior-microservices-architect` | opus | 679 |
| `aara-status-deck` | inherit | 790 |
| `aara-topology-visualizer` | inherit | 900 |

## Hooks

| Hook | Bytes |
|---|---|
| `block-dangerous-commands.json` | 3597 |
| `pre-commit-lint.json` | 1692 |
| `test-before-commit.json` | 1908 |

## Communication skills (instruction-os)

| Skill | Description chars | Over 1024? |
|---|---|---|
| `aaraminds-ai-agent-blueprint-advisor` | 988 | no |
| `aaraminds-ai-engineering-architect` | 824 | no |
| `aaraminds-content-strategist` | 814 | no |
| `aaraminds-executive-narrative-advisor` | 989 | no |
| `aaraminds-leadership-status-deck` | 720 | no |
| `aaraminds-project-planner` | 943 | no |

---

_Generated by `validation/tools/skill_audit.py`. Report-only — no source files were modified._