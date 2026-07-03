# Fixed costs: the Retail Prices API and standing SKU fees

Fixed costs are the standing, time-based fees that accrue whether or not a byte flows: gateway SKUs, the Azure Firewall base fee, Private Endpoints, NAT gateway base, public IPs, Application Gateway. They are **exact** — pull them live and compute to the cent. Never hardcode a rate; prices change and vary by region.

## The Azure Retail Prices API

Unauthenticated REST endpoint; query with OData `$filter`. No key, no SDK required.

```
GET https://prices.azure.com/api/retail/prices?api-version=2023-01-01-preview
    &$filter=serviceName eq 'Azure Firewall' and armRegionName eq 'eastus' and priceType eq 'Consumption'
```

```python
import requests

def price(service, region, meter=None):
    f = [f"serviceName eq '{service}'", f"armRegionName eq '{region}'", "priceType eq 'Consumption'"]
    if meter:
        f.append(f"meterName eq '{meter}'")
    r = requests.get("https://prices.azure.com/api/retail/prices",
                     params={"api-version": "2023-01-01-preview", "$filter": " and ".join(f)})
    return r.json()["Items"]
```

Key response fields: `meterName`, `retailPrice` / `unitPrice`, `unitOfMeasure`, `armRegionName`, `productName`, `skuName`, `currencyCode`. Filter tightly (`serviceName` + `armRegionName` + `meterName`) — it reduces the result set and avoids rate limits.

## The fixed meters to pull

| Component | What to query | Notes |
|---|---|---|
| VPN gateway | `serviceName eq 'VPN Gateway'`, the SKU (`VpnGw1`…`VpnGw5`, AZ variants) | Hourly per gateway; SKU drives both price and throughput |
| ExpressRoute gateway | `serviceName eq 'ExpressRoute'` (gateway SKU) | Hourly; separate from the circuit fee |
| Azure Firewall | `serviceName eq 'Azure Firewall'`, base/deployment meter | Standard vs Premium differ; **base fee is hourly per deployment** |
| Private Endpoint | `serviceName eq 'Private Link'`, the endpoint hour meter | Hourly per endpoint — multiplies with the private-link footprint |
| NAT gateway | `serviceName eq 'NAT Gateway'`, the resource-hour meter | Hourly base **plus** a per-GB meter (variable — see `cost-model-variable.md`) |
| Public IP | `serviceName eq 'Virtual Network'` / `'IP Addresses'`, Standard static | Hourly per address; cheap individually, adds up at fleet scale |
| Application Gateway / WAF | `serviceName eq 'Application Gateway'` | Fixed gateway-hour **plus** capacity-unit fees |

## Assemble the monthly fixed cost

For each standing resource in the (proposed) topology: `monthly = retailPrice × 730` for hourly meters (730 = average hours/month). Sum across resources. This figure is exact for the stated region and SKUs — cite the meters and the region with it.

Indicative only (always re-pull): an Azure Firewall Standard base is roughly **$1.25/hr ≈ $912/mo** `[VERIFY]`, a Standard static public IP a few dollars/month `[VERIFY]`, a Private Endpoint about **$0.01/hr ≈ $7.30/mo** `[VERIFY]`. These move — the number that ships comes from the API call, not this table.

## Caveats

- **SKU is load-bearing.** A gateway's price is meaningless without its SKU; pull the exact one in the design.
- **Region matters.** Same meter, different `armRegionName`, different price — always pass the target region.
- **Reserved/EA discounts** are not in retail prices. If the org has an Enterprise Agreement, the fixed numbers are an upper bound; note that and let FinOps (`azure-microservices-cost-review`) apply the negotiated rates.
