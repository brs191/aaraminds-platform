---
name: prompt-engineering
description: Engineer, optimize, and teach prompts for AI coding assistants across Anthropic Claude, GitHub Copilot, and OpenAI Codex. Use when the task is to (a) GENERATE a prompt, system prompt, or instruction file (CLAUDE.md / copilot-instructions.md / AGENTS.md) from a task spec, (b) OPTIMIZE an existing prompt that under/over-triggers, rambles, stops early, or ignores rules, or (c) TEACH which pattern to apply and why. Covers system-vs-user prompting, XML structuring, multishot, tool-use and agentic patterns, and per-platform customization surfaces. Do NOT use for writing the underlying engineering content (use the relevant domain skill) or for non-prompt LLM-API integration code.
version: 1.0.0
last_updated: 2026-06-16
---

# Prompt Engineering

## When to use

Trigger this skill when the deliverable is a **prompt or a prompt-governing artifact**, on any of the three supported platforms:

- **Generate** — "write a system prompt for X," "draft a `CLAUDE.md` / `copilot-instructions.md` / `AGENTS.md` for this repo," "give me a prompt that makes the agent do Y," "scaffold a reusable prompt file / custom agent / skill."
- **Optimize** — "this prompt is underperforming — fix it," "the agent over-triggers / ignores my rules / rambles / stops early," "tighten this system prompt," "why isn't this prompt working."
- **Teach** — "which pattern do I use for this," "should this go in the system prompt or the user turn," "explain few-shot vs chain-of-thought here," "what's the right way to structure tool descriptions."

Do **not** use this skill for: writing the substantive engineering content a prompt is *about* (route to the relevant domain skill — e.g. `microservices-architecture-design` for the architecture, this skill for the prompt that elicits it); general marketing/copywriting; or LLM-API integration plumbing that isn't prompt text (retry logic, streaming, token accounting).

## The critical decision rule — match the platform, then the job

Two questions decide everything before you write a single line:

1. **Which platform is the prompt for?** Claude, GitHub Copilot, and OpenAI Codex have *materially different* conventions — different file names, different customization surfaces, and in several cases **opposite** advice (on Codex, durable instruction files should not force rollout-plan/preamble narration — it can stop the rollout early — whereas GPT-5 and Claude benefit from prompted preambles in specific cases; note `gpt-5.3-codex`+ *does* support promptable preambles when the `phase` field is preserved — see `references/codex-patterns.md`). A prompt tuned for one platform degrades on another. Never produce "generic" cross-platform prompt text and assume it ports. Identify the platform first; if the user hasn't said, ask.

2. **Which job is it — generate, optimize, or teach?** Generation starts from a task spec and produces new prompt text. Optimization starts from an existing prompt and a symptom, and produces a *diagnosis plus the smallest fix* — not a rewrite. Teaching produces an explanation and a pattern recommendation, not a finished prompt unless asked.

The two failure modes this rule prevents: **platform drift** (shipping Anthropic XML-tag idioms into an `AGENTS.md` that Codex parses as plain Markdown) and **rewrite drift** (responding to "my prompt over-triggers" by rewriting the whole prompt instead of dialing back the one aggressive directive that caused it).

## Platform dispatch

| Platform | Customization surfaces (exact names) | Signature idioms | Reference |
|---|---|---|---|
| **Anthropic Claude** | System prompt (`system=`), `CLAUDE.md` (project memory), `SKILL.md` (on-demand skills), subagents | XML tags (`<instructions>`, `<example>`, `<thinking>`), multishot in `<examples>`, role in system prompt, `effort` parameter, adaptive thinking | `references/claude-patterns.md` |
| **GitHub Copilot** | `.github/copilot-instructions.md` (repo-wide), `.github/instructions/*.instructions.md` (`applyTo` glob), `.github/prompts/*.prompt.md`, `.github/agents/*.agent.md` (formerly `.chatmode.md`) | Markdown natural-language rules, `applyTo` frontmatter, slash commands, `#file`/`@workspace` context vars, least-privilege `tools` lists | `references/copilot-patterns.md` |
| **OpenAI Codex** | `AGENTS.md` (nested, closest wins), `config.toml`, `SKILL.md`, `~/.codex/prompts` (deprecated) | Plain Markdown, **no frontmatter** in AGENTS.md, `reasoning_effort` knob, `<persistence>` / `<context_gathering>` blocks, "less is more" | `references/codex-patterns.md` |
| **Cross-platform core** | — | The principles that DO transfer: clarity, context/motivation, examples, structure, explicit scope, verification loops | `references/cross-platform-core.md` |

