# Call and Dependency Graphs

The call graph and the dependency graph are the structural edges of a code model — what makes "if this changes, what breaks" answerable. This reference covers building them from resolved symbols, the limits of static call resolution, precision versus recall, and the blast-radius query they exist to serve.

## What the graphs are

The **call graph** has an edge from each method to each method it invokes. The **dependency graph** sits a level up — package depends on package, component depends on component — derived from the call graph, the import graph, and dependency injection. Together they answer reachability: forward ("what does this entry point reach"), backward ("what reaches this method" — blast radius), and structural ("where are the cycles and the layering violations").

## Build from resolved symbols, not names

A call-graph edge requires knowing which method declaration a call site binds to — so it is built from the *resolved* AST (`ast-extraction-and-parsing.md`), not from method names. Name-based edges are wrong in both directions: they merge unrelated methods that share a name, and miss calls whose target name differs from the binding. The import graph comes from resolved `import` statements; the injection graph from constructor parameters and `@Autowired` sites resolved to their bean types.

## The hard limit — static resolution cannot see everything

Static analysis resolves what the code states explicitly. It cannot fully resolve:

- **Dynamic dispatch** — a call through an interface or base type can land in any implementor; static analysis sees the *possible* targets, not the runtime one.
- **Reflection** — a method invoked by name through reflection has no static call edge at all.
- **Dependency-injection indirection** — the call goes through an injected interface; the concrete bean is wired by the framework, and resolving it means modeling the DI configuration.
- **Dynamic proxies** — Spring wraps beans for transactions and AOP; the call path runs through generated proxy code (`generated-code-handling.md`).

This is a limit to **state in the model**, not to hide. A blast-radius answer that silently omits reflective or dynamically-dispatched calls is worse than one that marks them as a known gap.

## Precision vs recall — choose deliberately

Two ways to be wrong. **Low recall** — a real call edge is missing; a blast-radius query under-reports and a change looks safer than it is. **Low precision** — a spurious edge is present (every interface implementor treated as a callee); blast radius over-reports and the result is noise. For an impact or blast-radius use case, bias toward recall: a missed edge is a missed risk, and over-reporting is recoverable by review while under-reporting is silent. Record the choice; do not let it be an accident of the parser's defaults.

## Dependency injection — resolve it, because it carries the architecture

In a Spring backend the most architecturally meaningful edges run through DI: a `@Service` constructor-injecting a `@Repository` *is* the component-to-data-tier dependency. Resolve injection edges from constructor parameters and `@Autowired` fields to their bean types, and treat them as first-class structural edges — not skipped because they are "just framework wiring." See `spring-stereotype-modeling.md`.

## Blast radius — the query the graphs exist for

Blast radius is the backward reachability query: from a changed method, walk caller edges transitively to every method, type, component, and endpoint affected. It is the highest-value output of the graphs — it drives change-impact analysis and risk-ranked test scope. Its quality is capped by call-graph recall (above) and, when the graph is large, by the traversal engine — deep variable-length backward traversal is exactly the workload that decides graph-database choice (`azure-data-tier-design`, `references/graph-databases.md`).

## Worked example — the Code Intelligence Factory

The CIF schema models `CALLS` (method to method), `IMPORTS` (package to package), and `INJECTS` (type to type, Spring DI) as deterministic edges, plus `DEPENDS_ON` between inferred components. Its stated highest-value query is blast radius — backward traversal from a changed `Method` — which the QA agent uses to rank test scope by risk. The schema is explicit that the call graph is deterministic while the design-layer `DEPENDS_ON` edges are inferred: the precision/recall and the deterministic/inferred split are decided per edge type, not once for the whole graph.

## Verification questions

1. Are call edges built from resolved symbol bindings, not method-name matching?
2. Are the static-resolution gaps — dynamic dispatch, reflection, DI indirection, proxies — represented in the model rather than silently dropped?
3. Was the precision-versus-recall bias chosen deliberately (recall, for blast-radius use) and recorded?
4. Are dependency-injection edges resolved to bean types and treated as first-class structural edges?
5. Does the model answer backward reachability — blast radius — from any method?

## What to read next

- `ast-extraction-and-parsing.md` — the resolved symbols edges are built from
- `spring-stereotype-modeling.md` — injection and the deterministic design edges
- `generated-code-handling.md` — proxies and generated members in the call path
- `azure-data-tier-design`, `references/graph-databases.md` — traversing the graph at scale
