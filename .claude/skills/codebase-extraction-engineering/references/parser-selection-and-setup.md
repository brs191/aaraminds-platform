# Java Parser Selection and Setup

The parser is the extractor's foundation, and the property that decides everything downstream is symbol resolution. This reference covers the realistic Java parser options, what each gives you, and how to set up the resolution the rest of the extractor depends on.

## The choice that caps everything — resolution

A parser does two jobs, and they are not the same. **Parsing** turns source text into a syntax tree — declarations, statements, expressions. **Resolution** binds names in that tree to what they refer to: this `OrderService` identifier is *that* declared type; this method call binds to *that* declaration. A syntax tree alone tells you a call happens; only resolution tells you what it calls. The call graph, the dependency graph, blast radius — all of it is built from bindings. So the parser is chosen for resolution quality first, and a syntax-only parser is disqualified as the core extractor regardless of how fast or convenient it is.

## Eclipse JDT — the strongest resolution

The Eclipse Java Development Tools parser is the most complete option: full type bindings, complete resolution of method invocations, generics, and inheritance, hardened by two decades of use inside a production IDE against real-world Java. It is the reference choice for a serious extractor. The cost is a heavier, lower-level API and the need to configure a name environment and classpath. When resolution accuracy is the priority — and for a code-comprehension product it is — JDT is the safe pick.

## JavaParser with the symbol-solver — the ergonomic default

JavaParser is a clean, well-documented library with a pleasant AST API; its companion symbol-solver adds resolution. The combination resolves types and binds method calls well enough for the large majority of extraction work, with a far gentler API than JDT. It is the pragmatic default — reach for JDT over it only when you hit a resolution case the symbol-solver handles poorly. Configure the symbol-solver with a type-solver per classpath source: the project's own source, its dependency JARs, and the JRE.

## Spoon — when you also transform

Spoon builds a complete, resolved, *transformable* model of a Java program. If the extractor only reads, Spoon's transformation power is unused weight; if you also rewrite code, it is the right tool. For a pure extractor, JDT or JavaParser is lighter.

## tree-sitter — syntax only, know its place

tree-sitter is fast, incremental, and multi-language — and a *syntax* parser with no resolution at all. It cannot bind a call to its target. It is genuinely useful for cheap structural passes — file inventory, rough symbol lists, language detection — and for editor-grade incremental reparsing, but it is never the resolving extractor. Using it as the core extractor is the syntax-only anti-pattern named in the SKILL.md.

## Setting up resolution — the classpath is the work

Resolution needs the same inputs a compiler needs: the project's source roots, every dependency JAR, and the JRE. Assembling that classpath is most of the setup effort, and it is why extraction runs against a *built* checkout (`build-integration-and-generated-code.md`) — the build is what produces the resolved dependency set. Wire it once, in the parse/resolve front end, and hand the rest of the extractor a fully resolved AST so no downstream pass re-does resolution.

## Java version and language level

Set the parser's language level to the target codebase's Java version — a parser told "Java 11" will choke on or mis-parse Java 21 records, sealed types, and pattern switches. Read the version from the build file (`maven.compiler.release`, the Gradle toolchain) rather than guessing, and fail loudly on an unsupported level rather than extracting a silently incomplete model.

## Verification questions

1. Does the chosen parser resolve symbols — type bindings and method-call resolution — not just produce a syntax tree?
2. Is the classpath (project sources, dependency JARs, JRE) assembled and handed to the resolver?
3. Is the parser's language level set from the build file to match the target codebase's Java version?
4. Is resolution done once in a front end, producing a resolved AST the rest of the extractor consumes?
5. Is any syntax-only parser (tree-sitter) confined to genuinely syntax-level passes, never the resolving extractor?

## What to read next

- `extractor-architecture.md` — the pipeline the resolved AST feeds
- `build-integration-and-generated-code.md` — assembling the classpath from a build
- `resolving-the-call-graph.md` — what resolution buys downstream
- `codebase-comprehension`, `references/ast-extraction-and-parsing.md` — the model-design view
