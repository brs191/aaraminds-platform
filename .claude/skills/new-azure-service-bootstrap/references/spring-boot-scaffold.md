# Spring Boot Scaffold (Java 21+, Spring Boot 3.4+)

The standard scaffold for a new Spring Boot service in the Azure estate. Copy this layout, rename, customize where the domain genuinely requires it — do not invent your own layout.

## Repository layout

```
<service-name>/
├── README.md
├── pom.xml                              # Parent POM
├── Makefile                             # Standard targets
├── Dockerfile                           # Multi-stage, distroless
├── .github/workflows/
│   ├── pr.yml                           # PR-gate workflow
│   ├── main.yml                         # Post-merge build + staging deploy
│   └── release.yml                      # Production deploy (manual approval)
├── infra/
│   ├── main.tf                          # Container Apps, identities, RBAC
│   ├── variables.tf
│   ├── outputs.tf
│   └── modules/                         # If service has its own modules
├── domain/
│   ├── pom.xml
│   └── src/main/java/<group>/<svc>/domain/
│       ├── model/                       # Aggregates, value objects, domain events
│       ├── service/                     # Domain services
│       └── repository/                  # Repository interfaces (not implementations)
├── infrastructure/
│   ├── pom.xml
│   └── src/main/java/<group>/<svc>/infrastructure/
│       ├── persistence/                 # Spring Data JPA / Mongo implementations
│       ├── messaging/                   # Service Bus consumers / producers
│       └── external/                    # HTTP clients for downstream services
├── app/
│   ├── pom.xml
│   └── src/
│       ├── main/
│       │   ├── java/<group>/<svc>/app/
│       │   │   ├── Application.java     # Main class
│       │   │   ├── config/              # Spring @Configuration classes
│       │   │   ├── web/                 # Controllers, exception handlers
│       │   │   └── security/            # Spring Security config
│       │   └── resources/
│       │       ├── application.yml      # Non-secret config only
│       │       ├── application-prod.yml
│       │       └── logback-spring.xml   # JSON logging
│       └── test/                        # Tests (unit + slice + integration)
└── docs/
    ├── adr/                             # Architecture Decision Records
    ├── runbook.md
    └── slo.md
```

## Parent `pom.xml` (key parts)

```xml
<?xml version="1.0" encoding="UTF-8"?>
<project xmlns="http://maven.apache.org/POM/4.0.0">
  <modelVersion>4.0.0</modelVersion>

  <parent>
    <groupId>org.springframework.boot</groupId>
    <artifactId>spring-boot-starter-parent</artifactId>
    <version>3.4.0</version>
    <relativePath/>
  </parent>

  <groupId>com.aaraminds.<svc></groupId>
  <artifactId><svc>-parent</artifactId>
  <version>1.0.0-SNAPSHOT</version>
  <packaging>pom</packaging>

  <modules>
    <module>domain</module>
    <module>infrastructure</module>
    <module>app</module>
  </modules>

  <properties>
    <java.version>21</java.version>
    <spring-cloud-azure.version>5.18.0</spring-cloud-azure.version>
    <opentelemetry.version>1.42.0</opentelemetry.version>
  </properties>

  <dependencyManagement>
    <dependencies>
      <dependency>
        <groupId>com.azure.spring</groupId>
        <artifactId>spring-cloud-azure-dependencies</artifactId>
        <version>${spring-cloud-azure.version}</version>
        <type>pom</type>
        <scope>import</scope>
      </dependency>
    </dependencies>
  </dependencyManagement>
</project>
```

Re-verify versions quarterly per `../../mcp-go-server-building/references/ecosystem-facts.md`.

## `app/pom.xml` essential dependencies

