# AaraMinds Instruction OS

This folder contains the AaraMinds modular instruction system: canonical persona source files, role-based assistants, validation artifacts, dated references, platform exports, and inactive archives.

## Folder Map

| Folder | Purpose | Use |
| --- | --- | --- |
| `Persona/` | Source of truth for active AaraMinds personas and modules | Edit and validate active instruction files here |
| `Persona/Testing/` | Stress tests, audits, generated outputs, and validation results | Use to verify changes before promoting a module or persona |
| `Persona/References/` | Dated reference maps for volatile AI ecosystem information | Use as search anchors and orientation, not as current evidence |
| `Exports/` | Platform-ready copies generated from active persona files | Use when loading instructions into ChatGPT or another platform |
| `Archive/` | Historical inactive versions | Keep for reference only; do not load by default |

## Operating Model

The active source lives in `Persona/`.

Use this load order:

```text
01_Layered_Base_System_v1.1.md
+ relevant specialist module
+ role-based persona when needed
```

Do not edit exported files as the source of truth. Update the active persona/module first, then regenerate exports if needed.

## Key Files

- `Persona/README.md` — active load order and role-persona list
- `../Ranking.md` — current score snapshot (the workspace-wide master ranking at the `aaraminds/` root; supersedes the former `Persona/Rankings.md`)
- `Persona/Validation_History.md` — dated score and validation history
- `Persona/Persona_WIP.md` — operating board and next actions
- `Persona/Feedback.md` — retrospective notes and system learnings

## Current Principle

AaraMinds instructions should be modular, grounded, validated, and useful.

The goal is not to collect prompts.

The goal is to maintain a reliable AI advisory system for content strategy, enterprise AI architecture, AI agent blueprinting, systems review, trend scanning, and founder strategy.
