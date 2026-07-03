# Pattern: Service Mesh

## Problem

Every service ends up implementing the same cross-cutting concerns — mTLS, retries, timeouts, circuit breakers, distributed tracing, traffic shifting. Each service does it slightly differently, in different languages, with different bug profiles. A service mesh extracts these into a sidecar proxy (or a node-level daemon) that every service traffic passes through, applying policies uniformly without app code changes.

## Use When

- The cluster has 10+ services in multiple languages/runtimes
- Cross-cutting concerns (mTLS, observability, retries) must be enforced uniformly
- You need fine-grained traffic shifting (10% to v2, 90% to v1) without app changes
- Security policies (zero trust, mTLS) must apply to every service automatically

## Avoid When

- Small cluster (<5 services) where the mesh overhead outweighs benefits
- Latency budget too tight (mesh adds 1–5ms per hop)
- Platform already provides equivalent features (Container Apps + Dapr is mesh-lite)
- Team can't operate the mesh control plane (Istio is non-trivial)

## Azure Implementation

### Implementation Steps

1. Choose the mesh: Dapr (light, app-aware), Istio (full-featured, complex), Linkerd (simpler than Istio), or Open Service Mesh
2. Inject sidecars: automatic on namespace label (AKS) or per-app annotation
3. Define mTLS policy: cluster-wide strict mTLS once sidecars are in place
4. Configure traffic policies: timeouts, retries, circuit breakers, in mesh config (not app code)
5. Set up traffic shifting for canary and blue-green deploys
6. Wire observability: mesh emits metrics, traces, logs without app instrumentation
7. Apply network policies: zero-trust authorization between services

### Azure Services

| Component | Service | Configuration |
|---|---|---|
| Light mesh / runtime | Dapr (Container Apps or AKS) | Sidecar with app-level APIs (state, pub-sub, invoke) |
| Full mesh | Istio on AKS | Sidecar (Envoy), control plane (istiod) |
| Simpler mesh | Linkerd on AKS | Rust-based, lower overhead than Istio |
| Azure-native | Open Service Mesh | Microsoft-backed SMI implementation (deprecated 2024 — check status) |
| Observability | Application Insights | Mesh emits OpenTelemetry traces |

## Trade-offs

| Aspect | Trade-off |
|---|---|
| Uniformity | Cross-cutting policies enforced everywhere |
| App code simplicity | Apps lose responsibility for mTLS, retries, tracing |
| Latency | Adds 1–5ms per hop (sidecar proxy) |
| Operational complexity | Mesh control plane to operate (significant for Istio) |
| Resource cost | Each pod has a sidecar (~50–100MB memory, ~0.1 vCPU baseline) |
| Debugging | Network path now goes through a proxy; debugging requires mesh knowledge |

## Common Failure Modes

- **Sidecar startup race** — App starts before sidecar is ready; outbound calls fail until sidecar comes up.
  - Detection: Pod logs show network errors in the first few seconds.
  - Prevention: Use `holdApplicationUntilProxyStarts` or equivalent; or have app retry initial calls.

- **mTLS enforcement breaks unrelated traffic** — Strict mTLS enabled cluster-wide; legacy services without sidecars get blocked.
  - Detection: Sudden cross-service connection failures after policy change.
  - Prevention: Permissive mode first; verify all services have sidecars; then switch to strict.

- **Sidecar resource exhaustion** — Sidecars get OOM-killed under load; restart cycles break traffic.
  - Detection: Sidecar restart count rising; correlation with traffic spikes.
  - Prevention: Right-size sidecar memory; monitor sidecar metrics separately.

- **Mesh as smart pipe** — Business logic creeps into mesh policies (e.g., header-based routing logic that should be in app).
  - Detection: Mesh config grows complex; understanding traffic flow requires reading both app and mesh.
  - Prevention: Keep mesh for cross-cutting (security, retries, tracing); business logic stays in app.

## Decision Signals

Adopt a service mesh when:
- Cluster reaches 10+ services or multiple languages
- Security requires mTLS everywhere (zero-trust)
- Cross-cutting concern duplication has become a bug source

Skip when:
- Small cluster; Container Apps + Dapr provides enough
- Latency-critical workload that can't afford mesh hop

## Azure Mapping

| Service | Role | Why |
|---|---|---|
| Dapr | App-aware runtime | State, pub-sub, invoke APIs; lighter than full mesh |
| Istio | Full mesh | Most features, most complex |
| Linkerd | Simpler mesh | Lower overhead, smaller feature set |
| Container Apps | Built-in Dapr | Mesh-lite without standalone mesh setup |

## Go Implementation Notes

With Dapr, the app calls localhost; sidecar handles the rest:
```go
// Service-to-service invocation via Dapr
resp, _ := http.Post(
    "http://localhost:3500/v1.0/invoke/payment-service/method/charge",
    "application/json", body)
```
Mesh handles mTLS, retries, tracing, discovery. App code is unaware.

With Istio, the app code is unchanged; sidecar transparently intercepts traffic. Policies are CRDs (VirtualService, DestinationRule, AuthorizationPolicy).

## MCP Tool Opportunities

- `recommend_microservice_patterns` — recommends mesh when describing 10+ services or uniform security needs
- `detect_architecture_risks` — flags duplicate cross-cutting logic across services, missing mTLS, ad-hoc retries
- `map_patterns_to_azure_services` — picks Dapr vs. Istio vs. Linkerd based on team and feature needs
- `generate_mesh_policy_plan` — drafts initial traffic and security policies

## Related Patterns

- **Sidecar** — the mesh's underlying mechanism
- **Service Discovery** — usually included in mesh
- **Circuit Breaker** — mesh-applied, not app-coded
- **Zero-Trust Service Access** — mesh enforces mTLS and authorization
- **API Gateway** — mesh handles internal traffic; gateway handles external

## References

- Skill: `../azure-mapping.md` — Dapr in Container Apps as mesh-lite
- Skill: `../../../azure-microservices-security/references/security-design.md` — mTLS via mesh
- Pattern: `sidecar.md` — the building block
- Pattern: `../../../azure-microservices-security/references/patterns/zero-trust-service-access.md` — mesh enables zero-trust enforcement
