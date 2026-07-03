# Hooks

This directory contains three Claude Code hook configurations. Hooks are event-driven shell commands that fire on specific Claude Code lifecycle events (tool use, message submission, stop). Each hook here is a stand-alone JSON file that maps directly to a `settings.json` `hooks` block.

## The three hooks

| Hook | Event | Purpose |
|---|---|---|
| [`pre-commit-lint.json`](pre-commit-lint.json) | `PreToolUse` on `Bash` matching `git commit` | Runs `gofmt -l` / `go vet` (Go) or `mvn spotless:check` (Java) before any commit. Blocks the commit if lint fails. |
| [`test-before-commit.json`](test-before-commit.json) | `PreToolUse` on `Bash` matching `git commit` | Runs `go test -race -count=1 ./...` (Go) or `mvn test` (Java) before any commit. Blocks if tests fail. Can be skipped with `TEST_BEFORE_COMMIT_SKIP=1` for genuine docs-only commits. |
| [`block-dangerous-commands.json`](block-dangerous-commands.json) | `PreToolUse` on `Bash` | Inspects every Bash command Claude is about to run; blocks a denylist of dangerous patterns: `rm -rf /` / `~`, force-push to protected branches, `DROP DATABASE`, `az group delete` on `*prod*` resource groups, `kubectl delete` against prod, `docker system prune -a --volumes`, fork bombs, `curl ... | bash`. |

## How to install

Hooks live in Claude Code's `settings.json`, not in this directory's files directly. The JSON files here are *templates* you merge into your settings.

### Project-level install (recommended for this pack)

Create or open `.claude/settings.json` in your project. Merge the `hooks.PreToolUse` arrays from each template into it:

```json
{
  "hooks": {
    "PreToolUse": [
      <entries from pre-commit-lint.json>,
      <entries from test-before-commit.json>,
      <entries from block-dangerous-commands.json>
    ]
  }
}
```

When merging, preserve each hook's `matcher` and `hooks` array.

### User-level install (cross-project)

If you want the hooks active across every project (especially `block-dangerous-commands`, which is universally useful), merge them into `~/.claude/settings.json` instead. Project-level settings can override user-level.

### Quick merge command

A jq one-liner to merge `block-dangerous-commands.json` into user-level settings:

```bash
jq -s '.[0].hooks.PreToolUse += .[1].hooks.PreToolUse | .[0]' \
   ~/.claude/settings.json \
   .claude/hooks/block-dangerous-commands.json \
   > ~/.claude/settings.json.new && \
   mv ~/.claude/settings.json.new ~/.claude/settings.json
```

Repeat for the other two. Inspect the merged settings before saving over your existing config.

## How the hooks behave

### Exit codes

A hook's shell command exits:

- **`0`** — proceed; tool call goes through.
- **`2`** — block; Claude sees the failure and does not run the tool. The hook's `>&2` output is surfaced to Claude as feedback.
- **Other non-zero** — depends on Claude Code's hook semantics; treat as proceed-with-warning.

The hooks here use `2` to mean "block this commit / command." That gives Claude actionable feedback rather than crashing the session.

### Input

Each hook receives `$CLAUDE_TOOL_INPUT` as an environment variable containing the full tool invocation as JSON. The hooks parse it with `python3` (universally available; no extra dependency to install):

```bash
cmd=$(printf '%s' "$CLAUDE_TOOL_INPUT" | python3 -c 'import sys,json;d=json.load(sys.stdin);print(d.get("command",""))')
```

### Fail-closed behavior (2026-05-27)

The hooks **fail closed**: if `python3` is missing, if `CLAUDE_TOOL_INPUT` cannot be parsed, or if the parsed `.command` is empty, the hook exits `2` (block) with a diagnostic on stderr. A security hook that fails silently is worse than no hook — earlier versions used `jq` and fell through cleanly when `jq` was absent, which let dangerous commands through unchecked. That bug is fixed.

`test-before-commit` still honors `TEST_BEFORE_COMMIT_SKIP=1` as the *first* check, so an intentional bypass never depends on `python3` being available.

## Trade-offs and limitations

### What hooks can't catch

Hooks operate on the *invocation* (the Bash command string). They don't run the command in a sandbox, so:

- A command can be obfuscated to evade the denylist (`r''m -rf /` or aliasing). The denylist is for *honest mistakes*, not adversarial intent.
- A safe-looking command can have destructive side effects deep in a script. The hook doesn't see what the script does.
- Commands launched by other tools (Edit writing a script then Bash running it) are detectable only at the Bash invocation step.

The hooks are guardrails, not security boundaries. Treat them as cheap defense-in-depth against accidents.

### Performance

- `pre-commit-lint` and `test-before-commit` add latency to commits. Lint is usually under 5 seconds; tests can be much longer.
- `block-dangerous-commands` is a single shell parse — sub-millisecond.

If `test-before-commit` becomes too slow for your iteration cadence, set `TEST_BEFORE_COMMIT_SKIP=1` for the rapid-iteration period and rely on the PR-level CI gate as the actual quality check. The hook is a fast-feedback nudge, not the system of record.

### False positives

`block-dangerous-commands` will occasionally flag a legitimate command. When that happens:

1. Inspect the matched pattern in the hook source.
2. If the command is genuinely safe, run it outside Claude Code (the hook only fires on tool calls Claude makes), OR
3. Adjust the denylist patterns if a class of safe commands keeps matching. Err on the side of keeping the deny strict; loosen only with clear rationale.

## Composing with the pack's agents

The hooks compose with the agents:

- `aara-senior-microservices-architect` and `aara-mcp-server-builder` will make Bash calls during design / implementation work. `block-dangerous-commands` keeps the destructive subset gated.
- After code changes, when Claude runs `git commit`, `pre-commit-lint` and `test-before-commit` enforce the bar before the change lands.

The agents don't need to know about the hooks; the hooks fire automatically at the tool-call layer.

## When to disable

Disable a hook (remove from `settings.json`) only when:

- You're running a one-off bulk operation where the hook causes excessive friction (e.g., committing 50 doc-only files where `test-before-commit` is wasted work — but `TEST_BEFORE_COMMIT_SKIP=1` is the better answer).
- The hook produces a false positive often enough that it's training you to ignore it. Fix the pattern instead.

Do not disable `block-dangerous-commands`. It's protecting you from a class of mistakes that happen rarely but expensively. The 1-millisecond cost is negligible.

## Adding a new hook

Future hooks should follow the same shape:

1. One JSON file per hook in this directory
2. Top-level `$schema` and `comment` fields for self-documentation
3. `hooks.<EventName>` array matching Claude Code's hook spec
4. Inline `comment` in each hook command explaining what it does
5. Conservative deny patterns; explicit skip-mechanism (env var, file marker) for emergency bypass

Events available include `PreToolUse`, `PostToolUse`, `UserPromptSubmit`, `Stop`, and others — check Claude Code's hook documentation for the current set. Pick the event that fires closest to where you want to enforce.
