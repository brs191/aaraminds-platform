# Layers 2 & 3 — Authorization and Service-to-Service Identity

> Authentication says *who*; authorization says *what*. The breach that actually happens is object-level (BOLA/IDOR), not a missing role check. And services must authenticate each other with Managed Identity — never a shared secret.

## The two authorization checks you owe every request

1. **Coarse — scope/role at the edge or method.** Does this token carry `Orders.Read`? Cheap, necessary, insufficient.
2. **Fine — object-level ownership inside the service.** May *this subject* act on *this specific resource*? `GET /orders/{id}` must confirm the order belongs to the caller (or the caller is support/admin). Skipping this is BOLA — the top API security risk, and a coarse scope check does nothing to stop it.

```
GET /orders/8123  (token subject = user-42)
  scope check:   token has Orders.Read         -> pass (necessary)
  object check:  order 8123.ownerId == user-42 -> the check that matters
```

## RBAC vs ABAC — default to RBAC

- **RBAC** (default): Entra **app roles** and **group** membership flow into the token; map role → permitted operations. Covers the large majority of services.
- **ABAC** only when rules are genuinely attribute-driven (`user.department == resource.department`, clearance tiers). Implement with **Open Policy Agent (Rego)** or **Entra custom security attributes** — but ABAC is infrastructure you must operate. Do not reach for a policy engine before the rules demand it.

### Entra claim mechanics that bite

- `roles` = application permissions (daemon); `scp` = delegated (user) permissions. Read the correct claim for the caller type.
- **Group overage:** when a user is in more than ~150–200 groups, Entra omits the `groups` claim and emits an overage indicator; resolve groups via Microsoft Graph instead of assuming the claim is complete. Auth that "works in dev, fails for the VP in 300 groups" is this bug.

## Layer 3 — service-to-service identity (zero-trust east-west)

If one compromised service can call any other with no authentication, you have a flat trust domain. Every inter-service call authenticates with a workload identity.

### Managed Identity — system- vs user-assigned

- **System-assigned**: lifecycle tied to the resource; dies with it. Fine for a singleton.
- **User-assigned**: a standalone identity you attach to one or more resources; survives redeploy and blue/green. **Prefer user-assigned per service** so role assignments are stable across deployments and you get per-service least privilege. One shared identity for the whole fleet is an anti-pattern — you lose the ability to scope permissions per service.

Available on Container Apps, App Service, and AKS.

### Workload Identity on AKS

Federate a Kubernetes service account to a user-assigned Managed Identity via the cluster's OIDC issuer:

1. Enable OIDC issuer + workload identity on the cluster.
2. Create a **federated credential** on the Managed Identity trusting `system:serviceaccount:<ns>:<sa>`.
3. Annotate the service account; the pod gets tokens with no secret mounted.

This replaces the deprecated pod-identity add-on. No credentials in the cluster, ever.

### One credential path in code — `DefaultAzureCredential`

```java
// Java: azure-identity
TokenCredential cred = new DefaultAzureCredentialBuilder().build();
// local: developer login; in-cluster: the workload's Managed Identity — same code
```

```go
// Go: azidentity
cred, err := azidentity.NewDefaultAzureCredential(nil)
```

The same code authenticates locally (developer identity) and in production (Managed Identity). No branch, no secret.

### Data-plane authorization — least privilege per identity

Grant each service identity only the data role it needs:

| Resource | Role (data-plane) |
|---|---|
| Azure SQL / Postgres | Entra auth + database role (e.g. `db_datareader` + scoped writes), not server admin |
| Cosmos DB | Cosmos DB Built-in Data Contributor scoped to the database/container |
| Service Bus | Azure Service Bus Data Sender / Data Receiver (split send vs receive) |
| Storage | Storage Blob Data Contributor on the container, not the account |
| Key Vault | Key Vault Secrets User (read at runtime) — see `secrets-and-encryption.md` |

```hcl
# Terraform: scope the role to the resource, assign to the service's MI
resource "azurerm_role_assignment" "orders_sb_send" {
  scope                = azurerm_servicebus_queue.orders.id
  role_definition_name = "Azure Service Bus Data Sender"
  principal_id         = azurerm_user_assigned_identity.orders_svc.principal_id
}
```

### Transport identity / east-west policy

When you need authenticated *and* authorized transport between services, use a service mesh (Istio `AuthorizationPolicy` on AKS, or Open Service Mesh): mTLS gives each service a cryptographic identity; mesh policy enforces who may call whom. See `network-segmentation-and-zero-trust.md`.

## Failure modes

- **Edge-only authorization** (scope check, no object check) → BOLA; any authenticated user reads any record by guessing an id.
- **One shared user-assigned identity for all services** → cannot scope least privilege; a compromise of one pod has the union of every service's data rights.
- **Long-lived service principal + client secret** instead of Managed Identity → the secret leaks, rotation is a redeploy, and it is the classic SOC 2 finding.
- **Over-broad role at resource-group scope** (`Contributor` on the RG) instead of the specific data role on the specific resource.

## Brownfield: service principal + secret → Workload Identity

Inventory every credential (connection strings, keys), switch each store to Entra auth with `DefaultAzureCredential`, enable Workload Identity, create the federated credential, assign least-privilege data roles, run both paths behind a flag for 1–2 weeks watching auth errors, then revoke the old secret. Full recipe in `references/patterns/zero-trust-service-access.md`.

## Read next

- `secrets-and-encryption.md` — where unavoidable secrets live and how data is protected
- `audit-logging-and-compliance.md` — proving every allow/deny decision to an auditor
