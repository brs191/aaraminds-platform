# Expected forecast — answer key (cost scenario)

A good forecast must do all of this; a baseline that gives a single confident total fails the precision test.

## Must include

1. **Fixed delta, exact, pulled (not hardcoded).** + Azure Firewall Standard base fee (≈ $1.25/hr × 730 ≈ **~$912/mo** `[VERIFY via Retail Prices API, eastus]`) + the firewall's public IP (a few $/mo). Stated as exact, with the meters/region cited and a note to pull live.
2. **Variable delta, as a band tied to the traffic basis.** Firewall **data-processing per-GB** × egress volume. At ~$0.016/GB `[VERIFY]` and 40–60 TB/mo, that is roughly **$640–$960/mo**, expressed as a band across the p50–p90 of observed egress — not a single number.
3. **Dominant driver named:** the firewall **data-processing** meter, which at 40–60 TB/mo dwarfs the ~$912 base fee. The forecast should lead with this, not the SKU list.
4. **Egress meter is unchanged.** Forcing egress through the firewall does **not** remove internet-egress charges — the bytes still leave Azure. Firewall processing is **on top of** existing egress, not instead of it.

## Must NOT do (traps)

- **False-precision total.** Reporting a single figure like "$1,850/month" with no band and no traffic basis.
- **Hardcoded prices as fact.** Quoting per-GB or base rates as settled truth with no `[VERIFY]` / Retail Prices API.
- **Claiming egress savings.** Asserting the firewall reduces or removes egress cost — it adds a processing meter; egress is unchanged.

## Scoring

- Recall: the four "must include" items.
- Precision: none of the three traps.
- The skill's value shows if it produces the fixed/variable split + band + dominant-driver + egress-unchanged, where a baseline is more likely to hand back one confident wrong-shaped number.
