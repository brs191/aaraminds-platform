# Benchmark — azure-network-topology-analysis, iteration 1

**Setup:** 3 synthetic fixtures, each run twice (with-skill vs baseline/no-skill), same model. Graded against `expected-findings.md` for recall of planted findings and avoidance of planted traps. Token/time from the subagent task notifications.

## Headline

**The skill did not beat the baseline.** Both arms avoided every planted precision trap, and the baseline matched the skill on all headline findings while catching two real issues the skill missed — at ~19% fewer tokens. This is a non-discriminating result: as drafted, the skill does not yet earn its keep over a strong base model. That is a useful finding, not a failure — it tells us exactly what to fix.

| Eval | With-skill | Baseline | Tokens (with/base) | Verdict |
|---|---|---|---|---|
| 1. internet exposure (real vs latent) | recall 2/2, trap avoided | recall 2/2, trap avoided | 45,885 / 37,824 | **Tie** |
| 2. segmentation + transitive peering | 2/2 planted, trap avoided | 2/2 + 1 real extra, trap avoided | 49,240 / 42,498 | **Baseline ahead** |
| 3. CIDR overlap + AVNM precedence | 2/2, trap avoided | 2/2 + 2 real extra, trap avoided | 48,454 / 40,353 | **Baseline ahead** |
| Total | | | 143,579 / 120,675 (+19%) | |

## What the skill got right

- All planted headline findings, with clean rule + route + exposure evidence.
- Every precision trap avoided: spoke-b latent (not high), spoke-z non-transitive (not flagged), mgmt RDP not called a live internet exposure.
- The standout was eval 3's headline: it correctly applied AVNM precedence to see that the `edge` 443 path is **open** despite the NSG deny (Always-Allow override) — the hardest call in the set.

## Where the baseline beat it (the two real misses)

1. **Eval 2 — default-rule flat-open db.** `nsg-db` carries the default `AllowVnetInBound` (65000) with no overriding deny, so the sensitive db is reachable from the **entire VNet on all ports**, not just `web:1433`. The baseline flagged this (Critical); the skill reported only the narrow planted `web→db` rule. The skill's own `nsg-route-evaluation.md` lists the default rules — but the run didn't operationalize them to surface the broader exposure.
2. **Eval 3 — AVNM Deny source-scope.** The admin `deny-rdp-from-internet` rule matches the `Internet` service tag only, so it closes Internet-sourced RDP but **not** intra-VNet/peered RDP that the NSG's `0.0.0.0/0` allow still permits. The skill called the path fully closed; the baseline correctly noted RDP is still open east-west (and that effective-rules tooling doesn't show AVNM rules at all).

## Improvement items for iteration 2

1. **Operationalize default NSG rules in reachability.** Add an explicit step/checklist item: when evaluating a subnet's exposure, account for `AllowVnetInBound` — a sensitive target with no `DenyVnetInBound` above 65000 is reachable VNet-wide, regardless of the narrow allow rules present. Add a fixture-2-style worked example.
2. **Make AVNM rule source-scope explicit.** An admin `Deny`/`AlwaysAllow` only governs the traffic its source/destination matches (e.g., the `Internet` service tag ≠ all sources). A `Deny Internet:3389` does not close intra-VNet/peered RDP. Add this as a named sub-rule under Gate 1 and a trap in the worked example.
3. **Add an observability check.** Note that Network Watcher effective security rules do **not** include AVNM admin rules — they must be pulled and applied separately, or exposure reads will be wrong in both directions.

## Answer-key gaps this run exposed (fix the eval too)

The baseline surfaced two legitimate findings my answer key didn't list (the AllowVnetInBound flat-open and the east-west RDP). Iteration 2's `expected-findings.md` should add both as required findings, so the eval rewards catching them. The traps were well-designed (both arms handled them); the recall key was under-specified.

## Recommendation

Don't promote to `1.0.0` or trust on a real subscription yet — the skill is currently non-discriminating. Apply the three improvement items, tighten the answer key, and re-run iteration 2. If the skill then catches the two misses the baseline caught while keeping its clean trap avoidance, it earns its strength rating. The honest current status stays `strength: n/t → not yet beating baseline`.

---

# Iteration 2 (v0.2.0)

**Result: the three fixes worked.** The with-skill runs now catch both findings they missed in iteration 1 — the default-`AllowVnetInBound` flat-open db (eval 2) and the east-west RDP under AVNM source-scope (eval 3) — while keeping every trap avoided. With-skill and baseline are now **at parity**.

