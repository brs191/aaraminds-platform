# TokenOptimizer Architecture — Mermaid Version

## Architecture summary

TokenOptimizer is a multi-agent system that analyzes LLM traces from connected
systems and produces ranked token-reduction recommendations across six waste
domains (prompts, context, memory, RAG, tool outputs, inter-agent messages).
The orchestrator fans out to six specialist agents in parallel, aggregates and
ranks findings by impact/risk ratio, and emits draft GitHub PRs that owners
review and merge. Cross-cutting controls (Entra ID, Key Vault, RBAC,
observability, audit) span the full stack. The system never modifies production
code directly — recommendations only.

## Mermaid diagram (initial)

```mermaid
flowchart LR
    classDef channel fill:#fef3c7,stroke:#d97706,color:#92400e
    classDef copilot fill:#e0e7ff,stroke:#4f46e5,color:#312e81
    classDef api fill:#dbeafe,stroke:#2563eb,color:#1e3a8a
    classDef orch fill:#bfdbfe,stroke:#1d4ed8,color:#1e3a8a
    classDef agent fill:#fecaca,stroke:#dc2626,color:#7f1d1d
    classDef mcp fill:#d1fae5,stroke:#059669,color:#064e3b
    classDef ent fill:#e9d5ff,stroke:#7c3aed,color:#5b21b6
    classDef data fill:#fde68a,stroke:#d97706,color:#78350f
    classDef gov fill:#f3f4f6,stroke:#6b7280,color:#374151
    classDef obs fill:#fed7aa,stroke:#ea580c,color:#7c2d12

    subgraph CH[1. USER CHANNELS]
        direction TB
        T[MS Teams]
        W[Web Client]
    end

    subgraph CP[2. COPILOT / EXPERIENCE]
        direction TB
        MC[MS Copilot]
        RV[Owner Review UI]
    end

    subgraph AP[3. API and ACCESS]
        direction TB
        APIM[Azure APIM]
        EID[Entra ID]
        RB[RBAC]
        KV[Key Vault]
    end

    subgraph OR[4. LANGGRAPH ORCHESTRATION]
        direction TB
        LGO[LangGraph Orchestrator]
        AST[Agent State Store]
        HRQ[Human Review Queue]
    end

    subgraph AG[5. AI AGENT LAYER]
        direction TB
        PLN[Planner Agent - Orchestrator]
        RVW[Reviewer Agent]
        subgraph SPEC[Domain Specialists]
            direction LR
            S1[Prompt Analyzer]
            S2[Context Pruner]
            S3[Memory Compressor]
            S4[RAG Optimizer]
            S5[Tool Output Truncator]
            S6[Inter-Agent Compressor]
        end
    end

    subgraph MC2[6. MCP SERVER and TOOL]
        direction TB
        MGW[MCP Gateway]
        LSM[LangSmith MCP]
        LFM[Langfuse MCP]
        GHM[GitHub MCP]
        SLM[Slack MCP]
        OAS[OpenAPI Connector]
    end

    subgraph ES[7. ENTERPRISE SYSTEMS]
        direction TB
        LSP[LangSmith Platform]
        LFP[Langfuse Platform]
        GHE[GitHub Enterprise]
        SLE[Slack Enterprise]
    end

    subgraph DK[8. DATA and KNOWLEDGE]
        direction TB
        AOI[Azure OpenAI Gateway]
        VDB[pgvector]
        DBS[Postgres]
        RDS[Redis Cache]
    end

    subgraph OBS[9. OBSERVABILITY, AUDIT, GOVERNANCE]
        direction TB
        AZM[Azure Monitor]
        OTL[OpenTelemetry]
        AUD[Audit Logs - WORM]
    end

    T --> MC
    W --> MC
    MC --> APIM
    RV --> APIM
    APIM --> LGO
    EID -.-> APIM
    RB -.-> APIM
    KV -.-> APIM
    LGO --> PLN
    LGO --> AST
    LGO --> HRQ
    PLN --> SPEC
    PLN --> RVW
    SPEC --> MGW
    RVW --> MGW
    MGW --> LSM
    MGW --> LFM
    MGW --> GHM
    MGW --> SLM
    MGW --> OAS
    LSM --> LSP
    LFM --> LFP
    GHM --> GHE
    SLM --> SLE
    SPEC --> AOI
    PLN --> AOI
    AOI --> VDB
    LGO --> DBS
    LGO --> RDS
    OBS -.-> AG
    OBS -.-> OR
    OBS -.-> MC2

    class T,W channel
    class MC,RV copilot
    class APIM,EID,RB,KV api
    class LGO,AST,HRQ orch
    class PLN,RVW,S1,S2,S3,S4,S5,S6,SPEC agent
    class MGW,LSM,LFM,GHM,SLM,OAS mcp
    class LSP,LFP,GHE,SLE ent
    class AOI,VDB,DBS,RDS data
    class AZM,OTL,AUD obs
```

