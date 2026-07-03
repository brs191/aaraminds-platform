# Spring Boot Anti-Patterns to Flag in PR Review

Named patterns to scan for when reviewing a Spring Boot 21+ PR. Most are fast to spot. For each: the pattern, detection cue, why it fails, the fix.

## 1. Property-file secrets

**Pattern:** `application.yml` / `application.properties` containing actual secret values.

**Detection cue:** any value in a property file that looks like a password, connection string with `password=`, API key (GUID, base64), or bearer token.

**Why it fails:** secrets in property files get committed to source control, leak via container introspection, and don't rotate. The Spring Cloud Azure Key Vault property source pattern eliminates this by *naming* secrets in property files while resolving values from Key Vault at startup.

**Fix:**
```yaml
spring:
  datasource:
    url: ${POSTGRES_JDBC_URL}        # POSTGRES_JDBC_URL is a Key Vault secret name
```
plus `spring-cloud-azure-starter-keyvault` in `pom.xml`.

## 2. `@Autowired` field injection

**Pattern:**
```java
@Service
public class CustomerService {
    @Autowired
    private CustomerRepository repository;
}
```

**Detection cue:** `@Autowired` on a field declaration.

**Why it fails:** field injection makes the dependency invisible from the constructor, breaks immutability, complicates testing (need reflection or `@InjectMocks`), and hides the cost of cyclic dependencies.

**Fix:** constructor injection with `final` fields:
```java
@Service
public class CustomerService {
    private final CustomerRepository repository;

    public CustomerService(CustomerRepository repository) {
        this.repository = repository;
    }
}
```

Spring auto-wires single-constructor classes; the `@Autowired` annotation is unnecessary.

## 3. Lombok in new code

**Pattern:** `@Data`, `@Getter`, `@Setter`, `@Builder`, `@RequiredArgsConstructor` on new classes.

**Detection cue:** any Lombok annotation in a diff touching new code.

**Why it fails:** Java 21 records cover most of Lombok's value with cleaner syntax and no annotation processor. Lombok introduces build complexity, IDE compatibility friction, and (less commonly) classloader issues. The scaffold forbids Lombok in new services.

**Fix:** use records for data carriers (`public record Customer(String id, String name, Instant createdAt) {}`); use constructor injection (see above) instead of `@RequiredArgsConstructor`.

## 4. `RestTemplate` in new code

**Pattern:** `new RestTemplate()` or `@Bean RestTemplate restTemplate()`.

**Detection cue:** import `org.springframework.web.client.RestTemplate` in a new file.

**Why it fails:** Spring announced `RestTemplate` is in maintenance mode in 5.x; no new features. `RestClient` (Spring 6.1+) provides a modern, fluent, immutable API with the same semantics.

**Fix:** `RestClient.create()` or inject a configured `RestClient` bean. For reactive code, `WebClient`.

## 5. Spring Boot Actuator on the application port

**Pattern:** `management.server.port` not set, or set to the same port as the application (`server.port`).

**Detection cue:** `application.yml` missing `management.server.port`, or set to `${server.port}`.

**Why it fails:** Actuator endpoints (`/actuator/*`) get exposed to the same traffic surface as the application. Health probes, traffic, and management endpoints compete for the same thread pool. Worse, accidentally exposing `/actuator/env` or `/actuator/configprops` to the internet leaks configuration.

**Fix:** separate management port (8081 by default in the scaffold); Container Apps health probe targets the management port; ingress only routes application port to traffic.

## 6. `@Transactional` on private or final methods

**Pattern:**
```java
@Service
public class OrderService {
    @Transactional
    private void processOrder(Order o) { ... }
}
```

**Detection cue:** `@Transactional` on `private`, `final`, `static`, or `protected` methods.

**Why it fails:** Spring's `@Transactional` uses CGLIB or JDK proxies. Proxies only intercept calls via public methods on the bean's public interface. `@Transactional` on private methods *does nothing* — no warning, no error, just silently no transaction. Same for self-invocation (`this.processOrder(o)` from another method in the same class).

**Detection signal in review:** `@Transactional private`, `@Transactional final`, or — harder to spot — a call to a `@Transactional` method from within the same class.

**Fix:** make the method public, or extract to a separate bean if the call is from within the same class.

## 7. Catching `Exception` (or `Throwable`)

**Pattern:**
```java
try {
    repository.save(entity);
} catch (Exception e) {
    log.error("save failed", e);
}
```

**Detection cue:** `catch (Exception ...)` or `catch (Throwable ...)` in business logic.

**Why it fails:** swallows everything including `InterruptedException`, `OutOfMemoryError`, programming errors. Hides bugs and breaks transaction rollback (a swallowed exception inside `@Transactional` may not trigger rollback).

**Fix:** catch specific exceptions and re-throw or transform. If the goal is logging + rethrow, just don't catch — let it propagate to the exception handler.

## 8. `String` concatenation in queries

**Pattern:**
```java
@Query("SELECT c FROM Customer c WHERE c.name = '" + name + "'")
```

