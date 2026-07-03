---
name: codebase-extraction-engineering
description: Implements the static-analysis extractor that turns a Java codebase into a structural model — parser choice with symbol resolution, the AST-to-graph pipeline, call/dependency resolution, build integration for generated code (Lombok, MapStruct), node embedding generation for GraphRAG, and incremental rebuilds. The build-side companion to codebase-comprehension. Use when implementing an extraction pipeline, choosing a Java parser, wiring symbol resolution, generating node embeddings, or building incremental re-extraction. Do not use for designing the model (codebase-comprehension), storing/traversing the graph (data-access-engineering), or AI features (ai-application-architecture).
version: 1.1.0
last_updated: 2026-05-30
---

# Codebase Extraction Engineering

## When to use

Trigger this skill when the task is to *build* the extractor — the program that reads a Java codebase and emits the structural model. Common triggers: "which Java parser do we use," "how do we wire symbol resolution," "how is the extractor code structured," "the call graph is missing edges," "Lombok getters aren't showing up," "how do we re-extract only what changed."

This is the implementation companion to `codebase-comprehension`. That skill decides *what* the model contains — the layers, the provenance rule, the identity scheme. This skill builds the program that produces it. Use them together: design with `codebase-comprehension`, implement with this.

Do **not** use this skill for: designing the code model itself (`codebase-comprehension`); storing or traversing the extracted graph (`azure-data-tier-design` for the engine, `data-access-engineering` for the queries); the AI features built on the model (`ai-application-architecture`). Primary target: extracting a **Java Spring Boot** codebase.

## The critical decision rule — the parser choice is the extractor's quality ceiling

The single decision everything else rests on is the parser, and the property that matters is **symbol resolution**. A parser that produces a syntax tree but does not resolve names gives you structure without meaning — you can see *that* there is a method call, not *what method it binds to*. Every downstream fact — the call graph, the dependency graph, blast radius, the whole product — inherits the resolution quality of the parser. So: pick the parser first, pick it for resolution, and accept the consequence that follows — **extraction runs against a buildable checkout with dependencies on the classpath, not a pile of loose `.java` files.** A team that picks a fast syntax-only parser to avoid the build dependency has capped the product's accuracy before writing another line.

## The Java parser landscape

| Parser | Symbol resolution | Use when |
|---|---|---|
| **Eclipse JDT** | Full type bindings, the most complete | You want the strongest resolution and can accept a heavier API; the reference choice for a serious Java extractor |
| **JavaParser** (+ symbol-solver) | Good, via the companion symbol-solver | You want an ergonomic API and resolution that is good enough for most extraction |
| **Spoon** | Full, with a rich transformable model | You also need to *transform* code, or want a high-level model; heavier |
| **tree-sitter** | None — syntax only | Fast, incremental, multi-language *syntax*; use for cheap structural passes, never as the resolving extractor |

The honest default for a Java extractor is **Eclipse JDT or JavaParser-with-symbol-solver**. tree-sitter is a real tool but it is a syntax parser — it has no notion of what a name binds to, so it cannot build a correct call graph. Choosing it for the core extractor is the parser version of the anti-pattern below. Parser depth is `references/parser-selection-and-setup.md`.

## Extractor architecture

