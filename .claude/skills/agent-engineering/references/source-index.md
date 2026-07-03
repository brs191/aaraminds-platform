# Source index

Primary sources behind this skill, with last-verified dates. Agent-design/eval/governance guidance and
model/parameter IDs move quarterly — re-verify at the cadence below and flag drift `[VERIFY]`.

Last verified: **2026-06-18.** Next review: **2026-09 (quarterly).**

## Agent design
- OpenAI — *A Practical Guide to Building Agents* (model · tools · instructions; earn the agent; guardrails) — https://cdn.openai.com/business-guides-and-resources/a-practical-guide-to-building-agents.pdf
- Anthropic — *Building Effective Agents* (workflows vs agents; start simple; 5 patterns) — https://www.anthropic.com/engineering/building-effective-agents
- Anthropic — *Claude Code subagents* (runnable agent frontmatter) — https://code.claude.com/docs/en/sub-agents

## Evaluation
- Anthropic — *Demystifying Evals for AI Agents* (transcript vs outcome; grader taxonomy; capability vs regression; pass@k/pass^k) — https://www.anthropic.com/engineering/demystifying-evals-for-ai-agents
- LangChain — *LangSmith Evaluations* + *Trajectory evals* (datasets→evaluators→experiments; strict/unordered/subset/superset) — https://docs.langchain.com/langsmith/trajectory-evals
- DeepEval / Confident AI — *LLM Agent Evaluation* (task completion, tool correctness, G-Eval) — https://www.confident-ai.com/blog/llm-agent-evaluation-complete-guide
- Ragas — *agentic/tool metrics* (topic adherence, tool-call accuracy/F1, goal accuracy) — https://docs.ragas.io/en/stable/concepts/metrics/available_metrics/agents/

## Governance / security
- OWASP — *Top 10 for Agentic Applications / Agentic Security Initiative* (ASI01–10) — https://genai.owasp.org/
- OWASP — *Top 10 for LLM Applications 2025* (LLM01 Prompt Injection, LLM06 Excessive Agency) — https://genai.owasp.org/llm-top-10/
- CSA — *MAESTRO* threat-modeling framework — https://cloudsecurityalliance.org/
- Simon Willison — *the lethal trifecta* — https://simonwillison.net/2025/Jun/16/the-lethal-trifecta/
- NIST — *AI Risk Management Framework* + GenAI Profile (AI 600-1) — https://www.nist.gov/itl/ai-risk-management-framework
- OpenTelemetry — *GenAI semantic conventions* (invoke_agent / execute_tool spans) — https://opentelemetry.io/docs/specs/semconv/gen-ai/

## Packaging / interop
- agents.md — *AGENTS.md* open standard — https://agents.md/
- A2A — *Agent Card* (`/.well-known/agent-card.json`) — https://a2a-protocol.org/
- Model cards (Mitchell et al.) → Anthropic *system cards* — lineage for AGENT_SPEC.md.

> Model/parameter IDs (e.g. Codex `.toml` fields, A2A `protocolVersion`) are the most volatile — verify
> against the live spec at use time; treat exact values as `[VERIFY]`.
