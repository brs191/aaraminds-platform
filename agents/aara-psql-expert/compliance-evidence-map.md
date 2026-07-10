# Compliance Evidence Map — aara-psql-expert

Regulatory classification is agent-specific and requires qualified legal review; this map is evidence, not legal advice.

## AI Act Role Assessment

Assessed role: deployer. Rationale: the agent uses a foundation model as an assistive coding aid without substantial modification, produces SQL/PL-pgSQL drafts for human review, and takes no autonomous action — it does not meet an Annex III high-risk use case. Autonomy level 2 (Drafting); no fully autonomous execution. Reassess if the model is substantially fine-tuned (which could reclassify AaraMinds as a provider) or if the agent is deployed into a regulated-sector high-risk workflow. This assessment is evidence for legal review, not legal advice.

## ISO 42001 Registry Fields

| Field | Value |
|---|---|
| Purpose | Reviewed, evidence-grounded PL/pgSQL and SQL drafts produced from provided schema |
| Owner | Raja Shekar Bollam |
| Lifecycle state | draft |
| Risk tier | medium |
| Review date | 2026-10-05 (quarterly; next review coincides with pilot approval) |

## NIST AI RMF Function Mapping

- Govern: catalog record, owners, approval boundaries, kill switch.
- Map: intake business problem, data domains, risk classification inputs.
- Measure: evaluation plan categories and release-gate thresholds.
- Manage: readiness verdict, sign-offs, incident and rollback paths.

## Open Compliance Questions

- Jurisdiction: primary users are internal engineering (EU/US). As a deployer of a non-high-risk assistive tool, transparency obligations apply (users know they are working with AI-generated drafts); no Annex III classification identified. Reassess before any external or regulated-sector deployment.
- Data retention: engagement-scoped memory is retained for the engagement duration and purged on engagement close; PII is disallowed by the manifest (pii_allowed=false, memory scope=engagement). Retention aligns with the client's data-handling terms; confirm per engagement.
