# Network Segmentation and Zero-Trust

> Zero-trust means no implicit trust from network location. Combine identity (Managed Identity, mTLS) with network controls (private endpoints, deny-by-default). Network is a *containment* layer, not the primary gate — a private network with no identity checks is still one breach from flat.

## The principle, made concrete on Azure

A service inside the cluster is treated exactly like one on the public internet: every call is authenticated, every call is authorized, every call is logged. Network controls limit blast radius and exfiltration; identity controls (`authentication.md`, `authorization-and-service-identity.md`) decide trust.

## Private data plane — no public PaaS endpoints

Every PaaS data service gets a **Private Endpoint / Private Link** and has **public network access disabled**: Postgres, Cosmos DB, Key Vault, Storage, Service Bus. Resolve through **Private DNS zones**. The data plane should have no public IP and no "firewall allow-list of our egress IPs" — that is a weaker substitute for Private Link.

```hcl
resource "azurerm_postgresql_flexible_server" "db" {
  public_network_access_enabled = false
}
# + azurerm_private_endpoint + azurerm_private_dns_zone for privatelink.postgres.database.azure.com
```

## VNet and subnet design

- Hub-spoke; each service tier in its own subnet.
- **Container Apps** environment with VNet integration and **internal-only ingress** for east-west services; only the public-facing app gets external ingress.
- **AKS** with Azure CNI and **NetworkPolicies** (Azure or Cilium) in **deny-by-default** mode; allow specific paths explicitly.

## Ingress edge — one public door

`Front Door / Application Gateway (WAF) → APIM → service`. Run the **WAF** with the OWASP ruleset and DDoS protection at the edge. Only the ingress is public; everything behind it is private. Authentication still happens at and behind the gateway (`authentication.md`) — the WAF is not an auth control.

## East-west policy

- **Service mesh** (Istio on AKS, or Open Service Mesh) for mTLS + `AuthorizationPolicy`: each service gets a cryptographic identity and an allow-list of who may call it.
- **NSGs** are coarse subnet guards; use them, but do fine-grained allow/deny with mesh policy or per-service identity, not IP rules.

## Egress control

Default-deny egress through **Azure Firewall** (or a NAT gateway with FQDN allow-lists). Most data exfiltration and command-and-control depends on open egress; an allow-list of required FQDNs closes it.

## The zero-trust checklist

- Inter-service calls use Managed Identity / Workload Identity or mesh mTLS — no shared secrets.
- Data plane is private (Private Link), public access disabled.
- Ingress is the only public surface, behind WAF.
- East-west traffic is authenticated and authorized (mesh policy or per-call identity).
- Egress is default-deny with an FQDN allow-list.
- Every authenticated action is logged (`audit-logging-and-compliance.md`).

## Failure modes

- **Perimeter-trust fallacy** — "it's on the internal VNet, so it's safe." Network isolation alone fails the moment one pod is compromised; pair it with identity.
- **Public data-plane + firewall rules** instead of Private Link — fragile, and a misconfigured rule exposes the store.
- **Allow-all egress** — turns a foothold into exfiltration.
- **Mesh mTLS but the app still trusts `X-Forwarded-User`** — transport identity established, then thrown away by application code.

## Brownfield: collapsing a flat VNet to zero-trust

Add Private Endpoints and disable public access on data services one service at a time (DNS first, then flip public-access off); introduce the mesh in permissive mode, watch the traffic graph, then switch `AuthorizationPolicy` to deny-by-default; add default-deny egress last. Each step is reversible and observable. Full recipe: `references/patterns/zero-trust-service-access.md`.

## Read next

- `references/patterns/zero-trust-service-access.md` — the full pattern with trade-offs and failure modes
- `audit-logging-and-compliance.md` — logging east-west auth decisions for forensics and evidence
