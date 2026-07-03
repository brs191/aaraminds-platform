# Template — agent efficiency scorecard

Functional + safety say *whether* the agent works; efficiency says *what it costs* to work. Measured
from traces (OTel GenAI spans), reported per the `eval-result.schema.json` `efficiency` block. Feeds the
Production-Readiness dimension and the behavioral eval.

| Metric | Definition | Target | Measured | Verdict |
|---|---|---|---|---|
| Success rate | % tasks completed correctly (functional pass) | ≥ {{target}} | | |
| Latency p50 / p95 | wall-clock per task | p95 ≤ {{target}} | | |
| Total tokens / task | input + output + reasoning | ≤ {{budget}} | | |
| Tool calls / task | count | ≤ {{expected}} | | |
| Redundant tool calls | unnecessary/repeated (from trace review) | ~0 | | |
| Retries / task | retried steps | ≤ {{n}} | | |
| Loops detected | repeated tool/step cycles | 0 | | |
| Cost per **successful** task | $ ÷ successful tasks (not raw $) | ≤ {{budget}} | | |
| Human rework rate | % outputs needing human correction before use | ≤ {{target}} | | |

## Rules
- **Cost is per *successful* task, not per call** — a cheap agent that fails is not cheap.
- Loops > 0 or rising trace-depth/token-cost vs baseline → a behavioral finding; cap with max-turns.
- Track these as **baselines from the first run** and watch for drift on sampled production traces.
- Efficiency never overrides correctness or safety — a faster, cheaper agent that violates policy fails.

## Baseline record
```md
Agent: {{name}}  Version: {{v}}  Date: {{date}}  Trials: {{k}}
Success: __%  p95: __ms  tokens/task: __  tool-calls/task: __  loops: __  cost/success: $__  rework: __%
Drift vs last baseline: {{better | same | worse — note}}
```
