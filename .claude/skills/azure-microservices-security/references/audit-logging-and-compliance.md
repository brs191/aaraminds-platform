# Layer 6 — Audit Logging and Compliance Evidence

> Every state-changing action emits a structured audit event — identity, action, resource, result, correlation id — shipped to Log Analytics with tamper-evident retention. That log *is* your SOC 2 / ISO 27001 evidence. "We have logs" is not the same as "we can answer who accessed order X at 14:03."

## The audit event schema

A separate stream from application logs (different retention and access controls), one event per security-relevant action:

```json
{
  "ts": "2026-05-30T14:03:11Z",
  "actor": { "oid": "8a1f...", "kind": "user|service", "name": "orders-svc" },
  "action": "order.read",
  "resource": { "type": "order", "id": "8123" },
  "result": "allow",
  "reason": "owner-match",
  "trace_id": "0af7651916cd43dd8448eb211c80319c",
  "src_ip": "10.2.3.4"
}
```

- **Both allow and deny** are logged — denies are the signal an auditor and a detection rule care about most.
- **Correlation:** carry the W3C `traceparent` trace id so a request can be reconstructed across services. Without it, a multi-service request is unreconstructable.
- **Never log the secret/PII value** — log that a secret was accessed, by whom, not its contents.

## Pipeline (on the pack's observability stack)

Structured JSON → **OpenTelemetry** → **Log Analytics workspace** (the pack standardizes on OTel + Grafana + Prometheus, with audit/security logs to Azure Monitor / Log Analytics). Retention: **≥1 year** for audit; export to immutable Storage (WORM / legal hold) for longer regulatory windows. Keep the audit stream in a workspace (or table) with tighter RBAC than app logs.

## Platform-side audit (you get these for free — turn them on)

- **Azure resource logs + Activity Log** → Log Analytics (control-plane changes).
- **Microsoft Entra sign-in and audit logs** (who authenticated, what changed in directory/roles).
- **Key Vault logs** (every secret/key access — pairs with `secrets-and-encryption.md`).
- **Microsoft Defender for Cloud** — posture (CSPM) + workload threat protection.
- **Microsoft Sentinel** — SIEM over the above for correlation, analytics rules, and incident workflow.

## Detection and alerting

Alert on the access *patterns* that signal compromise, not just thresholds:

```kql
// Key Vault secrets read by an identity that is not the rotation pipeline
AzureDiagnostics
| where ResourceType == "VAULTS" and OperationName == "SecretGet"
| where identity_claim_oid_g !in (dynamic(["<pipeline-mi-oid>"]))
| summarize count() by identity_claim_oid_g, bin(TimeGenerated, 1h)
```

Also: spike in authorization denials, Entra impossible-travel/risky sign-ins, privilege-escalation role assignments.

## Enforcement as evidence — Azure Policy

Preventive controls double as audit evidence: deny public network access on data services, require HTTPS, require Key Vault RBAC mode, require CMK where mandated. A Policy compliance snapshot is itself an artifact you hand the auditor.

## Producing the evidence package

Do **not** restate SOC 2 Trust Services Criteria or ISO 27001 Annex A control text here — that mapping is the `soc2-iso27001-controls-mapping` skill's job. This layer's contribution is that for each control there is a **named, reproducible evidence source**:

| Control area | Evidence source |
|---|---|
| Logical access (who can read what) | KQL over the audit stream + Entra role assignments export |
| Secret access | Key Vault `SecretGet` query by identity |
| Encryption in transit/at rest | Azure Policy compliance snapshot; Terraform plan |
| Change management | Activity Log + GitHub Actions OIDC deployment records |
| Monitoring/alerting | Sentinel analytics rules + alert history |

Hand the full criterion-to-control mapping to `soc2-iso27001-controls-mapping`; this skill guarantees the evidence the mapping cites actually exists and is queryable.

## Failure modes

- **App stdout with 7-day retention** as the "audit log" → gone before the assessment window opens.
- **Audit log mutable / co-located with app logs** → no tamper-evidence; a compromise edits its own trail.
- **No correlation id** → cannot reconstruct a cross-service request; forensics stalls.
- **Logging the secret value** → the audit log becomes the breach.
- **"We have logs" untested** → nobody has ever run "who accessed order X at time T"; test the query before the auditor does.

## Read next

- `soc2-iso27001-controls-mapping` skill — the explicit TSC / Annex A control-to-implementation mapping
- `azure-microservices-observability` skill — the OTel + Grafana + Log Analytics pipeline these events ride on