**Detection cue:** any `+` in a JPQL/SQL string with user input.

**Why it fails:** SQL injection. Even in JPQL, parameter binding is the only safe pattern.

**Fix:** parameter binding: `@Query("... WHERE c.name = :name")` plus `@Param("name") String name`.

## 9. Public `@RestController` without auth

**Pattern:**
```java
@RestController
@RequestMapping("/v1/orders")
public class OrderController {
    @GetMapping("/{id}")
    public Order getOrder(@PathVariable Long id) { ... }
}
```

…with no Spring Security config matching the path, or no `@PreAuthorize`.

**Detection cue:** new `@RestController` without method-level or class-level security annotation, **and** the Spring Security config doesn't cover the path.

**Why it fails:** unauthenticated access to the endpoint. If Spring Security is configured to permit-all for unknown paths, this is wide open.

**Fix:** `@PreAuthorize("hasAuthority('SCOPE_order.read')")` (or `hasRole(...)`), and verify the security config defaults to deny.

## 10. Mutable static state

**Pattern:**
```java
@Service
public class OrderCache {
    private static final Map<Long, Order> cache = new HashMap<>();
}
```

**Detection cue:** `static` mutable fields in `@Service`, `@Component`, `@Controller` classes.

**Why it fails:** concurrent access without synchronization is a race; concurrent access *with* synchronization defeats Spring's instance-per-bean model and serializes requests. The right shape is an injected dependency (Caffeine, Redis, Cosmos as cache).

**Fix:** inject a cache abstraction (`com.github.benmanes.caffeine.cache.Cache` for in-process, Redis client for distributed).

## 11. `Optional<T>` as a method parameter

**Pattern:** `public Customer findCustomer(Optional<String> idOpt) { ... }`

**Detection cue:** `Optional<...>` as a parameter type (not a return type — returns are fine).

**Why it fails:** `Optional` is for return types ("this may not return a value"). As a parameter, it forces callers to wrap `Optional.of()` or `Optional.empty()`, which is noise. Overloaded methods or null-checking the parameter is cleaner.

**Fix:** two overloaded methods, or nullable parameter with documented contract.

## 12. Returning `null` from `@RestController` methods for "not found"

**Pattern:**
```java
@GetMapping("/{id}")
public Customer getCustomer(@PathVariable Long id) {
    return repository.findById(id).orElse(null);  // Returns 200 OK with empty body
}
```

**Detection cue:** controller method returns `null` for a missing resource.

**Why it fails:** Spring serializes `null` as an empty body with status 200. The caller has no way to distinguish "found but empty" from "not found." This is the HTTP semantics anti-pattern.

**Fix:** `throw new ResponseStatusException(HttpStatus.NOT_FOUND)` or return `ResponseEntity.notFound().build()` or use `Optional<Customer>` return with a `@ResponseStatus` exception handler.

## 13. N+1 queries via JPA lazy loading

**Pattern:**
```java
List<Customer> customers = repository.findAll();
for (Customer c : customers) {
    log.info("orders: {}", c.getOrders().size());  // Triggers a query per customer
}
```

**Detection cue:** loop over a JPA entity collection that accesses a `@OneToMany` or `@ManyToOne` relation; OR an `Open-In-View` warning in logs.

**Why it fails:** classic N+1. 1 query to fetch the list, N queries to fetch each customer's orders. Latency is linear in dataset size.

**Fix:** `@EntityGraph` or `JOIN FETCH` in the repository query; or DTO projection that pulls the needed fields in one query.

## 14. Ignoring `InterruptedException`

**Pattern:**
```java
try {
    Thread.sleep(1000);
} catch (InterruptedException e) {
    // ignored
}
```

**Detection cue:** empty catch on `InterruptedException`.

**Why it fails:** breaks cooperative cancellation. Shutdown signals, request timeouts, virtual-thread interrupts all stop working.

**Fix:** `Thread.currentThread().interrupt();` to re-raise the interrupt, then propagate or break out of the loop. Or simply: catch and re-throw as `RuntimeException` wrapping the original.

## 15. Custom thread pools

**Pattern:** `new ThreadPoolExecutor(...)` or `Executors.newFixedThreadPool(...)` in business code.

**Detection cue:** any direct instantiation of `Executor` / `ExecutorService` outside `@Configuration` classes.

**Why it fails:** thread pools have lifecycle (shutdown), monitoring (pool saturation), and sizing (fixed vs. cached vs. virtual) concerns that are easy to get wrong. Java 21 virtual threads change the default answer.

**Fix:** if the work fits virtual threads (blocking I/O), use `Executors.newVirtualThreadPerTaskExecutor()` in a `@Bean`. If true parallel CPU work, use a sized `@Bean` `ThreadPoolTaskExecutor` with metrics wired to Micrometer.

## How to use this list in review

Scan the diff for these patterns by name; spend more time on the matches than on the rest. Most of these are fast to identify — minutes of review for hours of bug prevention. If a pattern doesn't apply (e.g., the PR doesn't touch concurrency), skip it. The goal is targeted attention, not exhaustive parsing.
