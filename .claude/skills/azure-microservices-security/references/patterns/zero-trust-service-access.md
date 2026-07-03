# Pattern: Zero-Trust Service Access

## Problem

The classic "network perimeter" model assumes anything inside the cluster is trusted. Once an attacker breaches the perimeter — through a compromised dependency, a leaked credential, or a misconfigured ingress — they can move laterally across services unchecked. Zero-trust flips the model: no service is trusted by default; every call is authenticated and authorized, regardless of network origin.

## Use When

- Compliance requires defense-in-depth (PCI-DSS, HIPAA, SOC 2, government)
- Cluster hosts services with mixed sensitivity levels (PII-handling next to public)
- Multi-tenant systems where one tenant compromise shouldn't reach others
- Cloud-native deployments where "network perimeter" is fuzzy by design

## Avoid When

- Tiny cluster with one service — added complexity dwarfs benefit
- Internal-only batch tools where breach surface is minimal
- Team can't operate the auth and policy infrastructure
- Latency budget too tight (mTLS adds ~1ms; policy lookup adds more)

## Azure Implementation

### Implementation Steps

1. Enforce mTLS between every service — no plaintext internal traffic
2. Give every service a workload identity (managed identity, SPIFFE/SPIRE)
3. Define authorization policies per service: who can call which endpoints
4. Authenticate at every hop — don't trust upstream's identity assertion alone
5. Apply network segmentation: services in their own subnets; deny-by-default policies
6. Encrypt data at rest (TDE, Customer-Managed Keys); rotate keys regularly
7. Audit every service-to-service call: who called what, when, with what result
8. Centralize secrets in Key Vault; nothing in config files or env vars

### Azure Services

| Component | Service | Configuration |
|---|---|---|
| Workload identity | Azure Managed Identity | Per-service identity, no credentials in code |
| mTLS | Service Mesh (Istio/Dapr/Linkerd) | Automatic mTLS between sidecars |
| Authorization | Mesh policies + Microsoft Entra ID RBAC | Service-to-service allow lists |
| Network segmentation | Container Apps internal ingress, AKS NetworkPolicies | Deny-by-default, allow specific paths |
| Secrets | Azure Key Vault | All secrets, certs, keys; pulled via CSI driver or SDK |
| Audit | Azure Monitor, Application Insights | Every auth event logged |

## Trade-offs

| Aspect | Trade-off |
|---|---|
| Security | Strongly improved — defense in depth, lateral movement blocked |
| Operational complexity | High — identities, policies, certificates everywhere |
| Latency | Adds 1–3ms per hop (mTLS + policy check) |
| Cost | Mesh, Key Vault, monitoring all add line items |
| Developer experience | Requires understanding of workload identity, mesh policies |
| Debugging | Network errors more cryptic ("policy denied" vs. "connection refused") |

## Common Failure Modes

- **Wildcard policies** — Lazy policy `allow service X from *`; defeats zero-trust.
  - Detection: Policy audit shows broad allow rules.
  - Prevention: Policy review process; deny-by-default templates; minimum scope per allow rule.

- **Long-lived tokens** — Service identities have tokens valid for days; stolen token enables prolonged access.
  - Detection: Token expiry inspection shows long TTLs.
  - Prevention: Short-lived tokens (1 hour or less) with auto-renewal via managed identity.

- **Secrets in env vars / config** — A service hardcodes a connection string; rotation requires redeploy.
  - Detection: `git grep` finds connection strings; env vars contain secrets.
  - Prevention: Key Vault for all secrets; CSI driver or runtime fetch; never config files.

- **Audit logging gap** — Auth events not logged or not retained long enough for post-incident forensics.
  - Detection: Compliance audit finds missing trails.
  - Prevention: Centralized auth log retention (≥90 days); test recovery of "who accessed X at time T".

## Decision Signals

Adopt zero-trust when:
- Compliance requires it (PCI, HIPAA, SOC 2)
- Multi-tenant or mixed-sensitivity workloads
- Past incident involved lateral movement

Skip when:
- Single-service, low-risk
- Team lacks bandwidth to operate the policy infrastructure

## Azure Mapping

| Service | Role | Why |
|---|---|---|
| Managed Identity | Service identity | No credentials in code; per-service |
| Istio / Dapr | mTLS + policy | Sidecar-enforced authn/authz |
| Key Vault | Secrets store | Single source for credentials, certs, keys |
| NetworkPolicies (AKS) | Segmentation | Deny-by-default network access |
| Microsoft Entra ID | Federated identity | Service-to-service auth via tokens |

## Go Implementation Notes

For Azure Managed Identity:
```go
cred, err := azidentity.NewManagedIdentityCredential(nil)
// Use cred to access Key Vault, Service Bus, SQL, etc.
```
No credentials in code; the runtime injects identity.

For mTLS via Dapr/Istio, the app is unaware — sidecars handle TLS. Policies are mesh CRDs (AuthorizationPolicy in Istio).

Audit logging: emit structured log line for every authentication / authorization decision (`event: auth.allow`, `event: auth.deny`).

## MCP Tool Opportunities

- `recommend_microservice_patterns` — recommends zero-trust when compliance or multi-tenancy is described
- `detect_architecture_risks` — flags secrets in code, wildcard policies, long-lived tokens, missing mTLS
- `generate_security_policies` — drafts AuthorizationPolicies / NetworkPolicies from described topology
- `generate_compliance_report` — produces audit trail summary for compliance review

## Related Patterns

- **Service Mesh** — provides mTLS and policy enforcement
- **Sidecar** — workload identity often delivered via sidecar
- **API Gateway** — external-facing zero-trust enforcement point
- **Audit Logging** — required complement; zero-trust without audit is half a defense

## References

- Skill: `../security-design.md` — defense-in-depth overview and layer map
- Reference: `../network-segmentation-and-zero-trust.md` — the network + zero-trust deep dive
- Pattern: `../../../azure-service-mapping/references/patterns/service-mesh.md` — typical zero-trust enforcer
- Pattern: `../../../microservices-api-design/references/patterns/api-gateway.md` — external boundary for zero-trust
