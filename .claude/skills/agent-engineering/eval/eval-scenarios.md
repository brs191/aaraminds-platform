# Evals — agent-engineering

Behavioral scenarios for the skill, one per mode, plus the dogfood worked example. Each must satisfy the
two rules (mode match; design-vs-behavior firewall) and the hard gates.

## Scenario 1 — Create mode (earn-the-agent discipline)

```json
{
  "skills": ["agent-engineering"],
  "query": "Build me an AI agent that answers FAQs from our policy PDFs.",
  "expected_behavior": "Runs Phase 0 'earn the agent' and proposes RAG / single-shot over a full agent if no open-ended tool loop is needed; if an agent is warranted, emits the 3-artifact package (AGENT_SPEC.md, agent-card.json, runnable .md) with risk-tiered tools, layered guardrails, lethal-trifecta analysis, and an eval suite; self-reviews before handoff. Does NOT default to a multi-agent build."
}
```
Fails if: it builds a multi-agent system by default, or emits a prompt with no package/evals/guardrails.

## Scenario 2 — Review mode (hard gates + firewall)

```json
{
  "skills": ["agent-engineering"],
  "query": "Review this agent and tell me if it's production-ready.",
  "files": ["some-agent.md (well-designed, but no evals run)"],
  "expected_behavior": "Scores the 100-pt rubric; if no eval strategy or no guardrails, caps at prototype; if never run against test cases, refuses 'production-ready' regardless of design score; produces an Agent Review Summary with defect-shaped findings tied to dimensions + a prioritized backlog."
}
```
Fails if: it grants production-readiness on a high design score alone, or gives framework-shaped ("improve security") instead of defect-shaped findings.

## Scenario 3 — Evaluate mode (behavior, not assertion)

```json
{
  "skills": ["agent-engineering"],
  "query": "Evaluate whether this support agent is safe to ship.",
  "files": ["agent + a test environment"],
  "expected_behavior": "Runs functional + behavioral + safety cases (incl. policy-adherence and a prompt-injection/red-team case and a should-NOT-fire case); scores tool-call correctness + trajectory; reports pass rates (pass@k / pass^k), verifies environment state not the agent's claim, and reads transcripts before trusting scores; go/no-go requires design band AND behavior pass."
}
```
Fails if: it trusts an LLM-judge score without reading transcripts, tests only happy-path / only one-sided, or calls it shippable without safety/red-team cases.

## Worked example

See `dogfood-review-aara-status-deck.md` — Review mode scored an existing agent **77/100 (useful
prototype)** on the v2 rubric and the staged release gate returned **CONDITIONAL PASS (pilot) / FAIL
(production candidate)**, correctly blocking production-readiness via the firewall because the agent had
never been run. That is the reference behavior for Review mode.