```xml
<dependencies>
  <!-- Web + Actuator -->
  <dependency>
    <groupId>org.springframework.boot</groupId>
    <artifactId>spring-boot-starter-web</artifactId>
  </dependency>
  <dependency>
    <groupId>org.springframework.boot</groupId>
    <artifactId>spring-boot-starter-actuator</artifactId>
  </dependency>

  <!-- Spring Security with OAuth 2.1 (Entra ID) -->
  <dependency>
    <groupId>org.springframework.boot</groupId>
    <artifactId>spring-boot-starter-oauth2-resource-server</artifactId>
  </dependency>

  <!-- Identity + Key Vault via Managed Identity -->
  <dependency>
    <groupId>com.azure.spring</groupId>
    <artifactId>spring-cloud-azure-starter-keyvault</artifactId>
  </dependency>
  <dependency>
    <groupId>com.azure.spring</groupId>
    <artifactId>spring-cloud-azure-starter-active-directory</artifactId>
  </dependency>

  <!-- Persistence (pick one set; not both) -->
  <dependency>
    <groupId>org.springframework.boot</groupId>
    <artifactId>spring-boot-starter-data-jpa</artifactId>
  </dependency>
  <dependency>
    <groupId>org.postgresql</groupId>
    <artifactId>postgresql</artifactId>
  </dependency>

  <!-- Messaging (only when used) -->
  <dependency>
    <groupId>com.azure.spring</groupId>
    <artifactId>spring-cloud-azure-starter-servicebus</artifactId>
  </dependency>

  <!-- Observability -->
  <dependency>
    <groupId>io.opentelemetry.instrumentation</groupId>
    <artifactId>opentelemetry-spring-boot-starter</artifactId>
    <version>${opentelemetry.version}-alpha</version>
  </dependency>

  <!-- JSON logging -->
  <dependency>
    <groupId>net.logstash.logback</groupId>
    <artifactId>logstash-logback-encoder</artifactId>
    <version>7.4</version>
  </dependency>

  <!-- Tests -->
  <dependency>
    <groupId>org.springframework.boot</groupId>
    <artifactId>spring-boot-starter-test</artifactId>
    <scope>test</scope>
  </dependency>
  <dependency>
    <groupId>org.testcontainers</groupId>
    <artifactId>postgresql</artifactId>
    <scope>test</scope>
  </dependency>
</dependencies>
```

## `Application.java`

```java
package com.aaraminds.<svc>.app;

import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;

@SpringBootApplication
public class Application {
    public static void main(String[] args) {
        SpringApplication.run(Application.class, args);
    }
}
```

That's it. The main class does nothing else. `@Configuration` classes wire beans; controllers handle requests. Do not stuff initialization into `main`.

## `application.yml` (non-secret config)

```yaml
spring:
  application:
    name: <svc>
  cloud:
    azure:
      keyvault:
        secret:
          property-source-enabled: true
          endpoint: ${AZURE_KEY_VAULT_ENDPOINT}
      credential:
        managed-identity-enabled: true
  datasource:
    url: ${POSTGRES_JDBC_URL}                  # Resolved from Key Vault at startup
    username: ${POSTGRES_USERNAME}             # Entra ID identity name
    hikari:
      maximum-pool-size: 10

management:
  server:
    port: 8081                                  # Separate from application port (8080)
  endpoints:
    web:
      exposure:
        include: health,info,prometheus,metrics
  endpoint:
    health:
      probes:
        enabled: true                           # /actuator/health/liveness, /readiness

server:
  port: 8080
  shutdown: graceful

logging:
  config: classpath:logback-spring.xml
```

Two things to note:

- Management port (`8081`) is separate from application port (`8080`). Container Apps probes hit `8081`; real traffic hits `8080`. This isolates probes from traffic load and avoids exposing actuator endpoints to the world.
- `POSTGRES_JDBC_URL` and `POSTGRES_USERNAME` are *Key Vault keys*, resolved at startup via the Spring Cloud Azure Key Vault property source. No `application.properties` with the actual values.

## `logback-spring.xml` — JSON logs to stdout

```xml
<?xml version="1.0" encoding="UTF-8"?>
<configuration>
  <appender name="JSON" class="ch.qos.logback.core.ConsoleAppender">
    <encoder class="net.logstash.logback.encoder.LoggingEventCompositeJsonEncoder">
      <providers>
        <timestamp/>
        <logLevel/>
        <loggerName/>
        <message/>
        <stackTrace/>
        <mdc/>
        <pattern>
          <pattern>
            {
              "service": "${spring.application.name}",
              "trace_id": "%X{traceId}",
              "span_id": "%X{spanId}"
            }
          </pattern>
        </pattern>
      </providers>
    </encoder>
  </appender>

  <root level="INFO">
    <appender-ref ref="JSON"/>
  </root>
</configuration>
```

