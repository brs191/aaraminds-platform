# Layers 4 & 5 — Secret Management and Data Protection

> Azure Key Vault is the *only* home for secret values — in RBAC mode, read by a Managed Identity, with config holding references and never values. Encrypt in transit (TLS 1.2+) and at rest (platform keys by default; customer-managed keys for regulated data).

## The Key Vault rule

One source of truth for secrets, no fallback. Configuration files and Terraform contain Key Vault **references** (vault URI + secret name); they never contain the value. There is no "Key Vault, or the env var if that fails" — a fallback is just a secret in an env var with extra steps. The best secret is the one that does not exist: prefer Managed Identity over any stored credential (see `authorization-and-service-identity.md`).

## Key Vault configuration

- **RBAC mode**, not legacy access policies (`enable_rbac_authorization = true`). RBAC gives per-identity least privilege and fine-grained Key Vault audit; access policies are coarse and unauditable.
- Runtime identities get **Key Vault Secrets User** (read only). Rotation pipelines get **Secrets Officer**. Nothing runs as vault admin.
- **Soft-delete + purge protection on** — without purge protection a deleted/compromised vault can be permanently destroyed, taking your CMKs with it.
- One vault per environment and per blast-radius; a **private endpoint** so the vault has no public data plane.

```hcl
resource "azurerm_key_vault" "kv" {
  name                       = "kv-orders-prod"
  enable_rbac_authorization  = true
  purge_protection_enabled   = true
  soft_delete_retention_days = 90
  public_network_access_enabled = false
}
```

## Access patterns by host

- **Container Apps:** Key Vault references in the app's `secrets` (`keyvaultref:<uri>,identityref:<mi>`), or the SDK with the app's Managed Identity. The platform resolves the reference at runtime.
- **AKS:** the **Secrets Store CSI driver** with the Key Vault provider + Workload Identity, mounting secrets into the pod. Enable sync-to-K8s-Secret only if a consumer truly needs a native Secret — every sync is another copy to protect.
- **Java:** `azure-identity` + `azure-security-keyvault-secrets` `SecretClient` with `DefaultAzureCredential`, or the Spring Cloud Azure Key Vault property source.
- **Go:** `azsecrets` client with a Managed Identity credential.

## Rotation

- Prefer credentials that rotate themselves: **Managed Identity tokens** are short-lived and auto-renewed — using them eliminates the rotation problem.
- For unavoidable secrets (third-party API keys, partner credentials): **event-driven rotation** — Event Grid emits a `SecretNearExpiry` event from Key Vault, a rotation function mints/sets the new version, consumers fetch by latest version. No redeploy.
- Version-aware retrieval; alert on secrets approaching expiry.

## Encryption in transit

- **TLS 1.2+ everywhere**; enforce HTTPS-only (`httpsOnly`/min TLS) on every front end and PaaS service.
- Terminate TLS at Front Door / Application Gateway / APIM and **re-encrypt to the backend** (end-to-end TLS) for sensitive paths — do not let the last hop run plaintext inside the VNet.
- East-west: mTLS via the service mesh for zero-trust transport (see `network-segmentation-and-zero-trust.md`).

## Encryption at rest

- **Platform-managed keys** are on by default — Azure Storage, Azure SQL (TDE), Cosmos DB, and Postgres all encrypt at rest with no action. This satisfies most controls.
- **Customer-managed keys (CMK / BYOK)** in Key Vault or Key Vault Managed HSM when a control requires *you* to own the key lifecycle: you gain revocation and key-level audit, you take on key-rotation and availability ops. Revoke the key and the data service stops — monitor the key as a production dependency.
- **Field/column level** for data even the DBA must not read: **Always Encrypted** (Azure SQL) keeps plaintext out of the engine; app-layer field encryption for Cosmos. Use for PII/PCI fields, not whole tables.
- **Infrastructure (double) encryption** for the highest regulatory bar.

## Certificates

Store certificates in **Key Vault Certificates**; integrate an issuer CA (e.g. DigiCert/GlobalSign) for automatic renewal, or use short-lived certs. App Service and Application Gateway pull directly from Key Vault — no PEM files in the repo or in pipeline variables.

## Failure modes

- **Key Vault in access-policy mode** → no least-privilege, weak audit; the lowest-effort fix with the highest control payoff is flipping to RBAC.
- **Secrets synced into K8s Secrets and then logged** → the value lands in Log Analytics. Limit sync; redact logs.
- **CMK with no monitoring** → someone rotates/revokes the key and the database goes dark with a cryptic error.
- **Connection string in `application.yml` "just for dev"** → the path of least resistance carries it to prod. (The skill's named anti-pattern.)

## Read next

- `network-segmentation-and-zero-trust.md` — keeping the data plane private and east-west traffic authenticated
- `audit-logging-and-compliance.md` — evidence that secrets are accessed only by the right identities
