#!/usr/bin/env bash
#
# setup-mac.sh — set up AaraMinds for VS Code + GitHub Copilot on an Apple Silicon Mac.
#
# This is the repeatable step. Re-run it any time:
#   - after pulling repo changes
#   - after editing the MCP server source
#   - on a new machine
# It is idempotent and backs up anything it overwrites.
#
# What it does:
#   1. Verifies Go >= 1.25 (the MCP server requires it).
#   2. Rebuilds the MCP server binary from source — native arm64, no committed binary trusted.
#   3. Smoke-tests the binary over stdio.
#   4. Registers the server in your VS Code USER config so the 13 tools work in EVERY repo.
#   5. Installs the 4 canonical agents into ~/.copilot/agents/ so they're available in EVERY repo.
#
# The committed .vscode/mcp.json + .vscode/settings.json already make everything work when you
# open the AaraMinds repo itself. This script adds the USER-level install for cross-repo use —
# i.e. reviewing your actual customer/work repos, which is the whole point.

set -euo pipefail

# ---- Resolve locations -----------------------------------------------------
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"   # skills-pack/copilot
PACK_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"                     # skills-pack
SERVER_DIR="$PACK_ROOT/examples/microservices-system-design-mcp-server"
AGENTS_SRC="$PACK_ROOT/.claude/agents"
BINARY="$SERVER_DIR/mcp-server"

VSCODE_USER="$HOME/Library/Application Support/Code/User"
MCP_USER_CONFIG="$VSCODE_USER/mcp.json"
AGENTS_DST="$HOME/.copilot/agents"

say()  { printf '\033[36m==>\033[0m %s\n' "$*"; }
warn() { printf '\033[33mWARN\033[0m %s\n' "$*" >&2; }
die()  { printf '\033[31mERROR\033[0m %s\n' "$*" >&2; exit 1; }

say "Pack root: $PACK_ROOT"

# ---- 1. Verify Go ----------------------------------------------------------
command -v go >/dev/null 2>&1 || die "Go not found. Install it:  brew install go"
GO_VER="$(go version | awk '{print $3}' | sed 's/^go//')"
GO_MAJOR="$(echo "$GO_VER" | cut -d. -f1)"
GO_MINOR="$(echo "$GO_VER" | cut -d. -f2)"
if [ "${GO_MAJOR:-0}" -lt 1 ] || { [ "${GO_MAJOR:-0}" -eq 1 ] && [ "${GO_MINOR:-0}" -lt 25 ]; }; then
  die "Go $GO_VER is too old; the server needs >= 1.25. Upgrade:  brew upgrade go"
fi
say "Go $GO_VER OK"

# ---- 2. Build the binary (native arm64 on Apple Silicon) -------------------
say "Building MCP server from source (native arch)..."
( cd "$SERVER_DIR" && go build -o mcp-server ./cmd/server )
[ -x "$BINARY" ] || die "Build did not produce an executable at $BINARY"
say "Built: $BINARY"
file -b "$BINARY" 2>/dev/null | sed 's/^/    /' || true

# ---- 3. Smoke-test over stdio ---------------------------------------------
say "Smoke-testing the binary over stdio..."
SMOKE_ERR="$(mktemp)"; REQ_FILE="$(mktemp)"
printf '%s\n' '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"setup","version":"0"}}}' > "$REQ_FILE"
"$BINARY" < "$REQ_FILE" > /dev/null 2> "$SMOKE_ERR" &
SPID=$!
sleep 2
kill "$SPID" 2>/dev/null || true
wait "$SPID" 2>/dev/null || true
if grep -q 'starting MCP server' "$SMOKE_ERR"; then
  say "Server starts cleanly under stdio transport"
else
  warn "Expected startup log not seen. First lines of stderr:"; head -3 "$SMOKE_ERR" >&2
  warn "Continuing — verify manually in VS Code."
fi
rm -f "$SMOKE_ERR" "$REQ_FILE"

# ---- 4. Register the server in VS Code USER config (cross-repo) ------------
mkdir -p "$VSCODE_USER"
say "Registering MCP server in user config: $MCP_USER_CONFIG"
python3 - "$MCP_USER_CONFIG" "$BINARY" <<'PY'
import json, os, shutil, sys, time
cfg_path, binary = sys.argv[1], sys.argv[2]
cfg = {}
if os.path.exists(cfg_path):
    bak = cfg_path + ".backup-" + time.strftime("%Y%m%d-%H%M%S")
    shutil.copy2(cfg_path, bak)
    print("    backed up existing mcp.json -> " + os.path.basename(bak))
    try:
        with open(cfg_path) as f:
            cfg = json.load(f)
    except Exception:
        print("    (existing mcp.json was unparseable; starting fresh)")
        cfg = {}
cfg.setdefault("servers", {})
cfg["servers"]["aaraminds-microservices"] = {
    "type": "stdio", "command": binary, "args": [], "env": {}
}
with open(cfg_path, "w") as f:
    json.dump(cfg, f, indent=2)
print("    registered 'aaraminds-microservices' (13 tools)")
PY

# ---- 5. Install the canonical agents (cross-repo) --------------------------
mkdir -p "$AGENTS_DST"
say "Installing agents into: $AGENTS_DST"
shopt -s nullglob
count=0
for f in "$AGENTS_SRC"/aara-*.md; do
  name="$(basename "$f")"
  dst="$AGENTS_DST/$name"
  [ -f "$dst" ] && cp "$dst" "$dst.backup-$(date +%Y%m%d-%H%M%S)"
  cp "$f" "$dst"
  echo "    installed $name"
  count=$((count + 1))
done
[ "$count" -gt 0 ] || warn "No agents found in $AGENTS_SRC — is the repo intact?"
say "$count agent(s) installed"

# ---- 6. Verification steps -------------------------------------------------
cat <<EOF

================================================================================
Done. Reload VS Code (Cmd+Shift+P -> "Developer: Reload Window"), then verify:

1. Cmd+Shift+P -> "MCP: List Servers"
   Expect:  aaraminds-microservices  (running)

2. Copilot Chat -> "Configure Tools" -> expect ~13 tools under aaraminds-microservices
   (review_microservice_design, detect_architecture_risks, generate_*, ...)

3. Chat -> agents dropdown, or type /agents -> expect:
     aara-senior-microservices-architect
     aara-mcp-server-builder
     aara-azure-cost-reviewer
     aara-network-topology-reviewer

4. Select @aara-senior-microservices-architect and ask a design question;
   confirm the verdict-first, brownfield, stack-pinned voice.

These now work in EVERY repo you open (user-level install), not just this one.
Re-run this script after pulling changes, editing the server, or on a new machine.

Full guide / troubleshooting:  $SCRIPT_DIR/README.md
================================================================================
EOF
