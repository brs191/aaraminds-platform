# Version Policy

## Cross-reference

Detailed ecosystem state lives in `.claude/skills/mcp-go-server-building/references/ecosystem-facts.md` with verification dates. This file states the policies; that file states the current facts.

## Go

- **Recommended for new examples:** Go 1.26.x (current stable as of May 2026)
- **Minimum baseline:** Go 1.25.x
- **Out of support:** Go 1.24 and earlier — Go's policy is to support only the two most recent major versions

Verify the current Go version before pinning. Go major versions ship roughly every six months and the support window shifts accordingly.

## MCP libraries — choose one

Two Go MCP SDKs are production-viable. Both implement MCP spec 2025-11-25 with backward compatibility to 2025-06-18, 2025-03-26, 2024-11-05. Spec **2026-07-28** finalizes July 28, 2026 (stateless core, extensions framework, 12-month deprecation window for legacy versions) — re-check SDK spec coverage after it lands; see `ecosystem-facts.md`.

### Official SDK

- **Module:** `github.com/modelcontextprotocol/go-sdk`
- **Status:** Stable since v1.0.0, currently v1.6.1; v1.7.0 ships spec 2026-07-28 support (verify before pinning)
- **Stability commitment:** Formal — no breaking API changes post-v1.0
- **Maintainer:** Anthropic in collaboration with Google
- **Default for new enterprise projects with no prior Go MCP code**

### Community SDK

- **Module:** `github.com/mark3labs/mcp-go`
- **Status:** Active development, no formal v1.0 stability commitment
- **Maintainer:** Originally Ed Zynda, community-maintained
- **Preferred when:** Migration from existing mcp-go code is significant, or your tooling specifically targets mcp-go

Treat library versions as **explicit dependencies**. Do not generate examples with floating or implicit versions. State the SDK and version pin in every example's README.

When unsure which SDK to use, see the decision framework in `.claude/skills/mcp-go-server-building/references/server-basics.md`.

## Transports

- **Local development:** stdio
- **Enterprise deployment:** Streamable HTTP behind an API gateway (Azure API Management, or equivalent)
- **Deprecated:** SSE — has 4-minute idle timeout issues on Azure Load Balancer; use streamable HTTP instead
- **Endpoint convention:** `/mcp` (single endpoint handling both POST and GET, per current MCP server samples on Microsoft Learn)

## Hosting

Verified May 2026 from Microsoft Learn `learn.microsoft.com/en-us/azure/container-apps/mcp-overview`:

- **Recommended:** Azure Container Apps (standalone) or Azure App Service
- **Min replicas:** 1 for interactive MCP clients (avoid scale-to-zero cold-start latency)
- **CORS:** Required when VS Code or browser clients connect
- **TLS:** Handled at ingress; container serves plain HTTP
- **Identity:** Managed Identity for Azure-internal; OAuth 2.1 or API key headers for clients

## Compatibility rule

When generating implementation code, always state in the README:

1. Go version pin
2. MCP SDK module and version pin
3. MCP spec version supported
4. Transport assumption
5. Hosting target

This makes the assumptions explicit and verifiable. Floating versions and unstated assumptions create silent compatibility breaks six months later.

## Production rule

Do not expose Streamable HTTP or SSE deployments without all of:

- Authentication (OAuth 2.1, API key, or Managed Identity — never anonymous)
- Per-tool authorization (the user being authenticated does not mean every tool should run)
- Rate limits (token bucket or sliding window, per-client and per-tool)
- Audit logging (every tool call, with input/output/decision/identity/timestamp)
- Output redaction (secrets, tokens, connection strings stripped before returning to model)
- Tenant-aware policy checks (cross-tenant data leakage is the most expensive failure mode)
- Dependency timeouts (every backend call bounded; no unbounded waits)

These are not optional for enterprise deployment. Missing any one of them is a production defect.

## Freshness

The ecosystem moves. Re-verify versions and Microsoft Learn guidance quarterly, or before any production deployment that depends on the specifics. The verification date in `.claude/skills/mcp-go-server-building/references/ecosystem-facts.md` is authoritative.
