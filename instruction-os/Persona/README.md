# AaraMinds Persona System

## Purpose

This folder contains the active AaraMinds Persona source files.

Use it as the canonical instruction system for base behavior, reusable modules, and future role-based assistants.

## Active Load Order

1. Load `01_Layered_Base_System_v1.1.md`.
2. Load only the relevant task module:
   - `02_Visual_Identity_System_v1.1.md`
   - `03_Newsletter_Editorial_System_v1.1.md`
   - `04_Framework_Creation_System_v1.1.md`
   - `05_AI_Systems_Review_System_v1.2.md`
   - `06_LinkedIn_Post_System_v1.1.md`
   - `07_AI_Engineering_Trend_Scan_System_v1.1.md`
   - `08_AI_Agent_Blueprint_System_v1.1.md`
   - `09_Project_Delivery_Planning_System_v1.0.md`
3. Adapt the combined instructions to the target platform.

## Role-Based Personas

Role-based personas are compositions.

They should load the canonical base, relevant modules, and a small role delta.

Active role-based personas:

- `AaraMinds_Content_Strategist_v1.0.md`
- `AaraMinds_AI_Agent_Blueprint_Advisor_v1.1.md`
- `AaraMinds_AI_Engineering_Architect_v1.2.md`
- `AaraMinds_AI_Business_Strategist_v1.1.md`
- `AaraMinds_Executive_Narrative_Advisor_v1.0.md`
- `AaraMinds_Project_Planner_v1.0.md`

Use role-based personas only after the relevant modules are stable.

## Rankings and Validation

Current scores and tier snapshot: [`Ranking.md`](../../Ranking.md) — the workspace-wide master ranking at the `aaraminds/` root, which supersedes the former `Rankings.md` and covers personas, modules, agents, skills, hooks, and MCP tools in one place.

Dated audit and validation history: `Validation_History.md` (append-only, per-pass entries).

`References/` holds dated snapshots that age fast and live outside the modules to prevent rot (e.g., `AI_Engineering_Trendsetters_2026-05.md`).

## Canonical Base

`01_Layered_Base_System_v1.1.md` is the only active Persona foundation.

Older base persona files should stay outside this folder or be treated as archive material.

## Module Rule

Modules refine the base system.
They should not redefine the base identity, voice, or decision logic unless explicitly required.
