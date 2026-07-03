<!-- doc-consistency: ignore — frozen point-in-time snapshot, not maintained. See validation/snapshots/README.md -->

# Action Plan — wshobson-inspired adoptions for `aaraminds-claude-skills-v10.0`

**Audience:** Claude Code running inside VS Code, executing this plan against the local checkout of `aaraminds-claude-skills-v10.0`.

**Scope:** Three discrete adoptions, executed in order. Each step is self-contained with explicit acceptance criteria. After each step, stop and report status; do not proceed to the next step without confirmation from the human.

**Context:**
- The pack at v10.0 has 18 Tier-1 skills, 3 agents, 3 hooks. Phase 1, 2, and 3 of the original build plan are complete.
- This plan implements three patterns borrowed from `wshobson/agents` (a 34.9k-star Claude Code reference pack): the `inherit` model tier, a static-analysis tool for skill governance, and a flat discovery index.
- Pack stack (binding): Azure-primary; Terraform; GitHub Actions + OIDC; Azure Key Vault; AKS; Grafana + Prometheus + OpenTelemetry; Spring Boot + Go backends; React/Next.js; Postgres + MongoDB + Cosmos DB; B2B SaaS with SOC 2 / ISO 27001 in scope.
- Voice: seasoned principal engineer talking to a peer. Lead with the verdict, no hedging, name tradeoffs and pick.

---

## Pre-flight

Before starting Step 1, verify the working directory and pack state:

```bash
# From the pack root (aaraminds-claude-skills-v10.0/)
test -f .claude/CLAUDE.md && echo "PACK ROOT: OK"
test -f .claude/agents/aara-mcp-server-builder.md && echo "AGENT FILE: OK"
ls .claude/skills/ | wc -l   # expect: 18
ls .claude/agents/*.md 2>/dev/null | grep -v README | wc -l   # expect: 3
ls .claude/hooks/*.json 2>/dev/null | wc -l   # expect: 3
```

If any of these fail, stop and report. Do not proceed.

---

## Step 1 — Switch `aara-mcp-server-builder` to `inherit` model tier

**Why:** Most MCP server work is scaffolding + tool additions, which Sonnet handles cleanly. Architecture-level MCP decisions (threat surface, transport choice, package layering) benefit from Opus. With `inherit`, the user chooses per session via `claude --model opus` vs. the default, without forking the pack.

### 1.1 — Edit the agent frontmatter

Open `.claude/agents/aara-mcp-server-builder.md`. In the frontmatter block (first YAML block), change the `model` field from `opus` to `inherit`:

```yaml
# Before:
model: opus

# After:
model: inherit
```

Leave all other frontmatter fields unchanged. Do not edit the body of the agent file.

### 1.2 — Update the agents README table

Open `.claude/agents/README.md`. Find the agents table (around line 7-11). Change the `aara-mcp-server-builder` row's Model column from `opus` to `inherit`.

Then add a short paragraph immediately after the table (before the "When Claude Code invokes which agent" section) explaining the `inherit` tier:

```markdown
**On the `inherit` tier:** `aara-mcp-server-builder` is marked `inherit`, meaning it uses the session's default model rather than a hard-pinned one. Most MCP server work (scaffolding, adding tools, embedding guardrails) is well within Sonnet's range; deep architecture decisions (transport choice, threat surface, package layering) benefit from Opus. Override per session with `claude --model opus` when the task warrants it; the default Sonnet is fine for routine work.
```

### 1.3 — Update the model-choice bullet at the bottom of the README

The README has a section listing principles (around line 65). Find the bullet that reads:

```
4. **Model choice**: `opus` for complex orchestration; `sonnet` for narrower or repeatable workflows.
```

Replace it with:

```
4. **Model choice**: `opus` for complex orchestration that always justifies the cost; `sonnet` for narrower or repeatable workflows; `inherit` when the workload range is wide enough that letting the user override per session is genuinely useful.
```

### 1.4 — Acceptance criteria for Step 1

Run these checks and report results:

