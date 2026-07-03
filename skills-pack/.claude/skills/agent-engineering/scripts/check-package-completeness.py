#!/usr/bin/env python3
"""Check a GENERATED agent package directory for the mandatory artifacts — pure stdlib.

Usage: check-package-completeness.py [--generated-package] <package_dir>

Excludes the engineering pack's own folders (templates/ references/ schemas/ scripts/ eval/) so the
skill pack itself is never mistaken for a generated agent package, and warns if a match came only from a
*-template.* file. Exit 0 if complete, 1 if any required artifact is missing.
"""
from __future__ import annotations
import sys, glob, os
from pathlib import Path

EXCLUDE_DIRS = {"templates", "references", "schemas", "scripts", "eval", "examples", ".github", ".git"}

# (label, list-of-acceptable-glob-patterns) — prefer specific names; templates are filtered out below.
REQUIRED = [
    ("runnable agent file", ["*.agent.md", "agent.md", "*.toml", "agents/*.md"]),
    ("AGENT_SPEC.md",       ["AGENT_SPEC.md", "agent-spec.md"]),
    ("agent-card.json",     ["agent-card.json"]),
    ("eval plan",           ["eval-plan.md", "evals.md"]),
    ("review scorecard",    ["review-scorecard.md"]),
    ("release gate",        ["release-gate.json", "release-gate.md"]),
]
RECOMMENDED = [
    ("tool-risk register",  ["tool-risk-register.md"]),
    ("AGENTS.md",           ["AGENTS.md"]),
    ("improvement backlog", ["improvement-backlog.md"]),
]

def _matches(base: Path, pats):
    hits = []
    for p in pats:
        for m in glob.glob(str(base / "**" / p), recursive=True):
            rel = os.path.relpath(m, base)
            parts = set(Path(rel).parts[:-1])
            if parts & EXCLUDE_DIRS:        # ignore anything under the pack's own folders
                continue
            hits.append(rel)
    return hits

def main() -> int:
    args = [a for a in sys.argv[1:] if a != "--generated-package"]
    if len(args) != 1:
        print(__doc__); return 2
    base = Path(args[0])
    if not base.is_dir():
        print(f"FAIL not a directory: {base}"); return 1
    ok = True
    print(f"Generated package: {base}")
    for label, pats in REQUIRED:
        hits = _matches(base, pats)
        only_templates = hits and all("-template." in h or h.endswith("-template.md") for h in hits)
        present = bool(hits) and not only_templates
        mark = "x" if present else ("~" if only_templates else " ")
        note = "  (only a *-template.* matched — not a populated artifact)" if only_templates else ""
        print(f"  [{mark}] required: {label}{note}")
        ok = ok and present
    for label, pats in RECOMMENDED:
        print(f"  [{'x' if _matches(base, pats) else ' '}] recommended: {label}")
    print("COMPLETE" if ok else "INCOMPLETE — missing required artifact(s)")
    return 0 if ok else 1

if __name__ == "__main__":
    sys.exit(main())
