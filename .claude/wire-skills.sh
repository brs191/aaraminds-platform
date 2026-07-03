#!/usr/bin/env bash
# Wire AaraMinds skills AND agents into .claude/ so Claude Code can discover them.
#
# Claude Code scans <workspace>/.claude/skills/<name>/SKILL.md and
# <workspace>/.claude/agents/<name>.md. The AaraMinds artifacts live in canonical
# homes outside those paths:
#   - skills-pack/.claude/skills/   (engineering skills)
#   - instruction-os/skills/        (communication skills)
#   - skills-pack/.claude/agents/   (subagent personas; README.md is skipped)
# This script symlinks each one into .claude/skills/ and .claude/agents/. Sources
# are enumerated from disk, so new skills/agents are picked up automatically on
# re-run — no counts are hard-coded here.
#
# Links are RELATIVE (e.g. ../../skills-pack/.claude/skills/<name>), so the wiring
# survives the workspace being moved, renamed, or cloned to a different path. The
# prior version wrote absolute links, which broke on every relocation. A relative
# link is portable and needs no `ln -r` (works under both GNU and BSD ln) because
# every destination sits exactly two levels below the repo root.
#
# Use this variant on WSL / Linux / macOS. On native Windows use wire-skills.ps1.
#
# Usage:
#   bash .claude/wire-skills.sh            # wire (or refresh) all skills + agents
#   bash .claude/wire-skills.sh --unwire   # remove all wired symlinks
#
# Re-runnable. The symlinks are local wiring and are gitignored — see .gitignore
# (/.claude/skills/ and /.claude/agents/). This script itself is tracked.
set -euo pipefail

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"   # .claude
repo="$(cd "$here/.." && pwd)"                          # workspace root

# Each spec: "<dest-subdir>|<root-relative-source-dir>|<kind>"
#   kind=dir -> link each child directory (skills)
#   kind=md  -> link each *.md file except README.md (agents)
specs=(
    "skills|skills-pack/.claude/skills|dir"
    "skills|instruction-os/skills|dir"
    "agents|skills-pack/.claude/agents|md"
)

if [[ "${1:-}" == "--unwire" ]]; then
    removed=0
    for dest in skills agents; do
        d="$here/$dest"
        [[ -d "$d" ]] || continue
        while IFS= read -r -d '' l; do
            rm -f "$l"; echo "unlinked  $dest/$(basename "$l")"; ((removed++)) || true
        done < <(find "$d" -maxdepth 1 -type l -print0)
    done
    echo ""
    echo "Unwired. Removed $removed symlink(s) from .claude/skills/ and .claude/agents/."
    exit 0
fi

# link_one <destdir> <name> <relative-target> <dest-label> -> 0 linked, 2 skipped
link_one() {
    local link="$1/$2"
    if [[ -L "$link" ]]; then
        rm -f "$link"                                   # refresh existing symlink
    elif [[ -e "$link" ]]; then
        echo "WARN  skip (real path, not a symlink): $4/$2" >&2; return 2
    fi
    ln -s "$3" "$link"
    echo "linked    $4/$2"
    return 0
}

linked=0; skipped=0
for spec in "${specs[@]}"; do
    IFS='|' read -r dest srcrel kind <<< "$spec"
    src="$repo/$srcrel"
    destdir="$here/$dest"
    if [[ ! -d "$src" ]]; then echo "WARN  source missing: $srcrel" >&2; continue; fi
    mkdir -p "$destdir"
    if [[ "$kind" == "dir" ]]; then
        for d in "$src"/*/; do
            [[ -d "$d" ]] || continue
            name="$(basename "$d")"
            if link_one "$destdir" "$name" "../../$srcrel/$name" "$dest"; then ((linked++)) || true; else ((skipped++)) || true; fi
        done
    else
        for f in "$src"/*.md; do
            [[ -e "$f" ]] || continue
            name="$(basename "$f")"
            [[ "$name" == "README.md" ]] && continue
            if link_one "$destdir" "$name" "../../$srcrel/$name" "$dest"; then ((linked++)) || true; else ((skipped++)) || true; fi
        done
    fi
done
echo ""
echo "Done. Linked $linked artifact(s) into .claude/skills/ and .claude/agents/ ($skipped skipped)."
echo "Restart Claude Code, then run /skills to confirm skill discovery; agents load from .claude/agents/."