```bash
# Confirm the agent frontmatter changed:
grep '^model:' .claude/agents/aara-mcp-server-builder.md
# expect: model: inherit

# Confirm the other agents are unchanged:
grep -H '^model:' .claude/agents/*.md
# expect:
#   .claude/agents/aara-azure-cost-reviewer.md:model: sonnet
#   .claude/agents/aara-mcp-server-builder.md:model: inherit
#   .claude/agents/aara-senior-microservices-architect.md:model: opus

# Confirm the README table reflects the change:
grep -E 'aara-mcp-server-builder.*inherit' .claude/agents/README.md
# expect: one line matching, in the table

# Confirm the new paragraph exists:
grep -c 'On the `inherit` tier' .claude/agents/README.md
# expect: 1
```

If all four checks pass, **stop and report success**. Wait for human confirmation before proceeding to Step 2.

---

## Step 2 — Build the static-analysis tool

**Why:** The pack has 18 skills, 3 agents, 3 hooks, and 55+ reference files. The original Phase 4 (validation cleanup) deferred a runner for the 34 per-skill evals. This tool replaces that backlog with cheap, fast, runnable static checks — adapting wshobson's PluginEval pattern. It catches the anti-patterns most likely to bite a personal pack on quarterly cadence: stale dates, description bloat, orphan references, dead cross-references, off-stack drift.

**Output:** a single Python script at `validation/tools/skill_audit.py`. Standard library only. Runs in seconds. Emits a markdown report.

### 2.1 — Create the tool directory and script

Create `validation/tools/skill_audit.py` with the contents below. Standard library only — no `pip install`.

