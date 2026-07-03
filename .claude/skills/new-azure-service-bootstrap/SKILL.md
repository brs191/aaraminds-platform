---
name: new-azure-service-bootstrap
description: Scaffolds a new Azure microservice end-to-end — repository layout, Spring Boot 21+ or Go service skeleton, Terraform AzureRM infrastructure, GitHub Actions OIDC pipeline, Container Apps deployment, OpenTelemetry instrumentation, Key Vault + Managed Identity wiring, and test scaffolding. Use when starting a new service in the existing estate, deciding between Java and Go for a service, replacing a legacy service with a fresh scaffold, or auditing whether an existing scaffold follows the standard. Do not use for designing the service's domain (use microservices-architecture-design) or for code-level review of an existing service (use pr-review-azure-microservices).
version: 1.0.0
last_updated: 2026-05-18
---

# New Azure Service Bootstrap

## When to use

Trigger this skill when the question is "I need a new microservice — what's the standard scaffold?" Common triggers: starting a new service in an existing Azure estate, choosing between Spring Boot and Go for a new domain, replacing a legacy service with a modern scaffold, onboarding a contributor by walking them through the standard project layout.

Do **not** use this skill for: designing the service's domain or bounded context (use `microservices-architecture-design`); code-level PR review (use `pr-review-azure-microservices`); cost / capacity sizing decisions (`azure-microservices-cost-review`).

## The critical decision rule — every new service starts from the standard scaffold

The standard scaffold encodes a decade of decisions you'd otherwise rediscover. New services do not invent their own structure, their own CI shape, their own IaC layout, or their own observability wiring. They start from the scaffold and customize only where the domain genuinely requires it.

The scaffold is opinionated:
- **Spring Boot 21+** for Java services. Maven multi-module, no Lombok dependency (use Java 21 records / `var`), Spring Boot Actuator with management on a separate port.
- **Go 1.25+** for Go services. Standard `cmd/<binary>` + `internal/<package>` layout. No `pkg/` directory; export through `internal/` consciously.
- **Terraform AzureRM** with RBAC mode. Modules under `infra/` per resource group. State in a dedicated Storage account.
- **GitHub Actions with OIDC** for deploys. No long-lived service principal secrets in CI.
- **Container Apps** as the compute platform. AKS only if `azure-service-mapping` justifies it for this specific service.
- **OpenTelemetry from day one.** No "we'll add tracing later."
- **Managed Identity + Key Vault** for every secret. Zero env-var secrets, zero config-file secrets.

## The bootstrap selector

| Question | Default | Reference |
|---|---|---|
| Java or Go for this service? | Decision rule depends on workload, team competence, latency budget | `references/language-choice.md` |
| Java scaffold (Spring Boot 21+) | Maven multi-module, Spring Boot 3.4+, Actuator, OpenTelemetry agent | `references/spring-boot-scaffold.md` |
| Go scaffold | `cmd/server` + `internal/...`, otelhttp / otelgrpc, `slog` to stderr | `references/go-scaffold.md` |
| CI / CD pipeline | GitHub Actions with OIDC, separate workflows per environment | `references/cicd-pipeline.md` |
| Infrastructure | Terraform AzureRM (RBAC), Container Apps, Postgres Flexible / Cosmos | `references/cicd-pipeline.md` |

## Bootstrap logic

1. **Pick the language first.** Java for services with rich domain models, JVM-ecosystem dependencies (Spring Data, Spring Security, Spring Cloud Stream), team Java skill, or compliance-heavy workloads where the JVM observability story is more mature. Go for services with tight latency budgets, high concurrency I/O patterns, low memory footprint requirements, or already-Go neighborhoods. See `references/language-choice.md` for the decision matrix.

2. **Generate the scaffold from the template.** Do not hand-write a new service tree; copy the standard scaffold (Spring Boot or Go), rename, then customize. The scaffold contents are:
   - **Source layout** with health, readiness, metrics endpoints already wired
   - **OpenTelemetry** auto-instrumentation (Spring Boot starter or Go otel SDK)
   - **`slog` (Go) / structured logback (Java)** writing JSON to stdout for ingestion
   - **Dockerfile** multi-stage, distroless runtime, non-root
   - **`Makefile`** with `make build / test / lint / run / docker-build / docker-run`
   - **`.github/workflows/`** with PR-gate + post-merge + release workflows
   - **`infra/`** with Terraform module skeleton (Container Apps, Postgres / Cosmos, Service Bus topic if needed)
   - **`README.md`** with the standard sections (purpose, ownership, runbook, ADR list)

3. **Wire identity and secrets before writing business logic.** The Container Apps revision gets a System-Assigned Managed Identity at creation time (Terraform). The MI is granted `Key Vault Secrets User` on the project's Key Vault via Azure RBAC. The application code reads secrets via `DefaultAzureCredential` (Azure SDK; Spring Cloud Azure Starter for Java, `azidentity` for Go). No `application.properties` secret values. No `os.Getenv("PG_PASSWORD")`. See `references/spring-boot-scaffold.md` and `references/go-scaffold.md`.

4. **Instrument before deploying.** OpenTelemetry SDK initialized at startup; HTTP server middleware emits server spans; HTTP client wrapper emits client spans; OTLP exporter targets the Grafana/Prometheus + Tempo stack. No service ships without traces. See `azure-microservices-observability` skill for SLO/alert design.

