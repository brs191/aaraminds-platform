# OPA / Conftest (Rego) — custom organizational policy

Stage 2. Checkov covers the generic security baseline; OPA/Conftest (both Apache-2.0, CNCF) covers AT&T-specific rules that no off-the-shelf check encodes — region allow-lists, mandatory tag schemas, private-only enforcement, approved SKUs/modules. Write the policy as Rego; run it with Conftest over the plan JSON.

## The input is the plan JSON

```
terraform plan -out tfplan.binary
terraform show -json tfplan.binary > tfplan.json
conftest test tfplan.json
```
Conftest evaluates Rego against the plan's `resource_changes[]` — each carries `type`, `change.after`, and the resolved values. Write rules over `change.after`, not raw HCL.

## Rego shape (illustrative)

```rego
package main

deny[msg] {
  rc := input.resource_changes[_]
  rc.type == "azurerm_virtual_network"
  loc := rc.change.after.location
  not allowed_regions[loc]
  msg := sprintf("VNet %s in disallowed region %q", [rc.address, loc])
}

allowed_regions := {"eastus2", "westus2"}
```

Patterns to encode for antr's domain: deny public IP on a `sensitive`-tagged subnet's resources; require an `environment` tag on every subnet; deny NSG inbound `Allow` from `Internet`/`*` to management ports; restrict module `source` to the approved AVM/ALZ registry; restrict VNet address space size.

## Keep policy as data where possible

Externalize allow-lists (regions, SKUs, approved modules) as Rego `data` documents or JSON, so the policy logic is stable and the lists are updated without touching rules. Unit-test rules with `opa test` / `conftest verify` — policies are code and deserve tests.

## Boundary with Checkov

Use Checkov for "is this a known insecure configuration" (maintained, broad). Use Conftest for "does this violate *our* governance" (custom, relational). Don't duplicate a check across both; if Checkov already covers it, suppress your Rego equivalent.

## Done when

Org rules are Rego, run via Conftest over plan JSON in CI, are unit-tested (`opa test`), externalize allow-lists as data, and don't duplicate Checkov's baseline.