```python
#!/usr/bin/env python3
"""
skill_audit.py — Static analysis for aaraminds-claude-skills-v10.0.

Walks .claude/skills, .claude/agents, .claude/hooks. Reports anti-patterns
adapted from wshobson/agents PluginEval framework. Report-only — never edits
source files.

Usage:
  python3 validation/tools/skill_audit.py
  python3 validation/tools/skill_audit.py --emit-index
  python3 validation/tools/skill_audit.py --report-path validation/skill-audit-2026-Q3.md

Exit code: 0 if no FAIL-tier findings; 1 if any FAIL-tier finding present.
"""

import argparse
import datetime as dt
import re
import sys
from dataclasses import dataclass, field
from pathlib import Path

# ---- Configuration ----------------------------------------------------------

PACK_ROOT_MARKERS = (".claude/CLAUDE.md",)
SKILLS_DIR = Path(".claude/skills")
AGENTS_DIR = Path(".claude/agents")
HOOKS_DIR = Path(".claude/hooks")

# Anti-patterns. Severity: FAIL (must fix), WARN (consider fixing).
CHECKS = {
    "EMPTY_DESCRIPTION":    ("FAIL", "description < 200 characters"),
    "MISSING_TRIGGER":      ("FAIL", "description does not contain 'Use when'"),
    "DESCRIPTION_OVERLOAD": ("WARN", "description > 700 characters"),
    "BLOATED_SKILL":        ("FAIL", "SKILL.md body > 130 lines"),
    "ANEMIC_SKILL":         ("WARN", "SKILL.md body < 70 lines"),
    "MISSING_SECTION":      ("FAIL", "SKILL.md missing one of: When to use / Worked example / Anti-pattern / Verification / What to read next"),
    "NO_BROWNFIELD_EXAMPLE":("WARN", "no Worked example mentions brownfield / existing / migrate / legacy"),
    "STALE_LAST_UPDATED":   ("WARN", "last_updated > 180 days ago"),
    "OFF_STACK_DRIFT":      ("WARN", "body references off-stack tool (AWS / Bicep / GitLab / Datadog / Pulumi / Azure DevOps / Node.js backend) without negation"),
    "ORPHAN_REFERENCE":     ("WARN", "file in references/ not linked from SKILL.md or any sibling reference"),
    "DEAD_CROSS_REF":       ("FAIL", "markdown link points at a non-existent file"),
}

# Required SKILL.md sections (case-insensitive header match)
REQUIRED_SECTIONS = [
    r"^##\s+When to use",
    r"^##\s+.*[Ww]orked example",
    r"^##\s+.*[Aa]nti-?pattern",
    r"^##\s+.*[Vv]erification",
    r"^##\s+.*([Rr]ead next|[Ww]hat to read next)",
]

# Off-stack tool tokens. Match standalone words; common false-positives (AWS keys
# as redaction target, "unlike AWS", "not Bicep") are stripped via context.
OFF_STACK_TOKENS = [
    r"\bAWS\b", r"\bDynamoDB\b", r"\bLambda\b(?!Test|Expr)",
    r"\bBicep\b", r"\bPulumi\b",
    r"\bGitLab\b", r"\bDatadog\b", r"\bNew Relic\b",
    r"\bAzure DevOps Pipelines\b",
    r"\bExpress\.js\b", r"\bnode\.js backend\b",
]

# Negation context — if any of these words appear near an off-stack token, skip.
NEGATION_CONTEXT = re.compile(
    r"(not |never |no |unlike |instead of |rather than |aws or gcp|aws/gcp|gcp/aws|other clouds|redaction target|redactor for|stripped from|do not (?:use|introduce))",
    re.IGNORECASE,
)


# ---- Data structures --------------------------------------------------------

@dataclass
class Finding:
    skill: str
    check: str
    severity: str
    detail: str


@dataclass
class SkillInfo:
    name: str
    skill_md_path: Path
    body_lines: int
    description: str
    last_updated: str
    version: str
    references: list = field(default_factory=list)


# ---- Pack-root resolution ---------------------------------------------------

def find_pack_root(start: Path) -> Path:
    """Walk up from `start` looking for the pack-root marker."""
    p = start.resolve()
    for parent in [p] + list(p.parents):
        if all((parent / m).exists() for m in PACK_ROOT_MARKERS):
            return parent
    sys.exit(f"ERROR: could not locate pack root (looked for {PACK_ROOT_MARKERS}). Run from inside the pack tree.")


# ---- Frontmatter parsing ----------------------------------------------------

FRONTMATTER_RE = re.compile(r"^---\n(.*?)\n---\n", re.DOTALL)


def parse_frontmatter(text: str) -> dict:
    m = FRONTMATTER_RE.match(text)
    if not m:
        return {}
    block = m.group(1)
    out = {}
    # Simple key: value parser. Multi-line values supported by indent continuation.
    current_key = None
    for raw in block.splitlines():
        if not raw.strip():
            continue
        if re.match(r"^[a-z_][a-z0-9_]*:", raw):
            key, _, val = raw.partition(":")
            current_key = key.strip()
            out[current_key] = val.strip()
        elif current_key and (raw.startswith(" ") or raw.startswith("\t")):
            # Continuation line — typically a tools list or wrapped value
            out[current_key] = (out[current_key] + " " + raw.strip()).strip()
    return out


def body_lines_count(text: str) -> int:
    m = FRONTMATTER_RE.match(text)
    if m:
        body = text[m.end():]
    else:
        body = text
    return len(body.splitlines())


# ---- Checks -----------------------------------------------------------------

def check_skill(skill_md: Path, all_md_paths: set) -> tuple[list, SkillInfo]:
    findings = []
    text = skill_md.read_text(encoding="utf-8")
    fm = parse_frontmatter(text)
    skill_name = skill_md.parent.name
    desc = fm.get("description", "")
    last_updated = fm.get("last_updated", "")
    version = fm.get("version", "")
    body_lines = body_lines_count(text)
    refs_dir = skill_md.parent / "references"
    refs = sorted(refs_dir.rglob("*.md")) if refs_dir.exists() else []

    info = SkillInfo(
        name=skill_name,
        skill_md_path=skill_md,
        body_lines=body_lines,
        description=desc,
        last_updated=last_updated,
        version=version,
        references=refs,
    )

    # EMPTY_DESCRIPTION
    if len(desc) < 200:
        findings.append(Finding(skill_name, "EMPTY_DESCRIPTION", *CHECKS["EMPTY_DESCRIPTION"][:1],
                                detail=f"{len(desc)} chars"))

    # MISSING_TRIGGER
    if "Use when" not in desc and "use when" not in desc:
        findings.append(Finding(skill_name, "MISSING_TRIGGER", CHECKS["MISSING_TRIGGER"][0],
                                detail="description lacks 'Use when' phrase"))

    # DESCRIPTION_OVERLOAD
    if len(desc) > 700:
        findings.append(Finding(skill_name, "DESCRIPTION_OVERLOAD", CHECKS["DESCRIPTION_OVERLOAD"][0],
                                detail=f"{len(desc)} chars (recommend <= 700)"))

    # BLOATED_SKILL / ANEMIC_SKILL
    if body_lines > 130:
        findings.append(Finding(skill_name, "BLOATED_SKILL", CHECKS["BLOATED_SKILL"][0],
                                detail=f"{body_lines} body lines"))
    elif body_lines < 70:
        findings.append(Finding(skill_name, "ANEMIC_SKILL", CHECKS["ANEMIC_SKILL"][0],
                                detail=f"{body_lines} body lines"))

    # MISSING_SECTION
    missing = []
    for pat in REQUIRED_SECTIONS:
        if not re.search(pat, text, re.MULTILINE):
            missing.append(pat)
    if missing:
        findings.append(Finding(skill_name, "MISSING_SECTION", CHECKS["MISSING_SECTION"][0],
                                detail=f"missing: {', '.join(missing)}"))

    # NO_BROWNFIELD_EXAMPLE
    # Find "Worked example" section, check next ~30 lines for brownfield language
    example_match = re.search(r"^##\s+.*[Ww]orked example.*?$", text, re.MULTILINE)
    if example_match:
        chunk_start = example_match.end()
        chunk = text[chunk_start:chunk_start + 2000]
        if not re.search(r"\b(brownfield|existing|migrat|legacy|retrofit|upgrade)", chunk, re.IGNORECASE):
            findings.append(Finding(skill_name, "NO_BROWNFIELD_EXAMPLE", CHECKS["NO_BROWNFIELD_EXAMPLE"][0],
                                    detail="worked example does not appear to be brownfield"))

    # STALE_LAST_UPDATED
    if last_updated:
        try:
            d = dt.date.fromisoformat(last_updated)
            age_days = (dt.date.today() - d).days
            if age_days > 180:
                findings.append(Finding(skill_name, "STALE_LAST_UPDATED", CHECKS["STALE_LAST_UPDATED"][0],
                                        detail=f"{age_days} days ago ({last_updated})"))
        except ValueError:
            findings.append(Finding(skill_name, "STALE_LAST_UPDATED", CHECKS["STALE_LAST_UPDATED"][0],
                                    detail=f"unparseable date: {last_updated!r}"))

    # OFF_STACK_DRIFT
    for token_re in OFF_STACK_TOKENS:
        for match in re.finditer(token_re, text):
            # Look at the surrounding context (±100 chars) for negation
            context_start = max(0, match.start() - 100)
            context_end = min(len(text), match.end() + 100)
            context = text[context_start:context_end]
            if NEGATION_CONTEXT.search(context):
                continue
            findings.append(Finding(skill_name, "OFF_STACK_DRIFT", CHECKS["OFF_STACK_DRIFT"][0],
                                    detail=f"'{match.group(0)}' at offset {match.start()}"))
            break  # one finding per token per skill is enough

    # DEAD_CROSS_REF — check all markdown links in SKILL.md and references
    link_re = re.compile(r"\[([^\]]*)\]\(([^)\s]+\.md[^)\s]*)\)")
    for source in [skill_md] + refs:
        try:
            content = source.read_text(encoding="utf-8")
        except (OSError, UnicodeDecodeError):
            continue
        for m in link_re.finditer(content):
            link = m.group(2).split("#")[0]
            if not link or link.startswith(("http", "mailto:")):
                continue
            try:
                target = (source.parent / link).resolve()
            except (OSError, ValueError):
                continue
            if not target.exists():
                rel_source = source.relative_to(skill_md.parent)
                findings.append(Finding(skill_name, "DEAD_CROSS_REF", CHECKS["DEAD_CROSS_REF"][0],
                                        detail=f"{rel_source} -> {link}"))

    # ORPHAN_REFERENCE — file in references/ not linked from SKILL.md or any sibling reference
    if refs:
        linked = set()
        all_text_in_skill = [skill_md.read_text(encoding="utf-8")]
        for r in refs:
            try:
                all_text_in_skill.append(r.read_text(encoding="utf-8"))
            except (OSError, UnicodeDecodeError):
                pass
        joined = "\n".join(all_text_in_skill)
        for ref in refs:
            ref_basename = ref.name
            # Look for any reference to this filename in any of the skill's files
            if ref_basename not in joined:
                rel = ref.relative_to(skill_md.parent)
                findings.append(Finding(skill_name, "ORPHAN_REFERENCE", CHECKS["ORPHAN_REFERENCE"][0],
                                        detail=str(rel)))

    return findings, info


# ---- Report writer ----------------------------------------------------------

def write_report(findings: list, infos: list, agents_info: list, hooks_info: list,
                 report_path: Path) -> None:
    today = dt.date.today().isoformat()

    fails = [f for f in findings if CHECKS[f.check][0] == "FAIL"]
    warns = [f for f in findings if CHECKS[f.check][0] == "WARN"]

    lines = []
    lines.append(f"# Skill audit — {today}")
    lines.append("")
    lines.append(f"- Skills scanned: **{len(infos)}**")
    lines.append(f"- FAIL findings: **{len(fails)}**")
    lines.append(f"- WARN findings: **{len(warns)}**")
    lines.append("")

    # FAILURES
    lines.append("## Failures (must fix)")
    lines.append("")
    if not fails:
        lines.append("_None._")
    else:
        lines.append("| Skill | Check | Detail |")
        lines.append("|---|---|---|")
        for f in fails:
            lines.append(f"| `{f.skill}` | `{f.check}` | {f.detail} |")
    lines.append("")

    # WARNINGS
    lines.append("## Warnings (consider fixing)")
    lines.append("")
    if not warns:
        lines.append("_None._")
    else:
        lines.append("| Skill | Check | Detail |")
        lines.append("|---|---|---|")
        for f in warns:
            lines.append(f"| `{f.skill}` | `{f.check}` | {f.detail} |")
    lines.append("")

    # Inventory
    lines.append("## Inventory")
    lines.append("")
    lines.append("| Skill | Body lines | Description chars | References | Version | Last updated |")
    lines.append("|---|---|---|---|---|---|")
    for info in sorted(infos, key=lambda i: i.name):
        lines.append(f"| `{info.name}` | {info.body_lines} | {len(info.description)} | {len(info.references)} | {info.version} | {info.last_updated} |")
    lines.append("")

    # Agents inventory
    lines.append("## Agents")
    lines.append("")
    if agents_info:
        lines.append("| Agent | Model | Description chars |")
        lines.append("|---|---|---|")
        for a in agents_info:
            lines.append(f"| `{a['name']}` | {a['model']} | {a['desc_chars']} |")
    else:
        lines.append("_No agents found._")
    lines.append("")

    # Hooks inventory
    lines.append("## Hooks")
    lines.append("")
    if hooks_info:
        lines.append("| Hook | Bytes |")
        lines.append("|---|---|")
        for h in hooks_info:
            lines.append(f"| `{h['name']}` | {h['size']} |")
    else:
        lines.append("_No hooks found._")
    lines.append("")

    lines.append("---")
    lines.append("")
    lines.append("_Generated by `validation/tools/skill_audit.py`. Report-only — no source files were modified._")

    report_path.parent.mkdir(parents=True, exist_ok=True)
    report_path.write_text("\n".join(lines), encoding="utf-8")


# ---- Index writer -----------------------------------------------------------

def write_index(infos: list, agents_info: list, hooks_info: list, pack_root: Path) -> None:
    """Emit .claude/INDEX.md — flat, generated, do-not-hand-edit discovery index."""
    today = dt.date.today().isoformat()
    lines = []
    lines.append("<!-- AUTO-GENERATED by validation/tools/skill_audit.py --emit-index — do not hand-edit. -->")
    lines.append("")
    lines.append("# Pack Index")
    lines.append("")
    lines.append(f"_Last generated: {today}._")
    lines.append("")
    lines.append("Flat discovery index of every skill, agent, hook, and pattern card in the pack. Regenerate with `python3 validation/tools/skill_audit.py --emit-index`.")
    lines.append("")

    # Skills
    lines.append("## Skills")
    lines.append("")
    lines.append("| Skill | One-line scope | References | Last updated |")
    lines.append("|---|---|---|---|")
    for info in sorted(infos, key=lambda i: i.name):
        # First sentence of description as one-line scope
        first_sentence = re.split(r"(?<=[.!?])\s+", info.description.strip(), 1)[0]
        # Trim if still too long
        if len(first_sentence) > 160:
            first_sentence = first_sentence[:157] + "..."
        lines.append(f"| [`{info.name}`](skills/{info.name}/SKILL.md) | {first_sentence} | {len(info.references)} | {info.last_updated} |")
    lines.append("")

    # Pattern cards
    lines.append("## Pattern cards")
    lines.append("")
    pattern_rows = []
    for info in infos:
        patterns_dir = info.skill_md_path.parent / "references" / "patterns"
        if patterns_dir.exists():
            for p in sorted(patterns_dir.glob("*.md")):
                pattern_rows.append((p.stem, info.name, p))
    if pattern_rows:
        lines.append("| Pattern | Owning skill |")
        lines.append("|---|---|")
        for pattern_name, owning_skill, path in sorted(pattern_rows):
            rel = path.relative_to(pack_root / ".claude")
            lines.append(f"| [`{pattern_name}`]({rel}) | `{owning_skill}` |")
    else:
        lines.append("_No pattern cards found._")
    lines.append("")

    # Agents
    lines.append("## Agents")
    lines.append("")
    if agents_info:
        lines.append("| Agent | Model | Description (first sentence) |")
        lines.append("|---|---|---|")
        for a in agents_info:
            first = re.split(r"(?<=[.!?])\s+", a['description'].strip(), 1)[0]
            if len(first) > 200:
                first = first[:197] + "..."
            lines.append(f"| [`{a['name']}`](agents/{a['name']}.md) | {a['model']} | {first} |")
    else:
        lines.append("_No agents found._")
    lines.append("")

    # Hooks
    lines.append("## Hooks")
    lines.append("")
    if hooks_info:
        lines.append("| Hook | Bytes | Event (from filename hint) |")
        lines.append("|---|---|---|")
        for h in hooks_info:
            event = "PreToolUse" if "commit" in h['name'] or "command" in h['name'] else "—"
            lines.append(f"| [`{h['name']}`](hooks/{h['name']}) | {h['size']} | {event} |")
    else:
        lines.append("_No hooks found._")
    lines.append("")

    lines.append("---")
    lines.append("")
    lines.append("_This file is generated. To change its content, edit the source frontmatter or rerun the generator. Manual edits will be overwritten._")

    index_path = pack_root / ".claude" / "INDEX.md"
    index_path.write_text("\n".join(lines), encoding="utf-8")


# ---- Collectors -------------------------------------------------------------

def collect_agents(pack_root: Path) -> list:
    out = []
    agents_dir = pack_root / AGENTS_DIR
    if not agents_dir.exists():
        return out
    for f in sorted(agents_dir.glob("*.md")):
        if f.stem.upper() == "README":
            continue
        text = f.read_text(encoding="utf-8")
        fm = parse_frontmatter(text)
        out.append({
            "name": fm.get("name", f.stem),
            "model": fm.get("model", "?"),
            "description": fm.get("description", ""),
            "desc_chars": len(fm.get("description", "")),
        })
    return out


def collect_hooks(pack_root: Path) -> list:
    out = []
    hooks_dir = pack_root / HOOKS_DIR
    if not hooks_dir.exists():
        return out
    for f in sorted(hooks_dir.glob("*.json")):
        out.append({
            "name": f.name,
            "size": f.stat().st_size,
        })
    return out


# ---- Main -------------------------------------------------------------------

def main() -> int:
    ap = argparse.ArgumentParser(description="Static analysis for the Claude Skills pack.")
    ap.add_argument("--emit-index", action="store_true",
                    help="Also write .claude/INDEX.md")
    ap.add_argument("--report-path", default=None,
                    help="Path to write the audit report (default: validation/skill-audit-<date>.md)")
    args = ap.parse_args()

    pack_root = find_pack_root(Path.cwd())
    skills_dir = pack_root / SKILLS_DIR

    if not skills_dir.exists():
        sys.exit(f"ERROR: {skills_dir} does not exist.")

    skill_mds = sorted(skills_dir.glob("*/SKILL.md"))
    if not skill_mds:
        sys.exit(f"ERROR: no SKILL.md files found under {skills_dir}.")

    # Build a set of every .md path under .claude for cross-ref validation
    all_md_paths = set((pack_root / ".claude").rglob("*.md"))

    all_findings = []
    infos = []
    for skill_md in skill_mds:
        findings, info = check_skill(skill_md, all_md_paths)
        all_findings.extend(findings)
        infos.append(info)

    agents_info = collect_agents(pack_root)
    hooks_info = collect_hooks(pack_root)

    if args.report_path:
        report_path = Path(args.report_path)
    else:
        report_path = pack_root / "validation" / f"skill-audit-{dt.date.today().isoformat()}.md"

    write_report(all_findings, infos, agents_info, hooks_info, report_path)
    print(f"Audit report written to: {report_path.relative_to(pack_root)}")

    fails = sum(1 for f in all_findings if CHECKS[f.check][0] == "FAIL")
    warns = sum(1 for f in all_findings if CHECKS[f.check][0] == "WARN")
    print(f"Findings: {fails} FAIL, {warns} WARN, {len(infos)} skills scanned.")

    if args.emit_index:
        write_index(infos, agents_info, hooks_info, pack_root)
        print(f"Index written to: .claude/INDEX.md")

    return 1 if fails > 0 else 0


if __name__ == "__main__":
    sys.exit(main())
```

