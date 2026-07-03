# Validation Tools

## skill_audit.py

Static analysis for the AaraMinds Claude Skills Pack. Walks `.claude/skills`, `.claude/agents`, `.claude/hooks` and emits a markdown audit report.

Inspired by [wshobson/agents PluginEval](https://github.com/wshobson/agents/blob/main/docs/plugin-eval.md), trimmed to the static-analysis layer only.

### Usage

```bash
# From the pack root:
python3 validation/tools/skill_audit.py                      # writes validation/skill-audit-<date>.md
python3 validation/tools/skill_audit.py --emit-index         # also writes .claude/INDEX.md
python3 validation/tools/skill_audit.py --report-path /tmp/foo.md
```

Exit code is 0 if zero FAIL-tier findings, 1 otherwise. Suitable as a quarterly governance ritual or as a pre-commit hook.

### What it checks

- **FAIL** (must fix): `EMPTY_DESCRIPTION`, `MISSING_TRIGGER`, `BLOATED_SKILL`, `MISSING_SECTION`, `DEAD_CROSS_REF`, `DOC_COUNT_DRIFT`, `INDEX_STALE`
- **WARN** (consider fixing): `DESCRIPTION_OVERLOAD`, `ANEMIC_SKILL`, `NO_BROWNFIELD_EXAMPLE`, `STALE_LAST_UPDATED`, `OFF_STACK_DRIFT`, `ORPHAN_REFERENCE`

### Doc-consistency pass

Beyond per-skill structure, the tool runs a doc-consistency pass: it derives the
true skill / agent / hook / reference / pattern-card counts from the filesystem,
then scans every *living* document (`README.md`, `ROADMAP.md`, `usage.md`,
`how-to-use-in-vscode.md`, `migration-map.md`, `copilot/README.md`, and the
`validation/` docs) for count claims that disagree.

- `DOC_COUNT_DRIFT` — a living document states a count that does not match disk.
- `INDEX_STALE` — `.claude/INDEX.md` row counts disagree with disk; rerun `--emit-index`.

Two escape hatches keep this honest without false positives. Dated point-in-time
records under `validation/snapshots/` are skipped wholesale, and any file
containing the marker `<!-- doc-consistency: ignore -->` is exempt — use it for a
document that deliberately carries a historical or rolling count. Numbers sitting
in a clearly historical context ("v9.0", "Phase 1", "migrated", "stopped at")
are also ignored, so migration history does not trip the check.

### What it does not do

- Does not modify any source files. Report-only.
- Does not run LLM judges or Monte Carlo simulation (the deeper PluginEval layers). For personal use, static checks catch most defects.
- Does not validate content quality — only structural quality.