| Eval | With-skill v0.2.0 | Baseline | vs iteration 1 |
|---|---|---|---|
| 1. internet exposure | recall 2/2, trap avoided | recall 2/2, trap avoided | held (clean) |
| 2. segmentation + peering | recall 3/3, trap avoided | recall 3/3, trap avoided | **fixed** (default-allow flat-open now caught) |
| 3. CIDR + AVNM | recall 3/3, trap avoided | recall 3/3, trap avoided | **fixed** (east-west RDP now caught) |
| Total | **8/8 recall, 3/3 traps** | **8/8 recall, 3/3 traps** | gap closed |

Tokens: with-skill 166,180 vs baseline 121,734 (+37%).

## The honest read

The skill went from **losing** to baseline (iteration 1) to **parity** (iteration 2). The fixes are validated: source-scope and default-rule reasoning now appear explicitly in the with-skill outputs and produce the catches. But parity is the ceiling here — these fixtures are now solved by both arms, so the eval no longer demonstrates the skill *beating* a strong base model, and the skill costs ~37% more tokens.

Two things this means:

1. **The skill's value on a frontier model is consistency, not raw uplift.** A careful frontier run already finds these issues; the skill makes the method explicit, repeatable, and tighter in signal (the with-skill output is more focused and uniformly evidence-structured; the baseline is more exhaustive but noisier). That consistency matters most on cheaper/weaker models and on org-specific severity rules — neither of which this eval exercises.
2. **To prove uplift, the next eval needs harder cases or a weaker model.** Add fixtures where a naive read genuinely fails (multi-hop NVA chains, ASG-based rules, overlapping service tags), and/or run the with/baseline comparison on a smaller model where the skill's scaffolding should separate them.

## Recommendation

The skill is now **correct and complete** on the planted cases and no longer regresses — safe to promote from `0.1.0` to a validated `0.2.0`/`1.0.0` and apply to the pack, recorded honestly as **`strength: parity with a strong baseline on the v1 fixtures; uplift unproven`**. Before relying on it for the actual product (or claiming it beats expert review), run iteration 3 with harder fixtures or a weaker model to isolate the skill's contribution.

---

# Iteration 3 (harder fixtures, v1.0.0)

Two fixtures built to probe likely failure points: inbound **firewall DNAT** to a no-public-IP backend (multi-hop reachability), a **UDR `0.0.0.0/0 → None`** black-hole, and the broad **`AzureCloud`** service tag.

| Eval | With-skill v1.0.0 | Baseline | Verdict |
|---|---|---|---|
| H1 firewall DNAT multi-hop | recall 1/1, trap avoided | recall 1/1, trap avoided | Tie |
| H2 black-hole + service tags | recall 2/2, trap avoided | recall 2/2, trap avoided | Tie |
| Total | **3/3 recall, 2/2 traps** | **3/3 recall, 2/2 traps** | **parity** |

Tokens: with-skill 101,718 vs baseline 76,197 (+33%).

## The result

