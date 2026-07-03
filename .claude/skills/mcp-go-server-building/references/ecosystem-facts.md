# Ecosystem Facts — Verified May 18, 2026

## Purpose

This file captures the current state of the Go and MCP ecosystem as of the verification date below. Skills in this pack reference these facts. When ecosystem facts shift (new Go release, new MCP spec version, new SDK), this file and the affected skills must be updated together.

## Verification date

**May 18, 2026.** All facts below were verified via web search of canonical sources on this date. Re-verify quarterly or before relying on any specific version pin in production.

## Go runtime

- **Latest stable Go release:** 1.26.3 (released 2026-05-07)
- **Currently supported Go versions:** 1.25.x and 1.26.x
- **Out of support:** Go 1.24 and earlier (Go's policy: only the two most recent majors get security patches)
- **Recommended for new projects:** Go 1.26.x
- **Minimum baseline:** Go 1.25.x

Notable Go 1.26 changes relevant to MCP servers:
- Green Tea garbage collector is enabled by default (10–40% GC overhead reduction in real-world programs that heavily use GC)
- Baseline cgo overhead reduced approximately 30%
- New `new(expr)` syntax allows allocating-and-initializing pointers in one expression
- Linker improvements reduce build time for projects with many dependencies

These changes are real, modest improvements. None of them is a reason by itself to upgrade. The reason to upgrade is that **Go 1.24 and earlier no longer receive security patches.**

## MCP protocol

- **Current MCP spec version:** 2025-11-25
- **Backward-compatible versions:** 2025-06-18, 2025-03-26, 2024-11-05
- **Specification home:** modelcontextprotocol.io

## Go MCP SDKs — both are viable

There are now **two production-viable Go MCP SDKs.** This is a meaningful change from earlier versions of this pack which only referenced one. A 9+ pack must surface this honestly because the choice has real consequences.

### Option A — `github.com/modelcontextprotocol/go-sdk` (official)

- **Status:** Stable since v1.0.0, currently v1.5.0
- **Maintainers:** Anthropic in collaboration with Google
- **Stability commitment:** Formal — no breaking API changes after v1.0.0
- **MCP spec coverage:** Implements 2025-11-25 with backward compatibility
- **Importers:** ~1,443 (as of verification date)
- **Transports:** stdio, SSE, streamable HTTP
- **License:** Apache 2.0 for new contributions, MIT for existing code
- **OAuth support:** Yes, with `oauthex` package for protected-resource metadata

Choose this SDK when:
- You want the longest-stability commitment available in the Go MCP ecosystem
- You prefer SDK design influenced directly by the protocol authors
- You value formal compatibility guarantees (post-v1.0 no breaking changes)
- Your team has no prior commitment to mcp-go

### Option B — `github.com/mark3labs/mcp-go` (community)

- **Status:** Active development (latest mcp package published May 13, 2026)
- **Maintainer:** Originally Ed Zynda, now community-maintained
- **Stability:** No formal v1.0 stability commitment as of verification date
- **MCP spec coverage:** Implements 2025-11-25 with backward compatibility
- **Importers:** ~1,880 (as of verification date — more than the official SDK)
- **Transports:** stdio, SSE, streamable HTTP
- **License:** MIT
- **OAuth support:** Yes, including RFC 9728 protected-resource metadata discovery (added in v0.49.0)
- **Pack pin (verified):** the `microservices-system-design-mcp-server` example pins `v0.52.0`. Verified building and running on Go 1.26.3 with `go mod tidy` / `go build ./...` / full test sweep and a live stdio `tools/list` + `tools/call` round-trip during session-3 verification (May 18, 2026). No version bump required; re-verify the pin per the freshness cadence.

Choose this SDK when:
- You have existing code on mark3labs/mcp-go and migration cost is significant
- You prefer the slightly more concise API surface (subjective)
- Your tooling, examples, or third-party integrations specifically target mark3labs/mcp-go

### Honest framing

The official SDK explicitly acknowledges mcp-go as inspiration and a viable alternative. There is no "wrong" choice here — both are real, both are maintained, both implement the current spec. The choice is dominated by your team's existing code, the stability commitment you value, and which SDK's API ergonomics you prefer.

For new enterprise projects with no prior Go MCP code, **the official SDK is the slightly safer default** because of the post-v1.0 no-breaking-changes commitment.

## Azure hosting for MCP servers

Microsoft Learn now has canonical documentation for hosting MCP servers on Azure:
- **Overview:** learn.microsoft.com/en-us/azure/container-apps/mcp-overview
- **Choosing the service:** learn.microsoft.com/en-us/azure/container-apps/mcp-choosing-azure-service

Key guidance from current Microsoft documentation:

- **Recommended services:** Azure Container Apps (standalone) or Azure App Service
- **Transport:** Streamable HTTP (SSE is deprecated due to 4-minute idle timeout issues on Azure load balancer)
- **MCP endpoint convention:** `/mcp` (single endpoint for both POST and GET)
- **Ingress transport setting:** `auto` or `http` (no special MCP transport value exists)
- **CORS:** Required when browser-based or VS Code MCP clients connect — configure allowed origins, methods, and headers explicitly
- **Min replicas:** Set to 1 for interactive MCP use to avoid cold-start latency (scale-to-zero is technically supported but degrades interactive UX)
- **TLS:** Container Apps ingress handles TLS termination — your container serves plain HTTP on the target port
- **Authentication:** Use Managed Identity for Azure-internal connections; use OAuth 2.1 or API key headers for client authentication (the latter is what VS Code currently supports best)

Alternative hosting:
- **Azure App Service:** Good fit when you prefer code-based deployment without a Dockerfile, or have an existing App Service Plan
- **Azure Kubernetes Service (AKS):** Good fit when you already run AI workloads on AKS, need GPU node pools, or require custom networking
- **Azure Container Apps dynamic sessions:** Platform-managed sandboxed environments for running code in isolation — useful when the MCP server's role is to execute LLM-generated code in a sandbox

## When to update this file

Re-verify and update when any of the following occur:

- A new Go major version is released (and the support window for current versions shifts)
- A new MCP spec version is released
- Either Go SDK has a major version change (v2.x, breaking changes)
- Microsoft Learn publishes guidance that changes the MCP-on-Azure default
- A new viable Go MCP SDK enters the ecosystem with significant adoption

Suggested cadence: quarterly review even if nothing obvious has shifted.

## How skills reference these facts

Skills in this pack should reference this file rather than hard-coding ecosystem facts. When a skill says "use the official MCP Go SDK or mark3labs/mcp-go," it does not hard-code version pins or feature sets — those live here, in one place, with one verification date.

This is the freshness mechanism the prior pack lacked.
