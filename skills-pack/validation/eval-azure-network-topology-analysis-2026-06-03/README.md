# Eval record — azure-network-topology-analysis (2026-06-03)

Skill eval folded in from the former top-level `skill-staging/eval/` on 2026-07-03.
This is the durable record: benchmark, expected findings, and eval definitions for
the `azure-network-topology-analysis` skill (base run + iter3), plus the
`azure-network-cost-forecasting` scenario (`expected-forecast.md`,
`scenario-firewall-egress.md`).

Not folded in (working artifacts, kept in `_archive/skill-staging/` locally,
untracked): eval workspaces, HTML review viewers, viewer-generation scripts,
fixtures. Regenerate viewers from `evals*.json` via `make_viewer*.py` if needed.

Files:

- `benchmark.md` — benchmark definition and results
- `expected-findings.md` / `evals.json` — base run
- `expected-findings3.md` / `evals3.json` — iter3
- `expected-forecast.md` / `scenario-firewall-egress.md` — cost-forecasting scenario
