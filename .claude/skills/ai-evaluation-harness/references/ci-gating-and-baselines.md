# CI Gating and Baselines

An evaluation that does not block a merge is decoration. This reference covers wiring the eval into CI as a gate, capturing baselines, hard versus soft gates, what triggers the eval, per-criterion gating, the eval's own cost and runtime, and the model-upgrade gate.

## An eval that does not gate is decoration

The eval suite's job is to stop a regression from shipping. If a failing score does not fail the build, the suite informs nobody at the moment it matters and stops nothing — it is a dashboard, not a guardrail. The gate is the point. This is the discipline `pr-review-azure-microservices` applies to code — a hard-fail finding blocks the merge — applied to AI output.

## The baseline — capture current behaviour first

A gate needs something to gate against. The baseline is the eval score of the current, accepted implementation, captured and recorded before any change; every subsequent run is compared to it. For a new feature, the baseline is the first implementation that clears the rubric thresholds. For a brownfield feature with no eval, the baseline is the current production behaviour captured as-is (the skill's worked example). The baseline is stamped with the golden-dataset version (`golden-datasets-and-fixtures.md`) and the model deployment — a baseline is only comparable within the same dataset and model.

## Hard gate vs soft gate

Not every signal should block. Split them:

- **Hard gate (blocks the merge)** — a per-criterion score regressing below its threshold; a safety or correctness metric dropping; schema-conformance failing.
- **Soft gate (warns, does not block)** — a small movement within normal variance; a new metric still being calibrated; a judge-scored criterion whose calibration is not yet trusted.

A gate that blocks on noise gets disabled by the team within a week; a gate that blocks on nothing protects nothing. Calibrate the hard/soft split against observed run-to-run variance.

## What triggers the eval

The eval runs on every change that can change behaviour — and that is more than the application code. A prompt template, a retrieval parameter, a model deployment, an orchestration change, and the golden dataset itself are all behaviour-affecting; all are version-controlled and all trigger the eval. Treat the prompt as code: the most behaviour-defining file in the repository cannot be the one file that ships unevaluated.

## Per-criterion gating, not one aggregate

The gate evaluates each rubric criterion against its own threshold — not one averaged score against one threshold. An aggregate can stay green while a critical criterion (faithfulness, safety) collapses and a cosmetic one improves enough to mask it. Per-criterion gating also localizes the failure: CI reports which criterion regressed, so the fix is targeted.

## The eval's own cost and runtime

The eval is itself a workload — it runs the feature over the golden dataset, which means model calls, on every gated change. Keep it affordable: size the dataset for signal not bulk (`golden-datasets-and-fixtures.md`), prefer cheap deterministic checks over judge calls where possible (`scoring-methods.md`), and run judge-heavy evals on the Azure OpenAI Batch API where the CI flow tolerates the latency. If the full eval is too slow for every commit, run a fast subset per commit and the full suite pre-merge — but the full suite must gate the merge.

## The model-upgrade gate

A model upgrade is the highest-stakes change an AI feature receives and the one most often made blind. It is exactly a gated change: run the full eval on the new model against the baseline captured on the old one, per criterion. The upgrade ships only if it holds or improves every gated criterion. "The new model is better" is a vendor claim until the feature's own eval confirms it on the feature's own golden dataset.

## Verification questions

1. Does a failing eval score actually fail the CI build and block the merge?
2. Is there a recorded baseline, stamped with the golden-dataset version and the model deployment?
3. Is the hard/soft gate split calibrated against observed run-to-run variance?
4. Does the eval trigger on prompt, retrieval, model, orchestration, and dataset changes — not only application code?
5. Is gating per-criterion against per-criterion thresholds, not one aggregate score?
6. Is the eval's own CI cost and runtime controlled — dataset sized for signal, deterministic checks preferred, batch where it fits?
7. Is a model upgrade run through the full eval against the prior baseline before it ships?

## What to read next

- `rubric-and-metric-design.md` — the thresholds the gate enforces
- `golden-datasets-and-fixtures.md` — the dataset version the baseline is stamped with
- `scoring-methods.md` — deterministic vs judge cost in CI
- `online-eval-and-drift.md` — the production half of the picture
- `pr-review-azure-microservices` — the CI gate discipline this mirrors
