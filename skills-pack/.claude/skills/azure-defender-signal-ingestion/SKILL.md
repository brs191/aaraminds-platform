---
name: azure-defender-signal-ingestion
description: Consume Microsoft Defender for Cloud's exposure and attack-path signals — internet-exposed resources, attack paths, and the cloud security graph — by querying them through Azure Resource Graph (the securityresources table) and Cloud Security Explorer, then reconcile them with antr's deterministic findings instead of recomputing what Defender already provides. Use whenever the task involves consuming or integrating Defender for Cloud / Microsoft Defender CSPM signals, attack-path data, internet-exposure analysis, the cloud security graph, or deciding whether antr should compute exposure itself or defer to Defender. This honors the project's "consume Defender, don't reimplement" rule. Do not use it to compute reachability from scratch (use azure-network-topology-analysis — the fallback when Defender CSPM is not licensed), to visualize the topology (use azure-network-topology-visualization), to gate Terraform on policy (use azure-iac-policy-as-code), or for non-Azure clouds.
version: 1.0.0
last_updated: 2026-06-15
---

# Azure Defender Signal Ingestion

## When to use

Use this when antr should **consume** Microsoft Defender for Cloud's already-computed security signals — internet-exposure analysis, attack paths, and the cloud security graph — rather than recomputing them. It covers querying those signals via Azure Resource Graph (ARG) and Cloud Security Explorer, authenticating read-only, detecting whether Defender CSPM is even licensed, and reconciling Defender's output with antr's deterministic findings so the two don't duplicate or contradict.

Do not use it to: compute reachability/exposure from config (that is `azure-network-topology-analysis` — the deterministic fallback for subscriptions without Defender CSPM); draw the topology (`azure-network-topology-visualization`); gate Terraform on policy (`azure-iac-policy-as-code`); or non-Azure clouds.

## Decision rule: consume where licensed, compute where not — dedupe, never duplicate

The one thing that, if forgotten, defeats the purpose: **where Defender for Cloud CSPM is licensed, consume its exposure/attack-path signals; do not recompute what Defender already provides.** Defender's internet-exposure analysis already evaluates control-plane config plus network-path reachability (routing, security and firewall rules), and its attack-path engine already ranks risk. Re-deriving that is the exact "reimplementation" the project forbids.

Two corollaries:

- **Determinism boundary.** Defender's scores are authoritative but **change over time and are not reproducible** — never put a raw Defender severity inside a hard, byte-reproducible CI gate. Use Defender signals for enrichment and prioritization; keep antr's deterministic engine as the gate of record.
- **Fallback, not dependency.** A large fraction of subscriptions run only **free foundational CSPM**, where attack-path analysis is unavailable. Detect licensing first; fall back to antr's deterministic engine when Defender data is absent. antr must work without Defender.

antr's engine still earns its keep on the findings Defender does **not** emit — CIDR overlap and missing tier segmentation — and as the deterministic, license-free, explainable path for everyone else.

## The work

| Stage | What you do | Reference |
|---|---|---|
| 1. Pull the signals | Query attack paths (`securityresources` / `microsoft.security/attackpaths`) and internet-exposure signals via ARG; expand graph components (entry point, target, connections) | `references/defender-arg-queries.md` |
| 2. Auth + licensing | Read-only access (Managed Identity / OIDC; Reader + Security Reader); detect whether Defender CSPM is enabled before relying on its data | `references/auth-and-licensing.md` |
| 3. Reconcile with the engine | Join Defender signals to antr findings by resource id; map Defender risk → antr severity; dedupe overlaps; keep antr-only findings (CIDR overlap, segmentation) | `references/reconcile-with-engine.md` |
| 4. Fallback + determinism | Where Defender is absent or for the CI gate of record, use antr's deterministic engine; keep Defender as enrichment | `references/fallback-and-determinism.md` |

## Worked example (enrichment + fallback)

Two subscriptions, one report.

- **`sub-prod` (Defender CSPM licensed):** ARG returns a `microsoft.security/attackpaths` instance — "Internet-exposed VM with high-severity vulnerabilities and read permission to a Key Vault." antr **consumes** it: it does not recompute the VM's internet exposure; it joins the attack path to the VM resource id, surfaces Defender's path + potential impact, and *adds* its own deterministic finding that the VM's subnet has a CIDR overlap with a peered VNet (which Defender doesn't emit). One reconciled report, no duplicated exposure verdict.
- **`sub-dev` (free foundational CSPM):** the ARG attack-path query returns nothing — attack-path analysis isn't licensed. antr **falls back** to its deterministic engine and computes internet-exposure reachability itself. The report notes the source ("antr engine, Defender CSPM not enabled") so the reader knows the provenance.

The CI exposure gate uses antr's deterministic engine in both cases; Defender data enriches, it does not gate.

## Anti-patterns

- **Recomputing exposure where Defender already provides it.** Running antr's internet-exposure analysis on a sub that has Defender CSPM and ignoring Defender's signal. Detection: two internet-exposure verdicts for the same resource. Fix: consume Defender's; reserve the engine for antr-only findings + unlicensed subs.
- **Putting Defender severity in a hard CI gate.** Defender scores change; the gate becomes non-reproducible. Fix: gate on the deterministic engine; Defender enriches.
- **Assuming Defender CSPM is licensed.** The query returns empty and the integration silently shows "no exposure." Fix: detect licensing; fall back explicitly and label provenance.
- **Ignoring ARG propagation lag.** Defender/ARG data can lag (hours). Fix: treat it as near-real-time enrichment, not a live oracle; note freshness.
- **Duplicating findings.** Emitting both Defender's and antr's verdict for the same exposure as two separate risks. Fix: dedupe by resource id; show one reconciled finding with both sources cited.

## Verification questions

1. For each subscription, did you **detect Defender CSPM licensing** before relying on its attack-path data, and fall back to the engine (with labeled provenance) when absent?
2. Are Defender signals used for **enrichment/prioritization**, with antr's **deterministic engine as the CI gate of record** (no Defender score in a hard gate)?
3. Are Defender exposure verdicts and antr findings **deduped by resource id** so the same exposure isn't double-reported?
4. Did you **keep antr-only findings** (CIDR overlap, missing tier segmentation) that Defender does not emit?
5. Is access **read-only** (Managed Identity / OIDC, Reader + Security Reader) with no `AZURE_CLIENT_SECRET`?
6. Did you account for **ARG data freshness/lag** and label it in the report?

## What to read next

- The four references above, in pipeline order.
- `azure-network-topology-analysis` — the deterministic engine that fills the gaps and is the gate of record (the fallback).
- `azure-network-topology-visualization` — paint reconciled severity (Defender + engine) onto the topology.
- `soc2-iso27001-controls-mapping` — map reconciled exposure/attack-path findings to audit controls.
