---
name: azure-microservices-security
description: Designs and reviews defense-in-depth security for Azure-hosted microservices: authentication (OAuth 2.1 / Entra ID), authorization and object-level access, service-to-service identity (Managed Identity, Workload Identity), secrets via Key Vault, encryption in transit and at rest, network segmentation, and zero-trust service access. Use when securing a new service, reviewing one against the SOC 2 / ISO 27001 bar, planning a zero-trust migration, or producing audit evidence. Do not use for MCP-server threat modeling (use mcp-go-threat-modeling) or code-level PR review (use pr-review-azure-microservices).
version: 1.1.0
last_updated: 2026-05-30
---

# Azure Microservices Security

## When to use

Trigger this skill when the question is about defending an Azure microservices system: authentication and authorization design, service-to-service identity (Managed Identity, Workload Identity, mTLS), secret management with Key Vault, encryption design, network segmentation, audit logging for SOC 2 / ISO 27001 evidence, or planning a zero-trust migration for an existing system.

Do **not** use this skill for: MCP-server-specific threat modeling (use `mcp-go-threat-modeling`); code-level security review of a PR (use `pr-review-azure-microservices`); penetration-testing tactics (out of scope for this design-focused pack).

## The critical decision rule — defense in depth, not perimeter

Security is not a single gate. Assume every layer can be breached. Design so that the failure of any one control still leaves attackers contained.

This is not philosophy. It is the difference between an incident where one breached credential gives the attacker your entire customer database, and an incident where the breached credential lets them read three rows of one table before audit logs trigger an alert and Key Vault rotation cuts them off.

The pack ships with SOC 2 / ISO 27001 in scope. Defense in depth is also the auditor's expectation; controls map to multiple Trust Service Criteria.

## The six defensive layers

| # | Layer | What it does | Azure primitives |
|---|---|---|---|
| 1 | Ingress authentication | Establishes who the client is | Entra ID with OAuth 2.1 (user-facing); API keys via APIM (service-to-service when OAuth is heavyweight); mTLS for B2B |
| 2 | Authorization | Decides what the authenticated client may do | Per-endpoint scope checks; RBAC inside the service; Entra ID groups + claims |
| 3 | Service-to-service identity | Lets services authenticate each other without shared secrets | System-assigned or user-assigned Managed Identity (Container Apps, AKS, App Service); Workload Identity on AKS |
| 4 | Secret management | Removes secrets from code, config, and environment | Azure Key Vault as single source of truth; RBAC mode (not access policies); rotation enforced |
| 5 | Data protection | Protects data in transit and at rest | TLS 1.2+ everywhere; encryption-at-rest via service-default keys (or customer-managed keys for regulated data); column-level encryption where needed |
| 6 | Audit and monitoring | Detects breaches and produces SOC 2 / ISO 27001 evidence | Every state-changing operation logged with who/when/what/result; Log Analytics workspace with retention sufficient for audit; alerts on anomalous access patterns |

If any layer is missing, the design is incomplete. Each layer has a specific failure mode it prevents. `references/security-design.md` maps the six layers; each layer's depth lives in a focused reference (see *What to read next*).

## The zero-trust principle

Zero-trust means no implicit trust based on network location. A service inside the cluster is treated the same as a service on the public internet: every call is authenticated, every call is authorized, every call is logged.

Concretely on Azure:

- **Inter-service:** Managed Identity (Container Apps, App Service) or Workload Identity (AKS) on every call; no shared secrets, no hardcoded API keys
- **Network:** private endpoints for PaaS services where the SKU supports them; service mesh (Istio on AKS, Open Service Mesh) for fine-grained policy on east-west traffic
- **Data plane:** Azure SQL with Entra ID auth + RBAC; Cosmos DB with Entra ID + role assignments; storage with Managed Identity access
- **Audit:** every authenticated action logged to Log Analytics with identity context

For the zero-trust pattern card with full Azure mapping, see `references/patterns/zero-trust-service-access.md`.

## Worked example — brownfield: upgrading an existing Spring Boot service to managed identity

Setup: existing Spring Boot order service on AKS, using a service principal with a client secret stored in a Kubernetes secret. Audit finding flagged the long-lived secret as a SOC 2 CC6.1 (Logical Access) concern. Migrate to Workload Identity without service downtime.

Decision walk:

