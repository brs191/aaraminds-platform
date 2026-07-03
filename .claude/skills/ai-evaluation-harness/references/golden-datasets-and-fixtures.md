# Golden Datasets and Reference Fixtures

A golden dataset is the specification of an AI feature made executable: representative inputs, each paired with what a correct response is. It is the artifact every other part of the harness depends on — the scorers score against it, the gate gates on it, the rubric is applied to it. This reference covers what a golden item contains, the reference-fixture pattern for large inputs, coverage, sourcing, versioning, size, and maintenance.

## The golden dataset is the specification

Before a golden dataset exists, "the feature works" is an opinion. After it exists, it is a measurement. The dataset defines what the feature must do — which makes writing it a design act, not a testing chore, and it belongs *before* the build (see the skill's critical decision rule). It is also reusable, durable work: the same dataset is the spec, the regression guard, the thing a model swap is measured against, and the basis of the CI gate. Treat it as a first-class asset of the feature, version-controlled beside the code.

## What a golden item contains

Each item is an input plus an expectation. The expectation takes one of three shapes, and which one the task allows drives the scoring method (`scoring-methods.md`):

- **Exact** — one correct output: a classification label, an extracted value, a normalized string. Scored by equality.
- **Acceptable set** — several outputs are correct. Scored by membership.
- **Rubric** — open-ended output with no single right answer; a written rubric defines what good looks like. Scored by a judge or a human.

Prefer exact and set expectations wherever the task allows — they are cheap and reproducible to score. Use rubric expectations only for genuinely open-ended output.

## Reference fixtures — when the input is large

For some features the "input" is too large to inline in a dataset row — a whole repository, a long document, a full conversation. The pattern is a **reference fixture**: a real, fixed, version-pinned artifact chosen once and kept permanently, paired with a hand-built golden output for it.

The Code Intelligence Factory is the clean example. Its M0 deliverable is a real, mid-size open-source Java Spring Boot repository chosen as the permanent test fixture, plus a hand-written *golden HLD* for it. That pairing is simultaneously the v1 spec and the evaluation target — "is the generated HLD as good as the golden one." A reference fixture costs more to build than a dataset row, but for a feature whose input is a large artifact it is the only honest way to have a golden expectation at all.

## Coverage — the easy-cases trap

A golden dataset of only easy, typical inputs passes at a high score and tells you almost nothing. Coverage is what makes the score mean something. Deliberately include:

- **Typical cases** — the common path, in rough proportion to real traffic.
- **Hard cases** — ambiguous inputs, inputs with missing or conflicting data, the long tail.
- **Adversarial cases** — inputs designed to trip the feature: injection-shaped content, edge formats, inputs where the correct answer is "I cannot answer this."
- **Abstention cases** — inputs where the right output is *no output* or an explicit "unknown." A feature that invents an answer rather than abstaining fails here, and only an abstention case catches it.

The score on a covered dataset is informative; the score on an easy one is a vanity metric (`rubric-and-metric-design.md`).

## Sourcing — build it from real inputs and real failures

Synthetic inputs invented at a desk drift from reality. Build the golden dataset from real production traffic where it exists, and from the feature's known failure reports — every bug a user found is a golden case that must never regress. For a feature not yet in production, draw on real artifacts of the kind it will process, not invented ones. Each new production failure is added to the dataset as it is found, so the dataset grows toward the real input distribution over time.

## Versioning

The golden dataset is version-controlled with the code, and every change to it is reviewed like code — adding, removing, or relabelling a golden item changes the spec. An eval run records which dataset version it ran against, so a score is always interpretable. When the dataset changes, the baseline (`ci-gating-and-baselines.md`) is recaptured against the new version.

## Size — enough for signal, not more

The dataset must be large enough that a score is statistically meaningful and small enough that the eval runs fast and cheap in CI (`ci-gating-and-baselines.md`). There is no universal number — it is set by how many distinct case classes the coverage above demands. Favour full coverage of the case classes over a large count of near-duplicate typical cases; a hundred well-chosen cases beat a thousand variations of the same easy one.

## Maintenance — a living artifact

A golden dataset is never finished. It grows from production failures, it is pruned when items go stale, and it is re-reviewed when the feature's intended behaviour changes. A dataset that has not changed in months on a feature that *has* shipped changes is a warning sign — either the feature is frozen or the dataset has stopped reflecting it.

## Verification questions

1. Is each golden item an input paired with an explicit expectation — exact, acceptable-set, or rubric?
2. For large-artifact inputs, is there a version-pinned reference fixture with a hand-built golden output?
3. Does coverage include hard, adversarial, and abstention cases — not only typical ones?
4. Is the dataset sourced from real inputs and real failure reports rather than desk-invented examples?
5. Is the dataset version-controlled, reviewed on change, and is each eval run stamped with the dataset version?
6. Is the dataset maintained — growing from new production failures, pruned of stale items?

## What to read next

- `rubric-and-metric-design.md` — defining the expectation for open-ended items
- `scoring-methods.md` — how each expectation shape is scored
- `ci-gating-and-baselines.md` — baselines and the dataset-version stamp
- `online-eval-and-drift.md` — turning production failures into new golden cases
