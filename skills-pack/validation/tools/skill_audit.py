#!/usr/bin/env python3
"""
skill_audit.py — Static analysis for aaraminds-claude-skills-v10.0.

Walks .claude/skills, .claude/agents, .claude/hooks. Reports anti-patterns
adapted from wshobson/agents PluginEval framework. Also runs a doc-consistency
pass: every living document (README, ROADMAP, usage, ...) must agree with disk
on skill / agent / hook / reference counts; dated records under
validation/snapshots/ and files carrying a "doc-consistency: ignore" marker are
exempt. Report-only — never edits source files.

Usage:
  python3 validation/tools/skill_audit.py
  python3 validation/tools/skill_audit.py --emit-index
  python3 validation/tools/skill_audit.py --report-path validation/skill-audit-2026-Q3.md

Exit code: 0 if no FAIL-tier findings; 1 if any FAIL-tier finding present.
"""

from __future__ import annotations

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
    "OFF_STACK_DRIFT":      ("WARN", "SKILL.md or a reference file names an off-stack tool (AWS / Bicep / GitLab / Datadog / Pulumi / Azure DevOps / Node.js backend) without negation"),
    "STALE_TERM":           ("WARN", "SKILL.md or a reference file uses a renamed/superseded product name (e.g. 'Azure AD' — renamed Microsoft Entra ID)"),
    "ORPHAN_REFERENCE":     ("WARN", "file in references/ not linked from SKILL.md or any sibling reference"),
    "DEAD_CROSS_REF":       ("FAIL", "markdown link points at a non-existent file"),
    "DOC_COUNT_DRIFT":      ("FAIL", "a living document states a skill/agent/hook/reference count that disagrees with disk"),
    "INDEX_STALE":          ("FAIL", ".claude/INDEX.md row counts disagree with disk — regenerate with --emit-index"),
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
    r"\bAWS\b", r"\bDynamoDB\b", r"\bAWS Lambda\b",
    r"\bBicep\b", r"\bPulumi\b",
    r"\bGitLab\b", r"\bDatadog\b", r"\bNew Relic\b",
    r"\bAzure DevOps Pipelines\b",
    r"\bExpress\.js\b", r"\bnode\.js backend\b",
]

# Negation context — if any of these words appear near an off-stack token, skip.
NEGATION_CONTEXT = re.compile(
    r"(not |never |no |unlike |instead of |rather than |aws or gcp|aws/gcp|gcp/aws|"
    r"other clouds|redaction target|redactor for|redact|stripped from|key shape|"
    r"drift|non-azure|secret-pattern|do not (?:use|introduce))",
    re.IGNORECASE,
)

# Renamed / superseded product names — an on-stack thing called by an old name.
# Not drift to the wrong tool; drift to a stale name for the right one. WARN-tier.
STALE_TERMS = [
    (re.compile(r"\bAzure Active Directory\b"), "Microsoft Entra ID"),
    (re.compile(r"\bAzure AD\b"), "Microsoft Entra ID"),
    (re.compile(r"\bAAD\b"), "Microsoft Entra ID"),
]


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
    # Find "Worked example" section; include the heading itself so a brownfield
    # keyword in the section title (e.g. "## Worked example — brownfield: ...")
    # is counted as evidence.
    example_match = re.search(r"^##\s+.*[Ww]orked example.*?$", text, re.MULTILINE)
    if example_match:
        chunk_start = example_match.start()
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

    # OFF_STACK_DRIFT + STALE_TERM — scan SKILL.md *and* every reference file.
    # The original check only saw SKILL.md, but drift overwhelmingly lives in the
    # Tier-2 references that progressive disclosure routes Claude into; the router
    # can be clean while the reference beneath it names AWS or "Azure AD". Both
    # checks share the negation-context filter ("not Bicep", "unlike AWS").
    scan_sources = [("SKILL.md", text)]
    for ref in refs:
        try:
            scan_sources.append((str(ref.relative_to(skill_md.parent)),
                                 ref.read_text(encoding="utf-8")))
        except (OSError, UnicodeDecodeError):
            continue
    for label, content in scan_sources:
        for token_re in OFF_STACK_TOKENS:
            for match in re.finditer(token_re, content):
                context = content[max(0, match.start() - 100):
                                  min(len(content), match.end() + 100)]
                if NEGATION_CONTEXT.search(context):
                    continue
                findings.append(Finding(skill_name, "OFF_STACK_DRIFT",
                                        CHECKS["OFF_STACK_DRIFT"][0],
                                        detail=f"{label}: '{match.group(0)}' "
                                               f"(offset {match.start()})"))
                break  # one finding per token per file is enough
        for stale_re, canonical in STALE_TERMS:
            for match in re.finditer(stale_re, content):
                context = content[max(0, match.start() - 100):
                                  min(len(content), match.end() + 100)]
                if NEGATION_CONTEXT.search(context):
                    continue
                findings.append(Finding(skill_name, "STALE_TERM",
                                        CHECKS["STALE_TERM"][0],
                                        detail=f"{label}: '{match.group(0)}' "
                                               f"— use {canonical}"))
                break  # one finding per stale term per file is enough

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


