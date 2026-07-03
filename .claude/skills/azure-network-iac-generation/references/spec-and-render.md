# Intent → spec → rendered Terraform

The model's deliverable is a **topology spec** (intent, structured) and the **module inputs** derived from it. A deterministic render turns spec + chosen modules into Terraform. The spec is the single source of truth; the render is mechanical.

## The topology spec (the model's output)

Capture the architect's intent as a structured object — not HCL:

```yaml
region: eastus
address_plan: 10.12.0.0/16          # must not overlap existing space (see below)
vnets:
  - name: spoke-pci
    cidr: 10.12.0.0/16
    subnets:
      - { name: web, cidr: 10.12.1.0/24 }
      - { name: app, cidr: 10.12.2.0/24 }
      - { name: db,  cidr: 10.12.3.0/24, sensitive: true }
connectivity:
  pattern: hub-spoke               # or mesh
  hub: hub-vnet
  forced_tunnel: true              # 0.0.0.0/0 -> hub firewall
isolation:
  inter_spoke: deny                # becomes an AVNM security-admin Deny
firewall: { use_hub: afw-hub }
avnm:
  network_group: pci-spokes
  security_admin: [ { name: deny-inter-spoke, action: Deny, direction: Inbound, priority: 100, source: spoke-cidrs, dest: spoke-cidrs } ]
```

This is what the LLM writes. Everything security-relevant in it maps to a *module input*, never to free HCL.

## Address planning (reuse the analyzer's math)

Before rendering, check the spec's CIDRs against the existing estate's address space — the same overlap math `azure-network-topology-analysis` uses (`reachability-and-severity.md`). Overlapping space silently breaks peering, so a CIDR clash is a hard stop at spec time, not a deploy-time surprise.

## Map spec → module inputs

Each spec field drives a pinned module's variable. For the hub-spoke AVM module: `vnets[]` → the spoke VNet/subnet inputs; `connectivity.pattern`/`hub` → the peering inputs; `forced_tunnel` → the spoke route-table input pointing `0.0.0.0/0` at the hub firewall; `isolation.inter_spoke` / `avnm.security_admin` → `azurerm_network_manager_admin_rule` inputs. The model selects the module and fills these; it does not write the `azurerm_*` resources directly.

## Render

Render is templating, not authorship: emit a `module` block per chosen module with its pinned `source`/`version` and the variable assignments from the spec. Example shape:

```hcl
module "spoke_pci" {
  source  = "Azure/avm-ptn-alz-connectivity-hub-and-spoke-vnet/azurerm"
  version = "0.6.0"            # pinned, never "latest"
  # ...inputs derived from the spec...
}
```

Because the security-bearing resources live inside the pinned module, the rendered HCL is auditable: a reviewer checks the spec and the module version, not a wall of hand-written rules. Keep the spec in the PR alongside the Terraform so intent and code travel together.
