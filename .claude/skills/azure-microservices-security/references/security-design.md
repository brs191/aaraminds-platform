# Defense-in-Depth Security — Overview and Map

> Design security as layers, not a single gate. Assume any one control can be breached; design so the failure of one still leaves attackers contained. This file is the overview and router; each layer's depth lives in a focused reference below.

## The defense-in-depth principle

Security is not a perimeter. The difference between an incident where one breached credential exposes the entire customer database and one where it reads three rows before audit logging alerts and Key Vault rotation cuts it off is *layering*. Each layer has a specific failure it prevents; if a layer is missing, the design is incomplete. The layers also map cleanly onto SOC 2 / ISO 27001 expectations — controls span multiple Trust Service Criteria.

## The six layers and where each is designed

| # | Layer | What it does | Azure primitives | Deep dive |
|---|---|---|---|---|
| 1 | Ingress authentication | Establishes who the caller is | Entra ID + OAuth 2.1; APIM `validate-jwt`; mTLS for B2B | `authentication.md` |
| 2 | Authorization | Decides what the caller may do (incl. object-level) | Entra app roles/groups; scope/role claims; OPA for ABAC | `authorization-and-service-identity.md` |
| 3 | Service-to-service identity | Lets services authenticate without shared secrets | System/user-assigned Managed Identity; Workload Identity (AKS); mesh mTLS | `authorization-and-service-identity.md` |
| 4 | Secret management | Removes secrets from code, config, env | Key Vault (RBAC mode); Managed Identity access; rotation | `secrets-and-encryption.md` |
| 5 | Data protection | Protects data in transit and at rest | TLS 1.2+; platform keys or CMK; Always Encrypted | `secrets-and-encryption.md` |
| 6 | Audit and monitoring | Detects breaches and produces audit evidence | Log Analytics; Defender for Cloud; Sentinel; Azure Policy | `audit-logging-and-compliance.md` |

Network segmentation and the end-to-end zero-trust model cut across all six — see `network-segmentation-and-zero-trust.md` and the `references/patterns/zero-trust-service-access.md` pattern card.

## Zero-trust in one line

No implicit trust from network location: a service inside the cluster is treated like one on the public internet — every call authenticated, every call authorized, every call logged. Identity controls (layers 1–3) decide trust; network controls (private endpoints, deny-by-default) limit blast radius. Neither alone is sufficient.

## How to use this skill

1. New service: walk layers 1→6, picking the Azure primitive per layer from the table; keep the data plane private from the start.
2. Review / audit: score each layer pass/soft-fail/hard-fail; a missing layer is a hard-fail.
3. Zero-trust migration: start from `network-segmentation-and-zero-trust.md` and the pattern card.
4. Evidence: `audit-logging-and-compliance.md` for the named sources, then `soc2-iso27001-controls-mapping` for the control mapping.

## Read next

- `authentication.md` · `authorization-and-service-identity.md` · `secrets-and-encryption.md` · `network-segmentation-and-zero-trust.md` · `audit-logging-and-compliance.md`
- `references/patterns/zero-trust-service-access.md` — zero-trust pattern card
- Related skills: `soc2-iso27001-controls-mapping` (control mapping) · `azure-microservices-observability` (the logging pipeline) · `mcp-go-threat-modeling` (MCP-server threat surface)