**Parity holds even on the hard cases.** The frontier base model, unaided, traced internet→backend reachability through the firewall DNAT, respected the `None` black-hole route (didn't over-flag the scary `0.0.0.0/0:22` rule), and flagged the `AzureCloud` tag as a broad cross-tenant source. The skill matched it on every call; it did not pull ahead.

Two honest takeaways:

1. **The skill's value on a frontier model is not capability — it's consistency, severity discipline, and the org-specific method.** Three iterations across eight fixtures now show parity with a careful frontier baseline, never a clear win. That is the real, unspun finding.
2. **A concrete content gap surfaced:** in H1 both arms caught the DNAT, but the **skill's references don't actually contain inbound-DNAT handling** — the base model supplied it. On a weaker model the skill would miss it too. So the next *content* fix (iteration 4) is to add firewall-DNAT reachability as an explicit path (extend Gate 3 / the routing reference), so the skill carries that knowledge rather than borrowing it from the model.

## Recommendation

The definitive uplift test is **not** harder fixtures (the frontier model keeps up) — it is a **weaker-model run**: rerun the with/baseline benchmark on a small model (e.g., Haiku) where the skill's scaffolding should separate them. If the skill lifts a weak model toward the frontier result, that is its proven value and the basis for the strength rating. Until then, the honest rating stands: **correct, non-regressing, parity-with-frontier, uplift-on-weak-models-unproven**, plus a known DNAT content gap to close in iteration 4.

---

# Weaker-model probe (Haiku requested) + DNAT fix (v1.1.0)

The DNAT content gap is closed: skill 1 `v1.1.0` adds explicit inbound-DNAT reachability (a backend with `publicIp: null` can be internet-reachable via a firewall DNAT rule) to `nsg-route-evaluation.md` plus a verification question.

Reran the two iteration-3 fixtures with `model: haiku` requested for both arms. **Findings result: parity again** — with-skill 3/3 recall, 2/2 traps; baseline 3/3, 2/2. Both caught the DNAT multi-hop, respected the `None` black-hole, and flagged the `AzureCloud` breadth. The with-skill runs visibly leaned on the new DNAT content ("Gate 3 (Routing/DNAT)", "`publicIp: null` not used to short-circuit").

**But this test is inconclusive on the model question.** The subagent outputs are as long, structured, and accurate as the frontier runs (44k–68k tokens, multi-page four-gate analyses with ASCII topology diagrams) — that does not look like Haiku. I can't confirm the `model: haiku` hint actually downgraded the general-purpose subagents; they may have inherited the parent model. So the weak-model uplift question is **still unanswered**, and the one test designed to settle it couldn't be verified to use a weak model.

A definitive run needs a **guaranteed-pinned** small model — skill-creator's `run_loop.py --model <haiku-id>` (which shells to `claude -p` with an explicit model) or a direct API call to a pinned `claude-haiku` — not a subagent model hint. Until that's wired, the honest status is unchanged: **parity with a capable model across the fixtures and four rounds; uplift unproven.**

---

# Skill 2 eval — cost forecasting (firewall-egress scenario)

One scenario: forecast the cost of forcing spoke egress through a new Azure Firewall, given a 40–60 TB/mo traffic basis.

- **With-skill:** clean structure — fixed exact (~$916, and it actually pulled the live meters), a variable band ($655–$983 from the basis), internet egress held at **$0 delta** (the planted trap), and a sharp honest call that fixed and variable are *co-dominant* at Standard's $0.016/GB — it even pushed back on the skill's own "variable dominates" worked example. **Missed:** the new inter-VNet peering cost.
- **Baseline:** also got fixed/variable/egress-not-a-delta right, looser on structure and quoting some prices from memory, **but caught the inter-VNet peering hop** the change introduces (spoke→hub, ~$0.01/GB each way) that the skill-run missed — a real component.

**Verdict: parity again.** The skill gives cleaner structure, discipline, and live prices; the baseline caught a real cost the skill missed. And — exactly the pattern from the topology evals — the baseline exposed a gap in *my answer key* (I didn't list the peering cost as a must-include) and a *skill* gap (the worked example should flag that rerouting egress through a hub adds a peering hop). Both go on the iteration-2 list for skill 2.

---

# Overall conclusion (5 eval rounds, 6 fixtures, 2 skills)

Across every eval — easy, hard, two skills, and a (best-effort) weak-model probe — the skills land at **parity with a capable baseline, never a demonstrated win**, and the baseline **repeatedly catches real things the skill misses** (AllowVnetInBound flat-open, east-west RDP under AVNM source-scope, inter-VNet peering cost). The skills' value is real but narrow: structure, consistency, evidence/severity discipline, and the org-specific method — not capability uplift on a strong model. The one test that could prove uplift (a weak model) couldn't be verified to actually run on a weak model. **Recommendation: before investing further, wire a pinned-small-model eval. If the skills lift a weak model, that is the justification; if they don't, these are documentation/consistency aids, not capability multipliers — still useful, but scope the investment accordingly.**

---

# Resolution — the weak-model test is now decisive

The open question was whether the subagent `model: haiku` hint actually downgraded the model. Calibration settled it: a `model: haiku` agent and a `model: opus` agent, each asked to self-identify, returned **"Haiku"** and **"Opus (claude-opus-4-8)"** respectively. The parameter genuinely pins the model.

That validates the earlier Haiku run — it really was Haiku — and its result stands: **with-skill Haiku 3/3 recall, 2/2 traps; baseline Haiku 3/3, 2/2.** Pinned Haiku, unaided, already caught the firewall-DNAT reach, the `None` black-hole, and the `AzureCloud` breadth.

**Conclusion (decisive, not provisional): no capability uplift on any fixture built, on either a frontier model or Haiku.** Both model sizes handle these cases with or without the skill. On the evidence, this is a **consistency play, not a capability play.** The skills' value is structure, evidence/severity discipline, reproducibility, and the org-specific method — useful for auditability and for keeping less-careful runs honest, but not a capability multiplier.

The only thing that would change the picture is fixtures that exceed *Haiku-baseline's* unaided ability — which, given Haiku handled even the DNAT/black-hole/service-tag cases, would have to be materially harder than anything realistic for this task. `run-haiku-uplift-eval.py` is delivered to reproduce this and to test any such future fixtures on a verified-pinned model.

**Strategic implication, tied back to the build plan:** if you want *capability* uplift from this investment, it has to come from the **deterministic engine** — the graph + reachability computed in code and exposed via the MCP server — not from skill prose. That is exactly what the build plan's "deterministic engine, LLM at the edges" architecture already says. The three skills are the documentation and consistency layer around that engine; they are correct, non-regressing, and worth keeping and governing — but the smarts live in the code you build, not in the prompts.
