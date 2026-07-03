# Validate the generated topology, then emit a PR (never apply)

Generation closes the loop only if the thing it produces is checked by the same engine that reviews deployed topology — and only if a human, not the agent, applies it.

## Validate before emit (hard gate)

Build the graph the rendered Terraform *would* produce (from the spec / `terraform plan` output) and run it through `azure-network-topology-analysis`. The generated topology must clear two checks before it can be emitted:

1. **Zero high-severity findings.** No internet-reachable path to a sensitive tier, no transitive-peering exposure, no over-permissive reachable NSG, no CIDR overlap. If the analyzer flags a high, the generation is wrong — fix the spec and re-render, don't emit and hope.
2. **Intent round-trips.** The analyzer's read-back must match what the architect asked for: if the spec said "spoke isolated from other spokes," the analyzer must confirm no spoke-to-spoke path exists; if it said "web reaches app only," the analyzer must confirm web→db is closed. Generating the intent and validating the intent on one engine is the whole point.

Run the mechanical checks too — `terraform validate`, `terraform plan`, `tflint` — but those catch syntax, not exposure. The analyzer is what catches a topology that deploys cleanly and is still wrong.

## Emit as a pull request

The output is a PR, full stop. Emit through CI:

- **GitHub Actions + OIDC** federated to Azure — no stored secrets, no service-principal password.
- The PR carries: the rendered Terraform, the topology spec, the analyzer's validation report (the proposed-topology findings), and the cost forecast from `azure-network-cost-forecasting`. A reviewer sees intent, code, security verdict, and price in one place.
- A human reviews and runs the apply (a second Actions job gated on approval). **The agent holds no write or apply identity** — its credential is read-only for the validation reads; it cannot `terraform apply`.

## Enforce at scale with AVNM, not sprawl

Where the design needs connectivity or a security baseline across many VNets, generate **AVNM** connectivity and security-admin configurations (`module-registry.md`) rather than per-VNet peering and NSG duplication. Security admin rules evaluate before NSGs and can't be overridden by spoke owners — the right tool for an enforced baseline, and far less HCL to review.

## What this skill hands off

- The **security/blast-radius verdict** on the proposed topology comes from `azure-network-topology-analysis` (this skill calls it; it doesn't reimplement it).
- The **cost delta** comes from `azure-network-cost-forecasting`.
- The **apply** comes from a human via CI.

This skill's own output is narrow and safe: a validated spec, vetted-module Terraform, and a PR. Generation proposes; humans dispose.
