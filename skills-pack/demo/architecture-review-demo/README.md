# Architecture Review Demo

Exercises the architecture-review tools in the v9.0 MCP server across three deliberately distinct architectures and writes per-architecture, per-tool JSON outputs.

The demo is **authentically driven by the MCP server**: the Python runner spawns the Go server over stdio, completes the MCP handshake, and calls five tools per architecture. The runner shapes per-tool inputs from a single master architecture description and forwards them; **it does not invent results**.

## Architectures covered

| Architecture | What it stresses | File |
|---|---|---|
| `ecommerce` | Customer-facing order flow with PCI-DSS card handling; saga / outbox / cache | [input/ecommerce.json](input/ecommerce.json) |
| `financial-services` | Online retail banking with event-sourced ledger, fraud scoring, SOX audit | [input/financial-services.json](input/financial-services.json) |
| `healthcare` | HIPAA-compliant patient records with CQRS read model and audit trail | [input/healthcare.json](input/healthcare.json) |

Because the rule-based Go server is deterministic, each architecture produces stable, byte-reproducible outputs that vary meaningfully — different boundary scores, different risk shapes, different Azure mappings — across the three.

## Tools exercised

For each architecture the runner calls five MCP tools and writes one JSON file per call:

| Output filename | Tool name | What it produces |
|---|---|---|
| `boundary.json` | `generate_service_boundary_canvas` | Per-service boundary assessment, risks, recommended changes, overall score |
| `apicontract.json` | `generate_api_contract` | OpenAPI-shaped contract sketch for services that expose an API |
| `archrisks.json` | `detect_architecture_risks` | Risks with severity, likelihood, mitigation |
| `azuremap.json` | `map_patterns_to_azure_services` | Pattern→Azure service mapping with rationale |
| `obsplan.json` | `generate_observability_plan` | SLOs, dashboards, alerts, coverage gaps |

Each tool's input schema is defined by the [implemented contracts](../../examples/microservices-system-design-mcp-server/contracts/architecture-tools/implemented/) in the example server.

## Prerequisites

- Python 3.8+ (stdlib only — no MCP SDK or pip install required).
- A built `mcp-server` binary from the v9.0 example.

```bash
cd ../../examples/microservices-system-design-mcp-server
go build -buildvcs=false -o ./mcp-server ./cmd/server
```

If your local Go is older than 1.25 (the example's `go.mod` requires `go 1.25.5`), build inside Docker:

```bash
docker run --rm --network=host \
  -v "$(pwd):/src" -w /src \
  golang:1.26-alpine sh -c \
  "go mod download && CGO_ENABLED=0 go build -buildvcs=false -o ./mcp-server ./cmd/server"
```

## Run it

```bash
# From this directory:
export MCP_SERVER_BIN=../../examples/microservices-system-design-mcp-server/mcp-server
make demo
```

Output lands in `out/<architecture>/<tool>.json`. The server's structured logs are forwarded to stderr (prefixed `[server]`) so you can see each tool call as it happens.

## Validate

```bash
make validate
```

This canonicalises generated JSON (sorts keys, normalises indentation) and compares to `golden/`. Mismatches are reported per file. A pass means: for the same input, the rule-based MCP server produced the same output as it did when the goldens were captured. Drift is a signal that something — input, server logic, or contract shape — changed.

## When goldens drift

If you intentionally change a tool's behaviour or alter an input, regenerate goldens:

```bash
make demo refresh
```

Then commit the updated `golden/` and document why in the same change.

## How the runner works

[`demo_runner.py`](demo_runner.py) is a stdlib-only MCP client over stdio. It:

1. Spawns the MCP server as a subprocess (default transport: stdio).
2. Performs the JSON-RPC `initialize` handshake and sends `notifications/initialized`.
3. For each architecture, shapes a per-tool input from the master architecture description and issues a `tools/call` request.
4. Re-emits the server's text-content response as a pretty-printed JSON file under `out/`.

The runner is deliberately small — about 250 lines — so a reader can trace the entire path from input fixture to MCP wire to output file without leaving the file. There is no LLM in the loop and no business logic in the runner.

## What this demo is and is not

It **is** evidence that the v9.0 example MCP server, fronted by a real client over the MCP protocol, produces structurally consistent, content-distinct outputs across realistic Azure microservices designs.

It **is not** a production architecture-review service. The Go tools apply deterministic rules (capability cohesion, dependency depth, ownership clarity, pattern recognition). They are useful as design-time guardrails and as MCP-tool exemplars; they are not a substitute for a human architect.

## Related

- Example server: [../../examples/microservices-system-design-mcp-server](../../examples/microservices-system-design-mcp-server)
- Tool contracts: [../../examples/microservices-system-design-mcp-server/contracts/architecture-tools/implemented](../../examples/microservices-system-design-mcp-server/contracts/architecture-tools/implemented)
- Roadmap entry: [../../ROADMAP.md](../../ROADMAP.md) (Gap 4)
