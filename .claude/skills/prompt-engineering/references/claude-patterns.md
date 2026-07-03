# Anthropic Claude — prompt patterns

Platform track for prompts targeting Claude (API, Claude Code, or Claude-powered agents). Anthropic has **consolidated** its prompt-engineering docs into one canonical page plus per-model pages; cite the consolidated page, not the old per-technique URLs (which now 404/redirect).

> Model-version note: the live docs reference Claude Opus 4.8 / 4.7 / 4.6, Sonnet 4.6, Haiku 4.5, the `effort` parameter, and adaptive thinking — all newer than mid-2025. Treat exact model IDs and parameter names as `[VERIFY]` against the docs at use time; the *guidance* below is stable.

## Technique priority order (Anthropic's stated ladder)

Be clear and direct → add context/motivation → use examples (multishot) → structure with XML tags → give Claude a role (system prompt) → long-context tips → thinking/CoT → chain complex prompts. Apply in this order; earlier levers have the highest return.

## System prompt vs user prompt

- **Role goes in the system prompt.** Even one sentence shifts behavior and tone: `system="You are a senior Go engineer specializing in MCP servers."` Put the task in the user turn.
- The **system prompt is the home for persistent behavioral steering**: action defaults, formatting rules, autonomy/safety rules, anti-hallucination rules. Anthropic writes all of these as named-tag snippets in the system prompt.
- **Current models are MORE responsive to the system prompt** than prior generations. Prompts written to fix *under*-triggering on older models now *over*-trigger. Fix: dial back aggressive language — replace `"CRITICAL: You MUST use this tool when…"` with `"Use this tool when…"`.
- **Prefill (shaping the assistant turn) is deprecated on current models.** From Claude 4.6+, a prefilled final assistant turn returns a 400 error. Migrate prefill use cases (forcing output format, killing preambles, avoiding bad refusals, role consistency) into **system-prompt instructions**. Older models still support prefill; non-final assistant messages are unaffected. `[VERIFY]` the exact cutoff model.

## XML tags — the Claude structuring idiom

- **Why:** clarity (separate parts), accuracy (less misinterpretation), flexibility (find/add/remove parts), parseability (tagged *output* is easy to post-process).
- **No magic tags** — Claude isn't trained on a fixed canonical set. Tag names should simply describe their content.
- **Commonly used names:** `<instructions>`, `<context>`, `<input>`, `<example>` / `<examples>`, `<formatting>`, `<thinking>`, `<answer>`; for documents, `<documents>` → `<document index="1">` → `<document_content>` + `<source>`.
- **Best practices:** be consistent (reuse the same names and refer to them in instructions — "using the contract in `<contract>` tags…"); nest for hierarchy (`<outer><inner></inner></outer>`).
- **Combine with multishot and CoT** for "super-structured" prompts: `<examples>` of input/output, plus `<thinking>`/`<answer>` separation.
- **Use XML as an output-format indicator too:** "Write the prose sections in `<prose>` tags."

## Examples / multishot

The most reliable lever for format, tone, structure. Wrap each example in `<example>`, the set in `<examples>`. Make them **relevant**, **diverse** (cover edge cases; vary enough to avoid accidental pattern-locking), and **structured**. 3–5 examples for best results. You can ask Claude to critique your examples for relevance/diversity or to generate more.

## Tool use / function calling

- **Treat tool definitions as first-class prompt engineering** — Anthropic spent more time optimizing tools than the overall prompt on their SWE-bench agent. Invest in the agent-computer interface (ACI) like you would a human UI.
- **Write tool descriptions like a docstring for a junior dev:** example usage, edge cases, input-format requirements, clear boundaries vs other tools.
- **Poka-yoke the tools** (make mistakes impossible): e.g., require absolute file paths so the model can't break after a `cd`.
- **Format choice matters:** keep tool I/O close to what appears naturally in text; markdown code is easier for the model than code-in-JSON (escaping overhead); avoid formats needing exact line counts.
- **Explicit triggering:** current models follow instructions literally — "can you suggest changes" yields suggestions, not edits. Steer the default with system-prompt tags like `<default_to_action>` (implement, infer intent, proceed) or `<do_not_act_before_instructions>` (research/recommend only).
- **Don't over-prompt tool use** on current models: "If in doubt, use [tool]" / "Default to [tool]" now over-trigger. Use "Use [tool] when it would enhance your understanding."
- **Effort raises tool usage.** On Opus 4.8, increasing `effort` (`high`/`xhigh`) yields substantially more tool use; prefer raising effort over nagging prose if a tool under-fires.
- **Parallel tool calls:** current models do this well. Push toward 100% with a `<use_parallel_tool_calls>` snippet (independent calls in parallel; dependent calls sequential; never guess missing params). Reduce with "execute operations sequentially."

## Extended / adaptive thinking (current models)