# ---- Doc-consistency check --------------------------------------------------
#
# The pack's most persistent failure mode is documentation drift: the same
# derived fact (skill count, reference count, ...) is re-typed by hand into a
# dozen prose documents, and the copies diverge. This pass treats the filesystem
# as the single source of truth and fails the audit when a *living* document
# disagrees with it. Dated point-in-time records under validation/snapshots/ and
# any file carrying the ignore marker below are deliberately exempt.

DOC_IGNORE_MARKER = "doc-consistency: ignore"

# Living-doc files to scan, as globs relative to the pack root. Skill content
# (.claude/skills/), the snapshots directory, the generated index, the feedback
# log, and generated audit reports are excluded by design.
DOC_SCAN_GLOBS = (
    "*.md",
    "copilot/*.md",
    "validation/*.md",
    "validation/governance/*.md",
    "validation/tools/*.md",
    "validation/prompts/*.md",
)
DOC_SCAN_EXCLUDE_PREFIXES = ("validation/snapshots/",)

# Count-claim patterns, each with the truth key it must match. The "skills"
# pattern requires the literal "Tier-1" so that subgroup headings such as
# "Azure platform (5 skills)" are not mistaken for a pack-total claim.
COUNT_CLAIM_PATTERNS = (
    (re.compile(r"(?<!Tier-)\b(\d{1,4})\s+Tier-1\s+skills?\b", re.IGNORECASE), "skills"),
    (re.compile(r"(?<!Tier-)\b(\d{1,4})\s+agents?\b", re.IGNORECASE), "agents"),
    (re.compile(r"(?<!Tier-)\b(\d{1,4})\s+hooks?\b", re.IGNORECASE), "hooks"),
    (re.compile(r"(?<!Tier-)\b(\d{1,4})\s+pattern\s+cards?\b", re.IGNORECASE), "pattern_cards"),
    (re.compile(r"(?<!Tier-)\b(\d{1,4})\s+(?:Tier-2\s+)?references\b", re.IGNORECASE), "refs"),
    (re.compile(r"(?<!Tier-)\b(\d{1,4})\s+reference\s+(?:markdown\s+)?files?\b", re.IGNORECASE), "refs"),
)

# A number sitting in a historical or negated context is a mention of a *past*
# value, not a current-state claim — skip it. Same technique as OFF_STACK_DRIFT.
HISTORICAL_CONTEXT = re.compile(
    r"(v9\.0|v9 |phase[ -]?1\b|phase[ -]?2\b|migrat|earlier|pre-2026|"
    r"previously|stopped at|no longer|formerly|not the\b|→|grew to|"
    r"described in pre|the other 7|originally)",
    re.IGNORECASE,
)


def disk_truth(infos: list, agents_info: list, hooks_info: list) -> dict:
    """Ground-truth counts derived from the filesystem."""
    refs = [r for info in infos for r in info.references]
    pattern_cards = [r for r in refs if "patterns" in r.parts]
    return {
        "skills": len(infos),
        "agents": len(agents_info),
        "hooks": len(hooks_info),
        "pattern_cards": len(pattern_cards),
        "refs": len(refs),
    }


