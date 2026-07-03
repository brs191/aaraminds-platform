# Pattern: Service Discovery

## Problem

Service instances come and go — autoscaling adds replicas, rolling deploys replace pods, failures kill nodes. A caller that hardcodes IP addresses or DNS names of specific instances breaks the moment the topology changes. Service discovery is the mechanism that lets a caller find healthy instances of a target service at runtime, without hardcoding network locations.

## Use When

- Services scale dynamically (autoscaler, blue-green, canary deploys)
- The cluster has more than a handful of services (manual configuration doesn't scale)
- Health-based routing matters — calls should skip unhealthy instances
- Multiple environments (dev, staging, prod) need symmetric service addressing

## Avoid When

- The system has 2–3 services with static addresses — direct DNS or env vars are simpler
- Operating overhead of a discovery system exceeds the benefit (very small clusters)
- Platform already provides built-in discovery (Container Apps, AKS, App Service)

## Azure Implementation

### Implementation Steps

1. Pick the discovery mechanism based on platform: Container Apps internal DNS, AKS DNS, or external (Consul)
2. Tag services with metadata: name, version, environment
3. Configure health checks (liveness, readiness) — only ready instances participate in discovery
4. Use logical service names in client code (`http://payment-service`), not IPs
5. Let the platform handle load balancing across instances of the same logical service
6. For multi-region, use Front Door or Traffic Manager as the regional discovery layer

### Azure Services

| Component | Service | Configuration |
|---|---|---|
| Container-native discovery | Container Apps internal ingress | Automatic DNS, internal-only flag |
| Kubernetes discovery | AKS cluster DNS | Service object provides stable DNS |
| Service mesh discovery | Dapr / Istio / Linkerd | Sidecar-managed discovery + routing |
| Cross-region | Azure Front Door / Traffic Manager | Global discovery, health-based routing |
| External (rare) | Consul on Azure VMs | When standard options don't fit |

## Trade-offs

| Aspect | Trade-off |
|---|---|
| Flexibility | Strong — services can scale and move without client changes |
| Operational complexity | Adds a discovery layer to operate (unless platform-provided) |
| Latency | DNS-based discovery is fast; sidecar-based adds 1–2ms |
| Coupling | Loose — clients don't know instance addresses |
| Failure mode | Discovery layer failure can break all routing |

## Common Failure Modes

- **Stale cache** — Client caches an instance address that's since been replaced; calls fail with connection refused.
  - Detection: Spikes in `connection refused` after deploys.
  - Prevention: Short DNS TTLs (30–60s); HTTP clients respect TTL; force refresh on connection failure.

- **Unhealthy instance in pool** — Discovery includes an instance that passes liveness but fails real requests.
  - Detection: Some requests fail intermittently; one instance shows higher error rate.
  - Prevention: Readiness probes test actual functionality; outlier detection ejects bad instances.

- **Cross-namespace leakage** — Dev environment service accidentally discoverable from prod (or vice versa).
  - Detection: Test traffic appears in prod logs; environments contaminate each other.
  - Prevention: Namespace/environment isolation in DNS; network policies enforce boundaries.

- **Discovery as single point of failure** — Discovery service down = nothing can route.
  - Detection: All services fail with "name resolution failed" simultaneously.
  - Prevention: Discovery layer redundancy; clients cache last-known-good addresses with bounded TTL.

## Decision Signals

Adopt explicit service discovery when:
- The cluster has >5 services with dynamic instance counts
- Manual config-based addressing is causing deploy bugs
- Health-based routing is required

Skip when:
- Container Apps, AKS, or App Service already provide built-in DNS-based discovery
- System is small enough that DNS env vars suffice

## Azure Mapping

| Service | Role | Why |
|---|---|---|
| Container Apps Ingress | Built-in DNS discovery | Zero-config, internal traffic stays in the VNet |
| AKS DNS | Cluster-internal discovery | Standard Kubernetes Service objects |
| Dapr | Sidecar discovery | Cross-runtime, app-agnostic |
| Azure Front Door | Global / multi-region | Health-based global routing |

## Go Implementation Notes

For Container Apps, just use logical names:
```go
resp, err := http.Get("http://payment-service/charge")
```
The platform resolves `payment-service` to the current set of healthy replicas.

For mesh-based (Dapr): call via sidecar:
```go
resp, err := http.Get("http://localhost:3500/v1.0/invoke/payment-service/method/charge")
```

Avoid client-side service-discovery libraries when the platform handles it — they add complexity for no benefit.

## MCP Tool Opportunities

- `recommend_microservice_patterns` — recommends discovery when describing dynamic scaling or multiple environments
- `detect_architecture_risks` — flags hardcoded IPs, missing health checks, missing namespace isolation
- `map_patterns_to_azure_services` — picks Container Apps Ingress vs. AKS DNS vs. mesh

## Related Patterns

- **Service Mesh** — typically includes discovery as a feature
- **API Gateway** — handles external-to-internal discovery
- **Health Check** — fundamental input to discovery decisions
- **Sidecar** — Dapr/mesh sidecars provide discovery transparently

## References

- Skill: `../azure-mapping.md` — platform-native discovery options
- Pattern: `service-mesh.md` — discovery as part of mesh
- Pattern: `../../../microservices-api-design/references/patterns/api-gateway.md` — external-facing discovery
