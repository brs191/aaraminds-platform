---
id: architecture-review/03-zero-trust-gap-review
area: architecture-review
exercises:
  - .claude/skills/azure-microservices-security/references/patterns/zero-trust-service-access.md
  - .claude/skills/azure-microservices-security/references/security-design.md
pass_threshold: 7/9
last_run: 2026-05-30
last_result: pass
---

# Find zero-trust gaps in a microservices design

## Context

Attach the zero-trust-service-access pattern card and `11-security-design.md`.

## Prompt

A healthcare platform's security posture, as described in their design doc:

> "All traffic is internal — services run in the same AKS cluster behind an API gateway. The gateway authenticates external requests with OAuth. Internal service-to-service calls use a shared API key passed in an HTTP header. Secrets are stored in environment variables loaded at pod startup from a config map. Network policies aren't applied — services can call each other freely within the cluster. Audit logging is at the API gateway only."

Identify the zero-trust gaps and propose the specific remediations. Be exhaustive.

## Rubric

A response passes if it covers at least 7 of the following 9 points:

- [ ] **Shared API key is a fail** — flags that one shared key across services means a compromise of any one service compromises all; rotation requires redeploying every service
- [ ] **mTLS or managed identity for service-to-service** — proposes per-service workload identity (Azure Managed Identity) or mTLS via a service mesh
- [ ] **Secrets in env vars / config map is a fail** — flags that secrets shouldn't live in config maps; proposes Key Vault + CSI driver or runtime fetch
- [ ] **No network policies = lateral movement** — flags that allowing all intra-cluster calls means a compromised service can hit any other; proposes NetworkPolicies with deny-by-default
- [ ] **Audit at gateway only is insufficient** — flags that PHI access happens inside services, not just at the gateway; proposes per-service audit logging of access decisions
- [ ] **HIPAA-specific concerns** — calls out that PHI access must be logged with the accessing identity, the record accessed, and the result; retention ≥7 years
- [ ] **Defense-in-depth framing** — explicitly notes that any one of these layers can be breached; the design currently has one layer (gateway auth), which is the opposite of defense-in-depth
- [ ] **Encryption in transit and at rest** — verifies (and where missing, requires) TLS for internal traffic and encryption-at-rest for stores holding PHI
- [ ] **Specific Azure-native remediation per gap** — names actual services (Managed Identity, Key Vault, Dapr or Istio, NetworkPolicies, Application Insights) per fix rather than abstract "use authentication"

## Reference output

> The design has one layer of authentication (gateway OAuth) and treats everything past that as trusted. That is the architectural shape zero-trust is meant to replace. Every gap is exploitable; given the workload is healthcare PHI, every gap is also a HIPAA finding.
>
> **Gap 1: Shared API key for service-to-service.**
> One key, every service has it, log it once and you've owned the cluster. Rotation requires touching every service and redeploying. This isn't authentication; it's a password sticky-note shared by the team.
> *Fix:* Azure Managed Identity per service. Each service gets its own identity in Azure AD. When service A calls service B, A's identity is included in the token; B's authorizer validates it against an explicit allow-list. No shared keys, no manual rotation, automatic credential lifecycle.
>
> **Gap 2: Secrets in environment variables loaded from a config map.**
> Anyone with cluster-read access to the config map (more people than you think — operators, CI, observability tools) can read every secret. Rotating a secret requires editing the config map and rolling pods.
> *Fix:* Azure Key Vault as the single source of truth for secrets. Use the Key Vault CSI driver to mount secrets as files at pod startup (without leaving them in env vars), or fetch at runtime via the SDK with Managed Identity. Pods see only the secrets they need; rotation in Key Vault propagates without redeploy.
>
> **Gap 3: No network policies — unrestricted lateral movement.**
> A compromise of any service (an SSRF, a deserialization bug, a leaked sidecar credential) gives the attacker freedom to call every other service. There's nothing stopping the catalog-service pod from talking directly to the patient-records database.
> *Fix:* Kubernetes NetworkPolicies (AKS) with deny-by-default for both ingress and egress. Each service declares the services it calls; the policy permits exactly those flows. Pair with namespace separation by sensitivity tier (PHI services in their own namespace).
>
> **Gap 4: No mTLS between services.**
> Even with the shared API key, the call itself is plaintext within the cluster. Anyone with packet capture or a misconfigured sidecar can observe PHI in flight.
> *Fix:* Service mesh (Istio or Linkerd on AKS, or Dapr in Container Apps) with cluster-wide strict mTLS. Sidecars terminate TLS on both ends; app code doesn't change. Verifies identity at the transport layer in addition to the application layer.
>
> **Gap 5: Audit logging at the gateway only.**
> The gateway sees the *external* request — it cannot see who inside the cluster touched which patient record. From a HIPAA perspective, this is the most damaging gap: "who accessed Mr. Smith's chart at 3pm?" is unanswerable.
> *Fix:* Per-service audit logging at the access-decision point. Every read or write of a record emits a structured log: `{timestamp, identity, action, resource, decision, reason}`. Pipe to a tamper-resistant store (Application Insights → Log Analytics with immutable archive). Retention configured to ≥7 years per HIPAA. Audit is its own pipeline, separate from operational logs.
>
> **Gap 6: Encryption posture not addressed.**
> The doc says "secrets in env vars" but doesn't mention what's encrypted at rest. PHI in databases must be encrypted at rest with Customer-Managed Keys (BYOK) for the compliance posture. Internal HTTPS must be TLS 1.2+.
> *Fix:* Verify all PHI-holding stores have TDE / SSE enabled with Key Vault-managed keys. Enforce TLS 1.2+ on all ingress and service-to-service traffic (mTLS gives you this).
>
> **Gap 7: No least-privilege authorization.**
> Even with proper authentication, what stops auth'd service X from calling sensitive endpoints on service Y? The design has nothing.
> *Fix:* Authorization policies per service. With a mesh: AuthorizationPolicy CRDs declare which identities can call which endpoints. Or, in-app: ABAC checks (`this clinician is in this care team for this patient`) before serving any PHI.
>
> **Defense-in-depth framing**
>
> Today the design has *one* layer that needs to fail for an attacker to win: gateway auth. The zero-trust posture is six layers that *all* need to fail for the same outcome: ingress authn, per-call authz, service-to-service authn (mTLS + identity), network policy, in-app authorization, audit + alerting. Each layer is cheap to add; together they make catastrophic compromise much harder.
>
> Don't ship to production until at minimum: per-service Managed Identity, Key Vault for secrets, NetworkPolicies deny-by-default, mTLS via mesh, and per-service audit logging are in place. The rest can be a follow-up iteration.

## Notes

The "exhaustive" framing tests whether the responder enumerates rather than picking the most obvious one or two gaps. The HIPAA context cues domain-specific concerns (audit retention, PHI handling) that go beyond generic zero-trust.