Stdout is consumed by the Container Apps log driver; `trace_id` and `span_id` come from OpenTelemetry MDC integration, joining logs and traces.

## `Dockerfile` (multi-stage, distroless)

```dockerfile
# Build stage
FROM eclipse-temurin:21-jdk-alpine AS build
WORKDIR /workspace
COPY pom.xml .
COPY domain/pom.xml domain/
COPY infrastructure/pom.xml infrastructure/
COPY app/pom.xml app/
RUN mvn -B -q dependency:go-offline
COPY . .
RUN mvn -B -q -DskipTests package

# Runtime stage — distroless Java
FROM gcr.io/distroless/java21-debian12:nonroot
ARG JAR=app/target/app-1.0.0-SNAPSHOT.jar
COPY --from=build /workspace/${JAR} /app/app.jar
USER nonroot
EXPOSE 8080 8081
ENTRYPOINT ["java", "-jar", "/app/app.jar"]
```

Image size lands around 250 MB; smaller with jlink, far smaller with Spring Native (different scaffold — only use it if cold start is critical).

## `Makefile`

```make
.PHONY: build test lint run docker-build docker-run

build:
	mvn -B clean package

test:
	mvn -B test

lint:
	mvn -B spotless:check
	mvn -B verify -DskipTests=true

run:
	mvn -B -pl app spring-boot:run

docker-build:
	docker build -t <svc>:dev .

docker-run:
	docker run --rm -p 8080:8080 -p 8081:8081 <svc>:dev
```

## What ships in the scaffold but is empty

- `domain/src/main/java/...` — package skeleton, no business logic. The team fills it.
- `docs/adr/0001-initial-architecture.md` — first ADR is "we used the standard scaffold." Future ADRs document divergence.
- `docs/runbook.md` — empty headings: "On-call rotation," "Common alerts," "Manual recovery procedures."
- `docs/slo.md` — empty SLO definition; the team fills it during their first sprint.

The scaffold is a chassis, not a finished car. But the chassis is opinionated enough that the team doesn't have to rediscover the right shape of CI, Dockerfile, logging, OpenTelemetry integration, or Key Vault wiring.

## Anti-patterns (these are violations of the scaffold)

- **Property-file secrets** — any `application.properties` or `application.yml` with an actual password / connection string. Use Key Vault via Spring Cloud Azure.
- **Lombok** — Java 21 records, sealed types, `var`, and pattern matching cover most of Lombok's value with less tooling friction. Lombok is forbidden in new services.
- **Single-module Maven** — every service is multi-module from day one (`domain/`, `infrastructure/`, `app/`). Single-module quickly tangles domain logic with Spring infrastructure.
- **Hand-rolled HTTP server** — use Spring Boot's `WebMvc` (or `WebFlux` if explicitly justified). `Spark`, `Javalin`, `Vert.x`, `Ratpack` are off-stack.
- **Actuator on the application port** — must be on a separate management port to keep probes and traffic isolated.
- **Spring Boot 2.x** — out of OSS support; only Spring Boot 3.4+ for new services.

## Verification

A new Spring Boot service passes the scaffold check if:

1. Multi-module `pom.xml` exists (`domain`, `infrastructure`, `app`)
2. `application.yml` has zero plaintext secrets; Key Vault property source is enabled
3. Management port is separate from application port
4. OpenTelemetry Spring Boot Starter is on the classpath
5. JSON logging to stdout via `logback-spring.xml`
6. Dockerfile is multi-stage with distroless runtime
7. GitHub Actions workflows use OIDC (no long-lived `AZURE_CREDENTIALS` secret)
8. `docs/adr/`, `docs/runbook.md`, `docs/slo.md` exist (even if not yet filled)
