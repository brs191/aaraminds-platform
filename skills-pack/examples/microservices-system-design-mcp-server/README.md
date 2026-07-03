# Microservices System Design MCP Server

A Go-language MCP server that exposes architecture-intelligence tools for microservices system design on Azure. Thirteen tools total — three classic design tools plus ten focused architecture-review tools — all rule-based, deterministic, and byte-reproducible given the same input.

## Tools exposed

### Design layer

| Tool | Purpose |
|---|---|
| `review_microservice_design` | Review a microservices design for Azure cloud readiness |
| `recommend_microservice_patterns` | Recommend patterns based on a problem statement |
| `score_well_architected_readiness` | Score a design against Azure Well-Architected pillars |

### Architecture review and generation layer

| Tool | Purpose |
|---|---|
| `generate_service_boundary_canvas` | Per-service boundary assessment, risks, recommendations, score |
| `generate_api_contract` | OpenAPI-shaped API contract sketches for services that expose an API |
| `detect_architecture_risks` | Named risks (severity, likelihood, mitigation), risk-posture score |
| `map_patterns_to_azure_services` | Pattern → Azure service mapping with rationale |
| `generate_observability_plan` | SLOs, dashboards, alerts, coverage gaps |
| `generate_architecture_decision_record` | Michael Nygard-shape ADR with quality score and warnings |
| `generate_deployment_topology` | Per-service placements, data placements, network boundaries |
| `generate_event_contract` | CloudEvents v1.0 contract with transport binding |
| `generate_resilience_plan` | Timeout / retry / breaker config, bulkheads, fallbacks, signals |
| `generate_diagram_assets` | Mermaid + PlantUML + draw.io prompt for the chosen diagram type |

Every tool has a contract under `contracts/architecture-tools/implemented/`. Inputs are validated server-side; outputs are structured JSON.

## Quick start

```bash
# Initialize modules
go mod tidy

# Run tests (includes -race in CI)
make test

# Build the binary
make build

# Run in stdio mode (for local MCP clients like Claude Code)
make run-stdio

# Run in streamable HTTP mode on port 8080
make run-http
```

If your local Go is older than the `go.mod` version, build inside Docker:

```bash
docker run --rm --network=host \
  -v "$(pwd):/src" -w /src \
  golang:1.26 sh -c \
  "go mod tidy && CGO_ENABLED=0 go build -buildvcs=false -o ./mcp-server ./cmd/server"
```

For full step-by-step verification see [../../VERIFICATION_CHECKLIST.md](../../VERIFICATION_CHECKLIST.md) at the pack root.

## Project structure

```text
microservices-system-design-mcp-server/
├── cmd/server/main.go                     # Entry point, transport selection
├── internal/
│   ├── mcpserver/server.go                # MCP server construction, tool wiring
│   ├── services/                          # Rule logic (one package per tool)
│   │   ├── adr/                           # generate_architecture_decision_record
│   │   ├── apicontract/                   # generate_api_contract
│   │   ├── archrisks/                     # detect_architecture_risks
│   │   ├── azuremap/                      # map_patterns_to_azure_services
│   │   ├── boundary/                      # generate_service_boundary_canvas
│   │   ├── design/                        # review / recommend / score
│   │   ├── diagrams/                      # generate_diagram_assets
│   │   ├── eventcontract/                 # generate_event_contract
│   │   ├── obsplan/                       # generate_observability_plan
│   │   ├── resilience/                    # generate_resilience_plan
│   │   └── topology/                      # generate_deployment_topology
│   └── tools/                             # MCP wiring (one package per tool, mirrors services/)
├── contracts/architecture-tools/
│   └── implemented/                       # 10 architecture-tool contracts
├── testdata/                              # Input fixtures + golden outputs per tool
├── go.mod
├── Makefile
├── Dockerfile                             # Multi-stage, distroless, non-root
└── .github/workflows/ci.yml               # Lint, vet, race, build
```

The layering follows the discipline named in [skills/mcp/06-mcp-go-project-structure.md](../../skills/mcp/06-mcp-go-project-structure.md). MCP-protocol concerns live in `internal/mcpserver/` and `internal/tools/`. Business logic lives in `internal/services/`. The layers are enforced by package boundaries: tool packages may import service packages, but service packages must not import tool packages.

## Configuration

The server reads two environment variables:

| Variable | Default | Values |
|---|---|---|
| `MCP_TRANSPORT` | `stdio` | `stdio`, `streamablehttp`, `sse` (deprecated) |
| `PORT` | `8080` | TCP port for HTTP/SSE modes |

For Azure Container Apps deployment, set `MCP_TRANSPORT=streamablehttp` and let Container Apps ingress terminate TLS. See [skills/mcp/00-ecosystem-facts.md](../../skills/mcp/00-ecosystem-facts.md) for verified Azure hosting guidance.

## Logging

The server uses `log/slog` with JSON output to stderr. Stdio transport requires stderr-only logging because stdout is the MCP protocol wire — see [skills/mcp/01-mcp-go-server-basics.md](../../skills/mcp/01-mcp-go-server-basics.md) for the failure mode this prevents.

In production, structured log events should flow to Application Insights or a SIEM. The current implementation writes to stderr only; production deployment requires log forwarding configuration at the container or orchestrator level.

## Tests

```bash
# All tests
make test

# Race-detected (matches the CI gate; requires cgo)
go test -race -count=1 ./...

# With verbose output
make test-verbose
```

Every service package has a table-driven test file in `service_test.go` next to the implementation. Coverage exercises every named rule with positive and negative cases.

## Contracts

Every tool has a contract file under `contracts/architecture-tools/implemented/`. The contract is the source of truth — implementation must match the contract, not the other way around. See [skills/mcp/02-mcp-go-tool-design.md](../../skills/mcp/02-mcp-go-tool-design.md) for the contract discipline.

## What this server is not

- **Not production-hardened.** The example is starter-grade. Production deployment requires per-tool authorization ([skills/mcp/07-mcp-go-enterprise-security.md](../../skills/mcp/07-mcp-go-enterprise-security.md)), rate limiting, audit log forwarding to a dedicated audit sink, secret management via Key Vault and Managed Identity, CORS configuration for browser clients.
- **Not load-tested.** The services are pure in-memory computation and should scale linearly, but no benchmarks are included.
- **Not LLM-augmented.** The tools are fully deterministic — same input, same output. A future variant could add LLM reasoning to the tools that benefit from unstructured input; for now, all guidance is rule-based and reproducible.
