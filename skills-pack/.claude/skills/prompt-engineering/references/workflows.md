# Workflows — generate / optimize / teach, with copy-ready templates

The three jobs this skill performs, each as a step-by-step playbook plus templates you can fill and ship. Always run the platform check first (`SKILL.md` → "match the platform, then the job").

---

## Job 1 — GENERATE (task spec → prompt)

### Steps
1. **Identify platform + surface.** Is the deliverable a system prompt, a persistent instruction file (`CLAUDE.md` / `copilot-instructions.md` / `AGENTS.md`), a reusable prompt file, a custom agent, or a one-off task prompt? Each has its own format.
2. **Extract the four anchors** from the spec: Goal, Context, Constraints, Done-when. If any is missing from the spec, ask — don't invent.
3. **Choose structure for the platform:** XML tags (Claude) or Markdown headings (Copilot/Codex).
4. **Add examples** (3–5, relevant + diverse) if output format matters.
5. **Set the reasoning/effort control** if the platform exposes one (Claude `effort`; Codex `reasoning_effort`; Copilot model picker).
6. **Add the verification hook** (Done-when made runnable: test, lint, expected output, rubric).
7. **State scope explicitly** and prefer positive phrasing throughout.

### Template A — Claude system prompt (XML)
```
You are <role: one sentence>.

<context>
<relevant files, prior decisions, the known-good example to imitate>
</context>

<instructions>
1. <step>
2. <step>
Apply this to <explicit scope, e.g. "every endpoint, not just the first">.
</instructions>

<constraints>
- <standard / convention / library to use or avoid> — because <reason>.
- <explicit do-not>
</constraints>

<examples>
<example>
<input>...</input>
<output>...</output>
</example>
<!-- 3–5 examples, diverse, covering edge cases -->
</examples>

<output_format>
<exact shape, positive phrasing: "Write the summary in flowing prose paragraphs.">
</output_format>

<verification>
Before finishing, <run the test / diff against the example / self-check against this rubric>. Show the evidence; do not assert success.
</verification>
```
Set `effort` (start `xhigh` for coding/agentic) and a large `max_tokens` at high effort. Avoid "CRITICAL/you MUST" — use neutral directives.

### Template B — Copilot `.github/copilot-instructions.md` (repo-wide, ≤ 2 pages)
```
# Project: <name>

## What this repo is
<one paragraph: purpose, stack, layout of important dirs>

## Build / test / run (exact commands, with tool versions)
- Install: `<cmd>`
- Build: `<cmd>`
- Test: `<cmd>`
- Lint: `<cmd>`

## Conventions (non-obvious only — skip what the linter enforces)
- <rule> — because <reason>.

## Constraints / do-not
- <rule>

Trust these instructions. Only search the codebase if something here is incomplete or proves wrong.
```
For language/path-specific rules, add `.github/instructions/<topic>.instructions.md` with frontmatter `applyTo: "**/*.ts,**/*.tsx"`.

### Template C — Codex `AGENTS.md` (plain Markdown, no frontmatter, closest-wins)
```
# <repo / subdir name>

## Overview
<what this code does, key directories>

## Commands
- Run: `<cmd>`
- Test: `<cmd>`   <!-- Codex runs these and fixes failures before finishing -->
- Lint: `<cmd>`

## Conventions
- <convention>

## Constraints / do-not
- <rule>

## Done means
<the verification signal>
```
Keep it short; add rules only after observing repeated mistakes. Do **not** add forced preamble/plan-narration prompting to `AGENTS.md` (it can stop the Codex CLI rollout early). On the `gpt-5.3-codex`+ API harness preambles are promptable when the `phase` field is preserved — see `codex-patterns.md`.

### Template D — reusable Copilot prompt file (`.github/prompts/<name>.prompt.md`)
```
---
description: <what this prompt does>
agent: agent
model: <optional>
tools: ['codebase', 'editFiles', '<mcp-server>/*']
---
Goal: <task, with ${input:feature:feature name}>.
Context: link a real instruction file here, e.g. `.github/instructions/standards.instructions.md`.
Steps: 1) ... 2) ...
Done when: <verification>.
```

---

## Job 2 — OPTIMIZE (existing prompt + symptom → diagnosis + smallest fix)

