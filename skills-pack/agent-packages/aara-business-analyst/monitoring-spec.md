# Monitoring spec — aara-business-analyst (production)

Metrics, thresholds, and alert routing for the production runtime (Foundry/LangGraph). This is the
**monitoring plan** (required for a production candidate); going live with it is the "production" stage.

| Metric | Source | Threshold / alert | Route to |
|---|---|---|---|
| Hallucinated-requirement rate | sampled output review vs. evidence | > 1% → alert | BA lead |
| Source-coverage (claims traced) | trace check on each deliverable | < 100% load-bearing → block delivery | agent (auto) + BA lead |
| Ambiguity-catch rate | labeled eval set, weekly | drop > 10% vs. baseline → review | eval engineer |
| Reviewer-override rate | review-routing outcomes | > 30% → prompt/regression review | BA lead |
| Injection-refusal rate | adversarial canary cases in prod traffic | any miss → page | security |
| Cycle time / rework | ticketing adapter | trend worse 2 periods → review | delivery lead |
| Cost per successful deliverable | trace tokens ÷ approved deliverables | > budget → review | platform |
| Latency p95 | OTel GenAI spans | > target → review | platform |

Instrumentation: OpenTelemetry GenAI spans (invoke_agent, execute_tool), per-action audit log, weekly
online eval on sampled traces for drift. Baselines (from the 3-trial run): ~23.5K tokens/run, 0 tool
calls, ~35s, 6/6 behavioral pass.
