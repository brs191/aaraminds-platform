# Architecture Review Report — Template

Use this layout for the report produced by the reviewer. Substitute placeholders. Keep the structure stable so reports across the estate are comparable quarter-over-quarter.

## Report frontmatter

```
System under review:   <product or estate name>
Scope:                 <whole estate | one product | one service>
Date:                  <ISO date>
Reviewer:              <name(s)>
Inputs reviewed:       <list of artifacts: ADRs, diagrams, Terraform, OpenAPI, dashboards, bill, runbooks>
Inputs missing:        <list — these are findings in themselves>
Verdict:               <Healthy | Healthy with risks | At risk | Unsound>
```

The verdict appears in frontmatter so a reader can find it without scrolling. If the verdict is "Healthy" — say so up top; do not bury it.

## Section 1 — Executive summary (max 200 words)

One paragraph stating: the verdict, the count of hard-fails and soft-fails by dimension, the single most load-bearing finding, and the recommended remediation arc with target horizon.

Example: *Verdict: At risk. Three hard-fails (Data consistency D2, Topology D3, Resilience D5) all concentrated in the Order→Payment leg. Six soft-fails distributed across observability and cost. Most load-bearing: synchronous Order→Payment with no timeout, breaker, or compensation — directly responsible for the May incident. Remediation arc: add resilience controls (1 sprint), move to outbox + compensation (2 sprints, strangler-fig), add Payment SLO and runbook (1 sprint, parallel). System recovers to "Healthy with risks" within one quarter.*

## Section 2 — Verdict and rationale

State the verdict explicitly with a 3-5 sentence rationale. The verdict tiers:

- **Healthy** — no hard-fails. Soft-fails tracked but the system meets its goals. Re-review in 12 months.
- **Healthy with risks** — no hard-fails, but ≥3 soft-fails clustered in one dimension, or a single soft-fail with significant blast radius. Re-review in 6 months.
- **At risk** — 1–3 hard-fails, each with an actionable remediation path of ≤2 quarters. Re-review when remediation completes.
- **Unsound** — 4+ hard-fails, or a hard-fail in Dimension 8 (security/compliance) with sensitive data in scope, or a structural anti-pattern (distributed monolith, shared DB across the estate) that cannot recover within a year. Rebuild path required. Re-review when the rebuild plan is approved.

Final-tier verdicts ("Unsound") are rare. Use only when remediation cannot recover the system within the named horizon. Most reviews will land in "At risk" or "Healthy with risks."

## Section 3 — Findings per dimension

For each of the 9 dimensions, one subsection with the same shape:

```
### Dimension N — <name>

Rating: <Pass | Soft-fail | Hard-fail>

Findings:
- <Specific named defect, with file / service / operation reference>
- <Next finding>

Remediation (for soft-fails and hard-fails only):
- <Smallest viable fix, with owner and target quarter>
```

If the rating is Pass, a one-line affirming note suffices ("All outbound calls have explicit timeouts and breakers; rollouts use blue-green via Container Apps revisions.").

If the rating is Hard-fail, every finding is named to the specific defect — not "improve security" but "OrderService → PaymentService has no timeout (`internal/payments/client.go:47`); one slow vendor call exhausts the OrderService thread pool."

## Section 4 — Cross-dimension findings

Anti-patterns that produce hard-fails across multiple dimensions get their own section. Each entry:

```
### <Anti-pattern name>

Symptoms across dimensions: <D2, D3, D5, ...>
Description: <2-3 sentences from the catalog>
Remediation: <named pattern, sequence, owner, horizon>
```

Cross-dimension findings are the most important section of the report — they explain *why* the per-dimension findings are clustered the way they are, and they make remediation prioritization obvious.

## Section 5 — Punch list

A flat numbered list of every hard-fail and soft-fail, in priority order (security and data consistency first, then resilience and observability, then the rest). Each entry has: identifier, dimension, defect, fix, owner, target quarter, success signal.

This is the section the steering committee uses to track remediation. Keep it terse and structurally consistent.

```
1. [D8 / Hard-fail] Plaintext secret in main.tf line 134. Fix: rotate, move to Key Vault, wire via Managed Identity. Owner: SRE. Target: this sprint. Signal: secret scan returns clean.
2. [D5 / Hard-fail] No timeout on OrderService → PaymentService client. Fix: add 2s timeout, 3-retry with jitter, breaker at 50% errors over 20 calls. Owner: Order team. Target: this sprint. Signal: synthetic Payment-slow test does not exhaust OrderService threads.
3. [D2 / Hard-fail] Order → Payment is sync without compensation. Fix: move to outbox + Service Bus; compensation handler for FulfillmentDispatched + PaymentFailed. Owner: Order team. Target: next quarter. Signal: synthetic Payment-failure test produces no orphan fulfillment.
4. [D7 / Soft-fail] No runbook for PaymentService. Fix: write the shortest viable template — purpose, alerts, first-response checklist. Owner: Payment team. Target: this sprint. Signal: runbook reviewed by on-call.
...
```

## Section 6 — Anti-pattern catalog hits

A short list of which anti-patterns from `anti-patterns.md` were detected in the system, with one-line evidence per hit. This lets future reviewers compare quarter-over-quarter and see whether the same anti-patterns are recurring.

## Section 7 — Inputs missing and their implications

Artifacts not provided to the review are themselves findings. List each with one line on what was missed because it was absent. Examples:

- No ADRs for the data ownership decision — Dimension 2 review relies on inference from Terraform.
- No runbook for Service X — Dimension 7 hard-fail recorded.
- No bill breakdown — Dimension 9 review limited to sizing inspection without cost-side validation.

## Section 8 — What was reviewed and how

A short evidence log: which dashboards were opened, which traces were inspected, which Terraform files were read, which conversations happened with which teams. The aim is reproducibility — a second reviewer should be able to verify the findings from the same inputs.

## Section 9 — Recommended next reviews

Based on the verdict tier, the next review date (re-review of this system) and any narrower follow-ups that are now indicated. Examples:

- Re-review of this system: 6 months from today (Healthy with risks tier).
- Cost-only review (`azure-microservices-cost-review`): suggested in Q3, after the right-sizing soft-fails land.
- Security-only review (`azure-microservices-security` deep): suggested before any sensitive-data feature.

## Optional appendices

- **A.** Trace snapshots and dashboard screenshots referenced inline.
- **B.** Service-to-team ownership table.
- **C.** Service-to-Azure-resource map.
- **D.** Diff against the previous review (when re-reviewing).

## Length and tone

Total report length target: 8–15 pages for a typical estate-level review; 3–5 pages for a single-service review. Reports longer than 20 pages are a smell — they have either drifted into redesign or padded with material that should be in references rather than the report.

Tone: principal engineer to a peer. State verdicts directly. No hedging. No "consider" or "you might want to." Findings are named or they are not findings.
