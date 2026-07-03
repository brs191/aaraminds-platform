# AST Extraction and Parsing

The abstract syntax tree is the deterministic ground truth of a code model — the layer everything else sits on. This reference covers parsing source into an AST, resolving symbols, what to extract, parser choice for Java, and the identity rules that let the model survive a rebuild.

## The AST is ground truth

Parse; do not pattern-match. A regex or text search over source has no concept of scope, type, or declaration-versus-reference — it cannot tell a method definition from a call from a string literal that happens to contain the name. The AST is the parsed, structured form of the code, and facts read from it are deterministic, confidence 1.0. Every higher layer of the model — call graph, design, requirements — is built on the AST and traces back to it. If AST extraction is wrong, the whole model is wrong; this is the stage to get exactly right.

## Parse, then resolve — two distinct steps

Parsing produces a syntax tree per file: the structure of declarations and statements. **Resolution** binds names to what they refer to — this `OrderService` token is *that* declared type; this call binds to *that* method declaration. A parser alone gives you syntax, not "method A calls method B," because the callee is a name that must be resolved. Symbol resolution needs the whole compilation unit and its dependencies on the classpath — which is why extraction runs against a *buildable* checkout, not a pile of loose files. Resolved symbols are the difference between a real model and a guess.

## What to extract — the code layer

From the resolved AST, extract the structural entities: repository, module (Maven or Gradle), package, file, type (class / interface / enum / record / annotation, with visibility and modifiers), method (with its full signature), and — for a web backend — HTTP endpoints. Extract the structural edges in the same pass: declaration and containment, inheritance, interface implementation, and the raw material for the call graph. All of it deterministic. Capture a source location — file and line span — on every entity: the location is what makes a model claim verifiable and what anchors evidence later.

## Identity — names are not unique, signatures are

A model entity needs a stable identifier, and the natural key is its fully-qualified name — with one Java-specific subtlety that bites every extractor that ignores it. **Java permits method overloading**, so a method's name alone is not unique within a type. The identity of a method is its type, its name, *and its ordered parameter-type list*. Get this wrong and two overloads collapse into one node, or call edges bind to the wrong method. Types are keyed by fully-qualified name (nested types qualified by the enclosing type); methods and test methods by type plus name plus ordered parameter types; endpoints by HTTP method plus path. Deterministic IDs hashed from these natural keys make the model rebuildable and diffable (`incremental-code-modeling.md`).

## Parser choice for Java

Use a real Java parser with symbol resolution — one that resolves types against the classpath — not a hand-rolled regex pass and not a syntax-only parser. The non-negotiable capabilities: full syntax support for the target codebase's Java version, type resolution against dependencies, and overload-aware method binding. Whether the parser is the compiler's own front end or a dedicated library is a tooling choice; that it resolves symbols is not.

## Worked example — the Code Intelligence Factory code layer

The CIF's knowledge-graph schema makes its Code layer entirely deterministic, `confidence = 1.0`: `Repository`, `Module`, `Package`, `File`, `Type`, `Method`, `Endpoint`, extracted straight from the Java Spring Boot repository. Its identity rules are exactly the ones above — types by fully-qualified name, methods by type plus name plus ordered parameter-type list, because overloading makes the bare name ambiguous. That deterministic code layer is the foundation the inferred Design layer and the generated documents are built on, and the schema's governing principle is that deterministic and inferred facts never share a confidence band.

## Verification questions

1. Is the code parsed to a resolved AST — not pattern-matched with regex or text search?
2. Is symbol resolution run against a buildable checkout with dependencies on the classpath, so call targets bind?
3. Does every method's identity include its ordered parameter-type list, so overloads do not collapse?
4. Is a source location captured on every entity, so model claims are verifiable?
5. Are entity IDs deterministic hashes of natural keys, stable across rebuilds?

## What to read next

- `call-and-dependency-graphs.md` — the edges built on resolved symbols
- `spring-stereotype-modeling.md` — reading framework annotations off the AST
- `generated-code-handling.md` — why source-only parsing misses members
- `incremental-code-modeling.md` — stable IDs and rebuild diffs
