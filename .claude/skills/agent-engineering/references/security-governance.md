# Security, governance & deployment readiness

The security review and the deployment-readiness gate. Routes deep controls to
`azure-microservices-security` and `soc2-iso27001-controls-mapping`; this is the agent-specific layer.
Threat-model with MAESTRO; score against the OWASP Agentic Top 10; gate deployment with the checklist.

## OWASP Top 10 for Agentic Applications (Dec 2025) — the review checklist

Score the agent against each; cite the ID in findings.

| ID | Threat | Check for |
|---|---|---|
| **ASI01** | Agent goal hijack | Indirect/hidden prompt injection via tool/RAG/email content redirecting the objective |
| **ASI02** | Tool misuse | Legitimate tools bent to destructive outputs; missing output validation |
| **ASI03** | Identity & privilege abuse | Over-broad/standing credentials; agent acts beyond intended scope |
| **ASI04** | Agentic supply chain | Poisoned MCP servers / A2A components / dependencies |
| **ASI05** | Unexpected code execution | NL paths to RCE; unsandboxed code/shell tools |
| **ASI06** | Memory & context poisoning | Tainted long-term memory reshaping later behavior |
| **ASI07** | Insecure inter-agent comms | Spoofed agent-to-agent messages |
| **ASI08** | Cascading failures | False signals propagating through multi-agent pipelines (need rate limits) |
| **ASI09** | Human-agent trust exploitation | Confident polished output rubber-stamped by humans (HITL theater) |
| **ASI10** | Rogue agents | Misalignment, concealment, self-directed action |

Grounded by OWASP LLM-2025: **LLM01 Prompt Injection** and **LLM06 Excessive Agency** are the two that
matter most for agents.

## Prompt injection is architectural — reject content filtering as the mitigation

LLMs can't separate trusted instructions from untrusted data in one token stream, so filters/classifiers
**raise attacker cost but don't close the hole** (adaptive attacks bypass published defenses >90%).
Require architectural controls, not "we filter malicious prompts":

- **Lethal-trifecta test (Simon Willison):** exfiltration is possible when an agent combines (a) access
  to private data, (b) exposure to untrusted content, (c) ability to communicate externally. **Flag any
  agent holding all three and remove or gate one leg.**
- **Dual-LLM / privileged-vs-quarantined:** the privileged LLM never sees untrusted content; the
  quarantined LLM processes untrusted content but can't call tools.
- **Agents Rule of Two:** don't let one trust context hold all three trifecta capabilities at once.
- Treat all tool outputs and RAG content as untrusted; gate high-impact actions behind deterministic
  policy (allow-lists, HITL), not model judgment.

## Guardrails — layered, and at the side effect

- **Input guardrails** validate user input before the expensive agent runs (blocking mode = no token
  spend / side effects on violation).
- **Output guardrails** validate the final output (PII, brand, policy).
- **Tool-level guardrails** run on the side-effecting call — skip/replace/reject before execution,
  redact/reject after. **Put validation next to the tool that creates the side effect, not only at the
  agent boundary.**
- **Stopping conditions:** max turns/retries; escalate on repeated failure to grasp intent.
- Combine LLM-based + rules-based (regex/blocklist/length) + a moderation pass; no single layer suffices.

## Identity, least privilege, audit

- **Agent as a first-class non-human identity** with **just-in-time, scoped tokens** (one resource,
  one action, one window) — not standing broad grants. (Caution: loosely-scoped agent roles have been
  abused for takeover.)
- **Audit logging** with per-action attribution (every tool call, model interaction, policy eval).
- For EU-exposed/regulated use, retain logs **≥ 6 months** (EU AI Act Art. 26 deployer duties; flag
  the moving high-risk timeline `[VERIFY]`).

## Observability / tracing

Trace **LLM generations, tool calls, handoffs, guardrail/tripwire events, custom events** — on
OpenTelemetry **GenAI semantic conventions** (`gen_ai.operation.name = invoke_agent`, `execute_tool`
spans, `gen_ai`/`mcp` attribute namespaces). Read traces of **successful** runs in dev, not only
failures — that's where you catch the right answer for the wrong reason. Tracing is the substrate for
debugging, audit, and drift alerting.

## Human-in-the-loop — when the agent must stop and ask

Pause for human approval on: irreversible/destructive actions, financial movement, actions outside
scoped permissions, low-confidence outputs, any action satisfying the lethal trifecta, and tool-guardrail
tripwires. The approval UI must surface the **evidence and the specific action**, not just a polished
summary (counter ASI09).

## Deployment-readiness gate (most production incidents are operability, not capability)

The eval gate is the *first* gate, not the only one. Require all eight before "production-ready":

1. **Eval gate** green (functional/behavioral/safety) — plus a separate operability review.
2. **Versioning** — prompt/config versioned, change history, env tags (dev/staging/prod).
3. **Rollback runbook** written before deploy — an unrelated engineer can restore the prior version cold.
4. **Kill switch** — documented, accessible, **tested**.
5. **Canary / progressive rollout** — 1% canary via feature flag, not 100%.
6. **Monitoring with the right thresholds** — error rates, token/cost spikes, **policy violations and
   eval/behavioral drift**, routed to the owning team.
7. **Audit trail** live (OTel GenAI, per-action).
8. **Guardrails + scoped identity + HITL** live in production, not just staging.

## Governance frameworks

- **NIST AI RMF** — GOVERN (policies/oversight) · MAP (context/impacts) · MEASURE (test/eval, drift,
  adversarial) · MANAGE (treat risks). Use the GenAI Profile (NIST AI 600-1) and the CSA Agentic
  Profile for agent-specific actions.
- **EU AI Act** — high-risk deployer duties (human oversight, logging ≥6 months, monitoring);
  multi-agent chains in scope (Recitals 99–100). Timelines are moving — mark `[VERIFY]`.
