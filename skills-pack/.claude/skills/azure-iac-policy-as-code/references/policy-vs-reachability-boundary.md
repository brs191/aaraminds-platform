# Policy-as-code vs the reachability analyzer — the boundary

Stage 4. The two gates are complementary axes. Getting the boundary right is what stops antr from either re-implementing Checkov or claiming its reachability engine "does compliance."

## Who owns what

| Question | Gate | Why |
|---|---|---|
| Is encryption-at-rest on? TLS min? public blob? mandatory tags? approved region/SKU/module? IAM scope? | **Policy-as-code** (Checkov + OPA) | properties of the *declared resource*; maintained rules + custom Rego |
| Can this be **reached from the internet** through NSG + route + peering + firewall? Is a sensitive subnet reachable VNet-wide? CIDR overlap? | **Reachability analyzer** (`azure-network-topology-analysis`) | a property of a *computed path*; a static linter cannot derive it |

A static scanner can flag a *pattern* ("public subnet + permissive SG") but cannot compute true end-to-end reachability across effective routes, peering transitivity, and firewall DNAT. The analyzer can prove a permissive-looking rule is *not* reachable (routed to a firewall, no public IP) — and conversely prove a benign-looking config *is* reachable. Neither subsumes the other.

## How they compose in `generate_topology`

Both gates run on the generated plan; the PR is emitted only if **both** pass:

```
generated Terraform
  ├─ policy gate:  Checkov + Conftest over plan JSON  → no HIGH/CRITICAL, no deny
  └─ reachability: project topology → analyze.Analyze() → zero high-severity reachable findings
→ PR (human applies; agent never `terraform apply`)
```

## Don't double-count or contradict

- If Checkov already flags "NSG allows 0.0.0.0/0 on 22," that is a *policy* finding. The analyzer separately decides whether that rule is *reachable*. Report both, but label them by axis so a reviewer isn't confused by "two findings for one rule."
- Severity scales differ: Checkov severities are per-check; the analyzer's severity is a property of reachability. Keep them in separate sections of the report; don't merge into one number.

## Done when

The generate pipeline runs both gates independently, each labeled by axis; the PR requires both green; and no finding is double-counted or presented as if one gate covers the other's axis.
