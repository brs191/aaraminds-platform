# Pre-Release Checklist

Run before tagging any pack release. Each item ties back to the gap that produced the artifact, so this list also serves as a tour of what the pack contains.

## Section 1 — Example MCP server (Gap 1, completed in session 3)

The example server at `examples/microservices-system-design-mcp-server/` is the executable proof that the pack's MCP guidance produces working code.

- [ ] `go mod tidy` succeeds; no missing or incorrect transitive dependencies
- [ ] `go build ./...` produces a binary; no compile errors
- [ ] `go test ./...` — all tests pass; failure rate 0%
- [ ] `go vet ./...` — no warnings
- [ ] `gofmt -l .` — no unformatted files
- [ ] `make docker-build` — image builds clean (skip if Docker unavailable; note exemption)
- [ ] Server starts in stdio mode and emits the expected startup log to stderr
- [ ] Tool catalog returns the full set of 13 tools (the complete list is in `VERIFICATION_CHECKLIST.md`, Step 6)

## Section 2 — Microservices content (Gap 2, completed May 18, 2026)

The migrated microservices skill references and pattern cards under `.claude/skills/microservices-*/references/` and `.claude/skills/*/references/patterns/`.

- [ ] Boilerplate grep returns zero hits for the canonical stub phrases in pattern cards: `grep -r "The problem exists in a real workflow" .claude/skills/*/references/patterns/ | wc -l` returns `0`
- [ ] Generic-step grep returns zero hits: `grep -r "Define the business capability and boundary" .claude/skills/*/references/patterns/ | wc -l` returns `0`
- [ ] Line counts in range: all skill files >200 lines, all pattern cards >100 lines
- [ ] Cross-reference spot-check: pick three pattern cards at random; each references at least one related pattern and at least one skill file
- [ ] No broken internal links: skill files that reference other skills or pattern cards use paths that resolve

## Section 3 — Demo (Gap 4, completed May 18, 2026)

The MCP-driven demo at `demo/architecture-review-demo/`.

- [ ] `make demo` runs to completion across all three architectures (e-commerce, financial-services, healthcare) without error
- [ ] `make validate` reports `Validation passed: 3 architecture(s) × 5 tool(s) all match golden fixtures`
- [ ] If goldens drift intentionally (e.g., a tool's rule logic changed): `make refresh`, document why in the release notes
- [ ] Server build instructions in `demo/architecture-review-demo/README.md` still work for the target environment (or the Docker fallback works)

## Section 4 — Validation pack

The 12 capability prompts under `validation/prompts/`.

- [ ] Run all 12 capability prompts against the current pack via your LLM of choice; record `last_run` and `last_result` in each prompt's front-matter
- [ ] Pass rate ≥ 80% (≥ 10 of 12 at their declared pass threshold). Workflow-level prompts are harder than per-skill evals; calibrate expectations accordingly.
- [ ] Any prompt that consistently fails: file a triage issue — is the skill drifting, or is the prompt drifting?
- [ ] Spot-check three prompts (one per area: MCP-server-building, microservices-design, architecture-review): manually score the response against the rubric; reference output still looks like quality
- [ ] If reference outputs feel stale (e.g., pattern cards have evolved): regenerate as a follow-up issue, don't block the release

## Section 5 — Freshness (Gap 6 continued)

- [ ] Ecosystem facts in `.claude/skills/mcp-go-server-building/references/ecosystem-facts.md` were verified within the last quarter
- [ ] `validation/governance/freshness-cadence.md` has named owners (no `[Owner: __________]` placeholders left)
- [ ] Last quarterly refresh date is within 90 days of today; if not, run the quarterly checklist before release

## Section 6 — Documentation

- [ ] `python3 validation/tools/skill_audit.py` reports **0 FAIL** — in particular no `DOC_COUNT_DRIFT`: every living document agrees with disk on skill / agent / hook / reference counts (dated records under `validation/snapshots/` are exempt)
- [ ] `.claude/INDEX.md` regenerated this release: `python3 validation/tools/skill_audit.py --emit-index`
- [ ] `README.md` reflects the current quality position and shipped gaps
- [ ] `ROADMAP.md` reflects current gap status (✅ where complete, accurate descriptions for pending)
- [ ] `versions.md` matches `ecosystem-facts.md`
- [ ] No `TODO`, `FIXME`, or `XXX` in user-facing documentation (skill files, READMEs); these belong in code or in issues, not in shipped docs

## Section 7 — Release-specific

- [ ] `CHANGELOG.md` (or equivalent) entry exists for this release with the gap deltas
- [ ] Version bump in `versions.md` and pack frontmatter
- [ ] Tag matches version
- [ ] Distribution: pack contents zipped or pushed to the canonical location

## Sign-off

```
Release version: __________
Date: __________
Released by: __________

Checklist completion: __ / 7 sections fully checked
Known exceptions (with reason and follow-up issue):
  -

Demo validation: pass / fail / N/A
Capability-prompt pass rate: __ / 12 (__%)
Quarterly refresh current: yes / no
```

## What to do when something fails

- Section 1 (example server build/test): hard block. Don't release.
- Section 2 (microservices content grep checks): hard block. Boilerplate regression is the same defect that motivated Gap 2.
- Section 3 (demo): hard block if `make demo` errors. If goldens drift intentionally, refresh and document; that's not a block.
- Section 4 (eval pass rate): soft block at <80%. At 80–90%, ship with a follow-up issue for the failing evals. At ≥90%, ship.
- Section 5 (freshness): if ecosystem facts are stale, run the quarterly refresh and re-run the relevant evals. Don't ship with knowingly stale ecosystem claims.
- Section 6 (docs): soft block. Fix and re-tag if needed.
- Section 7 (release plumbing): hard block. The artifacts must be discoverable.
