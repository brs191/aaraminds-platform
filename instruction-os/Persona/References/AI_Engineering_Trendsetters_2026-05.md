# AI Engineering Trendsetters Map (dated reference)

**As of:** 2026-05-21
**Used by:** `07_AI_Engineering_Trend_Scan_System_v1.1.md`

This file is a dated snapshot of representative AI engineering trendsetters by namespace. It exists as a separate file because vendor names age fast — a list inline in Module 7 would be stale within a quarter and would silently drag down the module's score. Refresh quarterly per the freshness cadence in `01_Layered_Base_System_v1.1.md` §3.1.

**Verification reminder:** treat every name below as a starting point for current research, not as decision-grade evidence. Capabilities, ownership, pricing, and market position shift constantly. Always run a fresh Module 7 trend scan before using these names in published content or investment / procurement decisions.

## Namespace Map (2026-05 snapshot)

| Namespace | Watch For | Representative Trendsetters (2026-05) |
| --- | --- | --- |
| AI Compute | Inference economics, custom silicon, GPU cloud capacity, energy-constrained compute | NVIDIA, CoreWeave, Crusoe, Google TPUs, AWS Trainium, Microsoft Maia, Cerebras |
| Inference and Serving | Latency, routing, batch efficiency, open-model serving, provider abstraction | Together AI, Fireworks AI, Baseten, Modal, vLLM, SGLang |
| Frontier Models | Capability frontier, enterprise APIs, multimodal models, reasoning models, model safety | OpenAI, Anthropic, Google DeepMind, Meta, Mistral, Cohere, DeepSeek |
| Agent Protocols | Tool connectivity, inter-agent communication, standardization, trust boundaries | Anthropic MCP, Google A2A, OpenAI tool ecosystem, Microsoft agent ecosystem |
| Agent Engineering | Workflow orchestration, agent runtime, state, memory, evaluation, supervision | LangGraph, LlamaIndex, CrewAI, OpenAI Agents SDK, Pydantic AI |
| Data and AI Platforms | Enterprise data control, model building, governed RAG, lakehouse AI, data quality | Databricks, Snowflake, Hugging Face, Scale AI |
| AI Coding and SDLC | Developer workflow automation, coding agents, code review, repo context, secure SDLC | Anysphere/Cursor, GitHub Copilot, Claude Code, Windsurf, Sourcegraph |
| Vertical AI SaaS | Domain workflows, proprietary data, compliance context, outcome ownership | Harvey, Abridge, OpenEvidence, Clay, Veeva, ServiceTitan, Procore |
| AI FinOps | Cost attribution, budget controls, model routing economics, outcome-based pricing | CloudZero, Apptio, Databricks cost controls, provider usage telemetry |
| Governance Runtime | Runtime policy, auditability, guardrails, compliance evidence, human approval | Microsoft Purview, NIST AI RMF ecosystem, EU AI Act tooling, Lakera, Guardrails AI, Llama Guard |

## Refresh notes

This snapshot was extracted from Module 7 v1.1's inline table during the 2026-05-21 hygiene pass. The previous pattern (inline vendor list in the module) was flagged in the Claude cross-module audit as a rot-risk: the names date fast but the module wasn't versioned often enough to keep them current.

Future refreshes:

- Quarterly review against current Trend Scan results.
- Move stale namespaces or sub-categories into an "Archive" section rather than deleting silently — the trajectory of who-was-relevant-when is itself useful signal.
- Add new namespaces when AI engineering surface area expands (likely candidates over the next year: AI infrastructure orchestration, AI security tooling, multi-modal-specific platforms).