Make the script executable:

```bash
chmod +x validation/tools/skill_audit.py
```

### 2.2 — Create a README for the tool

Create `validation/tools/README.md`:

```markdown
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

- **FAIL** (must fix): `EMPTY_DESCRIPTION`, `MISSING_TRIGGER`, `BLOATED_SKILL`, `MISSING_SECTION`, `DEAD_CROSS_REF`
- **WARN** (consider fixing): `DESCRIPTION_OVERLOAD`, `ANEMIC_SKILL`, `NO_BROWNFIELD_EXAMPLE`, `STALE_LAST_UPDATED`, `OFF_STACK_DRIFT`, `ORPHAN_REFERENCE`

### What it does not do

- Does not modify any source files. Report-only.
- Does not run LLM judges or Monte Carlo simulation (the deeper PluginEval layers). For personal use, static checks catch most defects.
- Does not validate content quality — only structural quality.
```

### 2.3 — Run the tool against current v10.0

```bash
python3 validation/tools/skill_audit.py
```

Read the generated `validation/skill-audit-<date>.md` end to end. Report what was found.

**Expected findings on v10.0 (from prior review):**
- `DESCRIPTION_OVERLOAD` (WARN) on roughly 6 skills with descriptions > 700 chars (`microservices-architecture-reviewer` is the longest at 922; `azure-data-tier-design` at 880; etc.)
- `OFF_STACK_DRIFT` (WARN) on `mcp-go-guardrails-and-safety` — the AWS-keys-as-redaction-target mention. This is a known false positive; do **not** edit the skill to remove it.
- Probably zero `FAIL`-tier findings.

