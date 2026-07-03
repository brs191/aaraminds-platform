# .pptx build recipe — repeatable monthly look

How to turn the mapped content into the actual file, so every month's deck looks identical. This
composes two things: the **`pptx` skill** (mechanics of building a .pptx — read its SKILL.md for the
current API and helpers) and **Module 2 `02_Visual_Identity_System_v1.1.md`** (the canonical AaraMinds
visual identity, which wins on any disagreement with the defaults below).

## Build order

1. **Read the `pptx` skill's SKILL.md first** for the current, supported way to generate the file
   (it carries the up-to-date build approach and helpers). Build with that, not from memory.
2. If **last month's deck** is available, open it and reuse it as the visual base — copy the theme,
   master, and layouts, then replace content. This is the most reliable way to keep the look stable.
3. If there is no prior deck, build from the default theme below and **save the generated .pptx as
   the template seed** for next month.
4. Apply the locked slide order from `monthly-deck-template.md`.
5. Save to the user's selected folder with a dated name: `Status_<Initiative>_<YYYY-MM>.pptx`.

## Default theme (used only when no prior deck exists; Module 2 is canonical)

AaraMinds enterprise palette (from Module 2 — semantic, restrained, ≤3 dominant colors per slide).
Hex values are sensible defaults; if Module 2 specifies exact values, use those.

| Role (Module 2 meaning) | Use in the deck | Default hex |
|---|---|---|
| Off-white — canvas / breathing space | Slide background | `#F7F8FA` |
| Navy — titles, primary structure, executive framing | Slide titles, headers, structure | `#1B2A4A` |
| Blue — AI workflow / platform layers | Section accents, links | `#2E5FAC` |
| Teal — flow / efficiency / operating signals | Secondary accent, progress bars | `#19A7A0` |
| Purple — reasoning / agents / advanced capability | Use sparingly, only if semantically right | `#6B4FA1` |
| Green — outputs / readiness / success | **RAG Green** | `#2E9E5B` |
| Orange — warning / caution | **RAG Amber** | `#E8A13A` |
| Red — risk | **RAG Red** | `#C7402F` |

Type: a single clean sans-serif family throughout (e.g., Calibri / Arial as a safe cross-machine
default, or the corporate font if specified). Titles in Navy, bold; body in dark grey `#33373D`;
never more than two type sizes per slide. No neon, no gradients-for-decoration, color is semantic.

## RAG chip + trend arrow

- **RAG chip:** a filled rounded rectangle in the RAG color with a single bold letter (R/A/G) or the
  word, white text. Same size and position every month (cover top-right; dashboard rows left-aligned).
- **Month-over-month arrow:** a small glyph next to the chip — `↑` (improved) in Green, `→`
  (unchanged) in Navy/grey, `↓` (worsened) in Red. Computed from last month's deck, never asserted.
- **Overall RAG = worst load-bearing dimension or workstream.** The build must not auto-average
  colors into a greener cover.

## Layout conventions (per `monthly-deck-template.md`)

- **Title = the message.** Title placeholder holds the message sentence, Navy bold, top-left. No
  topic-only titles.
- **Slide 3 (health dashboard):** two tables — (a) dimensional: Dimension (Scope/Schedule/Quality/
  Cost/Risk/Dependencies) | RAG chip | arrow | one-line reason; (b) per-workstream roll-up: workstream
  | RAG | arrow | reason (≤7 rows).
- **Slide 4 (accomplishments):** Top-5 list — win | business impact | evidence | planned-vs-actual.
- **Slide 5 (risks):** a table — Risk statement | Business impact | Probability (H/M/L) | Severity
  (H/M/L) | Mitigation | Owner | Open since. Red/Amber chips inline. Top 3–5 only.
- **Slide 6 (decisions):** Issue | Impact | Decision needed | Due date | Owner.
- **Footer:** initiative · month · "AaraMinds" · page number — same every slide.
- **One message per slide; ≤3 proof points** (except the dashboard). Detail → appendix.

## Repeatability checklist

- Theme, master, and layouts identical to last month (reused base or saved seed).
- RAG chips and arrows in the same position and size as prior months.
- Section order unchanged (cover → exec summary → health dashboard → accomplishments → risks →
  [decisions] → [outlook] → appendix).
- Filename follows `Status_<Initiative>_<YYYY-MM>.pptx`.
- The generated file is saved as the seed for next month if no prior base existed.

## Mandatory visual-QA pass (do not skip — Anthropic's `pptx` skill requires it)

python-pptx cannot measure rendered text, so `auto_size` / `fit_text` / `word_wrap` **do not
guarantee text fits** — silent overflow and clipping are the #1 recurring-deck failure mode. You must
verify visually, not assume:

1. **Render every slide to an image** (the `pptx` skill ships the render path; or
   `libreoffice --headless --convert-to pdf` then rasterize).
2. **Have a fresh subagent inspect the images** — a separate context, because the builder "sees what
   it expects." Give it this checklist: text overflow / clipping beyond the placeholder; shapes
   off-canvas or overlapping; RAG chips/arrows misaligned vs prior month; contrast too low to read;
   margins < 0.5"; leftover placeholder text (grep the content for `xxxx|lorem|ipsum|TBD|<…>`).
3. **Fix and re-verify at least once.** Do not declare the deck done before one fix-and-verify cycle.

## Overflow / content-budget guard

- Enforce the template's content caps **before** rendering: ≤3 proof points per slide (except the
  dashboard), ≤5 accomplishments, ≤5 risks, ≤7 workstream rows. If content exceeds a cap, **split or
  move to appendix — never shrink the font to cram.**
- Use PowerPoint-safe fonts (inherit the template master's stack; e.g., Calibri→Arial) so PowerPoint
  doesn't silently substitute and re-wrap.

## Evidence & verification reports (emit alongside the .pptx)

- **Evidence report:** for every load-bearing claim on slides 1–6, record which input it came from
  (status notes / RAID / Jira-ADO / milestone tracker / financials). A claim with no traceable source
  is downgraded to `[VERIFY]`.
- **Verification report:** list every `[VERIFY]` tag and every gap (missing metric, missing owner,
  inconsistent date, unclear trend mapping) for the user's manual review before the deck ships.

## Note on automation

For a hands-off monthly draft, this skill pairs with a scheduled task (e.g., "first business day of
the month, draft the status deck from <inputs source> and save to <folder>"). The schedule supplies
cadence; this skill supplies the method. Keep a human review before the deck goes to the leader —
metric integrity and risk honesty are judgment calls, not automation outputs.