The extractor is a pipeline, not a script: parse → resolve → walk → emit. Structure it as distinct passes — a parse/resolve front end producing a resolved AST, one or more visitor passes that extract entities and edges, and an emit stage that writes graph nodes and edges. Keep deterministic ID generation (`codebase-comprehension`'s identity scheme) in one place so every pass hashes keys the same way. The extractor's own language is a real choice — Go and Python can drive a Java parser as a subprocess or via a service, or the extractor is itself a JVM program using JDT/Spoon directly; the trade-offs are in `references/extractor-architecture.md`.

## Build integration is not optional

Because resolution needs the classpath, the extractor runs against a *built* project: it invokes (or consumes the output of) Maven or Gradle, assembles the dependency classpath, and analyses post-annotation-processing output so Lombok, MapStruct, and generated members are visible. It also runs against *untrusted* source — a repository under analysis is attacker-influenceable — so the build and parse happen sandboxed (network-denied, resource-capped). Build integration and generated code: `references/build-integration-and-generated-code.md`.

## Incremental extraction

A codebase changes constantly; re-parsing all of it on every commit is waste. An incremental extractor detects changed files, re-extracts them plus the entities whose resolution depends on them, and re-runs the affected slice of the graph and the inference — re-using stable deterministic IDs so the rebuild diffs cleanly. A full rebuild is reserved for when the extractor logic itself changes. Implementation in `references/incremental-rebuild-and-identity.md`.

## Determinism — same commit in, same model out

The extractor must be deterministic: the same commit produces a byte-identical model on every run. The regeneration diff (`codebase-comprehension`) is meaningless otherwise — non-determinism shows up as spurious changes that bury the real ones. The sources to control: unordered iteration (map and set order is not stable — sort by stable ID before emit), wall-clock values leaking into IDs or properties (stamp the build with the commit, never `now()`), and parallelism races in emit order (parallelize extraction but make emit collect-sort-write). Determinism is also what makes the extractor testable against a golden fixture — a pinned repo commit must yield a known model — so verify it as `test-engineering` and `ai-evaluation-harness` describe.

## Embedding generation

GraphRAG anchors on vector similarity, and those node embeddings are produced *here*, at emit time, written with the node at one `buildVersion` so the graph and its vectors never drift. Embed the node's semantic text (signature, name, doc) with a pinned Azure OpenAI embedding model — not raw source bytes — and re-embed only changed nodes (content-hash and cache). Generating embeddings at query time is slow and non-reproducible. `references/embedding-generation.md`.

## Beyond Java — design for more than one extractor

The first extractor targets Java, but a comprehension product rarely stays single-language — the CIF roadmap adds a React frontend and a Postgres schema in v2. Design so a second language is an *addition, not a rewrite*: keep the language-specific part (the parser, the resolver, the AST walk) behind a clean boundary, and keep the language-agnostic part (deterministic ID generation, the graph emit path, the incremental-rebuild machinery, the model schema) shared. A new language is then a new front end feeding the same emit and identity layers. The mistake is letting Java specifics — JDT types, Maven assumptions — leak through the whole extractor; that turns the second language into a second extractor.

## Worked example — brownfield: a grep-based "extractor" rebuilt around resolution

Setup: a first-cut "code model" is built by regex over `.java` files — it finds class and method *names* and guesses calls by matching method names. It works in a demo and is wrong in production: it merges overloaded methods, binds calls to the wrong target, and misses every call through an interface.

Decision walk: (1) Accept that the regex approach has no recoverable accuracy — scope, type, and binding cannot be pattern-matched; this is a rebuild, not a patch. (2) Adopt a resolving parser — Eclipse JDT or JavaParser-with-symbol-solver — and make extraction run against a Maven/Gradle build so the classpath is available. (3) Restructure into passes: resolve front end → entity visitor → edge visitor → emit. (4) Build the call graph from resolved bindings, representing the static-resolution gaps (dynamic dispatch, DI, reflection) explicitly rather than dropping them. (5) Run the build's annotation processors so Lombok and MapStruct members exist in the model. (6) Add change-detection so subsequent runs are incremental. The model is now correct and rebuildable.

The wrong move is to "improve the regexes." Resolution is not a quality you can approximate with patterns.

## Anti-pattern — the syntax-only extractor

**Bad:** the extractor uses a syntax-only parser (tree-sitter, or a hand-rolled regex pass) and never resolves symbols — chosen because it is fast and needs no build. **Why it fails:** without resolution the extractor cannot bind a call to its target, tell two overloads apart, or follow a call through an interface; the call graph is wrong in ways nothing downstream can detect. **Detection signal:** there is no classpath assembly and no build step; the extractor runs on loose files; call edges are matched by method name; overloaded methods collapse into one node. **Fix:** adopt a resolving parser and run against a built checkout — `references/parser-selection-and-setup.md`. Use a syntax-only parser only for genuinely syntax-level passes, never as the resolving extractor.

## Verification questions

1. Does the extractor use a resolving parser (Eclipse JDT or JavaParser-with-symbol-solver), not a syntax-only parser or regex?
2. Does extraction run against a buildable checkout with the dependency classpath assembled?
3. Is the extractor structured as distinct passes — parse/resolve, extract, emit — with deterministic ID generation in one place?
4. Are call edges built from resolved bindings, and are the static-resolution gaps represented rather than silently dropped?
5. Does the build step run annotation processors so Lombok/MapStruct members are in the model?
6. Does the extractor parse and build untrusted repositories in a sandbox (network-denied, resource-capped)?
7. Is re-extraction incremental — changed files plus their resolution dependents — with stable IDs across rebuilds?
8. Are node embeddings generated at emit time (not query time), from semantic text, with a pinned model and incremental re-embedding, at the node's `buildVersion`?

## What to read next

Tier-2 references: `references/parser-selection-and-setup.md` · `references/extractor-architecture.md` · `references/resolving-the-call-graph.md` · `references/build-integration-and-generated-code.md` · `references/incremental-rebuild-and-identity.md` · `references/embedding-generation.md`.

Related skills: `codebase-comprehension` (designs the model this extractor populates — read it first) · `data-access-engineering` (writing the extracted graph and querying it) · `azure-data-tier-design` (the graph engine choice) · `mcp-go-server-building` (if the extractor is exposed as an MCP tool).
