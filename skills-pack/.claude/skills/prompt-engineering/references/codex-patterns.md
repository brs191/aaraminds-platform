# OpenAI Codex — prompt patterns

Platform track for prompts targeting OpenAI Codex (the agentic coding tool: CLI + cloud/web + IDE + app), powered by the Codex-tuned GPT-5 family. This is NOT the deprecated 2021 code model.

> Model-version note: the current API model is `gpt-5.3-codex` (with `gpt-5.1-codex-max`, `gpt-5.4` in the family). Version numbers move quarterly — cite the *guidance* (stable) and flag exact model IDs / parameters as `[VERIFY]`.

## AGENTS.md — the primary custom-instructions surface

- **What:** an open-format "README for agents" — a predictable place for repo context/instructions. Plain **Markdown, no required fields, no YAML frontmatter, no special syntax.** Used by 60k+ projects; stewarded under the Linux Foundation; co-created by Codex, Amp, Jules, Cursor, Factory.
- **What goes in it:** repo layout / important dirs; how to run the project; build/test/lint commands; engineering conventions and PR expectations; constraints and "do-not" rules; what "done" means and how to verify. (Spec adds: project overview, code style, testing, security, commit/PR guidelines.)
- **Location & nesting (closest wins):** global personal defaults `~/.codex/AGENTS.md`; a repo-root file for shared standards; deeper files in subdirectories for local rules. Monorepos nest freely. **The closest AGENTS.md to the edited file wins; an explicit chat prompt overrides everything.**
- **How Codex loads it:** Codex CLI auto-enumerates these files and injects them; the model is **trained to closely adhere**. Files from `~/.codex` plus every directory root → CWD are merged root-to-leaf (deeper overrides shallower). Each becomes its own user-role message headed `# AGENTS.md instructions for <directory>`, injected near the top before the user prompt. `AGENTS.override.md` is supported.
- **Scaffold & maintain:** `/init` scaffolds a starter; then edit to match reality. Keep it **short and accurate over long and vague**; add rules only after you observe repeated mistakes. If it grows, keep the main file concise and reference task-specific files (`code_review.md`, planning docs). When Codex repeats a mistake, ask it for a retrospective and update AGENTS.md. Codex **runs the test commands listed in AGENTS.md** and fixes failures before finishing.

## reasoning_effort — the core thinking/tool-call knob

- Controls "how hard the model thinks and how willingly it calls tools." **Default `medium`.** Levels `minimal` → `low` → `medium` → `high` → `xhigh`. Scale up for complex multi-step/agentic tasks; down for latency.
- **Peak performance from splitting distinct tasks across multiple agent turns, one per task** — don't cram unrelated work into a single turn.
- Task mapping: Low = fast, well-scoped; Medium/High = complex changes or debugging; Extra-High = long, agentic, reasoning-heavy. `medium` is the all-around interactive default.

## Controlling agentic eagerness (steerability)

- **Reduce eagerness** (less tangential tool-calling, lower latency): lower `reasoning_effort`; give explicit context-gathering criteria with **early-stop conditions** and a **fixed tool-call budget** ("absolute maximum of 2 tool calls"); add an **escape-hatch** clause ("proceed even if it might not be fully correct").
- **Increase autonomy:** raise `reasoning_effort` and add a `<persistence>` block — "keep going until the query is completely resolved; never hand back on uncertainty; don't ask to confirm assumptions — decide, proceed, and document."
- **`<context_gathering>` block:** define what to read, when to stop, and the tool-call ceiling.

## GPT-5 vs GPT-5-Codex — the "less is more" inversion

This is the most important platform-specific delta, and it's the *opposite* of Claude/GPT-5 advice:

- **Codex is post-trained for the agentic coding harness, so it needs LESS prompting.** Start from OpenAI's standard Codex-Max prompt and make tactical additions only.
- **Remove, don't add:** *"Remove all prompting for the model to communicate an upfront plan, preambles, or other status updates during the rollout, as this can cause the model to stop abruptly before the rollout is complete."* (GPT-5 base, by contrast, *benefits* from prompted tool preambles.)
- **Mid-rollout updates are model-managed, not promptable** for Codex versions **before** `gpt-5.3-codex` — don't add instructions about intermediate plans/messages there. From `gpt-5.3-codex` on, updates are communicative and *are* promptable (Preambles & Personality).
- **Conflicting/vague instructions are uniquely costly.** GPT-5/Codex follow instructions "with surgical precision" and burn reasoning tokens reconciling contradictions. Audit prompts and remove conflicts; use OpenAI's prompt-optimizer.

## gpt-5.3-codex specifics

- **`phase` parameter (required):** the Responses API adds a `phase` field on **assistant** items (`null` / `"commentary"` / `"final_answer"`) to prevent early stopping. Persist and pass back assistant items *with* their `phase`; dropping it causes "significant performance degradation." Don't add `phase` to user messages. `[VERIFY]`.
- **Preambles & Personality:** acknowledge-then-plan (1-sentence ack + 1–2-sentence plan) before tool calls; mostly 1–2-sentence updates; cadence every 1–3 steps (hard floor every 6 steps / 10 tool calls). Two shippable personalities: **Friendly** (warm, "we/let's"; onboarding, ambiguous, high-stakes) and **Pragmatic** (terse, ship-focused; latency-sensitive).
- Use the **Responses API** (not Chat Completions) and pass `previous_response_id` to reuse reasoning traces (OpenAI measured Tau-Bench Retail 73.9% → 78.2% from that switch alone).

