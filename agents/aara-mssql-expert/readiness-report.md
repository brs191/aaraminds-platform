# Agent Readiness Report — aara-mssql-expert

Generated 2026-07-10T01:42:12Z · rubric 0.2.0 · agent version 0.1.0

## Verdict

**PASS** — score 95.0/100

Every field below is populated from verifiable checks, never self-attestation.

## Agent

| Field | Value |
|---|---|
| Business owner | Raja Shekar Bollam |
| Technical owner | Raja Shekar Bollam (acting engineering lead) |
| Autonomy level | 2 |
| Risk tier | medium |
| Justification | classifier: Drafting (risk tier medium, score 4, policy allowed) |

## Area Scores

| Area | Weight | Checks | Score |
|---|---:|---:|---:|
| Business scope and ownership | 10 | 3/3 | 10.00 |
| Autonomy and approval boundaries | 15 | 4/4 | 15.00 |
| MCP tool contract completeness | 15 | 4/4 | 15.00 |
| Identity and permissions | 15 | 3/3 | 15.00 |
| Data and source-of-truth mapping | 10 | 3/3 | 10.00 |
| Evaluation plan and test coverage | 15 | 3/4 | 11.25 |
| Security and governance controls | 10 | 5/5 | 10.00 |
| Compliance evidence | 5 | 2/2 | 5.00 |
| Export and build readiness | 5 | 3/4 | 3.75 |

## Failing Checks and Required Fixes

- `eval-runs-pass` (Evaluation plan and test coverage, eval-run)
  - Evidence: ../agents/aara-mssql-expert/eval-runs
- `artifacts-todo-free` (Export and build readiness, schema-validation)
  - Evidence: ../agents/aara-mssql-expert

## Check Evidence

| Check | Result | Mechanism | Evidence |
|---|---|---|---|
| intake-valid | pass | schema-validation | ../agents/aara-mssql-expert/agent-intake.yaml |
| owners-named | pass | schema-validation | ../agents/aara-mssql-expert/agent-intake.yaml |
| outcomes-stated | pass | schema-validation | ../agents/aara-mssql-expert/agent-intake.yaml |
| classification-current | pass | schema-validation | ../agents/aara-mssql-expert/classification.json |
| signoffs-recorded | pass | catalog-record | ../agents/aara-mssql-expert/signoffs.json |
| manifest-valid | pass | contract-lint | ../examples/mssql-expert.manifest.yaml |
| write-boundaries | pass | contract-lint | ../examples/mssql-expert.manifest.yaml |
| contracts-exist | pass | contract-lint | ../tool-contracts |
| contracts-lint | pass | contract-lint | ../tool-contracts |
| contracts-pinned | pass | contract-lint | ../examples/mssql-expert.manifest.yaml |
| manifest-agent-match | pass | schema-validation | ../examples/mssql-expert.manifest.yaml |
| identity-valid | pass | schema-validation | ../agents/aara-mssql-expert/agent-identity-spec.json |
| identity-complete | pass | schema-validation | ../agents/aara-mssql-expert/agent-identity-spec.json |
| identity-scopes-match | pass | schema-validation | ../agents/aara-mssql-expert/agent-identity-spec.json |
| evidence-contract-valid | pass | schema-validation | ../agents/aara-mssql-expert/data-evidence-contract.json |
| domains-mapped | pass | schema-validation | ../agents/aara-mssql-expert/data-evidence-contract.json |
| memory-citation-gate | pass | harness-gate | aapctl prove: memory-citation gates |
| eval-plan-sections | pass | schema-validation | ../agents/aara-mssql-expert/evaluation-plan.md |
| eval-safety-section | pass | schema-validation | ../agents/aara-mssql-expert/evaluation-plan.md |
| eval-gate-configured | pass | contract-lint | ../examples/mssql-expert.manifest.yaml |
| eval-runs-pass | fail | eval-run | ../agents/aara-mssql-expert/eval-runs |
| asi-checklist-complete | pass | schema-validation | ../agents/aara-mssql-expert/security-governance-checklist.md |
| proof-tool-denial | pass | harness-gate | aapctl prove: tool-denial gates |
| proof-memory-isolation | pass | harness-gate | aapctl prove: memory gates |
| proof-audit-chain | pass | harness-gate | aapctl prove: audit gates |
| prompt-injection-gate | pass | harness-gate | aapctl prove: prompt-injection tool-escalation gates |
| compliance-map-sections | pass | schema-validation | ../agents/aara-mssql-expert/compliance-evidence-map.md |
| compliance-complete | pass | schema-validation | ../agents/aara-mssql-expert/compliance-evidence-map.md |
| artifacts-complete | pass | schema-validation | ../agents/aara-mssql-expert |
| artifacts-todo-free | fail | schema-validation | ../agents/aara-mssql-expert |
| telemetry-payload-mode | pass | contract-lint | ../examples/mssql-expert.manifest.yaml |
| export-roundtrip | pass | export-roundtrip | ../agents/aara-mssql-expert/export-verification.json |

## Approvals Required

- business-owner: [unassigned]
- enterprise-ai-architect: [unassigned]

Source of truth: readiness-report.json (validated against schemas/readiness-report.schema.json).
