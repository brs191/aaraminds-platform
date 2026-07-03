# Skill — MCP-Go Transport Selection

## Purpose

Choose the right MCP transport for a deployment. The MCP spec defines several transports (stdio, streamable HTTP, the deprecated SSE), each with different operational characteristics. Picking incorrectly costs you either latency, debuggability, security, or compatibility — sometimes all four. This skill is the decision framework, the implementation notes per transport, and the migration story when you need to switch.

## The three transports, briefly

| Transport | What it is | Where it shines | Where it fails |
|---|---|---|---|
| **stdio** | The MCP server runs as a subprocess of the client; communication via stdin/stdout newline-delimited JSON-RPC | Local agent tooling (Claude Code, IDE plugins, single-user CLIs) | Multi-client serving; remote deployments |
| **streamable HTTP** | HTTP-based, server is a long-running service; clients connect over the network | Remote / multi-client deployments; cloud services | Local single-user setups (over-engineered) |
| **SSE** (server-sent events) | The legacy HTTP-based transport | Backward compatibility with old clients | New work — **deprecated in spec 2025-11-25** |

The decision is usually: **stdio for local agents, streamable HTTP for services, never SSE for new work.**

## When to pick stdio

- The MCP server is invoked by a local client (Claude Code, an IDE, a CLI workflow).
- One client process consumes one server process.
- Latency budget is sub-millisecond per call.
- The server has no need to serve multiple disconnected clients.
- You don't want to operate a network endpoint.

Stdio is the default in this pack's examples because most MCP servers are agent tooling.

**Critical operational rule for stdio:** stdout *is* the MCP wire. Logging to stdout will corrupt every message frame and the server appears broken. Log to stderr only.

## When to pick streamable HTTP

- The MCP server is a long-running service accessed by multiple clients across the network.
- You need horizontal scaling, load balancing, or geographic distribution.
- The server is part of a microservices estate and benefits from standard HTTP tooling (gateways, observability, deploys).
- The client cannot or should not spawn the server as a subprocess (security, lifecycle, multi-tenancy).

When streamable HTTP is right, treat the server like any HTTP service: TLS, authentication, rate limiting, structured logging, blue-green deploys. The MCP layer sits on top of HTTP norms.

## Why SSE is deprecated

Server-sent events was the original HTTP-based transport. The 2025-11-25 spec marks it deprecated in favour of streamable HTTP, which provides bidirectional streaming on a single endpoint with cleaner semantics. Don't build new servers on SSE; if you maintain one, plan migration.

The mark3labs/mcp-go SDK still includes `server.NewSSEServer(...)` for legacy clients; treat it as backward-compatibility scaffolding, not the default.

## Implementation: stdio

```go
func main() {
    logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
        Level: slog.LevelInfo,
    }))
    slog.SetDefault(logger)

    s := mcpserver.NewServer(logger)

    logger.Info("starting MCP server", slog.String("transport", "stdio"))
    if err := server.ServeStdio(s); err != nil {
        logger.Error("stdio server failed", slog.String("error", err.Error()))
        os.Exit(1)
    }
}
```

- `os.Stderr` for the slog handler. Always.
- `server.ServeStdio` is blocking; the process exits when stdin closes (the client disconnected).
- No port to manage; no TLS to configure. The client handles lifecycle.

## Implementation: streamable HTTP

```go
func main() {
    logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
    s := mcpserver.NewServer(logger)

    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }

    logger.Info("starting MCP server",
        slog.String("transport", "streamable_http"),
        slog.String("port", port),
    )
    if err := server.NewStreamableHTTPServer(s).Start(":" + port); err != nil {
        logger.Error("http server failed", slog.String("error", err.Error()))
        os.Exit(1)
    }
}
```

Operational additions you must layer on:
- **TLS termination.** Behind an Azure Front Door, Application Gateway, or Container Apps ingress — or terminate in the app with `Start(":" + port, certFile, keyFile)`. Plain HTTP is acceptable only in trusted internal networks.
- **Authentication.** OAuth/JWT validation in middleware before the MCP handler. Anonymous MCP endpoints are a serious security problem.
- **Rate limiting.** Per-client quotas to prevent one tenant from saturating the server.
- **Health endpoints.** `/healthz` for readiness/liveness probes; don't conflate with MCP.
- **Observability.** HTTP-layer metrics (latency, status codes) plus MCP-layer metrics (tool call counts, errors).

## Choosing in Azure

| Hosting | Transport | Why |
|---|---|---|
| Container Apps as an agent backend | streamable HTTP | Built-in TLS, autoscale, ingress |
| Container Apps job invoked by orchestrator | stdio | Lifecycle is per-job |
| Function App | streamable HTTP | Functions don't fit stdio's process model |
| Bundled with Claude Code or local CLI | stdio | Subprocess model |
| AKS pod serving multiple agents | streamable HTTP | Standard k8s service shape |

## Migration: stdio → streamable HTTP

When demand grows from local-only to remote/multi-client, the migration is small if the server was structured correctly:

1. Verify the server initialisation (`NewServer(logger)`) doesn't depend on stdin/stdout for anything other than transport.
2. Add an `MCP_TRANSPORT` environment variable that selects the transport in `main.go`.
3. Add HTTP-layer concerns: TLS, auth, rate limit, healthcheck.
4. Test from a remote client (or `curl` for sanity); validate the JSON-RPC frames over HTTP.
5. Deploy. The MCP service layer is unchanged.

The package layering from `project-structure.md` makes this trivial: `main.go` chooses the transport; everything else is transport-agnostic.

## Common failure modes

- **Logging to stdout under stdio.** Corrupts the wire; server appears unresponsive. Detection: client sees malformed messages or hangs at startup. Fix: `slog.NewJSONHandler(os.Stderr, ...)`.
- **SSE for new builds.** Adopting a deprecated transport. Detection: spec compliance checks fail; new clients can't talk to the server. Fix: use streamable HTTP.
- **stdio with TLS in mind.** Adding TLS configuration to a stdio server — meaningless; stdio has no network layer. Detection: dead code in `main.go`. Fix: pick one transport per binary.
- **HTTP server without auth.** Anonymous MCP over the network is a security hole. Detection: no JWT validation middleware. Fix: require auth before reaching the MCP handler.
- **Mixed transports in one binary without clean separation.** `main.go` has three branches doing slightly different setup. Detection: code drift between branches. Fix: each branch should do only the transport-specific bit; shared setup happens before.

## Verification questions

1. What transport does this server run on, and why?
2. If stdio: are all logs going to stderr?
3. If HTTP: is there authentication, TLS, rate limiting, health endpoint?
4. If SSE: when's the migration scheduled?
5. Could this server be re-hosted on a different transport with under a day of work?

## What to read next

- `server-basics.md` — minimal server skeleton on each transport
- `project-structure.md` — keeping transport choice isolated in `main.go`
- `enterprise-security.md` — auth and TLS for HTTP-based servers
- `observability.md` — observability differs per transport
