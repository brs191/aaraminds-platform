# The vetted module registry: select, parameterize, pin

The generator's job is to **choose an approved module and fill its inputs**, not to write network HCL. This file is the allowed set. Authoring `azurerm_network_security_rule`, route tables, or firewall policy by hand is the anti-pattern — those come from modules whose blast radius someone has already vetted.

## Preference order

1. **The org's own module registry** (if one exists) — org-specific patterns win, because they encode the standards the analyzer enforces.
2. **Azure Verified Modules (AVM)** — Microsoft-maintained, the default for greenfield.
3. **CAF community modules** — fallback where no AVM exists.

Never use a module at `latest`. Pin a version.

## Topology modules

| Need | Module | Notes |
|---|---|---|
| Hub-spoke connectivity (ALZ-aligned) | `Azure/avm-ptn-alz-connectivity-hub-and-spoke-vnet/azurerm` | Hub VNet, bastion, VPN/ER gateway, peering, private DNS — the default hub-spoke pattern |
| Virtual WAN connectivity | `Azure/avm-ptn-alz-connectivity-virtual-wan/azurerm` | When the design is vWAN rather than classic hub-spoke |
| Full landing zone | `Azure/terraform-azurerm-caf-enterprise-scale` | Management groups + policy + connectivity in one module call — use for whole-LZ generation, not a single spoke |
| Single hub / single spoke | CAF community (`kumarvna/terraform-azurerm-caf-virtual-network-hub` / `-spoke`) | Lighter-weight alternative where AVM is too much |

## Azure Virtual Network Manager (the enforcement layer)

For connectivity and security at scale, target AVNM rather than hand-rolled peering and NSG sprawl. Terraform (azurerm provider) resources:

- `azurerm_network_manager` — the manager.
- `azurerm_network_manager_network_group` — the groups topology/rules apply to.
- `azurerm_network_manager_connectivity_configuration` — hub-spoke or mesh connectivity, applied to network groups.
- `azurerm_network_manager_security_admin_configuration` → `azurerm_network_manager_admin_rule_collection` → `azurerm_network_manager_admin_rule` — the security admin rules that evaluate **before** NSGs. `action` (Allow / AlwaysAllow / Deny), `direction` (Inbound/Outbound), `priority` (1–4096), `protocol` (Tcp/Udp/Icmp/Esp/Ah/Any), source/destination blocks.

Wrap these via the AVM resource module `Azure/avm-res-network-networkmanager/azurerm` where possible rather than raw resources, so the security-admin-rule shape is vetted.

## The rule that keeps this safe

Whatever the generator emits for an NSG, route, firewall policy, or admin rule must be **traceable to a pinned module input**. If a piece of generated HCL has no module behind it, that is hand-authored network security — stop and route it through a module instead. This is verification question 1.

Sources: [AVM hub-spoke pattern](https://registry.terraform.io/modules/Azure/avm-ptn-alz-connectivity-hub-and-spoke-vnet/azurerm/latest), [azurerm_network_manager_admin_rule](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/network_manager_admin_rule).