### Steps
1. **Get the symptom and the prompt.** What is it doing wrong, on which platform, on which model generation?
2. **Map symptom → cause** using the table below. Do NOT rewrite — find the one or two lines responsible.
3. **Apply the smallest fix**, preserving everything that works.
4. **State how to verify** the fix (sample of the failing case; measure the behavior changed).

### Diagnosis table
| Symptom | Likely cause | Smallest fix |
|---|---|---|
| Over-triggers a tool/behavior | Aggressive directive ("CRITICAL", "you MUST", "if in doubt, use…"); current Claude/Codex over-respond | Neutralize to "Use X when it would help…" |
| Ignores a documented rule | Instruction-file bloat; rule buried deep | Prune file; move rule up; convert to a deterministic hook/gate |
| Rambles / too long | No format/length spec; adaptive-verbosity default | Add a positive concision spec + a short desired-output example |
| Stops early mid-task (Codex) | Preamble/plan-narration prompting; no persistence | Remove plan-narration prompting; add `<persistence>` block |
| Stops early / hands back (Claude) | Low effort; asks to confirm assumptions | Raise `effort`; add "proceed without confirming reasonable assumptions" |
| Contradictory / inconsistent output | Conflicting instructions (very costly for GPT-5/Codex) | Audit for contradictions; resolve to a single rule |
| Did the wrong action (acted vs advised) | Ambiguous action verb | Make explicit: "implement" vs "suggest only" |
| Hallucinates about unread code | No investigate-before-answering rule | Add "read the file before answering; never speculate about unopened code" |
| Over-engineers / adds unrequested abstractions | Eager current models (Claude 4.5/4.6) | Add scope-discipline snippet (no extra files/abstractions/defensive code) |
| Generic "AI slop" frontend | No concrete design direction | Specify a concrete palette/typography, or ask it to propose 3–4 directions first |
| Code-review recall seems to drop | "Only report high-severity" read as a filter | Tell it this stage is coverage not filtering; a later step will filter |

### Output shape for an optimization
Lead with the **diagnosis** (named cause), then the **exact change** (old → new), then the **verification**. One or two lines changed, not a rewrite. If the prompt genuinely needs a rebuild (rare — only when it's structurally incoherent), say so explicitly and justify it.

---

## Job 3 — TEACH (question → pattern recommendation)

### Steps
1. **Name the pattern** the question maps to (few-shot, CoT/thinking, role-setting, system-vs-user placement, tool-description design, persistence, context-gathering budget, instruction-file vs turn prompt).
2. **Give the one-line rule** and *why*.
3. **Show a minimal before/after** in the target platform's idiom.
4. **Recommend placement:** system prompt vs user turn; persistent instruction file vs per-task prompt.
5. **Cite the platform source.**

### Common teaching calls (quick reference)
- *"System prompt or user turn?"* → Role and persistent behavior in the system prompt; the specific task and inputs in the user turn (Claude). On Copilot/Codex, "persistent behavior" = the instruction file (`copilot-instructions.md` / `AGENTS.md`); the task = the chat prompt.
- *"Few-shot or chain-of-thought?"* → Few-shot when you need a specific *format/structure*; thinking/CoT when you need *reasoning* on a hard multi-step problem. They compose — put `<thinking>` inside few-shot examples (Claude).
- *"Why is my rule ignored?"* → Almost always instruction-file bloat or aggressive phrasing. Prune, or convert to a deterministic gate.
- *"How do I make it act vs just advise?"* → Explicit action verb; on Claude, a `<default_to_action>` vs `<do_not_act_before_instructions>` system-prompt stance.
- *"How much should I prompt the agent?"* → Claude/GPT-5: structured and explicit. **Codex: less is more** — start from the standard prompt, add tactically, remove plan-narration.

---

## A note on AaraMinds house use

When generating prompts/instruction files *for AaraMinds repos*, fold in the workspace's fixed-stack and anti-pattern rules (see the root `CLAUDE.md`): Azure-primary, Terraform AzureRM, GitHub Actions OIDC, Go/Spring Boot backends, no cloud/tool drift, no fabricated metrics (use `[VERIFY]`), brownfield-by-default. A generated `copilot-instructions.md` or `AGENTS.md` for an AaraMinds service should encode those as explicit constraints with reasons.