If FAIL-tier findings appear in skills other than `mcp-go-guardrails-and-safety`, **stop and report**. Do not auto-fix. The human decides what to do with the findings.

### 2.4 — Acceptance criteria for Step 2

```bash
test -f validation/tools/skill_audit.py && echo "TOOL: present"
test -f validation/tools/README.md && echo "README: present"
python3 validation/tools/skill_audit.py >/dev/null
ls validation/skill-audit-*.md 2>/dev/null | head -1
# expect: a skill-audit-<today>.md file exists
```

If the tool runs end-to-end without crashing, the report exists, and zero FAIL-tier findings (except as noted), **stop and report**. Wait for human confirmation before proceeding to Step 3.

---

## Step 3 — Generate the flat discovery index

**Why:** With 18 skills, 3 agents, 3 hooks, and 21+ pattern cards distributed across multiple skill folders, cross-skill discovery lives only in human memory. A flat, regenerated index at `.claude/INDEX.md` lets future-you (and Claude) find which skill owns which pattern, which agent invokes which skill, which hook fires on which event.

The index is generated, not authored. It regenerates from source-of-truth on every audit run.

### 3.1 — Run the tool with the `--emit-index` flag

```bash
python3 validation/tools/skill_audit.py --emit-index
```

This runs the same audit *and* writes `.claude/INDEX.md` as a byproduct.