The job-level playbooks (generate / optimize / teach) with ready-to-use templates live in `references/workflows.md`.

## The cross-platform core — what transfers everywhere

These principles hold on all three platforms; the *syntax* differs but the *substance* is shared. Apply these first, then layer platform-specific idioms on top. Full treatment in `references/cross-platform-core.md`.

1. **Be clear and direct.** Treat the model as a brilliant new hire with no context on your norms. The golden test: *show the prompt to a colleague with minimal context — if they'd be confused, the model will be too.* Specify output format and constraints explicitly; sequence multi-step work as numbered steps.
2. **Give context and motivation.** Explain *why* a rule matters — every platform's model generalizes better from the reason than from the bare instruction.
3. **Use examples.** Few-shot / multishot is the most reliable lever for format, tone, and structure. Make examples relevant, diverse (cover edge cases), and clearly delimited. 3–5 is the sweet spot.
4. **Structure the prompt.** Separate instructions / context / input / output-format into labelled sections so the model can't conflate them (XML tags on Claude; Markdown headings on Copilot/Codex).
5. **State scope explicitly.** Current-generation models follow instructions literally and do not silently generalize. "Apply this to every section, not just the first." "Implement the change" (not "can you suggest changes," which yields suggestions).
6. **Give the model a way to verify.** Tests, a build/lint exit code, an expected output, a rubric. "If you can't verify it, don't ship it." Have the model show evidence, not assert success.
7. **Move durable rules out of the turn-by-turn prompt** and into the platform's persistent surface (`CLAUDE.md` / `copilot-instructions.md` / `AGENTS.md`). Keep those files short — bloat gets ignored on every platform.
8. **Prefer positive instructions over negative ones.** "Write in flowing prose paragraphs" beats "don't use bullet points." Match the prompt's own style to the output you want.

## Job playbooks (summary — full templates in `references/workflows.md`)

**Generate (task spec → prompt).** (1) Identify platform + surface (system prompt? instruction file? reusable prompt?). (2) Extract the four anchors from the spec: **Goal**, **Context** (files/examples/constraints), **Constraints** (standards, conventions, do-nots), **Done-when** (verification signal). (3) Choose the structure for the platform (XML on Claude; Markdown sections on Copilot/Codex). (4) Add examples if format matters. (5) Set the reasoning/effort control if the platform exposes one. (6) End with the verification hook. (7) State scope explicitly throughout.

**Optimize (prompt + symptom → diagnosis + smallest fix).** Map the symptom to the cause, fix only that:

| Symptom | Likely cause | Smallest fix |
|---|---|---|
| Over-triggers a tool/behavior | Aggressive directive ("CRITICAL: you MUST…", "if in doubt, use…") | Dial back to neutral ("Use X when…") — esp. on current Claude/Codex |
| Ignores a rule | Instruction-file bloat; rule buried | Prune the file; move the rule up; convert to a deterministic hook/gate |
| Rambles / over-long | No length/format directive; aggressive verbosity default | Add a concise positive format spec; show a short desired example |
| Stops early mid-task | (Codex) preamble/plan prompting; missing persistence | Remove plan-narration prompting (Codex) or add a `<persistence>` block |
| Conflicting outputs | Contradictory instructions (costly for GPT-5/Codex) | Audit for contradictions; resolve to one rule |
| Acts when you wanted advice (or vice versa) | Ambiguous action verb | Make the action explicit ("suggest only" vs "implement") |
| Hallucinates about unread code | No investigate-before-answering rule | Add "read the file before answering; never speculate" |

