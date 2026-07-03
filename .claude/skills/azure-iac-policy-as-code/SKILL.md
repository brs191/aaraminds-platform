---
name: azure-iac-policy-as-code
description: Gate Azure Terraform (AzureRM) against organizational policy and security baselines using adopted policy-as-code tools — Checkov for the security baseline and OPA/Conftest (Rego) for custom org rules, evaluated over the terraform plan JSON in CI — never by hand-rolling a compliance engine. Use whenever generated or proposed Terraform must pass compliance/guardrail checks before merge or apply: encryption-at-rest, public-exposure denial, tagging, IAM least-privilege, naming, region allow-lists, or any "policy" / "compliance gate" / "OPA" / "Checkov" / "Conftest" / "Sentinel" / "block this misconfig" request. This is the BROAD compliance surface that runs ALONGSIDE the network reachability gate — do not use it for reachability/exposure analysis (use azure-network-topology-analysis), for generating the Terraform (use azure-network-iac-generation), for mapping findings to audit controls (use soc2-iso27001-controls-mapping), or for non-Azure clouds.
version: 1.0.0
last_updated: 2026-06-15
---

# Azure IaC Policy-as-Code

## When to use

Use this when Terraform (AzureRM) — hand-written or machine-generated — must clear a **policy/compliance gate** before it merges or applies: the broad surface of encryption, public-exposure denial, tagging, IAM scope, region allow-lists, SKU restrictions, and org naming. It covers selecting, configuring, and CI-gating the adopted tools — **Checkov** (security baseline) and **OPA/Conftest** (custom Rego) — over the `terraform plan` JSON.

Do not use it for: computing network reachability/exposure (that is `azure-network-topology-analysis` — a different axis, see the Decision rule); generating the Terraform itself (`azure-network-iac-generation`); mapping findings to SOC 2 / ISO 27001 controls for an auditor (`soc2-iso27001-controls-mapping`); or non-Azure clouds.

## Decision rule: adopt the engines, own only the policies — and never conflate the two gates

The one thing that, if forgotten, makes this skill counterproductive: **do not build a compliance engine; adopt Checkov + OPA/Conftest and write only the policies.** These tools have thousands of maintained checks and a Rego ecosystem; a hand-rolled scanner is a worse copy you will maintain forever.

The second load-bearing rule: **policy-as-code and the reachability analyzer are two different gates that compose, not substitutes.** Checkov/OPA answer "does this config violate a *rule*?" (a property of the declared resource). The reachability analyzer answers "can this actually be *reached* from the internet?" (a property of a computed path across NSG + route + peering + firewall). A static linter cannot compute true end-to-end reachability; the reachability engine does not check encryption-at-rest or tagging. Run **both** in the generate/CI pipeline; let each own its axis.

Scan the **plan JSON**, not just the HCL. `terraform show -json plan.binary` resolves variables, modules, and computed values that raw `.tf` hides — the difference between catching a misconfig and missing it.

## The work

Run the pipeline in order; each stage routes to a reference for depth.

| Stage | What you do | Reference |
|---|---|---|
| 1. Security baseline | Run Checkov over HCL **and** plan JSON; triage by severity; suppress with justification, never blanket-skip | `references/checkov-baseline.md` |
| 2. Custom org policy | Write Rego for AT&T-specific rules (region allow-list, mandatory tags, private-only, approved SKUs) and run via Conftest over plan JSON | `references/opa-conftest-rego.md` |
| 3. CI gate | Wire `terraform plan -out` → `terraform show -json` → Checkov + Conftest as a required check; pin tool + policy-bundle versions | `references/ci-gating-plan-json.md` |
| 4. Compose with reachability | Place the policy gate **beside** the analyzer's reachability gate in `generate_topology`; define which gate owns which failure | `references/policy-vs-reachability-boundary.md` |

Optional runtime backstop: Azure Policy enforces the same intents at deploy time (defense in depth) — note it, but pre-merge gating is this skill's job.

## Worked example (generate pipeline)

`generate_topology` produces Terraform for a new spoke with a storage account and an NSG. Two independent gates run on the plan JSON:

- **Policy gate (this skill):** Checkov flags `CKV_AZURE_*` — storage account allows public blob access and lacks `min_tls_version`; Conftest fails a custom rule because the subnet has no `environment` tag. These are *rule* violations, caught regardless of reachability.
- **Reachability gate (`azure-network-topology-analysis`):** the analyzer projects the topology and finds the NSG rule is not actually internet-reachable (routed through the firewall) — so it raises *no* high-severity reachability finding.

Both verdicts are correct and neither subsumes the other: the storage account is a real policy failure even though the NSG path is safe. The PR is blocked on the policy gate. Conflating the two would have shipped the public-blob storage account because "the network path was clean."

## Anti-patterns

- **Hand-rolling compliance checks in antr.** Re-implementing what Checkov/OPA already maintain. Detection signal: bespoke "does this resource have encryption" code in the engine. Fix: adopt the tools; contribute org rules as Rego/Checkov custom policies.
- **Scanning HCL only.** Missing misconfigs hidden behind variables/modules/computed values. Fix: scan `terraform show -json` plan output.
- **Conflating the policy gate with the reachability gate.** Claiming the reachability analyzer "covers compliance," or vice versa. Fix: two gates, two axes, both required.
- **Blanket suppressions.** `skip-check` without a justification comment erodes the baseline. Fix: per-resource suppressions with a reason and an owner.
- **Unpinned policy bundles.** Floating Checkov/policy versions make the gate non-reproducible. Fix: pin tool + bundle versions (mirrors the engine's determinism discipline).

## Verification questions

1. Are **both** gates present in the pipeline — policy-as-code (Checkov + OPA) **and** the reachability analyzer — with neither claiming to cover the other's axis?
2. Is the scan run over **plan JSON** (`terraform show -json`), not just raw `.tf`?
3. Are custom org rules expressed as **Rego/Conftest** (or Checkov custom policies), not hard-coded into antr?
4. Are tool and policy-bundle versions **pinned** so the gate is reproducible?
5. Are suppressions **justified per-resource** with an owner, not blanket skips?
6. Did you reserve Azure Policy for the **runtime** backstop rather than treating it as the pre-merge gate?

## What to read next

- The four references above, in pipeline order.
- `azure-network-iac-generation` — produces the Terraform this skill gates; the policy gate sits in its emit pipeline.
- `azure-network-topology-analysis` — the *other* gate (reachability); composes with this one, never replaces it.
- `soc2-iso27001-controls-mapping` — map policy/reachability findings to audit controls for reporting.
- `mcp-go-server-building` — to expose the policy gate result alongside `analyze_risks` / `generate_topology`.