## The recommended starter system-prompt skeleton

The Codex prompting guide ships a section-structured starter prompt (built on `gpt-5.1-codex-max`). Replicate this skeleton, adapting sections only as needed:

- **General** — prefer `rg`/`rg --files` over grep; prefer dedicated tools over raw shell; parallelize tool calls; deliver working code, not just a plan.
- **Autonomy and Persistence** — "autonomous senior engineer"; persist end-to-end within the turn; bias to action with reasonable assumptions; don't end on clarifications unless truly blocked; avoid re-editing the same files in a loop.
- **Code Implementation** — correctness/clarity/reliability over speed; conform to codebase conventions; tight error handling (no broad try/catch, no silent failures); preserve type safety; DRY/search-first; fix root cause not symptom.
- **Editing constraints** — default ASCII; rare meaningful comments; use `apply_patch` for single-file edits; never revert user changes / never `git reset --hard` without approval; stop and ask on unexpected changes.
- **Exploration / reading files** — "think first, batch everything," use `multi_tool_use.parallel` (only that); sequential only when a read depends on a prior result.
- **Plan tool** — skip planning for the easiest ~25%; no single-step plans; update after each subtask; never end with only a plan; reconcile every TODO (Done/Blocked/Cancelled) before finishing.
- **Frontend** — avoid "AI slop"; expressive typography (avoid Inter/Roboto/Arial defaults); no purple-on-white / dark-mode bias; meaningful motion; finish to a runnable state; preserve existing design systems.
- **Presenting work / final message** — concise plain text (the CLI styles it); lead with the change then context; reference file paths (don't dump files); for "review" requests, findings-first ordered by severity with file/line refs.

**Self-reflection / rubric (zero-to-one generation):** add a `<self_reflection>` block — think of a rubric, build 5–7 internal categories (don't show the user), iterate internally until top marks across all.

## Config / customization surfaces

- **`config.toml` precedence (highest wins first, per the official docs):** (1) CLI flags / `-c`/`--config` overrides → (2) project `.codex/config.toml` (root → CWD, closest wins; trusted projects only) → (3) profile files selected with `--profile` (`~/.codex/<profile>.config.toml`) → (4) user config `~/.codex/config.toml` → (5) system config `/etc/codex/config.toml` → (6) built-in defaults. Shared across CLI/IDE/app. Sets model, reasoning effort, sandbox mode, approval policy, MCP servers, feature flags. **Durable prompt behavior** belongs in user config for personal defaults and in project `.codex/config.toml` for repo-wide standards (it overrides user config); reserve CLI flags for one-offs.
- **AGENTS.md** — durable repo guidance (above).
- **Skills (`SKILL.md`)** — package a repeatable workflow (instructions + context + logic); works across CLI/IDE/app. Scope to one job; description must say *what it does and when to use it* with real trigger phrases. Personal `$HOME/.agents/skills`; shared `.agents/skills` in the repo; `$skill-creator` scaffolds.
- **Custom prompts (`~/.codex/prompts/`)** — Markdown slash commands; **deprecated in favor of Skills.**
- **Slash commands:** `/init`, `/plan`, `/review`, `/resume`, `/fork`, `/compact`, `/status`, `/agent`.
- Also: MCP servers (`codex mcp add`), Hooks, Rules, Subagents, Plugins, Automations (schedule a stable prompt/skill).

## Effective task prompts (the four anchors)

Default structure for a Codex task: **Goal** (what to change/build) · **Context** (files/folders/docs/errors — `@`-mention them) · **Constraints** (standards, architecture, safety, conventions) · **Done-when** (tests pass, behavior changed, bug no longer reproduces). Use **Plan mode** (`/plan` or `Shift+Tab`) on hard tasks; ask Codex to interview you for fuzzy ideas; one thread **per task, not per project**.

## Anti-patterns

- Overloading the prompt with durable rules → move them to AGENTS.md or a skill.
- Conflicting/vague/contradictory instructions → uniquely costly; audit and resolve.
- Over-prompting Codex with preambles/plan-communication → can stop the rollout early; remove it (Codex-specific).
- Dropping the `phase` field on `gpt-5.3-codex` assistant items → significant degradation.
- Not giving working build/test commands → the agent can't see its own work.
- Skipping planning on multi-step tasks.
- Granting full machine permissions before understanding the workflow → keep approval/sandbox tight, loosen later.
- Running live threads on the same files without git worktrees.
- Automating a workflow before it's reliable manually.
- One thread per project instead of per task → context bloat.
- Wiring in every MCP/tool up front → start with one or two that remove a real manual loop.
- Over-clever / code-golf output → prompt for clarity-first, readable, maintainable code.

## Sources

- Codex prompting guide (canonical): https://cookbook.openai.com/examples/gpt-5/codex_prompting_guide
- GPT-5 prompting guide: https://cookbook.openai.com/examples/gpt-5/gpt-5_prompting_guide
- Codex best practices: https://developers.openai.com/codex/learn/best-practices
- AGENTS.md standard: https://agents.md/
- Codex customization concept: https://developers.openai.com/codex/concepts/customization
- Codex skills: https://developers.openai.com/codex/skills
- Codex config basics: https://developers.openai.com/codex/config-basic
