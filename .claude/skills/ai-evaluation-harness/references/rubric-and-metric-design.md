# Rubric and Metric Design

A rubric is the written, criterion-by-criterion definition of what "correct" or "good" output means; a metric is a number that scores one criterion. This reference covers writing a rubric, selecting metrics, setting thresholds, reference-based versus reference-free metrics, the vanity-metric trap, and calibrating a rubric against human judgement.

## "Good" undefined is "good" ungoverned

If the team cannot write down what a good output is, it cannot tell whether the feature produces one — and "looks fine" becomes the acceptance bar, which is no bar. The rubric forces the definition to be explicit and shared *before* the feature is built. It is the hardest and most valuable part of the harness: most of the thinking is in deciding what good means, not in wiring the scorer. The Code Intelligence Factory makes the rubric an M0 deliverable for this reason — its roadmap notes that without a written definition of a "correct architecture map" and a "good HLD," the word "done" has no meaning.

## The rubric — named criteria, each independently scorable

A rubric is not a single "quality" score. It is a set of **named criteria**, each a specific, independently assessable property of the output. For a generated design document the criteria might be: structural accuracy, completeness against the source, traceability of claims to evidence, absence of unsupported inference, and readability. Each criterion is scored on its own, and the per-criterion breakdown is what makes a regression *localizable* — an aggregate score that moves tells you something broke; a per-criterion score tells you what.

Each criterion needs a written description concrete enough that two reviewers applying it to the same output land on the same score. "Is it accurate" is not a criterion. "Every component named in the document corresponds to a real module in the source, and no real module is omitted" is.

## Metric selection — measure the failure mode, not "quality"

A metric is chosen by asking *how does this feature fail* and measuring that — not by reaching for a generic quality number. The failure mode is task-specific: a retrieval feature fails by missing relevant context (measure recall and faithfulness); an extraction feature fails by inventing field values (measure field accuracy and abstention); an agentic feature fails by not completing the task or calling the wrong tool (measure task success and tool correctness). The per-archetype guidance is in `ai-application-architecture`, `references/evaluation.md`; the principle here is that a metric must name the specific way the output goes wrong.

## Thresholds — per criterion, set against the dataset

Each metric needs a pass threshold, and the threshold is per criterion, not one number for the whole feature. Set it against the golden dataset: run the current or a target implementation, look at the distribution of scores, and place the threshold where a real regression would cross it but normal run-to-run variation would not. Record the threshold and the reasoning; move it only deliberately. A threshold pulled from the air — "90% feels right" — gates nothing meaningfully.

## Reference-based vs reference-free metrics

A **reference-based** metric compares the output to a known-correct expectation in the golden dataset — field accuracy against an expected value, similarity to a golden document. A **reference-free** metric scores a property of the output with no reference — is the JSON well-formed, does every citation resolve to a real source, is the answer internally consistent. Reference-free metrics are doubly valuable because they also work on production traffic that has no golden answer (`online-eval-and-drift.md`). Use both: reference-based for correctness against the dataset, reference-free for properties you can check anywhere.

## Vanity metrics — the aggregate-score trap

A vanity metric looks like rigour and provides none. The signs: a single aggregate "eval score" with no per-criterion breakdown; a metric computed only over easy cases; a metric nobody can connect to a real failure mode; a number on a dashboard that no gate depends on. The fix is the discipline above — a per-criterion rubric, coverage including hard cases (`golden-datasets-and-fixtures.md`), metrics tied to named failure modes, and thresholds that gate. A metric that cannot fail is not measuring anything.

## Calibrating the rubric against human judgement

The rubric is itself a hypothesis about what good means, and it can be wrong. Calibrate it: have a qualified human score a sample of outputs, and check that the rubric applied mechanically agrees with the human. Where they diverge, either the criterion is underspecified or the human is catching something the rubric misses — fix the rubric until they agree. This calibration is also the prerequisite for trusting an LLM-as-judge, which must in turn agree with the calibrated rubric (`scoring-methods.md`).

## Verification questions

1. Is "good output" written down as named, independently scorable criteria — not a single undefined "quality"?
2. Is each criterion described concretely enough that two reviewers would score it the same way?
3. Does each metric measure a specific, named failure mode of this feature?
4. Is there a per-criterion pass threshold, set against the dataset's score distribution and recorded with its reasoning?
5. Are reference-free metrics included, so some scoring also works on production traffic?
6. Has the rubric been calibrated against human judgement on a sample?

## What to read next

- `golden-datasets-and-fixtures.md` — the inputs the rubric is applied to
- `scoring-methods.md` — turning rubric criteria into deterministic checks or judge prompts
- `ci-gating-and-baselines.md` — how thresholds become gates
- `ai-application-architecture`, `references/evaluation.md` — per-archetype metric guidance
