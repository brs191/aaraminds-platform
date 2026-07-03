#!/usr/bin/env python3
"""Validate demo outputs against golden fixtures.

Each architecture under --out/<arch>/ must match golden/<arch>/ byte-for-byte
after JSON normalisation. Failures are reported per file with a short reason.

The rule-based Go MCP server is deterministic, so any mismatch is meaningful:
either an input drifted, a tool's logic changed, or a golden needs to be
refreshed because the change was intentional.
"""

from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path

# Tool short keys this validator expects per architecture. Keep aligned with
# TOOLS in demo_runner.py — anything missing here would silently pass.
EXPECTED_TOOLS = ["boundary", "apicontract", "archrisks", "azuremap", "obsplan"]


def canonicalise_json(path: Path) -> str:
    """Normalise JSON to a canonical string for comparison.

    Sorting keys removes ordering differences that don't affect meaning.
    """
    return json.dumps(json.loads(path.read_text()), indent=2, sort_keys=True) + "\n"


def compare_arch(arch: str, out_dir: Path, golden_dir: Path) -> list[str]:
    errors: list[str] = []
    if not out_dir.is_dir():
        return [f"[{arch}] missing generated directory: {out_dir}"]
    if not golden_dir.is_dir():
        return [f"[{arch}] missing golden directory: {golden_dir}"]
    for tool in EXPECTED_TOOLS:
        generated = out_dir / f"{tool}.json"
        golden = golden_dir / f"{tool}.json"
        if not generated.exists():
            errors.append(f"[{arch}] missing generated output: {generated.name}")
            continue
        if not golden.exists():
            errors.append(f"[{arch}] missing golden: {golden.name}")
            continue
        try:
            gen_canon = canonicalise_json(generated)
            gold_canon = canonicalise_json(golden)
        except json.JSONDecodeError as e:
            errors.append(f"[{arch}] invalid JSON in {generated.name} or golden: {e}")
            continue
        if gen_canon != gold_canon:
            errors.append(f"[{arch}] mismatch in {tool}.json")
    return errors


def main() -> int:
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("--out", default="out", help="Directory containing generated outputs")
    ap.add_argument("--golden", default="golden", help="Directory containing golden fixtures")
    args = ap.parse_args()

    out_root = Path(args.out)
    golden_root = Path(args.golden)

    if not golden_root.is_dir():
        print(f"error: golden directory not found: {golden_root}", file=sys.stderr)
        return 2

    architectures = sorted(p.name for p in golden_root.iterdir() if p.is_dir())
    if not architectures:
        print(f"error: no architectures under {golden_root}", file=sys.stderr)
        return 2

    errors: list[str] = []
    for arch in architectures:
        errors.extend(compare_arch(arch, out_root / arch, golden_root / arch))

    if errors:
        print("Validation failed:")
        for e in errors:
            print(f"  - {e}")
        return 1
    print(
        f"Validation passed: {len(architectures)} architecture(s) "
        f"× {len(EXPECTED_TOOLS)} tool(s) all match golden fixtures."
    )
    return 0


if __name__ == "__main__":
    sys.exit(main())
