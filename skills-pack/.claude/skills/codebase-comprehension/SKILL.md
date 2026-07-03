---
name: codebase-comprehension
description: Designs the static-analysis pipeline that turns an existing codebase into a queryable structural model — the code knowledge-graph schema/ontology, AST extraction, call/dependency graphs, Spring-stereotype-aware modeling, generated-code handling, and incremental rebuilds. Primary target: Java Spring Boot. Use when building a comprehension capability, designing the graph schema, extracting a model from source, or modeling an annotation-driven framework. Do not use for PR review (pr-review-azure-microservices), reviewing a modeled architecture (microservices-architecture-reviewer), or the AI layer over the model (ai-application-architecture).
version: 1.1.0
last_updated: 2026-05-30
---

# Codebase Comprehension

## When to use

Trigger this skill when the task is to extract a structural model *from* an existing codebase — build a code-comprehension capability, model an undocumented system from source, design the AST or call-graph extraction, decide how to model an annotation-driven framework, or handle generated code in static analysis. Common triggers: "model this legacy service from the repo," "build the call graph and dependency graph," "how do we extract the architecture from code," "how do we use the Spring annotations," "Lombok-generated methods are missing from the AST," "how do we re-model the codebase on every change."

Do **not** use this skill for: human PR review of code changes (`pr-review-azure-microservices`); reviewing an architecture once it is modeled (`microservices-architecture-reviewer`); the AI/LLM layer that consumes or narrates the model (`ai-application-architecture`); or where the extracted model is stored — graph engine choice (`azure-data-tier-design`, `references/graph-databases.md`).

Primary target ecosystem: a **Java Spring Boot backend**. The principles generalize; the framework-specific depth is Spring.

## The critical decision rule — extract deterministically, infer separately, never blur them

A code model holds two kinds of fact, and the single most important rule is to keep them apart. A **deterministic fact** is read straight from the source — this type exists, this method calls that method, this class carries `@Service`. It is ground truth; confidence is total. An **inferred fact** is a judgement laid on top — this cluster of classes is "the payments component," this is the system's "ordering capability." It is a hypothesis; confidence is partial and it can be wrong.

Tag every fact in the model with which kind it is. A model that presents an inference with the same authority as a parsed fact produces confident fiction — and the entire value of a comprehension product is that its users can trust what it tells them. Extract everything deterministic first; infer on top of it, separately and visibly, never blended in.

## What a code model contains — layers

| Layer | Content | Provenance |
|---|---|---|
| Code | Repository, module, package, file, type, method, endpoint | Deterministic — parsed from source |
| Structural relationships | Calls, imports, inheritance, interface implementation, dependency injection | Deterministic — resolved from the AST |
| Design | Components and boundaries, data flows, external integrations | Inferred — *or* deterministic when annotation-derived |
| Requirement / capability | Business capabilities, requirements, actors | Inferred — and the weakest; needs non-code sources |

The lower layers are evidence; the upper layers are interpretation. The provenance column is the rule above made structural.

## The graph schema is the M0 deliverable

The node and edge taxonomy, the property and identity scheme, and how provenance and versioning are modeled are the *first* thing to settle — the graph is the system-of-record, and the write path, the traversals, and GraphRAG retrieval are all defined against it. Design the schema before the extractor writes a node; a wrong schema makes every downstream layer wrong. `references/graph-schema-and-ontology.md`.

## The extraction pipeline — build the thin slice first

The pipeline is Discover (repo scope, entry points) → Parse (source to AST) → Index (symbols, references) → Map (relationships, design). Do **not** build all four as polished stages before anything works end to end. Build the thinnest path through all four, for one real repository, first — a walking skeleton that produces a crude but genuine model — then deepen each stage. A perfected parser with no mapping stage models nothing; the end-to-end slice is what proves the approach.

## Framework-aware extraction beats generic parsing

A generic parser sees classes and methods. A Spring-aware extractor sees a `@RestController` as an API boundary, a `@Service` as a component, a `@Repository` as a data-access edge to a store, a constructor-injected dependency as a component edge. Annotations encode architectural intent *explicitly* — and exploiting them moves a large part of the Design layer from inferred to deterministic. This is the highest-leverage decision in the pipeline: in an annotation-driven framework, refusing to read the annotations is choosing to guess what the framework already told you. See `references/spring-stereotype-modeling.md`.