## Critical review of the diagram

What works:
- Nine layers are visible as labeled subgraphs with consistent ordering
- Six specialists are visually grouped without sprawling the agent layer
- Cross-cutting concerns (Entra ID, RBAC, Key Vault) use dotted lines to
  signal they wrap rather than flow through
- Observability connects to multiple layers via dotted edges, signaling
  cross-cutting

What's weak:
- The flow direction is left-to-right per the design rules, but the dense
  cross-cutting edges create visual noise — Entra ID/RBAC/Key Vault should
  ideally render as a vertical band, but Mermaid's auto-layout fights this
- 34 named components — over the 30-component soft limit. Specifically,
  the four "Enterprise Systems" boxes (LangSmith Platform, Langfuse Platform,
  GitHub Enterprise, Slack Enterprise) are end-states that don't change the
  architecture story; they could be implied by the MCP layer
- The Planner/Reviewer/Specialists relationship is shown as edges but the
  fan-out/fan-in pattern doesn't render visually distinctively — looks like
  a tree, not a fork-join
- Mermaid will render this with auto-routed edges that may cross; layout
  quality depends on the renderer

## Improved Mermaid diagram

```mermaid
flowchart LR
    classDef channel fill:#fef3c7,stroke:#d97706,color:#92400e
    classDef copilot fill:#e0e7ff,stroke:#4f46e5,color:#312e81
    classDef api fill:#dbeafe,stroke:#2563eb,color:#1e3a8a
    classDef orch fill:#bfdbfe,stroke:#1d4ed8,color:#1e3a8a
    classDef agent fill:#fecaca,stroke:#dc2626,color:#7f1d1d
    classDef mcp fill:#d1fae5,stroke:#059669,color:#064e3b
    classDef data fill:#fde68a,stroke:#d97706,color:#78350f
    classDef gov fill:#f3f4f6,stroke:#6b7280,color:#374151
    classDef obs fill:#fed7aa,stroke:#ea580c,color:#7c2d12

    subgraph CH[1. USER CHANNELS]
        T[MS Teams]
        W[Web Client]
    end

    subgraph CP[2. COPILOT EXPERIENCE]
        MC[MS Copilot]
        RV[Owner Review UI]
    end

    subgraph AP[3. API and ACCESS]
        APIM[Azure APIM + Entra ID + RBAC + Key Vault]
    end

    subgraph OR[4. LANGGRAPH ORCHESTRATION]
        LGO[LangGraph Orchestrator + State Store + Human Review Queue]
    end

    subgraph AG[5. AI AGENT LAYER]
        PLN[Planner Agent]
        subgraph SPEC[Six Domain Specialists]
            direction LR
            S1[Prompt Analyzer]
            S2[Context Pruner]
            S3[Memory Compressor]
            S4[RAG Optimizer]
            S5[Tool Output Truncator]
            S6[Inter-Agent Compressor]
        end
        RVW[Reviewer Agent]
    end

    subgraph MC2[6. MCP GATEWAY and SERVERS]
        MGW[MCP Gateway - LangSmith, Langfuse, GitHub, Slack, OpenAPI]
    end

    subgraph DK[7. DATA and KNOWLEDGE]
        AOI[Azure OpenAI Gateway]
        VDB[pgvector + Postgres + Redis]
    end

    subgraph OBS[8. OBSERVABILITY, AUDIT, GOVERNANCE]
        OBSALL[Azure Monitor + OpenTelemetry + Audit Logs WORM]
    end

    T --> MC
    W --> MC
    MC --> APIM
    RV --> APIM
    APIM --> LGO
    LGO --> PLN
    PLN --> SPEC
    SPEC --> RVW
    RVW --> MGW
    PLN --> AOI
    SPEC --> AOI
    AOI --> VDB
    OBSALL -.-> AG
    OBSALL -.-> OR
    OBSALL -.-> MC2

    class T,W channel
    class MC,RV copilot
    class APIM api
    class LGO orch
    class PLN,RVW,S1,S2,S3,S4,S5,S6,SPEC agent
    class MGW mcp
    class AOI,VDB data
    class OBSALL obs
```

Changes in the improved version: collapsed end-state Enterprise Systems
boxes into the MCP layer (they're implied), grouped API+identity+secrets
into a single APIM box (the access layer is one logical concern, not
four), made the fan-out pattern (Planner → Specialists → Reviewer)
visible as a left-center-right flow within the agent layer, dropped from
34 to 18 named components.

## Draw.io XML

See companion file `tokenoptimizer_drawio.xml` — import into draw.io via
File → Import → Device, then adjust layout and styling as needed.
