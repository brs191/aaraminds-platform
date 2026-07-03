# Language Choice — Spring Boot (Java 21+) vs. Go (1.25+)

The pack standardizes on Spring Boot and Go. Picking between them per-service is a real decision, not a stylistic preference. This document is the decision framework.

## The default: Spring Boot for domain-rich; Go for I/O-tight

- **Spring Boot 21+** when the service has rich domain logic, transactional integrity with Spring Data, ecosystem dependencies (Spring Cloud Stream, Spring Cloud Azure, Spring Security), or a Java-comfortable owning team. Spring Boot's strength is composing well-tested libraries that solve enterprise concerns out of the box.
- **Go 1.25+** when the service is mostly I/O orchestration, latency-sensitive (per-request budget < 50 ms), high-concurrency (10k+ concurrent connections), tightly memory-bounded, or sits in a Go-dense neighborhood. Go's strength is predictable resource use, fast startup, small images, and simple deployment.

If neither shoe fits cleanly, default to **Spring Boot** when the owning team is mixed-skill — the ecosystem covers more situations and the learning curve is gentler for an unfamiliar team member. Default to **Go** when the team is uniformly Go-fluent and the workload doesn't pull on Spring's ecosystem.

## The decision matrix

| Driver | Spring Boot | Go |
|---|---|---|
| Heavy domain model with relationships, transactions, rich validation | ✓ | (workable, more boilerplate) |
| Spring ecosystem dependency (Spring Security, Spring Cloud Stream, Spring Data JPA) | ✓ | n/a |
| Tight per-request latency budget (< 50 ms p99) | (achievable but tunable) | ✓ |
| High concurrency I/O (10k+ concurrent connections; reverse proxies, gateways) | (Reactor/WebFlux required, learning curve) | ✓ |
| Small container image (< 30 MB) | (jlink slims to ~80 MB; native via GraalVM is more work) | ✓ (10-20 MB typical) |
| Fast cold start (< 1 s) | (Spring Native compiles ahead, but build complexity rises) | ✓ |
| Memory footprint matters (< 100 MB resident) | (200-400 MB typical with JVM overhead) | ✓ |
| Team is uniformly Java-fluent | ✓ | (re-skill cost) |
| Team is uniformly Go-fluent | (re-skill cost) | ✓ |
| Service handles long-running workflows / sagas | ✓ (with Spring State Machine or workflow lib) | ✓ (with workflow lib; Durable Functions for orchestration) |
| Service must integrate with JVM-only library | ✓ | n/a |
| Service is an MCP server | n/a (this pack standardizes on Go for MCP) | ✓ — see `mcp-go-server-building` |
| Compliance evidence is heavy and a mature JVM APM-agent ecosystem is wanted | ✓ (mature agents) | (newer agent ecosystem; OpenTelemetry is on par for our stack) |

## The anti-decisions

- **"We use both already — pick whichever the team prefers."** Personal preference is the weakest signal. Pick based on the *workload*, then on the *owning team's* current skill, then on consistency with existing services in the same domain.
- **"Go is faster, so use Go."** Sometimes true, often irrelevant. The bottleneck of most microservices is the downstream call, not the in-process compute. If the downstream is a 50 ms Postgres round-trip, the choice between a 0.5 ms and 2 ms in-process processing time doesn't matter.
- **"Spring has more libraries, so use Spring."** Library availability matters when you actually need them. If the service is a simple CRUD on top of one database with no complex domain rules, Spring's library breadth is overprovisioning.
- **"Let's rewrite this Java service in Go for the perf win."** A brownfield rewrite has a 6-12 month time cost; "perf wins" rarely justify it unless the existing service is genuinely struggling. Profile first. If GC pause times are the bottleneck (rare), there's a real argument; if it's "we think Go would be faster," the bias is doing the talking.

## Inside Spring Boot — what we use, what we don't

