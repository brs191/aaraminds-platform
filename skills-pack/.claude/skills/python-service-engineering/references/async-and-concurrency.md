# Async and Concurrency

This reference covers `async`/`await` in a Python service: when async earns its place, the event-loop discipline it demands, structured and bounded concurrency, and when a synchronous service is the better choice.

## Async fits an I/O-bound AI tier

A service whose time is spent *waiting* — on model calls, retrieval queries, tool invocations, other services — is I/O-bound, and that is exactly what `async`/`await` is for: while one request waits on a model call, the event loop runs another. The AI orchestration tier is I/O-bound almost by definition, so async is usually the right model there. The benefit is concurrency without a thread per request.

## The event-loop discipline — one blocking call stalls everything

Async buys concurrency only if nothing blocks the event loop. A single synchronous, blocking call — a `requests` call instead of an async HTTP client, a synchronous database driver, a CPU-heavy loop, `time.sleep` — freezes the *entire* service for its duration, because there is one loop and that call is sitting on it. This is the defining hazard of async Python. The discipline: every I/O call on the async path uses an async library, and any unavoidable blocking or CPU-bound work is pushed off the loop with `run_in_executor` (a thread or process pool). A service that is "async" but calls a blocking SDK has the costs of async and few of the benefits.

## Structured concurrency

When a request fans out — several model calls, several retrievals — run them concurrently with structured concurrency: a task group that starts the tasks, waits for all of them, and propagates failures and cancellation as a unit, rather than loose `create_task` calls whose lifetimes and errors are untracked. Structured concurrency makes "all of these, concurrently, and clean up correctly on any failure" a single construct instead of manual bookkeeping.

## Bounded concurrency

Fan-out must be *bounded*. Firing 500 model calls concurrently because 500 items need processing will hit rate limits, exhaust connections, and blow cost and memory. Cap concurrency explicitly — a semaphore, a worker-pool pattern, or a batching layer — sized to the model's rate limit and the service's resources. Unbounded fan-out is the async equivalent of the agentic-loop cost runaway: the failure is not in any one call but in how many run at once.

## When not to use async

Async has a real cost: it colours the codebase (`async` propagates up every caller), it is harder to debug, and every dependency must have an async-compatible path. Do not pay it for a service that is not I/O-bound-concurrent. A CPU-bound service (heavy parsing, computation) wants processes, not an event loop. A simple, low-traffic, synchronous request-response service is simpler and perfectly fine sync. Choose async when the workload is concurrent I/O; choose sync when it is not, and record the choice.

## Verification questions

1. Is async used because the workload is I/O-bound and concurrent — a deliberate choice, not a default?
2. Is every I/O call on the async path an async library, with blocking or CPU-bound work pushed off the loop via an executor?
3. Is request fan-out run with structured concurrency (a task group), not loose untracked tasks?
4. Is fan-out concurrency bounded — a semaphore or worker pool sized to rate limits and resources?
5. For a CPU-bound or simple synchronous workload, was sync (or processes) chosen instead of async?

## What to read next

- `orchestration-code.md` — async in the orchestration framework
- `ai-application-architecture`, `references/model-and-inference-layer.md` — model-call rate limits and the Batch API
- `project-structure-and-packaging.md` — structuring the service
- `test-engineering` — testing async code
