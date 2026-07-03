# Expected findings — iteration 3 (harder fixtures)

These probe gaps a strong base model is most likely to miss. As before, precision (the traps) is weighted at least as heavily as recall.

## H1 — firewall DNAT multi-hop

Must catch:

1. **Backend1 internet-reachable on 443 via firewall DNAT (High).** The Azure Firewall NAT rule `dnat-https-to-backend1` maps `Internet → 20.70.0.10:443 → 10.8.1.4:443`. `nsg-backend` allows 443 from the firewall subnet `10.0.0.0/26`, so the translated traffic is permitted. **`nic-backend1` has no public IP of its own**, yet it is internet-reachable through the firewall — the multi-hop path is the finding. A "no public IP ⇒ not exposed" read misses this entirely.

Must NOT flag (trap):

- **Backend2 as internet-reachable.** `nic-backend2` has the identical NSG (allow 443 from the firewall subnet) but **no DNAT rule targets it**, so there is no inbound translation and no internet path. Flagging it = false positive (assumes "behind the firewall" means "exposed").

This is the key uplift probe: does the analysis trace inbound *through* the firewall's NAT rules, or stop at "no public IP"?

## H2 — UDR black-hole and service-tag breadth

Must catch:

1. **Edge 443 real internet exposure (High).** `nsg-edge` allows 443 from `Internet`; `nic-edge` has a public IP and effective route `0.0.0.0/0 → Internet`. Straightforward real path.
2. **API over-exposed via the AzureCloud service tag (Medium/High).** `nsg-api` allows 443 from `AzureCloud`. That tag is **every Azure public IP across all tenants**, not "our internal Azure" — combined with a public IP and an `Internet` route it is a broad cross-tenant exposure. Under-rating it as a safe internal allow is the miss.

Must NOT flag (trap):

- **Darkpool 0.0.0.0/0:22 as high/critical.** `nsg-darkpool` allows SSH from anywhere and `nic-dark` has a public IP — looks critical — but its effective route for `0.0.0.0/0` is **`None`** (UDR `drop-default` black-hole), so no traffic is routed to/from it. No reachable path → latent only. Flagging it high = ignoring the route gate (the `None` next hop).

## Scoring intent

- **Recall:** all three "must catch" findings (H1#1, H2#1, H2#2), with evidence.
- **Precision:** neither trap flagged at high severity (backend2 not internet-reachable; darkpool black-holed).
- **Uplift signal:** the skill *beats* baseline only if it catches what the baseline misses — most likely the DNAT multi-hop reach (H1#1) and/or the AzureCloud breadth (H2#2). If both arms catch everything, it remains parity (and we have harder work to do). If the **skill** misses H1#1, that is honest feedback that the skill needs explicit firewall-DNAT handling (an iteration-4 fix), since its current references don't cover inbound DNAT.
