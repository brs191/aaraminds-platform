# Online Evaluation and Drift Detection

Offline evaluation proves a feature is good against known inputs. Production sends inputs the golden set never had, and quality can slide for reasons the offline suite cannot see. This reference covers sampling production traffic, scoring without a golden answer, drift detection, the feedback loop back into the golden dataset, and alerting.

## Offline proves known inputs; production sends unknown ones

The offline harness gates change against a fixed golden dataset. Its blind spot is the input it has never seen — and production is mostly that. Real traffic shifts: users ask new things, upstream data changes, a dependency's behaviour moves. Quality can also slide with nothing in your repository changing — a model endpoint updated underneath you, retrieval-corpus drift. Online evaluation is the instrument for everything offline cannot see. It does not replace the offline gate; it covers the other half.

## Sampling production traffic

Online eval scores a sample of live requests, not all of them — scoring every request roughly doubles model cost. Sample a percentage, weighted toward the cases that matter: low-confidence outputs, new input shapes, high-stakes paths. The sampling rate is a cost-versus-signal decision — too low and rare failures never reach statistical signal, too high and the eval cost rivals the feature itself. Calibrate it against the first weeks of real data.

## Scoring without a golden answer

Production inputs have no golden expectation, so reference-based metrics do not apply. What works online:

- **Reference-free metrics** (`rubric-and-metric-design.md`) — schema validity, citation resolution, internal consistency, groundedness against the retrieved context.
- **LLM-as-judge against the rubric** — pointwise quality scoring that needs no reference (`scoring-methods.md`).
- **Implicit user signals** — corrections, retries, abandonment, thumbs-down, downstream acceptance or rejection.
- **The human approval gate**, where the feature has one — a reviewer rejecting a generated artifact is a high-quality online label, captured for free.

## Drift detection

Drift is a metric sliding over time rather than failing outright. Track the online metrics on a rolling window and alert when one moves past a threshold relative to its recent baseline — not when a single request scores low, which is noise, but when the *distribution* shifts. Drift is the early warning a static offline suite structurally cannot give, because the offline dataset does not move.

## The feedback loop — production failures become golden cases

Online eval is not only monitoring; it is the supply line for the offline harness. Every production case that scores low, every user correction, every rejected artifact is a candidate new golden item (`golden-datasets-and-fixtures.md`). Folding them back in means the offline gate grows toward the real input distribution, and the same failure cannot regress unnoticed twice. A harness without this loop has a golden dataset frozen at its launch-day guess of what production would look like.

## Alerting and ownership

An online eval signal needs a named owner and a defined response, or a drift alert is one more ignored dashboard. Route drift alerts like any production alert (`azure-microservices-observability`), with a severity and an owner. The response to drift is investigation — which input shape, which metric, which change — feeding the worst cases into the golden dataset and, if a real regression is confirmed, back through the offline gate.

## Verification questions

1. Is a sample of production traffic scored online — not only the offline golden dataset?
2. Is the sampling rate a deliberate cost-versus-signal decision, weighted toward cases that matter?
3. Does online scoring use reference-free metrics, judge scoring, and implicit user signals — given there is no golden answer?
4. Is drift detected on a rolling window — a distribution shift — rather than alerting on single low-scoring requests?
5. Do low-scoring production cases and user corrections feed back into the golden dataset?
6. Does a drift alert have a named owner and a defined response?

## What to read next

- `golden-datasets-and-fixtures.md` — where production failures feed back to
- `rubric-and-metric-design.md` — reference-free metrics that work online
- `ci-gating-and-baselines.md` — the offline half of the picture
- `azure-microservices-observability` — production telemetry and alerting