### 3.2 — Verify the index contents

Open `.claude/INDEX.md`. Confirm it contains four sections:

1. **Skills** table — 18 rows, each linking to `skills/<name>/SKILL.md`, with a one-line scope, reference count, and last-updated date
2. **Pattern cards** table — one row per pattern card found anywhere in any skill's `references/patterns/` folder
3. **Agents** table — 3 rows: `aara-senior-microservices-architect` (opus), `aara-mcp-server-builder` (inherit), `aara-azure-cost-reviewer` (sonnet)
4. **Hooks** table — 3 rows: `pre-commit-lint.json`, `test-before-commit.json`, `block-dangerous-commands.json`

### 3.3 — Add INDEX.md to CLAUDE.md governance

Open `.claude/CLAUDE.md`. Find the "How this pack is organized" section. Update the directory tree comment to mention `INDEX.md`:

```
.claude/
  CLAUDE.md                       # This file
  INDEX.md                        # Auto-generated discovery index — regenerate with skill_audit.py --emit-index
  FEEDBACK.md                     # Pack-usage feedback log
  skills/
    <skill-name>/
      SKILL.md
      references/
        <topic>.md
        ...
        patterns/
          <pattern>.md
  agents/
  hooks/
```

Then add a short subsection at the bottom of CLAUDE.md, before "What this file is not":

