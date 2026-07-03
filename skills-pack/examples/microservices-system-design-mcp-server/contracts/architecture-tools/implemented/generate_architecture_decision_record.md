# Tool Contract: generate_architecture_decision_record

## Status

Implemented

## Purpose

Generate an Architecture Decision Record (ADR) in the canonical Michael Nygard format. The tool takes a structured description of a design decision under consideration and produces both a structured representation and a ready-to-commit markdown document, with warnings for weak ADR shapes (short context, no alternatives, missing rejection reasons) and a quality score.

## Risk Level

Low. The tool is informational/generative; it does not modify any external state.

## Approval Required

No. Generated ADRs are drafts that humans review before adopting.

## Input Schema

```json
{
  "system_name": "string (required)",
  "title": "string (required) — imperative, e.g. 'Use Saga with Outbox for Order Workflow'",
  "context": "string — 2–4 sentences describing the forces. Warning if <120 chars.",
  "decision": "string (required) — single statement of the chosen direction",
  "status": "Proposed | Accepted | Deprecated | Superseded (default: Proposed)",
  "date": "ISO date (YYYY-MM-DD). Defaults to today (UTC) if omitted.",
  "decided_by": "string — team or person, optional",
  "drivers": ["string", "..."],
  "options": [
    {
      "name": "string",
      "pros": ["string", "..."],
      "cons": ["string", "..."],
      "rejected": "boolean",
      "rejected_because": "string — required if rejected is true and quality score should stay high"
    }
  ],
  "consequences": {
    "positive": ["string", "..."],
    "negative": ["string", "..."],
    "neutral":  ["string", "..."]
  },
  "references": ["string", "..."]
}
```

### Field semantics

- `title` should be imperative and name one decision. Inputs whose title appears to combine multiple decisions (`"...and another..."`) are rejected with an error suggesting the team split them.
- `context` should describe the forces — what's pushing in different directions — not just the situation. Short contexts trigger a warning.
- `decision` should be a single statement, not a list.
- `options` should include the rejected alternatives with a `rejected_because` reason. ADRs without this lose retrospective value.
- `consequences` may be omitted; the tool will derive a minimal set from drivers and rejected options. The derived set is flagged as a warning so the team replaces it.

## Output Schema

```json
{
  "system_name": "string",
  "title": "string",
  "status": "Proposed | Accepted | Deprecated | Superseded",
  "date": "ISO date",
  "drivers": ["string", "..."],
  "context": "string",
  "decision": "string",
  "options": [/* same shape as input */],
  "consequences": {/* same shape as input */},
  "references": ["string", "..."],
  "warnings": ["string", "..."],
  "markdown": "string — full ADR rendered as markdown",
  "quality_score": "integer 0-100"
}
```

### Quality score

Starts at 100; deducts:
- short context (-15)
- no drivers (-15)
- no options recorded (-20)
- empty consequences (input) (-15)
- no references (-5)

A score below 60 indicates the ADR is a draft that needs work; below 30 indicates the input was too thin to produce a useful ADR even after auto-derivation.

## Errors

Returned via `mcp.NewToolResultError` when the input cannot be processed:

- `system_name is required`
- `title is required`
- `decision is required`
- `title appears to combine multiple decisions; split into separate ADRs`

Weak shapes (short context, no alternatives, etc.) are surfaced in `warnings`, not as errors.

## Example use

Generate the canonical "Use Saga with Outbox for Order Workflow" ADR before merging the saga implementation. The team reviews the markdown, edits, and commits to the repo's `docs/adr/` directory.
