# Cross-platform core — the principles that transfer

These eight principles hold on Anthropic Claude, GitHub Copilot, and OpenAI Codex. The *substance* is shared; the *syntax* (XML vs Markdown, which file, which parameter) is platform-specific and lives in the per-platform references. Always apply this core first, then re-encode it in the target platform's idiom.

## The four anchors (the spine of any task prompt)

Every task-directed prompt — on any platform — should make these four things unambiguous. OpenAI states them explicitly for Codex; Anthropic and GitHub teach the same content under different headings. Use them as a checklist.

1. **Goal** — what to build or change, in one or two sentences, lead with the outcome.
2. **Context** — which files, folders, examples, errors, and prior decisions are relevant. Point to a known-good example to imitate ("`HotDogWidget.php` is the pattern to follow"). Name symbols, not "this."
3. **Constraints** — standards, conventions, architecture rules, libraries to use/avoid, and explicit do-nots. Include the *reason* for non-obvious constraints.
4. **Done-when** — the verification signal: tests pass, the bug no longer reproduces, the lint/type check is green, the output matches an example. If the model can't check it, "looks done" is the only signal and quality drops.

## The eight transferable principles

### 1. Be clear and direct
Treat the model as a brilliant new hire with zero context on your norms. The golden test (Anthropic): show the prompt to a colleague with minimal context; if they'd be confused, so will the model. Specify output format and constraints explicitly. Sequence multi-step work as numbered steps when order or completeness matters. If you want "above and beyond," ask for it — don't rely on inference.

### 2. Give context and motivation
Explain *why* a rule exists. "Use `date-fns` instead of `moment.js` because moment.js is deprecated and inflates bundle size" outperforms the bare "use date-fns," because the model generalizes the reason to edge cases the rule didn't enumerate. True on all three platforms.

### 3. Use examples (few-shot / multishot)
The single most reliable lever for steering format, tone, and structure. Make examples **relevant** (mirror the real case), **diverse** (cover edge cases so the model doesn't latch onto an accidental pattern), and **clearly delimited** (XML `<example>` on Claude; fenced code or a labelled section on Copilot/Codex). 3–5 examples is the sweet spot. Unit tests double as examples — write the tests first, then ask for the implementation that satisfies them.

### 4. Structure the prompt
Separate **instructions**, **context**, **input**, and **output format** into labelled sections so the model can't conflate them. Claude: XML tags. Copilot/Codex: Markdown headings. Structure also makes the prompt maintainable — you can find and change one part without disturbing the rest.

### 5. State scope explicitly
Current-generation models (Claude Opus 4.x, GPT-5/Codex) follow instructions *literally* and do not silently generalize. "Apply this formatting to every section, not just the first." Distinguish action from advice: "implement the change" produces edits; "can you suggest changes" produces only suggestions. Ambiguous action verbs are a top cause of "it did the wrong thing."

### 6. Give the model a way to verify
Provide a check it can run: a test command, a build/lint/type exit code, an expected output to diff against, or a rubric to self-score. Have it **show evidence** (command + result, test output, screenshot) rather than assert "done." The strongest setups escalate the gate: in-prompt check → re-checked goal condition → deterministic stop hook → a fresh-context reviewer. "If you can't verify it, don't ship it."

### 7. Move durable rules to the persistent surface
Anything true for the whole repo/project belongs in the platform's memory file, not re-typed each turn: `CLAUDE.md` (Claude), `.github/copilot-instructions.md` (Copilot), `AGENTS.md` (Codex). Keep these files **short** — every platform's model starts ignoring them when they bloat. The test (Anthropic, restated by all three): *would removing this line cause a mistake?* If not, cut it. Convert hard requirements into deterministic hooks/gates where the platform supports them, rather than hoping advisory prose is obeyed.

### 8. Prefer positive instructions; match the prompt's style to the output
"Write in smoothly flowing prose paragraphs" beats "do not use bullet points." Convert every "don't" into the "do" that replaces it. And match the prompt's own formatting to what you want back — a prompt written in heavy markdown nudges the model toward heavy-markdown output; strip it to reduce it.

## The universal prompt skeleton

A platform-agnostic skeleton you then render in the target idiom. (Claude version uses XML tags for each section; Copilot/Codex versions use `##` Markdown headings.)

```
ROLE / PERSONA        — who the model should act as (1 sentence is enough)
GOAL                  — the outcome, lead with it
CONTEXT               — relevant files, examples, prior decisions, the pattern to follow
CONSTRAINTS           — standards, conventions, libraries, explicit do-nots (+ reasons)
PROCESS               — numbered steps if order/completeness matters; else omit
EXAMPLES              — 3–5 delimited input/output pairs if format matters
OUTPUT FORMAT         — exact shape of the deliverable (positive phrasing)
DONE-WHEN / VERIFY    — the check the model runs before finishing; show evidence
SCOPE NOTES           — explicit "apply to all / only to X" guardrails
```

Not every prompt needs every section. The minimum viable task prompt is Goal + Context + Constraints + Done-when. Add Examples when format matters, Process when sequence matters, Role when tone/behavior matters.

## Where the platforms genuinely diverge (read the references for each)

These are *not* transferable — getting them wrong is platform drift:

- **Preambles / plan narration.** Claude and GPT-5 benefit from prompting an upfront plan in specific cases; on **Codex**, don't force rollout-plan/preamble narration in durable instructions — it can stop the rollout early. The nuance: this absolute applies to the Codex CLI / `AGENTS.md` and to Codex versions before `gpt-5.3-codex`; from `gpt-5.3-codex`+ preambles *are* promptable and supported when the API `phase` field is preserved (see `codex-patterns.md`). Opposite default from Claude/GPT-5.
- **Structuring syntax.** XML tags are an Anthropic idiom Claude is tuned for; `AGENTS.md` is plain Markdown with no frontmatter; Copilot instruction files use YAML frontmatter (`applyTo`).
- **Thinking/effort controls.** Claude: `effort` + adaptive thinking. Codex: `reasoning_effort` (+ `phase` on newer models). Copilot: model picker, no exposed effort knob.
- **Aggressive directive language.** Current Claude and Codex *over*-trigger on "CRITICAL/you MUST/if in doubt" — dial back. Older models needed the emphasis. Calibrate to the model generation.
- **Tool/context variables.** `@workspace`, `#file` are Copilot-only. `multi_tool_use.parallel` is a Codex harness construct. Don't cross-pollinate.

## Sources

- Anthropic — Prompting best practices (consolidated): https://platform.claude.com/docs/en/build-with-claude/prompt-engineering/claude-prompting-best-practices
- Anthropic — Building effective agents: https://www.anthropic.com/engineering/building-effective-agents
- Anthropic — Claude Code best practices: https://code.claude.com/docs/en/best-practices
- GitHub — Prompt engineering for Copilot Chat: https://docs.github.com/en/copilot/concepts/prompting/prompt-engineering
- VS Code — Best practices for using AI: https://code.visualstudio.com/docs/agents/best-practices
- OpenAI — Codex best practices: https://developers.openai.com/codex/learn/best-practices
- OpenAI — GPT-5 prompting guide: https://cookbook.openai.com/examples/gpt-5/gpt-5_prompting_guide
- agents.md standard: https://agents.md/
