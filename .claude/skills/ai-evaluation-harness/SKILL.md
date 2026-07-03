---
name: ai-evaluation-harness
description: Designs the evaluation harness for AI/LLM features — golden datasets, rubric and metric design, scoring (deterministic vs LLM-as-judge), retrieval/RAG evaluation (groundedness, citation accuracy), CI eval gating with baselines, and online drift detection. Use when standing up eval for a generative feature, defining what good output means, building a golden dataset, evaluating a RAG/GraphRAG pipeline, wiring an eval gate into CI, or monitoring drift. Do not use for AI application architecture (ai-application-architecture, which calls this), MCP-server gates (mcp-go-guardrails-and-safety), or PR review (pr-review-azure-microservices).
version: 1.1.0
last_updated: 2026-05-30
---

# AI Evaluation Harness

## When to use

Trigger this skill when the question is how to *measure* whether an AI or LLM feature is good — not how to build the feature, which is `ai-application-architecture`. Common triggers: "how do we know the output is correct," "what does a good answer / document / extraction even mean," "we need a golden dataset," "which metric for this," "wire an eval gate into CI," "is this model upgrade safe," "is production quality drifting."

Do **not** use this skill for: AI application architecture — archetype, model, retrieval, serving topology (`ai-application-architecture`, which calls this skill for the evaluation layer); MCP-server CI quality gates and the guardrail eval (`mcp-go-guardrails-and-safety`); human PR review of code (`pr-review-azure-microservices`).

## The critical decision rule — define "correct" before you build, because the eval is the spec

A generative model is non-deterministic: the same input can produce different output, and "different" can mean "wrong" in ways no type system catches. The evaluation harness is the only thing that turns "it works" from a hope into a measured claim. So the rule is: **write the evaluation before the feature.** The golden dataset and the rubric *are* the specification — they define what the feature must do. A team that builds first and evaluates later cannot tell a model upgrade from a regression, has nothing to tune retrieval or prompts against, and learns about a quality drop from a user instead of from CI.

The Code Intelligence Factory states this in its strongest form: its roadmap makes the evaluation rubric and a reference repository with a hand-written golden HLD the *first* deliverables — before any product code — because "if the architecture map is wrong, the product is worthless." Eval-first is not process overhead; for a product whose value is trustworthy output, it is the only safe build order.

## Anatomy of an evaluation harness

Four parts. Skip any one and the harness gives false confidence.

| Part | What it is | Reference |
|---|---|---|
| Golden dataset | Representative inputs paired with correct outputs or reference fixtures | `references/golden-datasets-and-fixtures.md` |
| Rubric and metrics | The written definition of "correct / good" and the metrics that score it | `references/rubric-and-metric-design.md` |
| Scorers | The deterministic checks and LLM-as-judge that produce the scores | `references/scoring-methods.md` |
| The gate | CI integration, baselines, and regression thresholds that block a bad change | `references/ci-gating-and-baselines.md` |

On top of the four, **online evaluation** scores live traffic and catches drift the offline harness cannot see — `references/online-eval-and-drift.md`.

## Offline and online — both, because they do different jobs

Offline evaluation runs the feature against the golden dataset on every change and gates CI: it proves the feature is good against *known* inputs and catches regressions before they ship. Online evaluation scores a sample of *production* traffic and alerts on drift: it catches the inputs the golden set never had and the quality slides the offline suite cannot see. They are complements, not alternatives — offline gates change, online catches the unknown. Ship neither alone.

## Scoring — deterministic first, judge only where you must

Two scorer kinds. A **deterministic check** — schema valid, value matches, citation resolves, test passes — is cheap, fast, reproducible, and not itself subject to model non-determinism; use it wherever the correctness criterion can be expressed as code. An **LLM-as-judge** scores open-ended quality where no deterministic check exists. The judge is necessary and powerful, but it is itself a non-deterministic model call: it must be rubric-anchored, calibrated against human labels, and given its own eval — an unvalidated judge is just a second opinion of unknown quality. Reach for the judge only after the deterministic checks are exhausted.

## Retrieval and RAG evaluation

