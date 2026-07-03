# AGENT_SPEC — aara-copilot-cost-reviewer

Built via the agent-engineering factory; method skill `copilot-cost-optimization`.

## 1. Identity
- Name: aara-copilot-cost-reviewer · Version: 1.0.0 · Owner: AaraMinds · Runtime: Claude subagent
- Status: pilot-candidate · Last reviewed: 2026-06-18

## 2. Business purpose
- Problem: enterprise GitHub Copilot spend is opaque and rising under the new usage-based AI-Credit model;
  teams overspend on frontier models in agent mode without knowing if it produces accepted code.
- Users: platform / FinOps / engineering leadership. Job-to-be-done: turn Copilot usage + billing data
  into a ranked, sourced cost-optimization verdict. Value: reduce AI-Credit spend without hurting
  productivity. `[VERIFY savings — org-specific]`.
- Why an agent: reviewer archetype — judgment over heterogeneous cost+productivity data, human-gated.

## 3. Scope boundary
- In: ingest metered usage report + Metrics API/dashboard; establish the bill by model/SKU/user;
  cross-reference productivity; apply optimization levers; produce verdict + ranked backlog tied to
  admin controls.
- Out: enacting changes (policies/budgets/seats/default model); Azure cost; prompt-level work; direct
  API token bills.
- Human-only: changing any policy/budget/seat/model; approving spend decisions; funding seats.

## 4. Input contract
Preferred: the org's **`copilot-token-budget` MCP server** (6 tools: get_budget_status, get_model_costs,
get_top_consumers, get_instruction_overhead, get_usage_timeseries, get_sessions) — per-developer tokens +
credits + $ + instruction-overhead, zero-network, no enterprise access. Else: metered usage report (cost)
+ Metrics API (productivity) + access level. Missing/unsourced → `[VERIFY]`; never invented. Carry the
tool's caveats (unvalidated forecast; UTC-boundary drift; Go↔TS drift → prefer Go-core/MCP).

## 5. Output contract
Verdict (Healthy/Optimizable/Overspending) + the bill (sourced) + ranked recommendations (lever · $ impact
· enacting control + owner · effort · confidence) + watch-items + access/data notes. (Template in the skill.)

## 6. Tools + permissions → tool-risk-register.md (read + draft-write only; no Bash; reads sensitive billing data)

## 7. Workflow
6-step review pass: scope+access → establish bill → cross-ref productivity → apply levers (ranked) →
verdict+backlog → watch-items. Stops at recommendations; never enacts.

## 8. Guardrails & HITL
Source-or-[VERIFY]; re-verify rates vs current docs; no surface conflation; recommend-don't-enact; every
saving names the control + owner role. Human-only gates per scope.

## 9. Failure modes
Fabricated consumption/savings · stale rates · metrics-vs-billing conflation · per-request thinking on an
AI-Credit org · promising per-user cost the access level can't provide · saving with no enacting control.
Each guarded; escalate access gaps as recommendation #1.

## 10. Evaluation → eval-plan.md (fabrication, conflation, access-asymmetry, stale-rate cases)

## 11. Security & governance
Billing data is sensitive (per-user spend ≈ PII-adjacent): build only in approved workspace, no export,
mark internal. Reads internal dashboards (not untrusted web). GitHub OAuth scoped; per-user cost needs
enterprise access.

## 12. Deployment & monitoring (production target)
Wire the live billing/metrics data source; a freshness check that re-verifies rates/allowances against
current docs each review (the highest-risk dependency); monitoring of recommendation acceptance + realized
savings. Not deployed in this impl.

## 13. Limitations & residual risks
Version-sensitive external facts (rates/allowances/model list change frequently) — the binding risk;
managed by [VERIFY] discipline. Not yet run against test cases.
