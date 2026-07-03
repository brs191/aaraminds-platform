# Evaluation

An AI feature without an evaluation suite is a prototype, not a product. This reference covers evaluation as it applies to an AI *application* — the golden dataset, offline CI gating, archetype-specific metrics, online production monitoring, and the regression gate. The general discipline of building evaluation harnesses — rubric design, scoring infrastructure, golden-fixture management as a craft — is broader than this skill; a standalone evaluation-harness skill is planned for the pack, and when it lands this reference defers the harness mechanics to it and keeps the AI-application-specific guidance.

## Why this is not optional

A generative model is a non-deterministic dependency. Every other dependency in the pack — a database, a queue, a downstream service — has defined, testable behaviour. The model does not: the same prompt can yield different output, and "different" can mean "wrong" in ways no type system catches. The evaluation suite is the only thing standing between "the feature works" as a measured claim and "the feature works" as a hope. A team that ships an AI feature without one cannot tell a model upgrade from a regression, cannot tune retrieval, and learns about a quality drop from a user, not from CI.

## The golden dataset comes before the build

Define the golden dataset and the pass threshold **before** the feature is built, not after the first incident. The golden dataset is a set of representative inputs each paired with what a correct response looks like — an exact answer, a set of acceptable answers, or a rubric a judge applies. It is real, reusable work, not throwaway overhead: it is simultaneously the spec (it defines what "correct" means), the regression guard, and the artifact that makes a model swap a measured decision.

Build it from real inputs, including the hard ones: ambiguous cases, inputs with missing data, adversarial phrasings, and the long tail. A golden set of only easy cases passes at 100% and tells you nothing. The Code Intelligence Factory's roadmap puts this first for exactly this reason — its M0 deliverables are an evaluation rubric and a reference repository with a hand-written golden HLD, and the roadmap calls the eval harness "existential": if you cannot define what a good HLD is, there is nothing to build toward.

## Offline evaluation — the CI gate

Offline evaluation runs the feature against the golden dataset and scores the results. Wire it into CI with **DeepEval or Ragas**, beside the pack's existing PR-review gates. It runs on every change to a prompt template, a retrieval parameter, a model deployment, or orchestration logic — all four are behaviour changes, and a behaviour change that does not run the eval is an untested change to the least predictable part of the system. Treat the prompt as code: it is version-controlled and it triggers the eval.

## Metrics by archetype

What you measure depends on the archetype (`patterns/`). Do not score every feature on "is the answer good" — measure the thing that actually fails for that archetype:

| Archetype | What to measure |
|---|---|
| Single-shot | Schema conformance, field accuracy, and abstention — does it return null/uncertain instead of inventing a value when the input lacks it |
| RAG | Faithfulness/groundedness (is the answer supported by the retrieved context), plus the retrieval metrics — recall@k, context-precision, context-recall (`retrieval-design.md`) |
| Agentic loop | Task success (was the goal reached), tool-call correctness (right tool, right arguments), and cost/steps per run — not output equality |
| LLM workflow | Each model node scored individually plus an end-to-end score, so a regression can be localized to a node |
| Conversational | Multi-turn dialogue success, not isolated turns — coherence and memory correctness across the conversation |

For open-ended output where there is no exact answer, use an **LLM-as-judge** scored against a written rubric — but where a deterministic check is possible (schema valid, value matches, citation resolves), use the deterministic check. Deterministic checks are cheaper, faster, and not themselves subject to model non-determinism; reserve the judge for genuine quality judgement.

## Online evaluation — production monitoring

Offline eval proves the feature is good against known inputs; production sends inputs the golden set never had. Use **Foundry Evaluations with continuous monitoring** to score a sample of live traffic into Azure Monitor, and alert on drift — a metric sliding over a rolling window is an early warning a static offline suite cannot give. Online and offline are complementary: offline gates change, online catches the unknown. The telemetry backbone is `azure-microservices-observability`.

## The regression gate

Evaluation only protects quality if a failing score actually blocks. Define the pass threshold per metric, and have CI **fail the build** when a score regresses past it — the same gate discipline `pr-review-azure-microservices` applies to code. A green "eval score" that nobody gates on is a dashboard, not a guardrail. The threshold is set once against the golden dataset and moved only deliberately, with the move recorded.

## Evaluating a brownfield feature with no eval

The common real case: an AI feature is already in production and has no eval suite (the notebook-to-production anti-pattern in the SKILL.md). Do not refactor it first. Build the golden dataset from its real traffic and known failures, stand up the offline eval against the *current* behaviour to capture a baseline, and only then change anything — now every change is measured against that baseline. Retrofitting the eval before touching the feature converts an unmeasurable system into a measurable one; that is the highest-value first move on inherited AI code.

## Verification questions

1. Does the feature have a golden dataset, defined before the build, including hard, ambiguous, and adversarial inputs?
2. Is there an offline eval (DeepEval or Ragas) in CI that runs on every prompt, retrieval, model, or orchestration change?
3. Are the metrics archetype-appropriate — not "is it good" but the specific failure mode for that archetype?
4. Are deterministic checks used where possible, with LLM-as-judge reserved for genuine quality judgement against a written rubric?
5. Is a pass threshold defined per metric, and does CI fail the build on regression past it?
6. Is a sample of production traffic scored online (Foundry Evaluations / continuous monitoring) with drift alerting?

## What to read next

- `retrieval-design.md` — retrieval-specific metrics (recall@k, context-precision/recall)
- `model-and-inference-layer.md` — why a model swap is unsafe without an eval
- `patterns/` — each archetype card names the failure mode to measure
- `azure-microservices-observability` — the production telemetry and alerting backbone