def check_doc_consistency(pack_root: Path, infos: list, agents_info: list,
                          hooks_info: list) -> list:
    """Scan living documentation for count claims that disagree with disk."""
    findings = []
    truth = disk_truth(infos, agents_info, hooks_info)

    scanned = set()
    for pattern in DOC_SCAN_GLOBS:
        for path in pack_root.glob(pattern):
            if not path.is_file():
                continue
            rel = path.relative_to(pack_root).as_posix()
            if any(rel.startswith(p) for p in DOC_SCAN_EXCLUDE_PREFIXES):
                continue
            if path.name.startswith("skill-audit-"):  # generated reports
                continue
            scanned.add(path)

    for path in sorted(scanned):
        rel = path.relative_to(pack_root).as_posix()
        try:
            text = path.read_text(encoding="utf-8")
        except (OSError, UnicodeDecodeError):
            continue
        if DOC_IGNORE_MARKER in text:
            continue
        for claim_re, key in COUNT_CLAIM_PATTERNS:
            expected = truth[key]
            for m in claim_re.finditer(text):
                stated = int(m.group(1))
                if stated == expected:
                    continue
                context = text[max(0, m.start() - 130): m.end() + 130]
                if HISTORICAL_CONTEXT.search(context):
                    continue
                line_no = text[:m.start()].count("\n") + 1
                findings.append(Finding(
                    rel, "DOC_COUNT_DRIFT", CHECKS["DOC_COUNT_DRIFT"][0],
                    detail=f"line {line_no}: states \"{m.group(0).strip()}\", "
                           f"disk has {expected} {key}",
                ))

    # INDEX.md staleness — its row counts must match disk.
    index_path = pack_root / ".claude" / "INDEX.md"
    if index_path.exists():
        try:
            itext = index_path.read_text(encoding="utf-8")
        except (OSError, UnicodeDecodeError):
            itext = ""
        index_counts = {
            "skills": len(re.findall(r"\]\(skills/[^)]*/SKILL\.md\)", itext)),
            "pattern_cards": len(re.findall(r"\]\(skills/[^)]*/references/patterns/", itext)),
            "agents": len(re.findall(r"\]\(agents/", itext)),
            "hooks": len(re.findall(r"\]\(hooks/", itext)),
        }
        for key, found in index_counts.items():
            if found != truth[key]:
                findings.append(Finding(
                    ".claude/INDEX.md", "INDEX_STALE", CHECKS["INDEX_STALE"][0],
                    detail=f"index lists {found} {key}, disk has {truth[key]} — "
                           f"run: python3 validation/tools/skill_audit.py --emit-index",
                ))
    return findings


# ---- Report writer ----------------------------------------------------------

def write_report(findings: list, infos: list, agents_info: list, hooks_info: list,
                 report_path: Path, comm_skills: list = None) -> None:
    today = dt.date.today().isoformat()

    doc_checks = {"DOC_COUNT_DRIFT", "INDEX_STALE"}
    doc_findings = [f for f in findings if f.check in doc_checks]
    fails = [f for f in findings
             if CHECKS[f.check][0] == "FAIL" and f.check not in doc_checks]
    warns = [f for f in findings if CHECKS[f.check][0] == "WARN"]

    lines = []
    lines.append(f"# Skill audit — {today}")
    lines.append("")
    lines.append(f"- Skills scanned: **{len(infos)}**")
    lines.append(f"- Skill FAIL findings: **{len(fails)}**")
    lines.append(f"- WARN findings: **{len(warns)}**")
    lines.append(f"- Doc-consistency FAIL findings: **{len(doc_findings)}**")
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

    # DOCUMENTATION CONSISTENCY
    lines.append("## Documentation consistency")
    lines.append("")
    lines.append("Living documents must agree with disk on skill / agent / hook / "
                 "reference counts. Dated records under `validation/snapshots/` are exempt.")
    lines.append("")
    if not doc_findings:
        lines.append("_No drift — every living document agrees with the filesystem._")
    else:
        lines.append("| Document | Check | Detail |")
        lines.append("|---|---|---|")
        for f in doc_findings:
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

    lines.append("## Communication skills (instruction-os)")
    lines.append("")
    if comm_skills:
        lines.append("| Skill | Description chars | Over 1024? |")
        lines.append("|---|---|---|")
        for c in comm_skills:
            over = "yes" if c["desc_chars"] > 1024 else "no"
            lines.append(f"| `{c['name']}` | {c['desc_chars']} | {over} |")
    else:
        lines.append("_None found (../instruction-os/skills not present)._")
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