```markdown
## Regenerating the index

The pack ships with a flat discovery index at `.claude/INDEX.md`. It is generated from frontmatter and directory structure by `validation/tools/skill_audit.py`. Regenerate it whenever skills, agents, or hooks are added, removed, or renamed:

\`\`\`bash
python3 validation/tools/skill_audit.py --emit-index
\`\`\`

Do not hand-edit `.claude/INDEX.md`. Manual edits are overwritten on next regeneration.
```

### 3.4 — Acceptance criteria for Step 3

```bash
test -f .claude/INDEX.md && echo "INDEX: present"

# Section presence
grep -c '^## Skills' .claude/INDEX.md           # expect: 1
grep -c '^## Pattern cards' .claude/INDEX.md    # expect: 1
grep -c '^## Agents' .claude/INDEX.md           # expect: 1
grep -c '^## Hooks' .claude/INDEX.md            # expect: 1

# Counts: 18 skills, 3 agents, 3 hooks
# Skills count: 18 SKILL.md links
grep -c 'skills/.*/SKILL.md' .claude/INDEX.md   # expect: 18

# Agent count
grep -c 'agents/.*\.md' .claude/INDEX.md        # expect: 3

# Confirm aara-mcp-server-builder shows inherit in the index
grep 'aara-mcp-server-builder.*inherit' .claude/INDEX.md
# expect: one matching line

# CLAUDE.md regeneration note
grep -c 'Regenerating the index' .claude/CLAUDE.md   # expect: 1
```