1. **Inventory all credentials the service holds.** Find them in `application.yml`, Kubernetes secrets, environment variables. For the order service: Azure SQL connection string, Service Bus connection string, Cosmos DB key, Storage account key.
2. **Replace each connection string with Entra-ID-based auth.** Azure SQL: switch from SQL auth to Entra ID auth, use `DefaultAzureCredential` in JDBC. Service Bus: use `azure-identity` + `azure-messaging-servicebus` with Managed Identity. Cosmos DB and Storage: same pattern with the Azure SDK for Java.
3. **Enable Workload Identity on the AKS cluster.** Annotate the service account; create a federated credential on the user-assigned Managed Identity that trusts the service account's OIDC issuer.
4. **Grant the Managed Identity RBAC roles** on each Azure resource (SQL DB Contributor, Service Bus Data Sender/Receiver, Cosmos DB Built-in Data Contributor, Storage Blob Data Contributor as needed). RBAC mode for Key Vault — no access policies.
5. **Deploy behind a feature flag.** Run the new auth code path in parallel with the old one for 1-2 weeks. Watch for auth errors in Log Analytics. Cut over fully once green.
6. **Rotate and revoke the old service principal's secret.** Do not leave it active "just in case" — that's the original audit finding regenerated.
7. **Update the SOC 2 evidence package.** Screenshot the federated credential, the RBAC role assignments, the Key Vault RBAC, and the Log Analytics audit query proving identity-based access is being used.

References: `references/authorization-and-service-identity.md` (layer 3 service-to-service identity), `references/secrets-and-encryption.md` (layer 4 secret management), `references/patterns/zero-trust-service-access.md` (full migration recipe with AKS specifics).

## Anti-pattern — "we'll use Key Vault but the connection string is also fine in config for now"

**Bad:** A service is set up to use Key Vault for "real" secrets, but the database connection string sits in `application.yml` committed to the repo "because it's only the dev environment for now." Three months later, the same service is in production with the same config file structure, and now the prod connection string is in the prod config.

**Why it fails:** Connection strings are credentials. The "dev environment for now" pattern means the path of least resistance to production is "copy the dev structure." The first prod outage that requires a hotfix will introduce the prod connection string into a Kubernetes secret, where it lives forever, granting full DB access to anyone with cluster access.

**Detection signal:** any `application.yml`, `appsettings.json`, Terraform variable, or Kubernetes manifest contains a value that looks like a connection string, an API key, or a token. Or: a Key Vault is referenced in code but the actual secret is also in an environment variable as a "fallback."

**Fix:** Key Vault is the only source of secret values, ever. No fallback. Configuration files contain references (Key Vault URIs, secret names) but never values. Local development uses the developer's own Managed Identity or a `.env` file that is git-ignored and never produced in a deployment artifact. If a developer needs to know a secret value to debug, they have IAM access to read it from Key Vault directly, not from config.

## Verification questions

1. Are there any secret values in code, config files, environment variables, or Kubernetes manifests? (Should be zero.)
2. Does every service-to-service call authenticate via Managed Identity / Workload Identity rather than a shared secret?
3. Is Key Vault in RBAC mode (not the legacy access-policies mode)?
4. For every state-changing operation: is there an audit log with identity, action, resource, and result?
5. For data at rest: is encryption enabled, and for regulated data, are customer-managed keys in use?
6. For SOC 2 / ISO 27001: does each control have a named evidence source (a query in Log Analytics, a screenshot of a portal page, a Terraform plan)?

## What to read next

- `references/security-design.md` — the six-layer overview and map
- `references/authentication.md` — ingress authentication: OAuth 2.1 / Entra ID, token validation, APIM, mTLS
- `references/authorization-and-service-identity.md` — authorization (incl. object-level) and Managed Identity / Workload Identity
- `references/secrets-and-encryption.md` — Key Vault (RBAC mode), rotation, encryption in transit and at rest
- `references/network-segmentation-and-zero-trust.md` — private endpoints, segmentation, the zero-trust model
- `references/audit-logging-and-compliance.md` — audit event schema, Log Analytics, Defender/Sentinel, evidence sources
- `references/patterns/zero-trust-service-access.md` — concrete zero-trust pattern with AKS + Workload Identity recipe
- `microservices-architecture-design` skill — for stage 10 (security as part of the design sequence)
- `mcp-go-threat-modeling` skill — when the service in question is an MCP server (different threat surface)
- `soc2-iso27001-controls-mapping` skill — the explicit control-to-implementation mapping required for audit
