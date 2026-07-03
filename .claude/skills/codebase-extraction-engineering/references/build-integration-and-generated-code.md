# Build Integration and Generated Code

The extractor cannot work from loose source files — resolution needs the classpath, and a large part of the running program is generated at build time. This reference covers driving the build, assembling the classpath, handling annotation-processor output, and sandboxing the untrusted repository.

## Resolution needs a built project

Symbol resolution needs the project's dependencies on the classpath, and the only reliable source of the exact dependency set is the project's own build. So the extractor either invokes the build (`mvn` / `gradle`) or consumes its output. There is no shortcut — "just parse the `src/` directory" leaves every library type unresolved, which means no call graph into framework code. The buildable-checkout requirement (`parser-selection-and-setup.md`) is a consequence of resolution, not a preference.

## Driving Maven and Gradle

Two integration points. **Dependency resolution** — get the classpath: `mvn dependency:build-classpath` or the Gradle equivalent emits the resolved dependency paths the parser's type-solver needs. **Compilation** — run the build's `compile` so annotation processors run and generated sources and classes exist. Detect the build tool from the repo (`pom.xml` vs `build.gradle`), and handle the common failures explicitly: a build that needs credentials for a private repository, a build that needs a specific JDK, a build that simply fails. A repo whose build fails yields a partial model — extract what resolves and record the gap (`extractor-architecture.md`), do not abort.

## Annotation processors — run them, or miss a layer

Lombok, MapStruct, and other annotation processors generate methods that are not in the source AST (`codebase-comprehension`, `references/generated-code-handling.md`). The implementation consequence: the extractor must analyze **post-annotation-processing** output — the generated sources the build produces, or the compiled bytecode — not the raw `.java` files. In practice, run the build's `compile` and point the extractor at the generated-sources directory and the compiled classes, so a Lombok `@Getter`'s `getX()` and a MapStruct mapper implementation are real nodes with real call edges. An extractor that reads only `src/main/java` has a silent hole exactly at construction, mapping, and transaction boundaries.

## Classpath assembly

Assemble three things for the resolver: the project's own source roots (including the generated-sources directory), the dependency JARs from the build, and the JDK matching the project's Java version. Completeness matters — a missing JAR turns every type from that library into an unresolved symbol. Validate the assembled classpath before extraction and fail loudly on gaps rather than producing a quietly under-resolved model.

## Sandboxing — the repository is untrusted

The repository under analysis is attacker-influenceable content, and the extractor *builds* it — which runs the repo's own build scripts, plugins, and annotation processors as code. That is arbitrary code execution by definition. Run the build and the extraction in a sandbox: network-denied (or tightly allowlisted to the dependency repository only), filesystem-isolated, resource-capped (CPU, memory, wall-clock, output size), and disposable per run. A build script that exfiltrates secrets or fork-bombs is not a hypothetical for a product whose entire input is "someone else's repository." This is the boundary discipline `ai-application-architecture`'s safety reference and `mcp-go-guardrails-and-safety` apply, at the build layer.

## Verification questions

1. Does extraction run against a built project, with the dependency classpath taken from the project's own build?
2. Does the extractor detect the build tool and handle build failure by extracting a partial model with the gap recorded?
3. Does extraction run against post-annotation-processing output, so Lombok/MapStruct members and their edges exist?
4. Is the classpath (sources + generated sources + dependency JARs + JDK) validated before extraction?
5. Does the build-and-extract run in a sandbox — network-denied, resource-capped, disposable — because it executes the untrusted repo's build?

## What to read next

- `parser-selection-and-setup.md` — why resolution needs the classpath
- `extractor-architecture.md` — handling partial extraction
- `codebase-comprehension`, `references/generated-code-handling.md` — the model-design view of generated code
- `ai-application-architecture`, `references/safety.md` — the untrusted-input boundary
