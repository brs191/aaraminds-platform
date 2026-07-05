# Reference Architecture (MVP → Phase 2)

Implements BRD v2.1 §14 with the existing AAP harness as the foundation. Build the differentiating layers; adopt the commodity ones.

## Component view

```mermaid
flowchart TB
  subgraph Experience [Experience — Phase 2+]
    CLI[aapctl CLI - MVP]
    UI[React console - Phase 2]
  end
  subgraph Factory [Agent Factory — BUILD, MVP]
    INTAKE[Intake and Classifier]
    GEN[Artifact Generators<br/>blueprint, contracts, identity,<br/>data/evidence, eval plan, compliance]
    READY[Readiness Engine<br/>rubric + verdict + state enforcement]
  end
  subgraph Existing [AAP Harness — EXISTS, platform/]
    LOADER[loader.go - YAML/JSON + schema validation]
    ENGINE[engine.go - manifest and tool boundary enforcement]
    PROOF[proof.go - release-gate proofs]
    OTEL[otel.go - OTel GenAI projection]
    MEM[memory.go - scoped memory + citation]
  end
  subgraph Adopted [Adopt, not build]
    IDP[Entra ID / Agent ID - Phase 1 spike]
    GW[MCP Gateway - Phase 2 adoption]
    RT[Runtimes: Agent Framework / LangGraph - Phase 4]
    OBS[Grafana + Prometheus + OTel Collector]
  end
  STORE[(Git repo = system of record - MVP<br/>PostgreSQL catalog - Phase 2)]

  CLI --> INTAKE --> GEN --> READY
  GEN --> LOADER
  READY --> PROOF
  READY --> STORE
  ENGINE --> OTEL --> OBS
  ENGINE -.enforces.-> GW
  READY -.gates status active.-> ENGINE
  IDP -.identity per agent_id.-> ENGINE
  RT -.Phase 4 integration.-> ENGINE
```

## Decisions

| Area | Decision | Rationale |
|---|---|---|
| System of record (MVP) | Git repo, folder per agent | Artifacts are files; validation in CI; no DB before the model stabilizes. PostgreSQL catalog in Phase 2 when the web console needs queries. |
| Control plane language | Go (extend `platform/`) | Harness is Go; stack standard; single binary `aapctl` distribution. |
| Validation | JSON Schema (existing `schemas/`) + Markdown section validator | Already proven by loader/engine; extend, don't replace. |
| Readiness | New `internal/readiness` package consuming `proof.go` gate results | Design-time rubric composes runtime proof gates (see rubric doc). |
| Identity | Entra Agent ID pattern; managed identity per `agent_id`; local dev fallback without shared prod credentials | Open item in runtime-verification-notes — Phase 1 spike. |
| Gateway | Adopt in Phase 2; selection criteria per OQ-004 | Do not build. Harness tool-boundary enforcement remains the design-time proof. |
| Observability | OTel GenAI semconv (`invoke_agent`, `execute_tool` spans; `aap.*` for governance) → Collector → Grafana/Prometheus | Matches existing `otel.go`; validate collector compatibility before production (open item). |
| Deployment | `aapctl` binary + GitHub Actions (OIDC) CI validating agent folders; containerized services only when the Phase 2 console arrives (AKS/Container Apps, Terraform AzureRM, Key Vault via managed identity) | Don't stand up infrastructure the MVP doesn't need. |

## What deliberately does not exist yet

No database, no web UI, no gateway integration, no runtime execution, no A2A. Each has a named phase and an open verification item; building them early is the R-001 (platform too broad) failure mode.