If all checks pass, **stop and report**. The plan is complete.

---

## Final summary report

After Step 3 acceptance, generate a final summary for the human covering:

1. Which files were created (with paths)
2. Which files were modified (with one-line summary of the change per file)
3. The current count from the audit report: skills scanned, FAIL findings, WARN findings
4. Any unexpected findings worth surfacing
5. Suggested next action: commit on a feature branch (e.g. `wshobson-adoptions-2026-Q2`), open a PR with the diff, run the audit one more time after merge as a clean baseline

---

## Boundaries — what NOT to do

- **Do not** modify content of any existing `SKILL.md`, `reference.md`, pattern card, or agent file. Step 1 touches the frontmatter `model:` field on one agent and the README; nothing else.
- **Do not** auto-fix audit findings. The tool is report-only. If the audit surfaces a `DESCRIPTION_OVERLOAD` warning, do not trim descriptions yourself.
- **Do not** add new skills, agents, or hooks. The plan is structural infrastructure (tool + index + model-tier change), not content.
- **Do not** install Python dependencies. The tool must run on standard library only. If you find yourself wanting to `pip install`, stop and report.
- **Do not** proceed past an acceptance-criteria failure. Stop and report. The human decides whether to fix the failure or amend the plan.
- **Do not** commit changes. Leave the working directory dirty. The human will review and commit.
- **Do not** rebuild the audit report from the previously-attached `audit-2026-Q2.md` — that's a different document, kept for historical reference.

---

## End-to-end run order (for reference)

```bash
# Pre-flight
test -f .claude/CLAUDE.md && echo "PACK ROOT: OK"

# Step 1
# (edit .claude/agents/aara-mcp-server-builder.md frontmatter)
# (edit .claude/agents/README.md table + add paragraph + update bullet)
grep '^model:' .claude/agents/aara-mcp-server-builder.md   # confirm: model: inherit

# Step 2
mkdir -p validation/tools
# (write validation/tools/skill_audit.py)
# (write validation/tools/README.md)
chmod +x validation/tools/skill_audit.py
python3 validation/tools/skill_audit.py

# Step 3
python3 validation/tools/skill_audit.py --emit-index
# (edit .claude/CLAUDE.md to mention INDEX.md + add regeneration subsection)
test -f .claude/INDEX.md && echo "INDEX: OK"
```

If the human runs `python3 validation/tools/skill_audit.py --emit-index` later (e.g. quarterly), they get both the audit report and a fresh index in one command.
