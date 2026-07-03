# Expected findings — answer key

The grading key for the three eval fixtures. Each fixture lists the planted issues the skill **must catch** (recall) and the traps it **must not flag** (precision). After sign-off these become the assertions in `grading.json`. Severity bands follow `reachability-and-severity.md`.

Precision is weighted at least as heavily as recall — the build plan's thesis is that false positives kill adoption, so the traps matter as much as the findings.

## Fixture 1 — internet exposure: real vs latent

Must catch:

1. **Internet → SSH on spoke-a (High/Critical).** `nsg-web-a` allows `0.0.0.0/0:22`, `nic-vm-web-a` has public IP `20.51.10.10`, and its effective route for `0.0.0.0/0` is `Internet`. Real path. Evidence must cite rule + route + public IP.
2. **Orphaned public IP (Low/Medium).** `pip-orphan-01` has `ipConfiguration: null`.

Must NOT flag (traps):

- **Spoke-b SSH as high/critical.** `nsg-web-b` has the *identical* `0.0.0.0/0:22` rule, but the effective route for `0.0.0.0/0` is `VirtualAppliance` (firewall) and there is no public IP → no reachable internet path. Acceptable: report as latent/informational with a note. Flagging it High = a false positive and a precision miss.

Discriminator vs baseline: a no-skill run is expected to flag both spoke-a and spoke-b SSH rules as critical. The skill should separate them on reachability.

## Fixture 2 — segmentation and transitive peering

Must catch:

1. **Web → DB cross-tier (High).** `nsg-db` allows `1433` from `10.3.1.0/24` (web), bypassing the app tier; `nic-db1` is tagged `sensitive: true`. Cross-tier path to a sensitive workload.
2. **Spoke-x ↔ Spoke-y transitive exposure (Medium/High).** Both peer to the hub with `allowForwardedTraffic: true` and carry UDRs routing each other's prefix to the hub NVA (`10.0.0.4`). A spoke-to-spoke path exists that hub-spoke isolation normally forbids.
3. **DB reachable VNet-wide via the default allow (High/Critical).** `nsg-db` has no `DenyVnetInBound` above the default `AllowVnetInBound` (65000), so every source in `10.3.0.0/16` (the app tier included) reaches the sensitive db on all ports — not just `web:1433`. The narrow `allow-sql-from-web` denies nothing. More severe than the planted web→db rule, and a required catch.

Must NOT flag (trap):

- **Spoke-z reaching other spokes.** `spoke-z` peers to the hub with `allowForwardedTraffic: false` and has no inter-spoke UDR → peering is non-transitive; it cannot reach spoke-x/spoke-y. Reporting a spoke-z→spoke path = false positive.

## Fixture 3 — CIDR overlap and AVNM precedence

Must catch:

1. **Address-space overlap (Medium/High).** `ov-a-vnet` and `ov-b-vnet` both use `10.10.0.0/16`; the peering is `Disconnected`/out of sync as a direct consequence. Flag the overlap and the broken peering.
2. **Edge 443 open via AVNM Always-Allow (High).** `nsg-edge` *denies* `443`, but the security admin rule `always-allow-443` (`AlwaysAllow`) overrides the NSG; `nic-edge1` has a public IP and an `Internet` route → a real open internet path the NSG alone would hide. Tests that admin-rule precedence is applied for *recall*, not just suppression.
3. **East-west RDP still open despite the AVNM Deny (High).** `deny-rdp-from-internet` matches the `Internet` service tag only, so it closes internet-sourced RDP but not the intra-VNet/peered RDP that `nsg-mgmt`'s `0.0.0.0/0:3389` allow still permits. The admin Deny narrows the exposure; it does not eliminate it.

Must NOT flag (trap):

- **Mgmt RDP as a live *internet* exposure.** `nsg-mgmt` allows `0.0.0.0/0:3389` with a public IP and `Internet` route — looks critical — but `deny-rdp-from-internet` (`Deny`, priority 10, source `Internet`) closes the *internet* path. Do not report internet→RDP as open. The *east-west* RDP path IS open and is a required catch (must-catch 3 above) — the distinction is the whole point: the admin Deny is scoped to the `Internet` tag, not all sources.

## Scoring intent

- **Recall:** all eight "must catch" findings across the three fixtures, with path evidence.
- **Precision:** zero of the three traps flagged at high severity (latent spoke-b, non-transitive spoke-z, internet-sourced mgmt RDP closed by the admin Deny).
- **Evidence discipline:** every High/Critical finding cites rule + effective route + exposure, per the skill's anti-pattern rule.
- A finding that is correct but lacks evidence is a partial pass; a trap flagged high is a hard fail for that eval.