**Use:**
- Java 21+ (records, sealed types, pattern matching, virtual threads)
- Spring Boot 3.4+ (Spring 6.x, Jakarta EE)
- Spring Data JPA for relational, Spring Data MongoDB for document
- Spring Cloud Azure starters for Key Vault, Service Bus, Cosmos, Storage
- Spring Security (OAuth 2.1 resource server for Entra ID; method-level `@PreAuthorize`)
- Spring Boot Actuator (health, info, metrics, traces — on a separate management port)
- Logback with JSON encoder (Logstash logback encoder or Spring Boot's built-in JSON)
- OpenTelemetry Spring Boot Starter (OTLP exporter)
- Testcontainers for integration tests
- Maven multi-module (preferred over Gradle for new services — simpler to onboard)

**Do not use:**
- Lombok (Java 21 records cover most of the value; Lombok's annotation processing complicates IDE tooling and build pipelines for marginal gain)
- Spring Boot 2.x (out of OSS support; security patches only via paid Tanzu Spring)
- WebFlux unless explicitly justified (most services do not need reactive; the learning cost is real)
- XML configuration (use `@Configuration` classes only)
- `RestTemplate` (use `RestClient` or `WebClient`; `RestTemplate` is in maintenance mode)
- Property-file secrets (`application.properties` with passwords); use Managed Identity + Key Vault via Spring Cloud Azure

## Inside Go — what we use, what we don't

**Use:**
- Go 1.25+ (generics, `log/slog`, structured errors via `errors.Join`)
- Standard library first: `net/http`, `encoding/json`, `database/sql`
- `github.com/jackc/pgx/v5` for Postgres (better than `lib/pq`)
- `go.mongodb.org/mongo-driver/v2` for Mongo
- `github.com/Azure/azure-sdk-for-go/sdk/azidentity` for Managed Identity
- `github.com/Azure/azure-sdk-for-go/sdk/keyvault/...` for Key Vault
- `github.com/mark3labs/mcp-go` for MCP servers (this pack's standardization)
- `go.opentelemetry.io/otel` and `go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp` for tracing
- `log/slog` with JSON handler to stderr (`os.Stderr` for stdio MCP servers)
- Table-driven tests with subtests via `t.Run`
- `github.com/stretchr/testify/require` (sparingly — prefer stdlib assertions where readable)

**Do not use:**
- Heavyweight frameworks (Echo, Gin, Fiber) — `net/http` plus a thin router is sufficient for most services; framework lock-in is real
- `pkg/` directory (use `internal/` for everything; expose via `internal/` package design, not by promoting to `pkg/`)
- `go-kit` (over-abstracted for the size of services we build)
- `github.com/sirupsen/logrus` (use `log/slog` from stdlib)
- `github.com/dgrijalva/jwt-go` (unmaintained; use `github.com/golang-jwt/jwt/v5`)
- ORMs (`gorm`, `sqlboiler`) — use `pgx` and write the SQL; the team that learns SQL writes better Postgres than the team that learns gorm

## Cross-language consistency

Whichever language is picked, certain things stay constant across the estate:

- Container image is multi-stage with a distroless runtime, non-root user, stripped binary/jar
- OpenTelemetry instrumentation with the same resource attributes (`service.name`, `service.namespace`, `deployment.environment`)
- Health probes at `/healthz` (liveness) and `/readyz` (readiness)
- Structured logs to stdout (Spring) or stderr (Go MCP) — never both, never to files
- GitHub Actions OIDC for deploys
- Terraform AzureRM for infrastructure
- Container Apps for compute (unless `azure-service-mapping` justifies AKS)

The point of the standardization is: switching context between a Java service and a Go service should feel like a syntax change, not an architectural shift.

## Verification

Before committing to a language for a new service, the ADR or design doc should answer:

1. What is the workload shape (latency, concurrency, memory)?
2. What is the domain richness (rich Spring Data benefits, or simple CRUD)?
3. What is the owning team's current language fluency?
4. Are there ecosystem dependencies that lock the choice (Spring Cloud Stream, JVM-only library, etc.)?
5. Is this service in a Java neighborhood or a Go neighborhood?
6. What's the rollback if the choice turns out wrong — does the API contract abstract the language well enough that a future re-write is bounded?
