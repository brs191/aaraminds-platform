# Using the AaraMinds Skills Pack

The canonical copy of this pack is **this folder** — `/home/raja/projects/aaraminds-platform/skills-pack`. It is the single source of truth; all edits go here. The shell snippets below refer to it as `$PACK_ROOT`; set that once before running them:

```bash
PACK_ROOT="<path to this pack folder>"   # the directory containing this file
```

The pack is not globally discoverable on its own — Claude Code looks for `.claude/` at the workspace root, so you wire the pack into each repo where you want skills, agents, and (optionally) hooks available.

This file is the canonical reference for setup.

---

## Quick setup — repo with no existing `.claude/`

From inside the target repo:

```bash
ln -s "$PACK_ROOT/.claude" .claude
( grep -qxF ".claude" .gitignore 2>/dev/null || echo ".claude" >> .gitignore )
```

That's it. The pack is now active in this workspace.

## Setup — repo that already has its own `.claude/`

This is the common case (e.g., a repo that already has its own `.claude/settings.json`). Symlink the pack's content subdirectories alongside the existing config:

```bash
mkdir -p .claude
PACK="$PACK_ROOT/.claude"
ln -s "$PACK/skills"      .claude/skills
ln -s "$PACK/agents"      .claude/agents
ln -s "$PACK/CLAUDE.md"   .claude/CLAUDE.md
ln -s "$PACK/INDEX.md"    .claude/INDEX.md
ln -s "$PACK/FEEDBACK.md" .claude/FEEDBACK.md
# Symlink hooks/ only if you want the JSON templates available for reference.
# They do NOT auto-fire — see "Activating hooks" below.
ln -s "$PACK/hooks"       .claude/hooks

# If repo is git-tracked:
for f in skills agents hooks CLAUDE.md INDEX.md FEEDBACK.md; do
  grep -qxF "/.claude/$f" .gitignore 2>/dev/null || echo "/.claude/$f" >> .gitignore
done
```

The repo's own `.claude/settings.json` is untouched.

## What the pack provides

- **35 Tier-1 skills** under `.claude/skills/` — auto-discovered by Claude Code when the workspace has the symlinks in place
- **17 agents** under `.claude/agents/`:
  - `aara-senior-microservices-architect` (opus) — Azure microservices design & review
  - `aara-mcp-server-builder` (inherit) — Go MCP servers
  - `aara-azure-cost-reviewer` (sonnet) — FinOps / cost review
  - `aara-network-topology-reviewer` (inherit) — Azure network topology / reachability review
- **3 hook templates** under `.claude/hooks/` — JSON templates, **not auto-firing**

## Activating hooks

Hooks are templates. Symlinking `hooks/` makes the JSON files visible; it does not make them fire. To enable, merge the JSON content into a `settings.json`:

- **Per-workspace**: edit `<repo>/.claude/settings.json` (overrides global for that workspace)
- **Globally**: edit `~/.claude/settings.json` (fires in every Claude Code session)

The three hooks:

| File | What it does |
|---|---|
| `pre-commit-lint.json` | Runs `gofmt`/`vet` (Go) or `spotless` (Java) before `git commit`; blocks on failure |
| `test-before-commit.json` | Runs `-race` tests before commit; bypass with `TEST_BEFORE_COMMIT_SKIP=1` |
| `block-dangerous-commands.json` | Blocks `rm -rf /`, force-push to protected branches, prod kubectl delete, `curl ... \| bash`, fork bombs |

See `$PACK_ROOT/.claude/hooks/README.md` for the exact merge syntax.

## Verifying the pack is active

Open the symlinked repo in VS Code, start a Claude Code session, and ask:

> *"What skills do you have available?"*

You should see all 32 skills listed. Triggering examples:

- *"Design a new microservices system for an order management platform"* → `aara-senior-microservices-architect`
- *"Add a tool to my MCP server that generates ADRs"* → `aara-mcp-server-builder`
- *"Why did our Azure bill spike 35%?"* → `aara-azure-cost-reviewer`

You can also verify on the filesystem:

```bash
ls .claude/skills          # should show 35 skill directories
ls .claude/agents          # should show the 4 aara- agents
readlink .claude/skills    # should point to $PACK_ROOT/.claude/skills
```

## Removing from a repo

To deactivate the pack in a repo, remove the symlinks:

```bash
# If you used the quick-setup (whole .claude symlink):
rm .claude

# If you used per-subdirectory symlinks:
rm -f .claude/skills .claude/agents .claude/hooks .claude/CLAUDE.md .claude/INDEX.md .claude/FEEDBACK.md
```

The canonical pack at `$PACK_ROOT` is untouched. Your repo's own `.claude/settings.json` (if any) stays.

## Maintenance

**Canonical source of truth:** this folder (`$PACK_ROOT`). It is the only live copy — all edits, new skills, agent updates, hook changes, and FEEDBACK entries go here. If you find another copy on disk (an older checkout elsewhere, a `~/.claude/packs/` install), diff it against this folder and discard the divergent one; do not maintain two.

**After any structural change** (added/removed/renamed skill, agent, hook), regenerate the discovery index:

```bash
cd "$PACK_ROOT"
python3 validation/tools/skill_audit.py --emit-index
```

**Run the audit periodically** to catch description-length warnings, missing frontmatter, orphan references:

```bash
cd "$PACK_ROOT"
python3 validation/tools/skill_audit.py
```

**Feedback log:** `$PACK_ROOT/.claude/FEEDBACK.md` is the pack-wide observation log. Per the session protocol in `CLAUDE.md`, Claude reads it at the start of pack-related sessions. Append entries whenever a skill, agent, or hook gives bad guidance or a non-obvious workaround proves useful.

## Currently symlinked into

| Workspace | Activated on |
|---|---|
| _(none recorded yet)_ | — |

Update this list when you symlink the pack into a repo, so you have a single inventory of where it's active.