## The generated-code problem

Lombok, MapStruct, and other annotation processors generate methods — getters, builders, mappers — that are not in the source AST. An extractor that parses source text only will miss those members and every call edge that runs through them, silently. The model looks complete and is not. Either run extraction against post-annotation-processing output, or model the generated members explicitly from the processor's known behaviour. This is decided before the parser is written, not discovered after. See `references/generated-code-handling.md`.

## Static structure, not runtime behavior

This skill models a codebase's *structure* — what exists and what references what — from source. It does not model *execution*: real runtime paths, actual call frequencies, production data values, latency. State that boundary, because two failure modes follow from ignoring it. First, a static call edge is *possibility*, not observation — `methodA` may call `methodB` on an edge production never exercises; never present static reachability as proof a path runs. Second, do not wait for runtime data to begin — static analysis works from a repository clone on day one, while runtime evidence needs the target system's observability piped in, a heavy integration and a real trust ask. Build the static model first; treat runtime evidence as a later, separate layer that *annotates* the static model, never a prerequisite for it. The Code Intelligence Factory makes exactly this call — static analysis for v1 and v2, the runtime evidence layer deferred to v3.

## Worked example — brownfield: an undocumented Spring Boot service

Setup: a legacy Java Spring Boot service, original authors gone, no design docs. The task is a trustworthy structural model a team can act on.

Decision walk: (1) Discover — clone the repo, find the modules and entry points. (2) Parse — source to AST; extract the deterministic code layer (types, methods with overload-aware signatures, endpoints), every fact `confidence = 1.0`. (3) Resolve — call edges, imports, inheritance, constructor injection. (4) Read the Spring stereotypes — `@RestController`/`@Service`/`@Repository`/`@Entity` give a deterministic Design layer rather than a guessed one. (5) Cluster the un-annotated remainder into *inferred* components, tagged as inference, not blended with (4). (6) Handle Lombok/MapStruct so generated members and the calls through them are not missing. (7) Stamp the model with the commit SHA and assign stable IDs so the next build diffs cleanly. The model is now queryable — an HLD, a dependency map, a blast-radius answer all fall out as views over it.

The wrong move is to skip parsing and "summarize the repo" with text search and an LLM — see the anti-pattern.

## Anti-pattern — grep-and-guess "analysis"

**Bad:** "analyze the codebase" is implemented as text search across files plus an LLM summary, with no parsed model underneath. **Why it fails:** grep has no notion of scope, type resolution, method overloading, or call edges; it cannot tell a definition from a mention or a comment from code. The "analysis" is ungrounded — no claim traces to a resolved source fact — and unverifiable. **Detection signal:** there is no AST and no resolved symbol table; output claims cannot be traced to a specific declaration or call site; renamed identifiers silently break the "model." **Fix:** build the deterministic parsed model first (`references/ast-extraction-and-parsing.md`); every higher-level statement sits on it and traces back to it.

## Verification questions

1. Is every fact in the model tagged deterministic or inferred, with the two never blended?
2. Was a thin end-to-end slice (Discover → Parse → Index → Map) built before any one stage was deepened?
3. Does extraction resolve symbols — types, overload-aware method signatures, call edges — rather than text-matching?
4. Are framework annotations (Spring stereotypes, request mappings, injection) read as first-class deterministic signal?
5. Is compiler-generated code (Lombok, MapStruct) handled — extraction runs post-processing, or generated members are modeled explicitly?
6. Does every entity get a stable, deterministic ID so the model diffs cleanly across rebuilds?
7. Can every higher-level claim be traced back to a specific source location?
8. Is the graph schema — node/edge taxonomy, identity, provenance, versioning — explicitly designed, not improvised by the extractor?

## What to read next

Tier-2 references: `references/graph-schema-and-ontology.md` · `references/ast-extraction-and-parsing.md` · `references/call-and-dependency-graphs.md` · `references/spring-stereotype-modeling.md` · `references/generated-code-handling.md` · `references/incremental-code-modeling.md`.

Related skills: `microservices-architecture-reviewer` (reviews an architecture; consumes a model like this) · `ai-application-architecture` (the AI layer that narrates the model; its `references/retrieval-design.md` covers graph-RAG over a code model) · `azure-data-tier-design` (`references/graph-databases.md` — where the model is stored and traversed) · `pr-review-azure-microservices` (human review of code changes).
