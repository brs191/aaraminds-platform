---
name: azure-network-iac-generation
description: Generate validated Terraform (AzureRM) for an Azure network topology from an architect's intent by selecting and parameterizing vetted modules — never by free-writing network-security Terraform — then validating the generated topology through the reachability analyzer before emitting it as a pull request. Use when an architect wants a deployable hub-spoke or mesh topology, a new spoke or peering, an Azure Firewall or gateway, or AVNM connectivity/security-admin configuration produced as code from requirements; when turning an approved topology design into a PR; or when generating the network half of a landing zone. Do not use for reviewing or risk-analyzing an existing topology (use azure-network-topology-analysis), for cost forecasting (use azure-network-cost-forecasting), for applying changes (this skill emits a PR; humans apply), or for non-Azure clouds.
version: 0.1.0
last_updated: 2026-06-03
---

# Azure Network IaC Generation

## When to use

Use this to turn an architect's network requirements into deployable, validated Terraform — a hub-spoke or mesh topology, a new spoke, a peering, an Azure Firewall or gateway, Private Link, or AVNM connectivity and security-admin rules. It produces a topology spec, renders it from vetted modules, validates the result, and emits a pull request. It is the generation half of the reviewer: `azure-network-topology-analysis` reads deployed topology; this writes proposed topology.

Do not use it to review or risk-score an existing estate (that is `azure-network-topology-analysis`), to forecast cost (`azure-network-cost-forecasting`), to *apply* anything (this emits a PR; a human applies), or for non-Azure clouds.

## Decision rule: select and parameterize vetted modules; the model never authors network-security Terraform

The one thing that, if forgotten, makes this dangerous: **the LLM's output is a topology *spec* and a set of *module parameters* — never hand-written `azurerm_network_security_rule`, route table, or firewall-policy HCL.** Network security has too high a blast radius for free-written IaC. The model selects an approved module (Azure Verified Modules / CAF landing-zone modules / the org's registry), pins its version, and fills its inputs. A deterministic render turns the spec into Terraform from those modules.

Two non-negotiable gates around the output: **validate before emit** — run the generated topology through `azure-network-topology-analysis` and require zero high-severity findings before it leaves the skill; and **PR, never apply** — emit through CI (GitHub Actions + OIDC), where a human reviews and applies. The agent holds no write/apply identity.

## The work

| Stage | What you do | Reference |
|---|---|---|
| 1. Intent → spec | Translate requirements into a structured topology spec (VNets, subnets, peering pattern, firewall/gateway, AVNM config) | `references/spec-and-render.md` |
| 2. Select modules | Map the spec to vetted modules and pin versions; never author security HCL | `references/module-registry.md` |
| 3. Render | Deterministically render the spec + chosen modules into Terraform AzureRM | `references/spec-and-render.md` |
| 4. Validate + emit | Run the generated topology through the analyzer (zero high-severity), then emit a Terraform PR via Actions + OIDC | `references/validate-and-emit.md` |

## Worked example (brownfield)

An existing hub-spoke estate; the architect needs a new PCI spoke peered to the hub, forced-tunnelled through the hub firewall, isolated from other spokes. Generate it:

- **Spec:** one VNet (`10.12.0.0/16`), subnets per tier, hub peering with `allowForwardedTraffic`, a `0.0.0.0/0 → firewall` UDR, and an AVNM security-admin `Deny` for inter-spoke traffic.
- **Modules:** the Azure Verified Module `avm-ptn-alz-connectivity-hub-and-spoke-vnet` for the spoke + peering; `azurerm_network_manager_*` for the AVNM admin rule. Pin versions; fill inputs from the spec. No hand-written NSG HCL.
- **Validate:** run the rendered topology's graph through `azure-network-topology-analysis` — it must show the spoke isolated (no transitive path) and no internet exposure before emit.
- **Emit:** a PR; a human reviews and applies. The agent never runs `terraform apply`.

## Anti-patterns

- **LLM-authored network-security Terraform.** Free-writing NSG rules, route tables, or firewall policy. Detection: generated HCL not traceable to a pinned module input. Fix: spec + module selection only.
- **Auto-apply.** The agent running `terraform apply` or holding a write identity. Fix: emit a PR; humans apply through CI.
- **Emit without validation.** Producing Terraform without running it through the analyzer. Fix: zero high-severity findings is a hard gate before emit.

## Verification questions

1. Is every security-relevant resource produced by a pinned, vetted module — not hand-written HCL?
2. Did you validate the generated topology through `azure-network-topology-analysis` and clear all high-severity findings before emit?
3. Is the output a pull request (Actions + OIDC), with no apply and no write identity on the agent?
4. Are module versions pinned (not `latest`)?
5. For enforcement at scale, did you target AVNM (connectivity + security-admin rules) rather than hand-rolled peering/NSG sprawl?
6. Does the generated spec round-trip — would the analyzer read back the intent the architect asked for?

## What to read next

- The three references above, in stage order.
- `azure-network-topology-analysis` — the validator this skill calls before emit, and the reader of what it deploys.
- `azure-network-cost-forecasting` — price the generated change before the PR merges.
- `new-azure-service-bootstrap` and `pr-review-azure-microservices` — the pack's scaffolding and Terraform-review skills that complement the PR path.