**Teach (question → pattern recommendation).** Name the pattern, give the one-line rule, show a minimal before/after, cite the platform source. Recommend system-prompt vs user-turn placement, and whether the rule belongs in a persistent instruction file.

## Worked example — brownfield: optimize an over-triggering Claude system prompt

Setup (brownfield — an existing, in-production prompt, not a greenfield draft): a team's Claude-Code-based agent calls its `search_codebase` tool on nearly every turn, even for trivial lookups, burning latency. Their current system prompt contains: `"CRITICAL: You MUST use the search_codebase tool before answering ANY question about the code. Default to searching if in doubt."`

Diagnosis (do not rewrite the prompt): current Claude models are *more* responsive to the system prompt than prior generations, so reduce-undertriggering language now over-triggers. The fix is the single directive, not the architecture.

Fix: replace with `"Use search_codebase when it would improve your understanding of the code in question. For trivial or already-answered lookups, answer directly."` One sentence changed. Then add a verification note only if needed: confirm tool-call rate drops on a sample of trivial queries.

Why this is right: it targets the named cause (aggressive phrasing × higher system-prompt sensitivity), preserves everything that worked, and is measurable. Rewriting the whole prompt would have discarded working instructions and made the regression hard to attribute.

## Anti-pattern — the platform-agnostic mega-prompt

**Bad:** Asked for "a good agent instructions file," the author writes one Markdown file full of XML `<instructions>` tags and Claude-style `effort` directives, and tells the user to drop it into `AGENTS.md`, `copilot-instructions.md`, *and* `CLAUDE.md`.

**Why it fails:** Codex parses `AGENTS.md` as plain Markdown and ignores XML semantics; Copilot's instruction files want `applyTo` frontmatter and short scoped rules; the `effort` directive is an Anthropic API concept with no meaning to the other two. The file is mediocre everywhere and wrong in specifics on two of three platforms.

**Detection signal:** the same prompt text is proposed for more than one platform without per-platform adaptation, or platform-specific mechanisms (XML tags, `applyTo`, `reasoning_effort`, `phase`) appear on a platform that doesn't use them.

**Fix:** Start from the cross-platform core (the eight transferable principles), then branch to the platform reference and re-encode each principle in that platform's idiom and file format. One source of substance, three platform-specific renderings.

## Verification questions

1. Is the target platform identified, and is the prompt written in *that* platform's idiom and file format (not a generic blend)?
2. For generation: are all four anchors present — Goal, Context, Constraints, Done-when?
3. For optimization: is the fix the *smallest* change that addresses the named symptom, or did it drift into a rewrite?
4. Are instructions positive (do-this) rather than a pile of don'ts, and is scope stated explicitly where literal models would otherwise under-apply?
5. Is there a verification hook the model can actually run (test, lint, expected output, rubric)?
6. Are durable rules placed in the platform's persistent surface, and is that file short enough not to be ignored?
7. Are platform-specific mechanisms used correctly and *only* on the platform that supports them (XML on Claude; `applyTo` on Copilot; `reasoning_effort`/`phase` on Codex)?
8. Are exact model IDs and API parameters flagged `[VERIFY]` where they may have shifted since `last_updated`?

## What to read next

- `references/cross-platform-core.md` — the eight transferable principles in depth, plus the universal prompt skeleton and the four anchors.
- `references/claude-patterns.md` — Anthropic/Claude: XML, system prompts, multishot, tool-use, agentic patterns, `CLAUDE.md`, effort/adaptive thinking, current-model deltas.
- `references/copilot-patterns.md` — GitHub Copilot: custom instructions, `applyTo`, prompt files, custom agents, cloud-agent task scoping, context idioms.
- `references/codex-patterns.md` — OpenAI Codex: `AGENTS.md`, `reasoning_effort`, persistence/context-gathering blocks, "less is more," the starter system-prompt skeleton.
- `references/workflows.md` — the generate / optimize / teach playbooks with copy-ready templates per platform.
