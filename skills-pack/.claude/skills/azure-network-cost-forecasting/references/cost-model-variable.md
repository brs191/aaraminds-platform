# Variable costs: per-GB meters and the traffic basis

Variable costs are the per-GB data-processing and egress meters. They usually **dominate the real bill** and they depend on a traffic volume you have to source or assume — so they ship as a band, never a single number. Every figure here is indicative; pull live from the Retail Prices API (`cost-model-fixed.md`) and mark anything quoted `[VERIFY]`.

## The per-GB meters

| Meter | Roughly | What triggers it |
|---|---|---|
| Internet egress | first **100 GB/mo free**, then tiered from ~**$0.087/GB** down with volume `[VERIFY]` | Any byte leaving Azure to the internet |
| VNet peering, same region | ~**$0.01/GB each direction** (in *and* out) `[VERIFY]` | Traffic across a same-region peering — billed both sides |
| VNet peering, cross-region / global | ~**$0.035/GB** (US/EU/AU), **$0.09** (Asia), **$0.044** (SA/Africa) `[VERIFY zone]` | Traffic across regions — zone-dependent, much pricier than same-region |
| NAT gateway data processing | ~**$0.045/GB** (outbound + return) `[VERIFY]` | All data through a NAT gateway |
| Azure Firewall data processing | per-GB processed `[VERIFY]` | Every GB the firewall inspects — the big one behind a forced-tunnel design |
| Private Link data processing | per-GB inbound + outbound `[VERIFY]` | Traffic through a Private Endpoint |

The trap: same-region peering at $0.01/GB looks trivial, but a chatty cross-region peering or a firewall inspecting 50 TB/mo is where the bill actually lives. Lead with whichever meter the design's traffic concentrates on.

## The traffic basis

A per-GB meter is useless without GB. Get the volume, in order of preference:

1. **Measured — VNet flow logs + Traffic Analytics.** For a deployed estate, Traffic Analytics aggregates flow-log data into GB per flow/subnet/region over a window. Use the last 30 days per path. Build on **VNet flow logs** — NSG flow logs stop accepting new resources after 30 Jun 2025 and retire 30 Sep 2027.
2. **Derived.** Scale from a comparable workload's measured traffic.
3. **Assumed.** If nothing is measurable (pre-deployment greenfield), state an explicit assumption (e.g., "10 TB/mo egress, 80% east-west") and make the band wide.

## Express it as a band, not a number

Wherever traffic is uncertain, ship a range tied to the basis:

- **Measured:** band across the p50–p90 of observed monthly volume.
- **Assumed:** a low / expected / high triple with the assumption stated.

`variable_monthly = Σ over paths ( traffic_GB_path × per_GB_meter_path )`, evaluated at the low and high traffic figures to give the band. Always attach the basis ("derived from 30-day VNet flow logs" or "assumed 10 TB/mo") so the reader can challenge the input, not just the output.

## Caveats

- **Egress is not peering.** Forcing egress through a firewall adds the firewall per-GB meter but does **not** remove the internet-egress meter — the bytes still leave Azure. Count both.
- **Peering is billed both directions** — don't halve it.
- **Cross-zone vs cross-region** differ; use the correct `armRegionName`-pair rate.
- These rates exclude EA/reserved discounts — see `cost-model-fixed.md`.
