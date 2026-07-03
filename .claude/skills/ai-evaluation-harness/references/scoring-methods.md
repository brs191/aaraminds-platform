# Scoring Methods

A scorer turns an output and its expectation into a number. There are two kinds — deterministic checks and LLM-as-judge — and choosing correctly between them is most of what makes a harness trustworthy. This reference covers when to use each, how to build a judge you can trust, pointwise versus pairwise scoring, and the role of human evaluation.

## Two kinds of scorer

A **deterministic check** is code: it compares, validates, parses, or executes. Given the same output it returns the same score, always. An **LLM-as-judge** is a model call that scores an output against a rubric; it handles open-ended quality a deterministic check cannot express — but it is itself a non-deterministic model, with all the unreliability that implies. The rule that follows from this asymmetry: **deterministic first, judge only where you must.**

## Deterministic checks — use them wherever you can

Wherever a correctness criterion can be expressed as code, express it as code. Schema validity, exact-value or set membership, a citation resolving to a real source, a referenced entity existing, a numeric value in range, a generated test compiling and passing, two runs being self-consistent — all deterministic. They are cheap, fast, reproducible, and — critically — not themselves subject to model non-determinism, so they do not import a second source of noise into the harness. A surprising amount of "AI output quality" reduces to deterministic checks once the rubric criteria are made concrete enough (`rubric-and-metric-design.md`). Exhaust them before reaching for a judge.

## LLM-as-judge — for open-ended quality only

Some criteria genuinely cannot be coded: "is this explanation clear," "does this design document read coherently," "is this summary faithful to the source." For these, an LLM scores the output against the rubric. The judge is necessary and powerful. It is also a model call — non-deterministic, biased in known ways (toward longer answers, toward its own style, toward the first option shown), and capable of being confidently wrong. A judge used naively is a second opinion of unknown quality dressed up as a metric.

## Building a judge you can trust

A judge is a component that must itself be engineered and validated:

- **Rubric-anchored** — the judge scores against the explicit, per-criterion rubric, not a vague "rate this 1–10." It is given the criterion description and asked to apply it.
- **Calibrated against human labels** — the judge's scores are checked against a human-scored sample; it is trusted only on criteria where it agrees with the human. Where it does not, that criterion goes back to a human or a deterministic proxy.
- **Given its own eval** — the judge has a golden set too: outputs with known-correct scores, so judge drift on a model upgrade is caught.
- **Controlled for known bias** — randomize option order for pairwise judging; control for length; do not let the judge see which output is the "reference."
- **A capable model** — judging is a hard task; the judge should generally be a frontier model even when the feature under test runs on a smaller one.

An LLM-as-judge that has not been calibrated is not a measurement.

## Pointwise vs pairwise

**Pointwise** scoring rates one output in isolation against the rubric — necessary for an absolute gate ("is this good enough to ship"). **Pairwise** scoring asks which of two outputs is better — more reliable from a judge, because relative judgement is easier than absolute, and the natural fit for comparing a change against the baseline ("is the new output better than the old"). Use pairwise to detect regressions and improvements between versions, and pointwise for the absolute threshold. They answer different questions; a mature harness uses both.

## Human evaluation — the calibration anchor

Human evaluation is too slow and expensive to gate CI, but it is the anchor everything else is calibrated to: the rubric is calibrated against humans, and the judge is calibrated against humans. Budget a periodic human-scored sample — not to gate, but to keep the automated scorers honest. When the judge and the humans diverge, the humans are the ground truth and the judge is what gets fixed.

## Verification questions

1. Is every criterion that *can* be a deterministic check implemented as one, with the judge reserved for genuinely open-ended quality?
2. Is the LLM-as-judge rubric-anchored — scoring named criteria, not a vague overall rating?
3. Has the judge been calibrated against a human-scored sample, and is it trusted only on criteria where it agrees?
4. Does the judge have its own golden set, so judge drift on a model upgrade is caught?
5. Are known judge biases controlled — option order randomized, length controlled?
6. Is pairwise scoring used for regression-vs-baseline and pointwise for the absolute gate?
7. Is there a periodic human-scored sample anchoring the automated scorers?

## What to read next

- `rubric-and-metric-design.md` — the rubric the judge and checks score against
- `golden-datasets-and-fixtures.md` — the expectations scorers compare to
- `ci-gating-and-baselines.md` — how scores become a gate
- `online-eval-and-drift.md` — reference-free scoring on production traffic
