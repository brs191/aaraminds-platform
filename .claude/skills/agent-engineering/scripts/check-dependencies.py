#!/usr/bin/env python3
"""Report which delegated skills/agents this pack composes are present vs missing — pure stdlib.

Usage: check-dependencies.py <workspace_root>
Looks under <root>/.claude/skills and <root>/.claude/agents (and skills-pack/instruction-os).
Exit 0 always — this is informational (the pack degrades gracefully on missing deps).
"""
import sys
from pathlib import Path

SKILLS = ["aaraminds-ai-agent-blueprint-advisor", "ai-application-architecture",
          "prompt-engineering", "ai-evaluation-harness",
          "azure-microservices-security", "soc2-iso27001-controls-mapping"]
AGENTS = ["aara-prompt-engineer", "aara-ai-evaluation-engineer", "aara-project-planner"]

def exists(root: Path, kind: str, name: str) -> bool:
    cands = [root/".claude"/kind/name, root/".claude"/kind/f"{name}.md",
             root/"skills-pack"/".claude"/kind/name, root/"skills-pack"/".claude"/kind/f"{name}.md",
             root/"instruction-os"/"skills"/name]
    return any(p.exists() for p in cands)

def main() -> int:
    root = Path(sys.argv[1]) if len(sys.argv) > 1 else Path(".")
    print(f"Dependency check for: {root.resolve()}\n")
    miss = 0
    print("Skills (Evaluate mode's evaluator specialist is the load-bearing one):")
    for s in SKILLS:
        ok = exists(root, "skills", s); miss += 0 if ok else 1
        print(f"  [{'x' if ok else ' '}] {s}")
    print("Agents:")
    for a in AGENTS:
        ok = exists(root, "agents", a); miss += 0 if ok else 1
        print(f"  [{'x' if ok else ' '}] {a}")
    print(f"\n{('All composed dependencies present.' if miss==0 else str(miss)+' missing — pack still runs via inline core, but cannot claim run-tested behavior without aara-ai-evaluation-engineer.')}")
    return 0

if __name__ == "__main__":
    sys.exit(main())
