# GitHub Copilot — prompt patterns

Platform track for prompts and customization targeting GitHub Copilot (VS Code, JetBrains, GitHub.com cloud agent, Copilot CLI).

> Terminology/path note (2026 reorg): GitHub moved prompt-engineering docs under `/copilot/concepts/prompting/`; VS Code moved customization under `/docs/agent-customization/` and **renamed "custom chat modes" to "custom agents"** (`.chatmode.md` → `.agent.md`). Old `using-github-copilot/...` URLs redirect to `/copilot/concepts/...`.

## Prompt-engineering guidance (GitHub official)

- **Start general, then get specific.** Open with the broad goal, then list each specific requirement on its own line. ("Write a JS function that tells if a number is prime" → "takes an integer, returns true if prime" → "errors on non-positive input.")
- **Give examples** — example inputs, outputs, and implementations. Unit tests double as examples; have Copilot write tests first, then the function those tests describe.
- **Break complex tasks into simpler ones.** Decompose and request the pieces sequentially.
- **Avoid ambiguity.** Name the symbol (`What does the createUser function do?`), not "this." For uncommon libraries, describe the library or set the import at the top of the file to force it.
- **Indicate relevant code.** Open relevant files, close irrelevant ones (open files are context), highlight the code to reference, use `@workspace` (VS Code) / `@project` (JetBrains).
- **Be specific about inputs, outputs, constraints**; include expected output / test cases for verification — one of the highest-leverage moves.
- **Iterate, don't rewrite** — refine with follow-ups; course-correct a running request early. Tell Copilot to **ask clarifying questions** when ambiguous.
- **Keep history relevant** — one thread per task; delete stale turns (history is context). Run independent tasks in parallel.
- **Copilot mirrors your codebase quality** — consistent style, descriptive names, tests improve its output.

## Custom instructions — three types

1. **Repository-wide:** a single **`.github/copilot-instructions.md`** (Markdown, natural language). Applies to all requests in repo context. Whitespace between rules is ignored.
2. **Path-specific:** one or more **`*.instructions.md`** files under **`.github/instructions/`**. Filename must end `.instructions.md`. YAML frontmatter with **`applyTo`** (glob): `applyTo: "**/*.ts,**/*.tsx"`. Glob semantics: `*` = current dir, `**`/`**/*` = recursive, `src/*.py` = direct children, `src/**/*.py` = recursive. Optional `excludeAgent: "code-review"` / `"cloud-agent"`. (On GitHub.com, path-specific instructions currently apply only to cloud agent and code review.)
3. **Agent instructions:** **`AGENTS.md`** anywhere (nearest in the tree wins), or a single root `CLAUDE.md` / `GEMINI.md`.

**Precedence:** Personal (highest) → Repository → Organization (lowest); all relevant sets are still supplied — avoid conflicts. Code review uses the PR **base branch** instructions.

