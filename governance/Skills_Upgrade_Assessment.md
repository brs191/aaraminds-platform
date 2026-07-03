# Skills Upgrade — Workspace Assessment

**Author:** Claude (independent review pass)
**Date:** 2026-05-30
**Scope:** Full scan of `C:\aaraminds`, centred on the skills system.
**Companion docs:** [`../Ranking.md`](../Ranking.md) (master ranking) · [`../skills-pack/ROADMAP.md`](../skills-pack/ROADMAP.md) (open work) · [`Upgrade_Skills.md`](Upgrade_Skills.md) (author-deepening course plan)

---

## Verdict

The content is strong; the wiring and the proof are not. As of this scan the workspace held **26 engineering skills**, **3 communication skills**, 6 personas, 9 system modules, 3 agents, 3 hooks, and a 13-tool example MCP server — governed unusually well (a master ranking, a `skill_audit.py` with a drift check, a freshness cadence, honest caveats). But two facts cap the whole system: **none of the 29 skills could trigger** (the discoverability directory was empty), and **none have been behaviorally tested** (`strength: n/t` everywhere except the 13 MCP tools). Today it is an excellent knowledge library wearing a skills costume. The highest-leverage upgrades are plumbing and proof, not more authoring.

This pass fixed the plumbing (see *Changes applied* below). The proof is still outstanding and is now the top open item.

---

## What's genuinely strong

- **Governance discipline.** `Ranking.md` is honest about what is paper-rated vs behaviorally tested, carries an explicit persona-side 9.3 cap, and records known bugs rather than hiding them. `skill_audit.py` enforces doc-vs-disk consistency. This is the best part of the system and most skill packs have nothing like it.
- **Depth leaders.** `azure-data-tier-design` (20 references, ~4,400 lines), `mcp-go-server-building` (15 refs, ~2,800), and `mcp-go-guardrails-and-safety` (9 refs, ~2,700) are deep, specific, and stack-true.
- **The MCP server is the one proven artifact.** All 13 tools were invoked over stdio JSON-RPC with golden-output validation — the only artifacts in the workspace with real `strength` evidence.
- **Skill routing is well-formed.** Every `SKILL.md` carries a "Use when / Do not use" contract that cross-references sibling skills. The design/implementation pairing (e.g. `azure-data-tier-design` ↔ `data-access-engineering`) is coherent.

---

## Scope for improvement — prioritized

### P0 — convert latent assets to real ones

**1. Discoverability (fixed in this pass).** The workspace-root `.claude/skills/` was empty, so Claude Code could not discover any of the 29 skills — they were Read-only knowledge. This pass ships `.claude/wire-skills.ps1` (Windows junctions) and `.claude/wire-skills.sh` (WSL/Linux/macOS symlinks) to populate `.claude/skills/` from disk. **Action remaining: run the script in your environment**, restart Claude Code, and confirm with `/skills`.

**2. Behavioral validation — the real proof gap.** `strength` is `n/t` for all 26 engineering skills and all 3 agents. The 12 capability prompts under `skills-pack/validation/prompts/` have never been run end-to-end (confirmed: no recorded results; ROADMAP open-work #1). Until they run, the 8.3 average quality is a paper score. **Run the 12 prompts, record `last_run`/`last_result`, and live-test the 3 agents in a registered session.** This is the single biggest confidence increase available and needs no new content.

### P1 — sharpen routing and close the governance blind spot

**3. Audit blind spot + living-doc drift (partly fixed).** `skill_audit.py` scans only the 26 engineering skills, never `instruction-os/skills/`. That is why the third communication skill, `aaraminds-project-planner` (committed 2026-05-28), was missing from `Ranking.md` and `CLAUDE.md` while the audit still reported "No drift." This pass reconciled both documents. **Action remaining: extend `skill_audit.py` to also inventory `instruction-os/skills/` and the personas/modules**, so the drift check actually covers them. Then re-rate `aaraminds-project-planner` (currently unrated) and `azure-microservices-security` (its reference was fixed 2026-05-25 but the `claude` 6 predates the fix).

**4. Description overload.** 17 of 26 skills exceed the pack's own 700-char guidance (`skill-audit-2026-05-24.md`). Measured against Anthropic's ~1024-char practical limit, only `data-access-engineering` (~1,023) sits at the edge — so this is a *routing-precision* problem, not a hard violation. With 26 skills competing to trigger, long overlapping descriptions cause misfires. The 3 communication-skill descriptions (814–943 chars) are within limit. **Trim the 17 flagged descriptions to a tight trigger + a sharp "do not use" boundary.**

### P2 — content depth and strategic clarity

**5. Depth ceiling on the thin skills.** The 5 implementation skills (`codebase-extraction-engineering`, `data-access-engineering`, `frontend-engineering`, `python-service-engineering`, `test-engineering`) ship ~190–220 reference lines each — roughly a quarter of the design skills they pair with. `azure-microservices-security` (2 refs / 476 lines) and `mcp-go-threat-modeling` (2 refs / 321 lines) are the thinnest design skills. **Deepen the implementation cluster; consider merging `mcp-go-threat-modeling` into `mcp-go-guardrails-and-safety`** (proposed in `Upgrade_Skills.md`).

**6. Named-but-unbuilt content.** `Upgrade_Skills.md`'s closer names a **missing CIF document-generation skill** and the threat-modeling merge as the pack's real authoring gaps. These are identified, not done.

**7. Communication skills are thin wrappers.** All 3 take a hard dependency on `instruction-os/` persona files with no in-skill fallback (Codex 7.0–7.5 vs Claude 9.0–9.3); if invoked without those files present they degrade. And only 3 of 6 personas have skill wrappers — Executive Narrative Advisor, AI Business Strategist, and AI Agent Blueprint Advisor have none. **Add a minimal in-skill fallback, and wrap the remaining 3 personas if they are meant to be invocable.**

**8. Platform story unresolved (ROADMAP open-work #5).** The pack is formatted for Claude Code (auto-routing, hooks, progressive disclosure) but is also run under Copilot, where none of that applies. **Commit to one framing, or stop describing Copilot-inert machinery as live features.**

---

## Recommended sequence

1. Run `.claude/wire-skills.*` → confirm `/skills` discovery. *(plumbing — done once)*
2. Run the 12 capability prompts + live-test the 3 agents → record results. *(proof)*
3. Extend `skill_audit.py` to cover `instruction-os/`; re-rate project-planner and azure-microservices-security.
4. Trim the 17 overloaded descriptions.
5. Deepen the implementation cluster; decide the threat-modeling merge; author the CIF doc-gen skill.
6. Resolve the Claude Code vs Copilot platform story.

Steps 1–2 are the leverage. Everything else is incremental polish on an already-strong base.

---

## Changes applied in this pass (2026-05-30)

- **Added** `.claude/wire-skills.ps1` and `.claude/wire-skills.sh` — idempotent, disk-enumerated wiring of all 29 skills into `.claude/skills/`, with `-Unwire` / `--unwire`. Verified end-to-end (wire → all 29 resolve → idempotent re-run → clean unwire).
- **Updated** `.claude/CLAUDE.md` — the Skills-system section now points to the wiring scripts instead of saying the symlink "is not done, defer until needed"; the instruction-os summary now reflects 3 communication skills (was 2).
- **Updated** `Ranking.md` — communication-skills count 2 → 3; added the `aaraminds-project-planner` row (unrated, Draft).
- **Updated** `.gitignore` — ignores the machine-local generated links under `/.claude/skills/`.

_Not done in this pass (require a live Claude Code session or larger authoring effort): running the 12 validation prompts, live-testing agents, trimming descriptions, deepening references._