def collect_communication_skills(pack_root: Path) -> list:
    """Inventory the persona-derived communication skills under
    ../instruction-os/skills/. A different format from the engineering SKILL.md
    (no references/, no version), so they get a light inventory and a description-
    length note against the 1024-char hard limit, not the full engineering ruleset."""
    out = []
    comm_dir = pack_root.parent / "instruction-os" / "skills"
    if not comm_dir.exists():
        return out
    for skill_md in sorted(comm_dir.glob("*/SKILL.md")):
        try:
            fm = parse_frontmatter(skill_md.read_text(encoding="utf-8"))
        except (OSError, UnicodeDecodeError):
            continue
        out.append({
            "name": fm.get("name", skill_md.parent.name),
            "description": fm.get("description", ""),
            "desc_chars": len(fm.get("description", "")),
        })
    return out


def check_communication_consistency(pack_root: Path, comm_skills: list) -> list:
    """Targeted drift check: the workspace-root Ranking.md states a communication-
    skills count in its summary row and its section header. Flag if either
    disagrees with disk. Deliberately narrow - matches only the communication-
    skills claim, never the score or other-count numbers elsewhere in Ranking.md.
    This is the exact drift class that previously went undetected."""
    findings = []
    ranking = pack_root.parent / "Ranking.md"
    if not ranking.exists():
        return findings
    try:
        rtext = ranking.read_text(encoding="utf-8")
    except (OSError, UnicodeDecodeError):
        return findings
    truth = len(comm_skills)
    claims = re.findall(r"\|\s*Communication skills\s*\|\s*(\d+)\s*\|", rtext)
    claims += re.findall(r"##\s+Communication skills[^\(\n]*\((\d+)\)", rtext)
    for stated in claims:
        if int(stated) != truth:
            findings.append(Finding(
                "../Ranking.md", "DOC_COUNT_DRIFT", CHECKS["DOC_COUNT_DRIFT"][0],
                detail=f"states {stated} communication skills, disk has {truth}",
            ))
    return findings


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
    comm_skills = collect_communication_skills(pack_root)

    all_findings.extend(check_doc_consistency(pack_root, infos, agents_info, hooks_info))
    all_findings.extend(check_communication_consistency(pack_root, comm_skills))

    if args.report_path:
        report_path = Path(args.report_path)
    else:
        report_path = pack_root / "validation" / f"skill-audit-{dt.date.today().isoformat()}.md"

    write_report(all_findings, infos, agents_info, hooks_info, report_path, comm_skills)
    try:
        printable = report_path.relative_to(pack_root)
    except ValueError:
        printable = report_path
    print(f"Audit report written to: {printable}")

    fails = sum(1 for f in all_findings if CHECKS[f.check][0] == "FAIL")
    warns = sum(1 for f in all_findings if CHECKS[f.check][0] == "WARN")
    doc_drift = sum(1 for f in all_findings if f.check in ("DOC_COUNT_DRIFT", "INDEX_STALE"))
    print(f"Findings: {fails} FAIL ({doc_drift} doc-consistency), {warns} WARN, "
          f"{len(infos)} skills scanned.")

    if args.emit_index:
        write_index(infos, agents_info, hooks_info, pack_root)
        print(f"Index written to: .claude/INDEX.md")

    return 1 if fails > 0 else 0


if __name__ == "__main__":
    sys.exit(main())
