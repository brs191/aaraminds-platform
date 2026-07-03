# Spring Stereotype Modeling

In an annotation-driven framework, the framework has already told you the architecture — the job is to read it. This reference covers exploiting Spring Boot's stereotype and mapping annotations to make a large part of the design model deterministic instead of inferred.

## Annotations are architectural intent, made explicit

Spring's stereotype annotations are not decoration — they declare a type's architectural role. `@RestController` says "this is an API boundary." `@Service` says "this is a business-logic component." `@Repository` says "this is a data-access component, an edge to a store." `@Entity` says "this is a persistent domain type." `@Configuration` says "this is wiring." In a plain language those roles must be *inferred* from structure and guessed; in Spring they are *stated*. Reading them is the highest-leverage move in comprehending a Spring codebase — exploit the annotations, do not re-derive what they already declare.

## The deterministic-design payoff

This is what the stereotype move buys. A design-layer fact that is normally inferred — "these classes form the orders component" — becomes deterministic when it is annotation-derived. A `@Service`-annotated type maps to a component of kind `service` with full confidence; a `@RestController` to an API component; a JPA `@Repository` to a data-access component with an edge to its datastore. Per the skill's critical decision rule, those annotation-derived facts are tagged `deterministic`; only the un-annotated remainder is `inferred`. The more of the codebase Spring annotates, the more of the design model is fact rather than hypothesis.

## The signals to extract

- **Stereotypes** — `@RestController`, `@Controller`, `@Service`, `@Repository`, `@Component`, `@Configuration` on types → component role.
- **Request mappings** — `@RequestMapping`, `@GetMapping` / `@PostMapping` and the rest, on methods → HTTP endpoints, with method, path, consumes / produces.
- **Persistence** — `@Entity`, `@Table`, and JPA repository interfaces → the domain model and the data-access edges.
- **Injection** — constructor parameters and `@Autowired` → component dependency edges (`call-and-dependency-graphs.md`).
- **Configuration** — `@Configuration` classes and `application.yml` / `application.properties` → external integrations and datastores; the property files are the primary source for discovering which databases and external systems the service talks to.
- **Boundaries** — `@SpringBootApplication` marks the application entry point and the component-scan root.

## Sliced tests are a signal too

Spring's sliced test annotations — `@WebMvcTest`, `@DataJpaTest`, `@SpringBootTest` — declare *what layer a test exercises*. A `@DataJpaTest` is a repository-layer test; a `@WebMvcTest` a controller-layer test. Extracting the test slice classifies test coverage by architectural layer for free, which feeds risk-ranked test-scope analysis.

## The limit — annotations are not the whole story

Annotation-derived facts are deterministic, but they are not complete. Un-annotated plain classes still exist and still need clustering into inferred components. Custom or meta-annotations — a project's own annotation composed from Spring ones — must be resolved to their Spring meaning. And an annotation states intent, not correctness: `@Service` says "this was meant as a service," not "this class is well designed." Read the annotations as deterministic *role* signal; keep architectural *judgement* as inference on top.

## Worked example — the Code Intelligence Factory

The CIF schema makes this its design principle: "Spring annotations are exploited, not ignored." It captures `stereotype` as a first-class property on `Type` and models `Endpoint` as a node, both read straight from annotations and both `deterministic`. The schema is explicit that a `@Service` type becomes a `Component` of kind `service` with `provenance = deterministic`, while a cluster of un-annotated types also becomes a `Component` but `inferred` — the two kept distinct, never blurred. The `annotation` and `config` evidence kinds exist precisely so an inferred fact can be backed by the `@Service` site or an `application.yml` entry. This is what makes the CIF's Design layer higher-confidence on Spring than it could be on a language without stereotypes.

## Verification questions

1. Are Spring stereotypes read off types and mapped to component roles as `deterministic` facts?
2. Are request-mapping annotations extracted into endpoints with method, path, and content types?
3. Are `application.yml` / `application.properties` mined for datastores and external integrations?
4. Are custom and meta-annotations resolved to their underlying Spring meaning?
5. Are annotation-derived design facts tagged `deterministic` and un-annotated-cluster facts tagged `inferred`, never merged?
6. Are sliced-test annotations used to classify test coverage by layer?

## What to read next

- `ast-extraction-and-parsing.md` — extracting annotations off the AST
- `call-and-dependency-graphs.md` — dependency-injection edges
- `generated-code-handling.md` — Spring's own proxy generation
- `microservices-architecture-reviewer` — reviewing the design once it is modeled
