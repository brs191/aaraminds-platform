# Pattern: Batch / Async LLM Processing

## Problem

A large volume of items each needs model processing, but no single item is latency-sensitive — classify a year of tickets, summarize every method in a repository, embed an entire corpus. Running these through a synchronous online endpoint wastes money on online pricing and risks rate-limit exhaustion that degrades the interactive path. Batch processing decouples the work.

## Use When

- The item count is high and each item is independent.
- The work is latency-tolerant — minutes to hours to completion is acceptable.
- The work is offline: indexing, backfills, bulk extraction, evaluation runs.
- Cost matters and the workload can wait — the Azure OpenAI Batch API trades latency for roughly half the per-token cost.

## Avoid When

- Any item is user-facing and latency-sensitive → an online archetype.
- Volume is low enough that an online loop is simpler and the cost difference is noise.
- Items have ordering or cross-item dependencies — that is a workflow, not a batch.

## Shape

A Python job, not a request handler. Items flow off a queue (Service Bus or Storage Queue) or are submitted to the Azure OpenAI Batch API; results land in storage or a database. Run it on Azure Container Apps jobs — scheduled or event-driven — not the always-on serving tiers. Every item must be idempotent and individually retryable; a poison item goes to a dead-letter queue, it does not fail the batch. Stamp the batch with the input version (a repo commit, a dataset id) so results are reproducible.

## Trade-offs

| Aspect | Trade-off |
|---|---|
| Throughput and cost | The Batch API is roughly half the price and isolated from online quota contention |
| Latency | Minutes to hours — unusable for interactive work |
| Operational shape | A job and a queue, not an always-on endpoint |
| Partial failure | Must be designed for — at this volume, some items will fail |

## Common Failure Modes

- **All-or-nothing partial failure** — one bad item aborts thousands of good ones. Detection: per-item success / failure accounting. Prevention: isolate failures per item, dead-letter the poison ones, report a completion summary.
- **Non-idempotent items** — a retry double-writes or double-charges. Detection: re-running the batch changes results. Prevention: key every item by a stable id and check "already done" before processing.
- **Silent rate-limit degradation** — the batch competes with online traffic for the same quota. Detection: online latency rises whenever a batch runs. Prevention: a separate deployment / quota for batch, or the Batch API, which is quota-isolated.
- **Stale results** — the batch ran against an old corpus version. Detection: results carry no input-version stamp. Prevention: stamp every batch with the source version.

## Decision Signals

Use batch when volume is high and latency does not matter. If even one consumer needs the result *now*, that path is online — split it out; do not make interactive users wait on a batch.

## Worked signal — Code Intelligence Factory

The CIF's indexing — Discover → Parse → Index → Map over a whole repository — is batch work. Most of it is deterministic (AST extraction, no model). The model-touched parts — method-purpose summaries, component-role inference — should go through the Batch API: they are high-volume, offline, and latency-tolerant, and the commit SHA is the natural batch version stamp the regeneration diff already depends on.

## References

- `azure-data-tier-design` — where batch results land
- Pattern: `single-shot.md` — the per-item transform
- `../model-and-inference-layer.md` — Batch API vs online, quota isolation
- `../evaluation.md` — evaluation runs are themselves batch jobs