- **Adaptive thinking** (`thinking: {type:"adaptive"}`) lets Claude decide when and how much to think, calibrated by the **`effort`** parameter (`low`/`medium`/`high`/`xhigh`/`max`) and query complexity. Anthropic reports it beats fixed extended thinking. `effort` replaces the deprecated `budget_tokens`. `[VERIFY]` parameter names.
- **Thinking is OFF by default** when the `thinking` parameter is omitted.
- **Promptable both ways:** suppress over-thinking with "respond directly unless multi-step reasoning is needed"; deepen with "after receiving tool results, reflect on their quality before proceeding."
- **Four thinking rules:** (1) prefer general instructions ("think thoroughly") over prescriptive step lists — Claude's reasoning often exceeds a hand-written plan; (2) multishot works *with* thinking — put `<thinking>` inside examples; (3) manual CoT as fallback when thinking is off (`<thinking>` then `<answer>`); (4) ask Claude to self-check before finishing.
- **Word-sensitivity gotcha:** with thinking disabled, Opus 4.5 is sensitive to the word "think" — use "consider," "evaluate," "reason through" instead.
- **Opus 4.8 effort guidance:** start `xhigh` for coding/agentic; min `high` for intelligence-sensitive work; `max` can overthink. Raise effort rather than prompting around shallow reasoning. At high effort set a large `max_tokens` (~64k start) for room.

## Agentic patterns (Building Effective Agents + Claude Code best practices)

- **Start simple; add complexity only when it demonstrably helps.** Distinguish **workflows** (LLMs on predefined code paths) from **agents** (LLM directs its own process). Five building blocks: prompt chaining, routing, parallelization (sectioning/voting), orchestrator-workers, evaluator-optimizer.
- **Agents = LLMs using tools in a loop on environmental feedback.** Get ground truth each step (tool results, code execution); build in stopping conditions and human checkpoints. Three principles: simplicity, transparency (show planning steps), careful ACI design.
- **`CLAUDE.md` = persistent project memory**, read every session (generate with `/init`). Keep it short and human-readable. Test each line: *would removing it cause a mistake?* If not, cut — **bloated `CLAUDE.md` gets ignored.** Include: non-guessable bash commands, non-default code style, test runners, repo etiquette, architecture decisions, gotchas. Exclude: anything inferable from code, standard conventions, file-by-file descriptions. Boost adherence with "IMPORTANT"/"YOU MUST" sparingly. Locations: `~/.claude/CLAUDE.md` (global), `./CLAUDE.md` (team, committed), `./CLAUDE.local.md` (gitignored), parent/child dirs (monorepo). Move sometimes-relevant knowledge into `SKILL.md` skills so it loads on demand without bloating every turn.
- **Explore → Plan → Code → Commit.** Use plan mode to separate research from execution. Skip planning for one-line diffs; plan when uncertain/multi-file/unfamiliar.
- **Verification loop:** give Claude a runnable check (tests, build exit code, linter, fixture diff, screenshot vs design). Escalate the gate: in-prompt check → re-checked goal → deterministic stop hook → verification subagent. Have it show evidence, not assert success.
- **Subagents** run in separate context windows and report summaries — the key tool because context is the constraint. Use for scoped investigation and for fresh-context adversarial review of a diff against `PLAN.md` ("report correctness/requirements gaps, not style"). A gap-seeking reviewer over-reports — restrict it.
- **Manage context aggressively:** `/clear` between unrelated tasks; after two failed corrections, `/clear` and rewrite the prompt; `/compact` for targeted compaction.

## Current-model deltas that change the advice (Opus 4.x / Sonnet 4.x)

- **More literal** (esp. Opus 4.8 at low effort): state scope explicitly; it won't infer unrequested work.
- **Adaptive verbosity:** short on lookups, long on analysis; may skip post-tool summaries — re-request them if wanted. Positive concision examples beat "don't" instructions.
- **Higher system-prompt responsiveness → dial back aggressive language.**
- **Over-engineering tendency** (4.5/4.6): add an "avoid over-engineering" scope-discipline snippet (no extra abstractions, no docstrings on untouched code, no defensive code for impossible cases).
- **Subagent spawn rate differs by model** (4.6 over-spawns; 4.8 spawns fewer) — give explicit "when to use a subagent" guidance.
- **Effort matters more than on any prior Opus** — experiment when you upgrade.
- **Front-end house style** (Opus 4.8): defaults to cream/serif/terracotta editorial look — specify a concrete alternative palette for dashboards/fintech rather than just negating.

## Anti-patterns

- Prefilling the final assistant turn on 4.6+ (400 error) → use system-prompt instructions.
- Over-prompting tool/skill use ("if in doubt, use…", "CRITICAL: you MUST…") → neutral "use X when…".
- Telling Claude only what *not* to do → convert to positives; match prompt style to desired output.
- Bloating `CLAUDE.md` → prune ruthlessly; convert hard rules to hooks.
- Shipping unverified output → "if you can't verify it, don't ship it."
- Kitchen-sink sessions / repeated corrections in one context → `/clear` and rewrite.
- Unscoped "investigate X" → scope narrowly or delegate to a subagent.
- Letting Claude hard-code to test inputs → "tests verify correctness, they don't define the solution; tell me rather than work around."
- Adding agent/framework complexity prematurely → add only when it demonstrably helps.

## Sources

- Prompting best practices (consolidated): https://platform.claude.com/docs/en/build-with-claude/prompt-engineering/claude-prompting-best-practices
- Prompting Claude Opus 4.8 (current model page): https://platform.claude.com/docs/en/build-with-claude/prompt-engineering/prompting-claude-opus-4-8
- Use XML tags: https://platform.claude.com/docs/en/build-with-claude/prompt-engineering/use-xml-tags
- Building effective agents: https://www.anthropic.com/engineering/building-effective-agents
- Claude Code best practices: https://code.claude.com/docs/en/best-practices
- Extended/adaptive thinking: https://platform.claude.com/docs/en/build-with-claude/extended-thinking
