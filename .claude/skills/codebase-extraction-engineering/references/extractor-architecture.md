# Extractor Architecture

This reference covers how to structure the extractor as a program — the pass pipeline, the visitor passes, where identity generation lives, the extractor's own language, and how it emits to the graph.

## The extractor is a pipeline: parse → resolve → walk → emit

Structure the extractor as distinct stages, not one monolithic walk:

1. **Parse + resolve** — the front end (`parser-selection-and-setup.md`) produces a fully resolved AST per compilation unit.
2. **Walk** — one or more visitor passes traverse the resolved AST and extract entities and edges.
3. **Emit** — the extracted nodes and edges are written to the graph.

Separating these means the resolver can be swapped, the passes are independently testable, and emit can be batched or streamed without touching extraction logic. A single tangled walk that parses, resolves, decides, and writes in one loop is the structure that becomes unmaintainable first.

## The visitor passes

The walk is naturally more than one pass, because edges depend on nodes already existing. A workable split: a **declaration pass** emits the code-layer entities (types, methods, fields, endpoints) and the containment and inheritance edges; then a **reference pass** emits the edges between them (calls, injections, type uses) now that every node it might point at exists. The resolution-heavy call-graph work is its own pass (`resolving-the-call-graph.md`). Keep each pass single-purpose — a pass that does everything is as hard to test as the monolith.

## Where deterministic ID generation lives

Every pass refers to entities by a stable ID, so ID generation is one shared component, not logic copied per pass. It implements `codebase-comprehension`'s identity scheme — hash the natural key: fully-qualified type name; type plus method name plus *ordered parameter-type list*, because overloading makes the bare name ambiguous. One module, one function, called everywhere. If two passes hash keys differently, edges bind to the wrong nodes and the bug is maddening to find.

## The extractor's own language

The codebase under analysis is Java; the extractor need not be. Three shapes:

- **A JVM extractor** — the extractor is itself Java or Kotlin, using JDT or Spoon directly. Tightest parser integration, no process boundary.
- **Go or Python driving a JVM parser** — the extractor is Go or Python (the pack's tool and orchestration tiers) and runs the Java parser as a subprocess or a small JVM service that returns a resolved model. Keeps the extractor on-stack with the rest of the platform; pays a serialization boundary.
- **A polyglot resolve-service** — a JVM service exposes the resolved model over an API; the extractor consumes it.

Decide by where the extractor sits in the platform. If it is one component of a Go/Python system, driving a JVM parser keeps the stack coherent; if extraction is the whole product, a JVM extractor is simplest. Record the choice — it is not cheap to reverse.

## Emit: streaming, not one giant transaction

A large repository produces a large model. Emit incrementally — stream nodes and edges to the graph in batches as passes complete, rather than building the whole model in memory and writing it once. Batched emit bounds memory and lets a failed run resume. Keep emit deterministic (the SKILL.md's determinism rule): collect, sort by stable ID, then write, so two runs of the same commit produce the same write order. The graph write path itself is `data-access-engineering`.

## Errors and partial extraction

A real repository will not always fully resolve — a missing dependency, a generated source the build did not produce, a parse error in one file. The extractor must degrade, not abort: extract what resolves, mark what did not, and record the gap as a fact in the model rather than emitting a silently incomplete graph that looks complete. A file that failed to parse is a known unknown; a file silently skipped is a lie the model tells.

## Verification questions

1. Is the extractor staged — parse/resolve, walk, emit — rather than one tangled loop?
2. Are the walk passes single-purpose (declarations before references) and independently testable?
3. Is deterministic ID generation a single shared component used by every pass?
4. Is the extractor's language an explicit, recorded decision given where it sits in the platform?
5. Does emit stream in batches with a deterministic order, rather than one in-memory build?
6. On partial failure, does the extractor extract what it can and record the gaps as facts?

## What to read next

- `parser-selection-and-setup.md` — the front end this pipeline starts with
- `resolving-the-call-graph.md` — the resolution-heavy pass
- `incremental-rebuild-and-identity.md` — making the pipeline incremental
- `data-access-engineering` — the graph write path emit targets