**Authoring a good `copilot-instructions.md`** (from GitHub's onboarding prompt): keep it **≤ 2 pages**; **not task-specific**; document the exact bootstrap/build/test/run/lint command sequences *with tool versions* (validated by actually running them); document project layout and CI checks; and **explicitly tell the agent to trust the instructions and only search when they're incomplete or wrong.**

**VS Code specifics:** `.instructions.md` frontmatter `name` / `description` / `applyTo` (all optional). **No `applyTo` ⇒ not auto-applied** (attach manually); VS Code also semantically matches on `description`. Default locations `.github/instructions` (also `.claude/rules`, which uses `paths`); user-level `~/.copilot/instructions`. **Custom instructions do NOT affect inline completions — chat/agent only.** Tips: one short self-contained statement per rule; **include the reason** for each rule; show preferred/avoided code; **skip what linters/formatters already enforce**; split into topic-scoped files; commit to VCS; reference instruction files from prompt files/agents via Markdown links rather than duplicating.

## Prompt files (`.prompt.md`) — reusable, manually invoked

- **Extension** `.prompt.md`; **default location** `.github/prompts/`. Invoke with `/<name>` in chat, **Chat: Run Prompt**, or the editor play button.
- **Optional frontmatter:** `description`, `name` (the `/` name), `argument-hint`, `agent` (`ask` | `agent` | `plan` | a custom-agent name), `model`, `tools` (built-in, tool sets, MCP — `<server>/*` for all of a server's tools).
- **Body:** Markdown. Reference workspace files via relative Markdown links; reference tools inline `#tool:<name>`; take input via `${input:varName:placeholder}`; built-in vars like `${selection}`.
- **When to use:** lightweight single-task prompts (scaffold a component, generate tests, prep a PR). Use a custom agent instead for a persistent persona with tool restrictions; an agent skill for portable multi-file capabilities.
- **Tool precedence:** prompt-file `tools` > referenced custom-agent `tools` > selected agent's defaults.

## Custom agents (formerly custom chat modes) and built-in modes

- **Extension** `.agent.md`; **locations** `.github/agents/` (workspace), `.claude/agents/` (Claude format, plain `.md`), `~/.copilot/agents`. VS Code treats any `.md` in `.github/agents/` as a custom agent.
- **Frontmatter:** `description`, `name`, `argument-hint`, `tools` (array), `agents` (allowed subagents; `*` all, `[]` none — needs the `agent` tool), `model` (string or prioritized array), `user-invocable`, `disable-model-invocation`, `target` (`vscode` | `github-copilot`), `mcp-servers`, `hooks` (preview), `handoffs` (`label`/`agent`/`prompt`/`send`/`model`). `infer` is deprecated.
- **Handoffs** build guided sequential workflows (Plan → Implement → Review); buttons appear post-response; `send: true` auto-submits.
- **Why:** least-privilege tool lists per role — a Plan agent gets read-only tools to prevent edits; an implementer gets `edit`. Generate via `/create-agent`; share at workspace or org level.
- **Built-in modes** ("pick the right tool"): **Ask** (questions, exploring), **Inline/Edit** (targeted in-place edits), **Agent** (autonomous multi-file changes with planning + tools), **Plan** (read-only structured planning before implementation).

## Cloud / coding agent — task scoping

- **Treat the issue as the prompt.** A well-scoped task needs: a clear problem description, **complete acceptance criteria** (e.g., "are unit tests required?"), and direction on which files to change. Semantic code search means exact paths aren't mandatory but help.
- **Good first tasks:** bug fixes, UI tweaks, test coverage, docs, accessibility, tech debt.
- **Keep for yourself (don't delegate):** complex cross-repo refactors needing deep domain knowledge; sensitive/critical work (production-critical, security/PII/auth, incident response); ambiguous/open-ended; learning tasks where you want the understanding.
- **Plan-first:** Explore (ask mode reads code) → Plan (Plan agent produces a reviewable plan) → Implement (include tests/expected outputs so it self-verifies) → Review (checkpoints, rewind, request Copilot code review on the PR).
- **MCP:** extend the cloud agent with MCP tools; **GitHub MCP and Playwright MCP servers are enabled by default**; add more per-repo.
- **The single biggest merge-ability lever:** `copilot-instructions.md` documenting how to build/test/validate so the agent's PRs pass CI.

## Prompting idioms (slash commands, variables, participants)

- **Slash commands:** `/clear`, `/explain`, `/fix`, `/fixTestFailure`, `/tests`, `/new`, `/help`, `/rename`; authoring commands `/init`, `/create-instruction`, `/create-prompt`, `/create-skill`, `/create-agent`, `/create-hook`; session `/compact`, `/fork`.
- **Context variables (`#`):** `#file`, `#selection`, `#path`, `#sym`, `#function`, `#class`, plus `#<file>`/`#<folder>`/`#<symbol>` and **`#fetch`** to pull web pages / GitHub repos for current info.
- **Chat participants (`@`):** `@workspace` (project structure / cross-file; JetBrains `@project`), `@github`, `@terminal`, `@vscode`, `@azure` (preview).
- **Inline vs chat:** inline suggestions for flow (note: custom instructions don't touch inline); inline chat for targeted edits; chat panel for questions / multi-file / agentic work.

## Anti-patterns

- Vague prompts ("make this better") → specify the dimension ("reduce time complexity," "add null-input validation").
- Ambiguous referents ("this," "it") → name files/symbols.
- Piling unrelated tasks into one conversation → new sessions, delete stale history, `/compact`, `/fork`.
- One giant instruction file → scope with `applyTo`; keep concise (loads every chat turn); don't restate linter rules.
- Conflicting instructions across personal/repo/org layers.
- Delegating the wrong tasks to the cloud agent (security/PII/auth, production-critical, deeply ambiguous, design-consistency-critical refactors).
- Skipping review — AI output can carry bugs, injection flaws, hardcoded secrets; review, test, use checkpoints/rewind.
- Pasting credentials into prompts.
- Over-enabling tools — fewer active tools = faster, more relevant responses.
- Expecting custom instructions to steer inline completions (chat/agent only).

## Sources

- Prompt engineering for Copilot Chat: https://docs.github.com/en/copilot/concepts/prompting/prompt-engineering
- Adding repository custom instructions: https://docs.github.com/en/copilot/customizing-copilot/adding-repository-custom-instructions-for-github-copilot
- Copilot Chat cheat sheet: https://docs.github.com/en/copilot/reference/chat-cheat-sheet
- Best practices for the cloud agent: https://docs.github.com/en/copilot/tutorials/cloud-agent/get-the-best-results
- VS Code — custom instructions: https://code.visualstudio.com/docs/agent-customization/custom-instructions
- VS Code — prompt files: https://code.visualstudio.com/docs/agent-customization/prompt-files
- VS Code — custom agents: https://code.visualstudio.com/docs/agent-customization/custom-agents
- VS Code — best practices: https://code.visualstudio.com/docs/agents/best-practices
