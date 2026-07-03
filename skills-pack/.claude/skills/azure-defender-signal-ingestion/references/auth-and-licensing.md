# Auth and licensing detection

Stage 2. Two prerequisites before relying on Defender data: read-only access, and confirmation that Defender CSPM is actually enabled (or the signals simply won't exist).

## Read-only auth (AaraMinds standard)

- Authenticate with **Managed Identity / OIDC** (`DefaultAzureCredential`), never `AZURE_CLIENT_SECRET`.
- Roles: **Reader** (subscription/management-group scope) for ARG, plus **Security Reader** for Defender for Cloud assessments/recommendations. No write, no remediation actions from antr.
- ARG queries can span a management group — pass the subscription set, mirroring the topology adapter's scope.

## Detect Defender CSPM licensing FIRST

Attack-path analysis and the full exposure graph require the **Defender CSPM plan**; **free foundational CSPM does not include attack paths**. If you query `microsoft.security/attackpaths` on a sub without the plan, you get an empty result that looks identical to "no risk" — a dangerous silent failure.

Detect before relying:
- Check the Defender plans on the subscription (pricing/plan state) and treat `attackpaths` as available only where the CSPM plan is on.
- Record the licensing state per subscription so the report can label provenance ("Defender CSPM enabled" vs "antr engine fallback").

## Freshness

ARG/Defender data is near-real-time but can lag (hours) after config changes. Treat it as enrichment, capture a timestamp, and label freshness — do not present it as a live oracle.

## Done when

Access is Managed-Identity/OIDC read-only (Reader + Security Reader, no secret), Defender CSPM licensing is detected per subscription before its data is trusted, and data freshness is captured.
