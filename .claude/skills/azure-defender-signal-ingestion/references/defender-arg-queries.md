# Pulling Defender signals via Azure Resource Graph

Stage 1. Defender for Cloud exposes attack paths and exposure through the Azure Resource Graph (ARG) `securityresources` table and the Cloud Security Explorer. Query them; do not recompute them.

## Attack paths (`microsoft.security/attackpaths`)

```kusto
securityresources
| where type == "microsoft.security/attackpaths"
| where subscriptionId == "<SUBSCRIPTION_ID>"
| extend name = tostring(properties["displayName"]),
         impact = tostring(properties["potentialImpact"]),
         risk   = properties["riskCategories"]
```
Filter to a specific path by `properties.displayName` (e.g. *"Internet exposed VM with high severity vulnerabilities and read permission to a Key Vault"*).

ARG returns **only externally-driven, exploitable** attack paths (real threats, not every theoretical scenario). Key response fields to consume:

| Field | Use |
|---|---|
| `properties.displayName` / `description` | human-readable path |
| `properties.attackPathType` | category of path |
| `properties.potentialImpact` | blast-radius hint for prioritization |
| `properties.riskCategories` | risk classification |
| `properties.entryPointEntityInternalID` / `targetEntityInternalID` | the path's start (often internet) and crown-jewel target |
| `properties.graphComponent.{entities,connections,insights}` | the actual graph to reconcile against antr's nodes/edges |

## Internet-exposure signals

Consume Defender's internet-exposure analysis (control-plane config + network-path reachability over routing/security/firewall rules) rather than re-deriving it. Surface exposed resources and "exposure width" via Cloud Security Explorer queries / the exposure recommendations. (Defender EASM adds active external-scan confirmation where enabled.)

## Cloud Security Explorer

For ad-hoc/relational questions ("internet-exposed resources with a path to a sensitive data store"), use Cloud Security Explorer rather than hand-writing graph traversals — it is Defender's supported query surface over the security graph.

## Join key

Every consumed signal carries Azure **resource IDs** (entry point, target, graph entities). Keep those IDs intact — they are the join key to antr's findings (stage 3). Map Defender's internal entity IDs to ARM resource IDs via `graphComponent.entities`.

## Done when

You can pull attack paths + exposure for a subscription via ARG/Explorer, you preserve resource IDs and graph components for reconciliation, and you only ingest externally-exploitable paths.
