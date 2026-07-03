<!-- doc-consistency: ignore — frozen point-in-time snapshot, not maintained. See validation/snapshots/README.md -->

# Workspace session log — through 2026-05-21

Running log of work done in the workspace while building and wiring the pack. Rescued
from the former root-level `overview.md` when that file was retired as a duplicate of
`README.md`. Frozen as of 2026-05-21 — counts and paths reflect the pack at that date.

## Work done so far in this workspace

1. **Identified the original folder as a Copilot adapter only** — no `.claude/` directory was present; the three Copilot `.agent.md` files referenced a `~/aaraminds-pack/.claude/skills/` knowledge base that didn't exist locally.
2. **Located the full Claude Code pack** in a separate Linux checkout (since archived and superseded by this canonical copy).
3. **Verified no destructive conflict** between source and destination — every shared file was byte-identical; two local-only files would be preserved.
4. **Copied the full pack** with `cp -a`, making it self-contained. Preserved local files `usage.md` and `validation/SoftwareDevAgent_TestPlan_2026-05-21.md`. Skipped the 14 MB `.zip` archive as redundant.
5. **Confirmed a Linux x86-64 `mcp-server` binary is present** — no Go rebuild needed to wire up MCP on this Linux box.
6. **Produced `ranking.md`** — every skill, agent, hook, and MCP tool scored 1–10 for Claude Code vs Codex/Copilot. Ratings grounded against Microsoft's Azure Skills Plugin, Copilot's layered Instructions/Skills/Agents/Hooks architecture, the official MCP Go SDK, and the 1000+ entry awesome-agent-skills catalog.
7. **Wired a staleness hook** in user-level `~/.claude/settings.json` that prints a reminder whenever Claude Code edits a file under `.claude/skills/`, `agents/`, `hooks/`, or the MCP server tool registrations.
8. **Expanded reference depth on the two thinnest skills** — v3 added 4 references to `azure-microservices-cost-review` (1 ref / 346 lines → 5 refs / 1,119 lines); v4 added 4 references to `azure-microservices-observability` (1 ref / 248 lines → 5 refs / 1,173 lines). Both depth scores moved 5 → 8 in `ranking.md`.
9. **Wired MCP server into Claude Code** via project-root `.mcp.json` pointing at the Linux x86-64 binary. Claude Code registered all 13 tools as `mcp__aaraminds-microservices__*` deferred tools.
10. **Installed Go 1.25.5 to `~/.local/go/`** (user-scope, no sudo, reversible with `rm -rf ~/.local/go`) since the system Go 1.13/1.14 cannot parse the pack's `go 1.25.5` directive.
11. **Rebuilt the MCP server binary** to fix the 3 formerly-stub design tools (`recommend_microservice_patterns`, `review_microservice_design`, `score_well_architected_readiness`). The underlying `internal/services/design/service.go` was already 1,253 lines of real implementation; the defect was broken wiring between `register.go`/`service_test.go` and the new service API. Re-wired both files, rebuilt binary, `go test ./...` green across 11 service packages, direct stdio probes confirm all 3 tools are now input-aware.
12. **Installed `jq` 1.7.1 static binary** at `~/.local/bin/jq` (no sudo). Re-pipe-tested all 3 pack hooks; `block-dangerous-commands` now correctly blocks `rm -rf /`, force-push to protected branches, `DROP DATABASE`, `curl ... | bash`, etc. Safe commands still pass.
13. **Reverted `register.go` to expose typed args** (`problem`, `system_name`, optional `business_capability`/`deployment_target`/`services`) matching Claude Code's cached MCP schema. The handler synthesizes the rich JSON the service layer expects from the typed args. Verified all 3 design tools work end-to-end through Claude Code's `mcp__aaraminds-microservices__*` interface with no MCP restart needed.

## Current state — 2026-05-21

**Done:**

14. **Wired skills into workspace** — copied `skills-pack/.claude/skills/` into `aaramind/.claude/skills/` so all Tier-1 skills are auto-discoverable. On Windows/OneDrive symlinks are not viable; this is a managed copy. Re-sync after skill edits with `cp -r skills-pack/.claude/skills/ .claude/skills/` (or `rsync -a` on Linux).
15. **Compiled Windows-native MCP server binary** — cross-compiled `mcp-server.exe` (GOOS=windows GOARCH=amd64) and updated `.mcp.json` to point at the Windows-native path. The 13 MCP tools are now wired for Claude Code on Windows.

**Accepted debt — explicit decisions recorded:**

- **Beef up `microservices-api-design` and `microservices-resilience` depth** — Accepted deferred. The v3/v4 depth-expansion pattern applies. Add during next quarterly refresh. Tracked in `.claude/FEEDBACK.md`.
- **Re-test skill/agent strength** — Will happen organically now that skills are wired into the workspace.
- **Hook installation scope** — Hooks ship as JSON definitions in `.claude/hooks/`. To activate, reference each hook's `type`, `command`, and `matcher` fields from a workspace `settings.json` `hooks` array. Do not add hooks to the aaramind workspace `settings.json` by default — `block-dangerous-commands` could disrupt tool-call workflows that legitimately use those patterns.
- **`review_microservice_design` Container Apps false positive** — Logged in `.claude/FEEDBACK.md`. The pattern matcher in `internal/services/design/service.go` needs Azure-native vocabulary widened to include `Container Apps`, `ACA`, `Azure Container Apps`.
- **Delete the redundant archived Linux source folder** — the old pre-canonical Linux checkout is stale and superseded by the canonical copy; delete it from the Linux machine when next on it.
- **Regenerate `.claude/INDEX.md`** — Run `python3 validation/tools/skill_audit.py --emit-index` from the pack root after any skill edits.
