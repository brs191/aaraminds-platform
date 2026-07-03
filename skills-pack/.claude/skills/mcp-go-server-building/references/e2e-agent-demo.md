# Skill — MCP-Go End-to-End Agent Demo

## Purpose

Design an end-to-end demo that exercises an MCP server from a real client over the real protocol with realistic inputs and verifiable outputs. The demo is both a *verification artifact* (proof that the server works) and a *teaching artifact* (a runnable walkthrough new contributors can follow). This skill is about what makes a demo authentic versus theatrical, and how to keep one honest as the server evolves.

## Authentic vs. theatrical demos

| Authentic | Theatrical |
|---|---|
| The demo client speaks the real MCP protocol | The demo "calls" the server through a wrapper that bypasses the wire |
| Inputs vary; outputs vary in response | Inputs are ignored; outputs are hardcoded |
| Outputs are produced by the server's actual code paths | Outputs are crafted to look impressive but unrelated to inputs |
| The demo fails when the server has a regression | The demo always passes because outputs are constants |
| Goldens regenerate from current code; drift is visible | "Demo data" is hand-curated and decoupled from reality |

This pack's reference demo lives at `demo/architecture-review-demo/`: a stdlib-only Python MCP client over stdio that drives the Go server across three deliberately distinct architectures and captures real outputs as goldens — the authentic shape, not a hardcoded fixture.

## Anatomy of an authentic demo

### 1. Real client → server transport

The demo client speaks the MCP wire — JSON-RPC over stdio or HTTP. No shortcuts. The pack's reference implements stdio in stdlib-only Python (`demo_runner.py`), about 250 lines.

### 2. Realistic inputs

Inputs reflect actual systems someone would design. The v9.0 demo ships three: e-commerce order platform (PCI-DSS), retail banking (event-sourced ledger, SOX), HIPAA patient platform (PHI, audit). Each is a master JSON with ~10 services, declared constraints, NFRs.

Choose inputs that *differ meaningfully* — patterns that surface in one but not another, risks that vary, Azure mappings that diverge. A demo with three near-identical inputs proves nothing.

### 3. Outputs that vary by input

Because the server is deterministic and rule-based, the same input always produces the same output (byte-reproducible). Different inputs produce materially different outputs. The v9.0 demo verifies this implicitly: the boundary scores across three architectures are 84/100/77, risk counts 6/3/2 — variance is the proof of authenticity.

### 4. Per-tool, per-architecture goldens

For each (architecture, tool) pair, capture the output JSON. Store under `golden/<architecture>/<tool>.json`. A validator compares freshly generated outputs against goldens.

### 5. Validator with canonical comparison

The validator normalises JSON (sort keys, fixed indentation) before comparing. This isolates real changes from incidental formatting differences. Failures are reported per file with a specific mismatch reason.

### 6. Refresh procedure

When goldens drift intentionally (a tool's rule logic changed), regenerate:

```
make demo refresh
```

Then commit the new goldens with a rationale in the PR. Goldens should never silently change.

### 7. Makefile orchestration

```
make demo       # runs the demo against the built server
make validate   # compares outputs to goldens
make refresh    # promotes outputs to goldens (manual, deliberate)
make clean      # removes generated outputs
```

Familiar surface; one-command workflows.

## When the demo should fail

A failing demo means *something has changed*. The reasons are limited:

1. A tool's rule logic changed (intentional). Refresh goldens and commit.
2. A tool's input schema changed (intentional). Update inputs in `input/<arch>.json` to match.
3. The demo runner has a bug (rare; the runner is small and stable).
4. The MCP server changed in a way that broke the protocol exchange.

Each requires investigation; don't blindly refresh. A culture of "refresh and ship" undoes the demo's value.

## What the demo is and is not

It is:
- Evidence that the server works end-to-end.
- A reproducible artifact: same code, same goldens, every time.
- A walkthrough for new contributors.

It is not:
- A benchmark of agent capability — the demo is rule-based, not LLM-driven.
- A substitute for unit tests — they catch issues at finer granularity.
- An exhaustive test of every tool combination — it covers the major paths, not every edge case.

## Building a demo for a new server

1. **Define 2–4 architectures.** Pick deliberately distinct ones. E-commerce + financial + healthcare is a good template.
2. **Write master input JSONs.** One per architecture, with the superset of fields all tools will need.
3. **Implement the runner.** Use the pack's `demo_runner.py` as a template if Python; mirror in Go or another language if needed. The runner:
   - Spawns the server.
   - Initializes the MCP connection.
   - For each architecture, shapes per-tool inputs and calls each tool.
   - Writes outputs to a per-architecture directory.
4. **Run it once, capture goldens.** Move outputs to `golden/`.
5. **Implement the validator.** Canonical JSON comparison.
6. **Wire into CI.** Nightly or pre-release, not per-PR (the demo is moderately slow).

## Common failure modes

- **Hardcoded outputs masquerading as a demo.** The "demo" returns constants and never fails. Detection: outputs identical across inputs. Fix: drive the demo from the actual server.
- **Goldens too fragile.** Every formatting change breaks the demo. Detection: validator failures over whitespace. Fix: canonical JSON comparison (sort keys, fixed indent).
- **Single architecture demo.** One input proves the happy path only. Detection: regressions in edge cases pass the demo. Fix: ≥3 architectures with materially different shapes.
- **Demo without refresh procedure.** Drift means rebuilding goldens by hand, painful, skipped. Detection: stale goldens. Fix: `make refresh` is one command and documented.
- **Demo runner that bypasses the wire.** A Python wrapper that imports the Go service directly via FFI. Detection: protocol bugs don't surface. Fix: real subprocess, real MCP exchange.

## Verification questions

1. Does the demo actually call the server over the MCP protocol?
2. Do outputs vary meaningfully across architectures?
3. Is the validator deterministic (same outputs → same result)?
4. Is the refresh procedure one command with a documented rationale?
5. Does the demo fail when a tool's rule logic regresses?

## What to read next

- `demo/architecture-review-demo/README.md` — the pack's reference demo
- `client-integration.md` — the client side of the demo
- `../../mcp-go-production-review/references/testing.md` — where unit tests pick up where the demo leaves off
- `../../mcp-go-production-review/references/cicd-quality-gates.md` — wiring the demo into CI
