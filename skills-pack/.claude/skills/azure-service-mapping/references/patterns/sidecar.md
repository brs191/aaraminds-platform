# Pattern: Sidecar

## Problem

Cross-cutting concerns — logging, tracing, mTLS, secrets fetching, configuration reload — must be implemented in every service. Each language and framework does it slightly differently; bug fixes require updating dozens of services. The sidecar pattern moves these concerns into a separate process that runs alongside the application in the same pod, sharing its lifecycle and network namespace.

## Use When

- The same infrastructure concern must apply uniformly across services
- Services are written in multiple languages — uniform libraries are impractical
- Operational concerns (config, secrets, mTLS) should be decoupled from application code
- The team wants to upgrade infrastructure capabilities without redeploying every app

## Avoid When

- Single-language, small cluster — a shared library is simpler
- Latency budget is tight — sidecar adds 1–3ms per intercepted call
- Resource constraints — every pod gets an extra container (~50–100MB memory)
- The concern lives entirely inside the app (business logic) — sidecar adds no value

## Azure Implementation

### Implementation Steps

1. Identify the cross-cutting concern (logging, tracing, secrets, mTLS, service discovery)
2. Choose or build a sidecar implementation (Dapr for app APIs, Envoy for proxying, custom for niche needs)
3. Configure the orchestrator to inject the sidecar with the app (init container, sidecar container)
4. App communicates with sidecar over localhost (TCP or Unix socket)
5. Sidecar communicates with external systems (other services, Key Vault, observability backend)
6. Health checks include both app and sidecar; failures of either are visible

### Azure Services

| Component | Service | Configuration |
|---|---|---|
| App-aware sidecar | Dapr | Pub-sub, state, secrets, invoke APIs via localhost:3500 |
| Network proxy sidecar | Envoy (Istio) | mTLS, retry, circuit breaker, traffic shifting |
| Logging sidecar | Fluent Bit | Tails app logs, ships to Log Analytics |
| Secrets sidecar | Key Vault CSI driver (init pattern) | Mounts secrets as files for the app |
| Container Apps | Dapr built-in | Native sidecar support, no manual injection |

## Trade-offs

| Aspect | Trade-off |
|---|---|
| Uniformity | Strong — concerns apply identically across services |
| Language agnostic | Apps in any language benefit equally |
| Operational | Sidecar upgraded independently of app |
| Resource cost | Every pod has +1 container; CPU and memory adders |
| Latency | Adds 1–3ms per intercepted hop |
| Failure surface | Sidecar crash takes down app's external communication |

## Common Failure Modes

- **Startup ordering** — App starts and tries to call sidecar before sidecar is ready; first calls fail.
  - Detection: Network errors in first seconds after pod start.
  - Prevention: Use `holdApplicationUntilProxyStarts` (Istio) or app retries with backoff on init failures.

- **Sidecar resource starvation** — Sidecar throttled by tight CPU/memory limits; app's traffic slows.
  - Detection: Sidecar CPU at limit; app's outbound latency rises.
  - Prevention: Profile sidecar's resource needs under load; set adequate requests/limits.

- **Sidecar bug breaks all services** — Buggy sidecar version rolled out cluster-wide breaks every app.
  - Detection: Many services fail simultaneously after sidecar upgrade.
  - Prevention: Canary sidecar upgrades; per-namespace rollouts; quick rollback.

- **Sidecar as smart pipe** — Business logic ends up in sidecar config; complexity creeps in.
  - Detection: Sidecar config grows to hundreds of lines per service.
  - Prevention: Keep sidecars focused on cross-cutting infra; business logic stays in app.

## Decision Signals

Adopt sidecars when:
- 5+ services share an infra concern done inconsistently
- Multi-language cluster needs uniform observability or security
- Service mesh is being considered (mesh = mass sidecar)

Skip when:
- Single-language, small cluster
- Concern lives in business logic, not infrastructure

## Azure Mapping

| Service | Role | Why |
|---|---|---|
| Dapr in Container Apps | App-aware sidecar | Native support, pub-sub/state/invoke APIs |
| Istio on AKS | Mesh sidecar | mTLS, traffic shaping, observability |
| Fluent Bit sidecar | Log shipping | Decouples logging from app code |
| Key Vault CSI driver | Secrets sidecar (init) | Secrets as files, no SDK in app |

## Go Implementation Notes

With Dapr:
```go
// State via sidecar
resp, _ := http.Post("http://localhost:3500/v1.0/state/mystate",
    "application/json", body)
// Pub-sub via sidecar
http.Post("http://localhost:3500/v1.0/publish/mypubsub/topic1",
    "application/json", event)
```
App is unaware of Service Bus, Redis, mTLS — sidecar abstracts them.

Logging sidecar pattern: app writes structured JSON to stdout/file; Fluent Bit sidecar tails and ships.

## MCP Tool Opportunities

- `recommend_microservice_patterns` — recommends sidecar for cross-cutting concerns repeated across services
- `detect_architecture_risks` — flags startup ordering issues, missing sidecar health checks, business logic in sidecars
- `generate_sidecar_config` — drafts Dapr component or Istio config for described needs
- `map_patterns_to_azure_services` — picks Dapr vs. Envoy vs. custom

## Related Patterns

- **Service Mesh** — built on sidecars; sidecar is the building block
- **Ambassador** — a sidecar variant focused on outbound communication
- **Adapter** — sidecar that translates between protocols
- **Distributed Tracing** — typically applied via sidecar

## References

- Skill: `../azure-mapping.md` — Dapr in Container Apps
- Pattern: `service-mesh.md` — large-scale application of sidecars
- Pattern: `../../../azure-microservices-security/references/patterns/zero-trust-service-access.md` — sidecars enforce mTLS