A retrieval-augmented or GraphRAG feature has two failure surfaces — retrieval (did it fetch the right context?) and generation (did the answer use it faithfully?) — and they must be scored separately, or a regression cannot be localized to one. Faithfulness (every claim entailed by retrieved context) is the anti-hallucination metric the CIF lives on; citation accuracy is largely a *deterministic* check, because `data-access-engineering`'s GraphRAG carries a source id on every node. `references/retrieval-and-rag-evaluation.md`.

## Build sequence

1. Define the rubric — what "correct / good" means, with named criteria and thresholds — `references/rubric-and-metric-design.md`.
2. Build the golden dataset and reference fixtures, including hard, ambiguous, and adversarial cases — `references/golden-datasets-and-fixtures.md`.
3. Pick metrics and scorers per the rubric; deterministic wherever possible — `references/scoring-methods.md`.
4. Capture a baseline against current behaviour, then wire the eval into CI as a blocking gate — `references/ci-gating-and-baselines.md`.
5. Stand up online sampling and drift alerting once the feature is in production — `references/online-eval-and-drift.md`.
6. Maintain the harness — the golden set and rubric are living artifacts; grow them from real production failures.

## Worked example — brownfield: an AI feature already in production with no eval

Setup: an LLM feature ships and works "well enough" in demos. There is no golden dataset, no rubric, no eval gate. Leadership wants to upgrade the model and add a capability.

Decision walk: (1) Do not refactor or change anything first. (2) Build the golden dataset from real production traffic and the known failure reports — the inputs the feature actually sees, including the ones it got wrong. (3) Write the rubric against what good output looks like for those inputs. (4) Stand up the offline eval against the *current* behaviour and record that as the baseline. (5) Only now make changes — the model upgrade, the new capability — each measured against the baseline. (6) Add online sampling so the next quality slide is caught by monitoring, not by a user.

The wrong move is to do the model upgrade first and "watch for problems." That changes the least predictable dependency in the system with no instrument to read the result.

## Anti-pattern — vanity metrics and the eval that does not gate

**Bad:** a dashboard shows an "eval score" of 92%. Nobody can say what the other 8% is, the golden set is all easy cases, and a failing score does not block a merge. **Why it fails:** an eval that does not gate is decoration — it informs nobody and stops nothing; a golden set of only easy cases scores high and proves nothing; a single aggregate score with no per-criterion breakdown cannot localize a regression. **Detection signal:** the golden dataset has no adversarial or edge cases; CI does not fail on an eval regression; the team cannot describe what a failing case looks like. **Fix:** build the golden set from hard cases, score against a per-criterion rubric, set per-metric thresholds, and make CI block on regression — the gate discipline `pr-review-azure-microservices` applies to code, applied to AI output.

## Verification questions

1. Were the rubric and golden dataset defined *before* the feature was built — or at least before the next change?
2. Does the golden dataset include hard, ambiguous, and adversarial inputs, not only easy cases?
3. Are deterministic checks used wherever possible, with LLM-as-judge reserved for genuine open-ended quality — and is the judge itself calibrated and evaluated?
4. Is there a recorded baseline, and does CI *fail the build* on a regression past a per-metric threshold?
5. Is a sample of production traffic scored online with drift alerting — not only offline gating?
6. Are the golden dataset and rubric maintained as living artifacts, grown from real production failures?
7. For a retrieval/RAG feature: are retrieval and generation scored separately, with faithfulness and a deterministic citation check against a labeled relevant set?

## What to read next

Tier-2 references: `references/golden-datasets-and-fixtures.md` · `references/rubric-and-metric-design.md` · `references/scoring-methods.md` · `references/retrieval-and-rag-evaluation.md` · `references/ci-gating-and-baselines.md` · `references/online-eval-and-drift.md`.

Related skills: `ai-application-architecture` (the AI feature this harness measures; its `references/evaluation.md` is the per-archetype view) · `mcp-go-guardrails-and-safety` (the MCP-server CI eval gate) · `pr-review-azure-microservices` (CI gate discipline) · `azure-microservices-observability` (the telemetry backbone for online evaluation).
