# AaraMinds Agent Platform — Architecture

Layered container view of the platform as it stands (2026-07-05). Grounded in the shipped code under `platform/`, the schemas under `schemas/`, and the governance config under `governance/`. See `DOCUMENT-MAP.md` for document authority.

Legend: **built** = AaraMinds-built and tested (blue); **adopt** = adopted, external, or a later phase (gray, dashed).

```mermaid
flowchart TB
  classDef built fill:#E6F1FB,stroke:#185FA5,stroke-width:1.5px,color:#042C53;
  classDef adopt fill:#F1EFE8,stroke:#888780,stroke-width:1.3px,color:#2C2C2A,stroke-dasharray:5 3;
  classDef contract fill:#CECBF6,stroke:#534AB7,stroke-width:1.3px,color:#26215C;
  classDef gate fill:#FAEEDA,stroke:#BA7517,stroke-width:1.5px,color:#854F0B;

  subgraph EXP[Experience and interfaces]
    direction LR
    CLI[aapctl CLI]:::built
    UI[React console - Phase 2]:::adopt
    TEAMS[Microsoft Teams]:::adopt
    API[Platform API]:::adopt
  end

  subgraph FAC[Agent Factory - design-time pipeline - shipped]
    direction LR
    F1[Intake]:::built --> F2[Classify]:::built --> F3[Scaffold]:::built --> F4[Readiness]:::built --> F5[Export]:::built
  end

  subgraph GOV[Governance and evidence]
    direction LR
    G1[Rubric and thresholds]:::built
    G2[Approval boundaries]:::built
    G3[Blocked actions and kill switch]:::built
    G4[Evidence report]:::built
  end

  subgraph SUB[Runtime proof substrate - Go harness - shipped]
    direction LR
    S1[Manifest enforcement]:::built
    S2[Tool-contract gate]:::built
    S3[Scoped memory]:::built
    S4[Audit chain]:::built
    S5[Proof gates]:::built
  end

  SCHEMAS[13 JSON schemas - machine contracts of record]:::contract
  GATE{{activation gate - status active requires verdict = pass}}:::gate

  subgraph RUN[Runtime orchestration - adopt]
    direction LR
    R1[MCP gateway]:::adopt
    R2[Primary runtime - Claude Agent SDK]:::adopt
    R3[Alt runtimes - LangGraph, ADK, AgentCore]:::adopt
  end

  subgraph FND[Platform foundation]
    direction LR
    N1[Identity - Entra Agent ID]:::adopt
    N2[Observability - OTel GenAI]:::adopt
    N3[Data and SoR - git now, Postgres Phase 2]:::adopt
    N4[CI/CD - readiness gate, govulncheck]:::built
  end

  EXP --> FAC --> GOV --> SUB --> SCHEMAS --> GATE --> RUN --> FND
  SUB -. gate results as evidence .-> F4
  SCHEMAS -. validates .-> FAC
```

## How to read it

The platform is **built top-down and depends bottom-up**. The upper region is design-time and AaraMinds-built; the lower region is the adopted runtime and infrastructure. The two are separated by a single control — the **activation gate**.

**Experience.** Today the only shipped interface is the `aapctl` CLI. Console, Teams, and API are later phases (BRD §10.2).

**Agent Factory (the differentiator).** A deterministic, no-LLM pipeline: `intake → classify → scaffold → readiness → export`, one `aapctl` subcommand each. It turns an agent idea into a governed, evidence-scored artifact folder. Determinism is the point — a reproducible factory is what makes the readiness verdict trustworthy.

**Governance and evidence.** The `readiness` stage reads the versioned rubric (`governance/readiness-rubric.yaml`, 9 weighted areas), approval boundaries, the blocked-actions deny-list, and produces a per-check, evidence-cited report. No score is self-attested; every point traces to a file, gate result, or audit event.

**Runtime proof substrate (the Go harness).** The shipped enforcement engine: manifest control (no off-manifest tool calls), MCP tool-contract gating, engagement-scoped memory with citation enforcement, a tamper-evident audit chain, and proof gates for prompt-injection escalation and tool/memory denial. The readiness engine runs this harness and consumes its gate results as evidence (the dotted arrow).

**Machine contracts of record.** Thirteen JSON schemas are the shared source of truth; every generated artifact validates against one, and the factory validates against them (the second dotted arrow).

**Activation gate.** A manifest cannot move to `status: active` unless a current readiness report scored under the live rubric returns verdict `pass`. This is the one hard control separating design-time from production — enforced in code (`ActivationGate`), not policy.

**Runtime orchestration (adopt, not build).** The platform orchestrates existing runtimes and adopts an MCP gateway; it never rebuilds them. Claude Agent SDK is the primary runtime target ([VERIFY] per PRD §6); LangGraph, Google ADK, and Bedrock AgentCore are portable alternatives.

**Foundation.** Identity via the Entra Agent ID pattern (OAuth2 / workload identity federation); observability via OpenTelemetry GenAI semantic conventions into Grafana/Prometheus; the system of record is git today, PostgreSQL in Phase 2; CI/CD is shipped, running the readiness gate and `govulncheck` on every change. Deployment is Azure-first: AKS / Container Apps, GitHub Actions with OIDC, Terraform AzureRM, Key Vault via managed identity.

**Standards.** MCP (pinned `2025-11-25`), OTel GenAI, OWASP ASI 2026, NIST AI RMF, ISO/IEC 42001, and EU AI Act obligations; A2A is a future interoperability layer.

## What is real vs planned

Everything marked **built** is code with tests in `platform/` and is exercised by CI. Everything marked **adopt** is an integration point that is either a later phase or an external component the platform composes rather than owns. The reference agent (`agents/aara-business-analyst`) currently scores a readiness **pass** through the full pipeline.
