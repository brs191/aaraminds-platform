# Skill — MCP-Go Client Integration

## Purpose

Build a client that talks to a Go MCP server. Where `agent-integration.md` is the server-author's view ("how do agents see my server?"), this skill is the client-author's view ("how do I write the program that calls it?"). The most common form is a custom agent, a CLI, or a test harness — anything that drives the MCP server programmatically.

## Two client shapes

| Client | Lives in | Use case |
|---|---|---|
| Off-the-shelf MCP client | Claude Code, Claude Desktop, IDE extensions | The end-user-facing agent context |
| Programmatic / custom client | Your own Go, Python, or other code | Demos, tests, custom agent loops, batch tools |

This skill focuses on the second — when you write the client code yourself.

## Protocol primer for clients

MCP over stdio is newline-delimited JSON-RPC 2.0. The minimal interaction:

1. **Spawn the server.** Client launches the server binary as a subprocess; stdin/stdout are the wire.
2. **Initialize.** Client sends `initialize` request with `protocolVersion`, `capabilities`, and `clientInfo`. Server responds with its capabilities.
3. **Notify initialized.** Client sends `notifications/initialized` (notification, no response).
4. **Discover.** Client may call `tools/list`, `resources/list`, `prompts/list`.
5. **Call.** Client sends `tools/call` requests; reads responses keyed by `id`.
6. **Shutdown.** Client closes stdin; server exits.

Over streamable HTTP, the steps are the same; the wire is HTTP + a streaming response body. SSE is deprecated; don't build new clients on it.

## Minimal Go client (illustrative)

```go
// Pseudocode — actual SDK usage may differ.
client := mcp.NewStdioClient(exec.Command("./mcp-server"))
ctx := context.Background()

if err := client.Initialize(ctx, mcp.InitializeParams{
    ProtocolVersion: "2025-11-25",
    Capabilities:    mcp.ClientCapabilities{},
    ClientInfo:      mcp.Implementation{Name: "my-client", Version: "0.1.0"},
}); err != nil {
    return err
}
client.NotifyInitialized(ctx)

tools, err := client.ListTools(ctx)
if err != nil {
    return err
}

result, err := client.CallTool(ctx, "generate_service_boundary_canvas", map[string]any{
    "input_json": string(inputJSON),
})
if err != nil {
    return err
}
// result.Content is a list of content blocks (text, image, resource).
```

The pattern in any language: connect, initialize, discover, call, repeat, shutdown.

## Minimal Python client (stdlib only)

The pack's demo runner at `demo/architecture-review-demo/demo_runner.py` is a stdlib Python client. It implements just enough of the protocol — about 100 lines — to spawn the server, complete the handshake, and call tools. No `pip install` required.

Key implementation details:

- `subprocess.Popen` with `stdin=PIPE, stdout=PIPE, stderr=PIPE`.
- A background thread reads stderr and forwards to the client's stderr (so server logs are visible during runs).
- Newline-delimited JSON-RPC over stdin/stdout.
- Request IDs are auto-incremented; responses matched by ID; unrelated notifications discarded.

When you need a client that survives in environments without the official MCP SDK, this is the shape.

## What clients must handle correctly

### Request/response matching

The server may send notifications between request and response. The client must:

- Track the request ID it sent.
- Skip notifications and unrelated server-initiated messages.
- Match the response to its ID.

```python
while True:
    msg = recv()
    if msg.get("id") == my_request_id:
        return msg["result"]
    # else: notification or unrelated; discard
```

### Stderr handling

Server logs come on stderr. The client should:

- Capture stderr explicitly (don't let it inherit and pollute the client's stderr unintentionally — or do, if you want server logs visible).
- Forward to the client's logging at an appropriate level.
- Not interpret stderr as protocol data.

### Tool result shape

`tools/call` responses have:

```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "result": {
    "content": [
      {"type": "text", "text": "..."}
    ],
    "isError": false
  }
}
```

If `isError` is true, the content describes the error. The client should surface this to the caller, not raise an exception (it's a tool-level rejection, not a transport error).

### Graceful shutdown

Closing stdin signals the server to exit. The client should:

- Send any remaining messages.
- Close stdin.
- `Wait()` on the subprocess with a timeout.
- Send SIGTERM and then SIGKILL if the timeout is exceeded.

Don't let abandoned server processes accumulate.

## Configuration and auth

For stdio, client configuration is the command line and environment variables passed to the subprocess. For HTTP, the client must additionally handle:

- TLS verification (don't disable for convenience).
- Authentication: OAuth/JWT token included in HTTP headers.
- Connection pooling and timeouts.
- Retry on transient HTTP errors (with idempotency keys if tools require them).

## Demo runner as worked example

The reference client in this pack is `demo/architecture-review-demo/demo_runner.py`. It:

- Spawns the Go MCP server via stdio.
- Performs the initialize handshake with `protocolVersion: "2025-06-18"`.
- For each architecture in `input/`, shapes per-tool inputs and calls five tools.
- Re-emits server responses as JSON files for comparison against goldens.

Total: ~250 lines, stdlib only. Read it as a template when writing your own client.

## Common failure modes

- **Skipping notifications/initialized.** Server may refuse `tools/call` until it sees the initialized notification. Detection: first tool call fails or hangs. Fix: send the notification after initialize.
- **Mixing request IDs.** Client reuses ID 1 for every call. Detection: responses arrive in unexpected order; matching breaks. Fix: monotonic counter.
- **Ignoring stderr.** Server logs go nowhere; debugging post-incident is harder than it should be. Detection: silent server failures. Fix: capture stderr explicitly.
- **Treating `isError: true` as transport failure.** Client retries forever on a tool-level rejection. Detection: retry storms on validation errors. Fix: surface tool errors to the caller; don't retry.
- **Hanging on shutdown.** Client closes stdin but doesn't wait properly; orphaned server processes. Detection: zombie processes accumulate. Fix: structured shutdown with timeout and SIGTERM fallback.
- **Pinning the wrong protocol version.** Client declares it speaks `2024-11-05` against a `2025-11-25` server. Detection: capability negotiation fails or server returns errors. Fix: pin to a version the server supports; update both together.

## Verification questions

1. Does the client correctly send `initialize` and then `notifications/initialized` before any tool call?
2. Does it match responses to requests by ID?
3. Does it handle stderr deliberately, not by accident?
4. Does it distinguish tool errors (`isError: true`) from transport errors?
5. Does it shut down cleanly without orphaning the server process?

## What to read next

- `agent-integration.md` — the server-side mirror
- `transport-selection.md` — stdio vs. streamable HTTP from the client's view
- `demo/architecture-review-demo/demo_runner.py` — working reference implementation