5. **CI from day one, deploys from day two.** PR pipeline runs lint + vet/style + unit tests + `govulncheck` (Go) or `mvn dependency-check` (Java) + integration tests against testcontainers. Post-merge builds the image and pushes to ACR. Release workflow rolls out the new revision via Container Apps revisions (blue-green). See `references/cicd-pipeline.md`.

6. **Document the service.** README has: purpose (one paragraph), owners (team + on-call), local-dev runbook, deployment runbook, ADRs (or pointer to the ADR repo), and a "what this service is not" section. The README is what an oncall reads at 2 AM; treat it as a runbook, not marketing.

## Worked example — brownfield: replacing a legacy `accounting-service` with a fresh scaffold

Setup: an existing Spring Boot 2.7 service (`accounting-service`) is on App Service, using property-file secrets, no OTel, log4j-style logging, Maven single-module. The team is rewriting it to land on Container Apps and the modern scaffold. Existing API contract stays; internals change.

Decision walk:

1. **Language check.** Java is fine — the team is deeply Java, Spring Data JPA is doing real work, accounting domain is rich. Confirm Spring Boot scaffold; reject "let's rewrite in Go" without a workload argument. See `references/language-choice.md`.
2. **Generate the scaffold under a new repo (`accounting-service-v2`).** Spring Boot 3.4 with Java 21. Maven multi-module: `app/`, `domain/`, `infra/`. Actuator on a separate port (avoid traffic and management mixing). See `references/spring-boot-scaffold.md`.
3. **Identity migration.** Existing service uses connection strings in `application.properties` for Azure SQL and Service Bus. New service: System-Assigned Managed Identity on the Container Apps revision, Entra ID auth for Azure SQL (drop SQL auth), Managed Identity for Service Bus. Verify the Postgres / Azure SQL instance accepts Entra-ID logins (may need a one-time setup); see `azure-microservices-security` for the migration steps.
4. **CI rebuild.** New repo gets the standard `.github/workflows/`: `pr.yml` (lint, test, vulncheck, image build to a PR-scoped tag), `main.yml` (deploy to staging via OIDC, integration tests, deploy to prod after manual approval gate). See `references/cicd-pipeline.md`.
5. **OTel from commit zero.** Spring Boot Starter for OpenTelemetry; OTLP exporter to the cluster's tempo / OTel collector. Service span attributes: `service.name=accounting-service`, `service.namespace=finance`, `deployment.environment=staging|prod`. See `azure-microservices-observability` skill.
6. **Migration cutover.** Deploy `accounting-service-v2` to staging; run parity tests against the v1 service using the same API contract; cut over via Front Door route weight (10% → 50% → 100%) with rollback to v1 if KPIs degrade. Retire v1 after 30 days of clean v2 operation. See `microservices-resilience` skill for the rollout pattern.

## Anti-pattern — "we'll add observability later"

**Bad:** A new service ships to staging without OpenTelemetry, without per-endpoint SLO targets, without alerts. The team plans to "add observability before prod." Then prod ships under deadline pressure with the same gaps.

**Why it fails:** Observability is not retrofit-friendly. Every endpoint that ships without span instrumentation accumulates blind spots that are hard to retroactively add. Every alert thresholds that gets set "later" gets set under fire, after the first incident, with adrenaline-driven judgment. The cheap moment to add observability is when the endpoint is being written — not three months later when on-call is debugging at 2 AM.

**Detection signal:** new service's `pom.xml` / `go.mod` lacks the OpenTelemetry dependency. No `service.name` resource attribute set. No alerts file under `infra/observability/`. No SLO documented in the README.

**Fix:** OpenTelemetry dependency, OTel exporter config, baseline alerts (error rate, p99 latency, saturation), and a SLO section in the README are all part of the scaffold. They land with the first commit, not after the first incident.

## Verification questions

1. Does the service use the standard scaffold (Spring Boot or Go), and is its deviation from the scaffold documented in the README?
2. Is every secret retrieved via Managed Identity + Key Vault — zero env-var secrets, zero config-file secrets?
3. Is OpenTelemetry initialized at startup with proper `service.name` / `service.namespace` / `deployment.environment` attributes?
4. Does the CI pipeline use GitHub Actions OIDC (no long-lived service principal secrets)?
5. Is the deployment Container Apps with Terraform AzureRM modules, not hand-crafted Bicep / Pulumi / Azure DevOps templates?
6. Does the README cover purpose, ownership, runbook, SLO, and the "what this service is not" section?

## What to read next

- `references/language-choice.md` — when to pick Java, when to pick Go
- `references/spring-boot-scaffold.md` — full Spring Boot 21+ scaffold, Maven layout, Actuator, OTel agent
- `references/go-scaffold.md` — full Go scaffold, `cmd/server` + `internal/`, otelhttp, slog
- `references/cicd-pipeline.md` — GitHub Actions OIDC workflow, Terraform deploy, Container Apps rollout
- `microservices-architecture-design` skill — for the upstream domain / boundary decisions
- `azure-microservices-security` skill — for Entra ID auth, Key Vault wiring, network segmentation
- `azure-microservices-observability` skill — for SLO definition, alert design, dashboard standards
- `pr-review-azure-microservices` skill — what to enforce in code review once the scaffold is in use
