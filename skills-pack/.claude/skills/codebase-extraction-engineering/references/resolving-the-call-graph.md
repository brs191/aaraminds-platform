# Resolving the Call Graph

The call graph is the highest-value and most expensive output of the extractor — it is what makes blast radius answerable. This reference covers building it from resolved bindings, representing what static analysis cannot resolve, the precision/recall knob, and the dependency graph.

## Edges come from bindings, not names

A `CALLS` edge requires knowing which method *declaration* a call site binds to. That is a resolution result (`parser-selection-and-setup.md`), not a name match. Building call edges by matching method names is wrong twice over: it merges unrelated methods that share a name, and it misses calls where the bound target's name differs from the call expression. Take the call edge straight from the resolved invocation's target binding, and key both endpoints by the deterministic ID (`extractor-architecture.md`).

## The unresolvable — represent it, do not drop it

Static resolution cannot fully determine some call targets, and the extractor's job is to *model that uncertainty*, not hide it:

- **Dynamic dispatch** — a call through an interface or base type binds, statically, to a *set* of possible implementors. Emit edges to the candidates, marked as dispatched (not a single definite call), or the model claims a certainty it does not have.
- **Dependency injection** — the call goes through an injected interface; the concrete bean is wired by Spring. Resolve it by modeling the DI configuration (`INJECTS`, below), or mark the call DI-indirect.
- **Reflection** — a method invoked reflectively has no static call edge at all. It cannot be recovered; where you can detect the reflection site, record that the method *is reachable reflectively*.
- **Proxies** — Spring wraps beans for transactions and AOP; calls run through generated proxy code (`build-integration-and-generated-code.md`).

Each is a property to put on the edge or node. A blast-radius query that silently omits dispatched and reflective calls under-reports risk; one that marks them lets the consumer decide.

## Precision vs recall — a code knob

Two failure directions, and the extractor should make the choice deliberate, not accidental. **Recall** loss — a real edge missing — makes blast radius under-report; a change looks safer than it is. **Precision** loss — a spurious edge — makes blast radius noisy. For an impact-analysis product, bias to recall: include every candidate of a dynamic dispatch, because a missed edge is a missed risk while over-reporting is recoverable by review. Make this an explicit configuration of the call-graph pass — recorded — not a side effect of how the resolver happens to behave.

## INJECTS and the dependency graph

In a Spring codebase the architecturally meaningful edges run through dependency injection: a `@Service` constructor-injecting a `@Repository` *is* the component-to-data-tier dependency. Resolve injection from constructor parameters and `@Autowired` sites to their bean types, emit `INJECTS` edges, and derive the component-level dependency graph from `INJECTS` plus `CALLS` plus the import graph. Treat injection as first-class — skipping it as "framework wiring" discards the clearest signal of the architecture.

## Performance — the call graph is the expensive pass

Resolution is the costly part of extraction, and the call-graph pass does the most of it. Practical levers: cache resolution results per compilation unit; parallelize across files — the resolved ASTs are independent — but keep emit deterministic; and on an incremental run, re-resolve only the changed files and their dependents (`incremental-rebuild-and-identity.md`). Measure it — the call-graph pass is where an extractor that "got slow" almost always got slow.

## Verification questions

1. Are call edges built from resolved invocation bindings, never from method-name matching?
2. Are dynamic dispatch, DI indirection, reflection, and proxies represented on the model as properties, not silently dropped?
3. Is the precision-vs-recall bias an explicit, recorded configuration of the call-graph pass — recall, for impact analysis?
4. Are `INJECTS` edges resolved from constructor and `@Autowired` sites and treated as first-class?
5. Is resolution cached and parallelized, with the call-graph pass measured?

## What to read next

- `parser-selection-and-setup.md` — the resolution the call graph consumes
- `extractor-architecture.md` — where the call-graph pass sits
- `codebase-comprehension`, `references/call-and-dependency-graphs.md` — the model-design view
- `data-access-engineering` — querying the call graph for blast radius
