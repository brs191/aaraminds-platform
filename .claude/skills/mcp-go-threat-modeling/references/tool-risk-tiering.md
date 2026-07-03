# Tool Risk Tiering

> Every MCP tool gets a risk tier before it gets a threat model. The tier sets the *minimum* controls; STRIDE then finds the tool-specific gaps. Default a new tool to the highest plausible tier and justify it *down* — never up after an incident.

## The four tiers

| Tier | Definition | Examples |
|---|---|---|
| **Informational** | Returns static or non-sensitive derived data; no caller-specific data, no side effects | `get_server_version`, `list_tool_schemas` |
| **Read** | Reads caller- or tenant-scoped data; no mutation | `get_pod_logs`, `query_orders`, `read_config` |
| **Write** | Mutates state, reversible, bounded blast radius | `create_ticket`, `update_record`, `tag_resource` |
| **Destructive** | Irreversible or wide blast radius — deletes, money movement, infra changes, bulk ops | `delete_namespace`, `issue_refund`, `scale_cluster`, `bulk_export` |

## Controls required by tier (the minimum floor)

| Control | Informational | Read | Write | Destructive |
|---|---|---|---|---|
| Transport auth (HTTP) | required | required | required | required |
| Per-tool authorization | — | required | required | required |
| Input validation (types/size/pattern) | required | required | required | required |
| Audit log (identity, args, decision) | recommended | required | required | required (immutable sink) |
| Output redaction | — | required | required | required |
| Rate limit / resource cap | recommended | required | required | required (tight) |
| Idempotency key | — | — | required | required |
| Human-in-the-loop confirmation | — | — | — | **required** (or a hard policy gate) |
| Dry-run / preview mode | — | — | recommended | recommended |

A control marked required for a tier is a hard finding if absent. This table is the contract `mcp-go-guardrails-and-safety` implements (`../../mcp-go-guardrails-and-safety/references/tool-authorization.md`).

## How tier steers the STRIDE pass

The tier tells you which STRIDE rows carry the most weight, so the threat model spends effort where it matters:

- **Destructive** → Elevation of Privilege and Repudiation dominate. Who can invoke it? Is every invocation non-repudiably logged? Is there a confirmation step an injected instruction cannot silently satisfy?
- **Read** → Information Disclosure dominates. What sensitive content can the output carry? Is redaction proven by a test?
- **Write** → Tampering and idempotency. Can a replay or malformed arg corrupt state?
- **Informational** → DoS and supply-chain only; keep the surface boring.

## The composition trap

Tier each tool *and* the reachable chains. Two individually-safe tools — a read tool that returns a resource id and a write tool that acts on any id — compose into an unsafe capability. Tier a *workflow* by its most destructive reachable step, and correlate audit logs across the chain. This is the STRIDE "tool composition abuse" row made concrete.

## Anti-pattern — defaulting new tools to Read

Teams ship a new tool tiered "read" because that is the least work, then discover it can mutate via a side effect (a "read" that triggers a cache rebuild, a "query" that runs arbitrary SQL). Default to the **highest plausible tier**, wire the controls, then demote with a written reason once the surface is proven bounded.

## Read next

- `threat-modeling.md` — the STRIDE pass the tier prioritizes
- `prompt-injection-and-output-handling.md` — least privilege as the capability-control primary defense
- `../../mcp-go-server-building` — design-time placement of risk tier and audit emission
