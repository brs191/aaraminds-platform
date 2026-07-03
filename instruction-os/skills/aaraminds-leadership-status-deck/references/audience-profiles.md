# Leadership audience profiles (opt-in)

The base skill calibrates by **altitude** (VP vs manager). This adds calibration by **role** — a VP of
Engineering and a CFO consume the same program differently. An `Audience Profile` input re-weights
*emphasis and ordering*, never the facts: the RAG, the metrics, and the risks are identical; what
changes is which lens leads.

## The hard rule

Profiles change **emphasis, ordering, and translation lens — not the truth.** Never fabricate a
role-specific metric to satisfy a profile. If a profile wants a number you don't have (e.g., CFO wants
burn rate and it isn't in the inputs), mark it `[VERIFY]` and ask — don't invent it.

## Profiles

| Profile | Leads with | Dashboard dimensions to foreground | Translation lens |
|---|---|---|---|
| **VP Engineering** | Delivery confidence, risk, dependencies, quality/reliability | Schedule · Quality · Dependencies · Risk | "what's shipping, what's fragile, what's blocked" |
| **VP Product** | Customer impact, revenue, adoption, roadmap | Scope · Schedule (to launch) · Risk-to-adoption | "what this means for customers and the roadmap" |
| **CFO / Finance** | Cost, burn rate, forecast vs budget, ROI | **Cost** (lead) · Schedule (as cost driver) · Risk (as financial exposure) | "what it costs, what it saves, where the spend is going" |
| **CIO** | Strategic alignment, governance, risk, security/compliance | Risk · Dependencies · Cost · Governance | "how this aligns to strategy and what we're exposed to" |
| **CEO / Board** | Outcomes, strategic bets, money, the ask | Overall RAG · top risks · the decision | "the outcome, the bet, the one decision" |
| **(default) AVP / VP generic** | Overall health, what changed, top risks, the ask | all six, balanced | business meaning + decision |

## How a profile re-weights the deck

- **Executive summary (slide 2):** the headline and key wins lead with the profile's primary lens
  (e.g., for a CFO, the headline carries the cost/forecast read first).
- **Health dashboard (slide 3):** foreground the profile's dimensions; the others still appear (RAG is
  never hidden) but order/emphasis shifts.
- **Accomplishments (slide 4):** the business-impact translation uses the profile's lens (the same win
  is framed as "protects Q3 launch" for Product vs "avoids $X rework" for Finance).
- **Appendix:** unchanged — full detail for any reader.

If no profile is supplied, use the generic AVP/VP default. If a deck serves multiple roles in one
meeting, default to generic and add a one-line role-specific note where a lens materially differs.

## Verify (profile)

- Profile applied to emphasis/ordering only — RAG, metrics, and risks unchanged from the facts?
- No role-specific metric fabricated; missing ones marked `[VERIFY]`?
- All six dashboard dimensions still present (foregrounded, not deleted)?
