# The Threat-Model Document

> The deliverable. STRIDE in your head is not a threat model; the artifact is. It is also the named evidence source the compliance skill cites for SOC 2 CC3.2 and ISO 27001 A.5.8 / A.8.29 — so it has to exist, be current, and be queryable.

## Where it lives

`docs/threat-model-<server-or-tool>.md`, in the server's own repo, versioned with the code. One per MCP server; large servers may have one per high-risk tool. Reviewed in PRs like any other doc.

## Structure

1. **System summary** — what the server does, transport (stdio / HTTP), who calls it, what it can reach downstream.
2. **Trust boundaries** — the three MCP boundaries (LLM→server, server→downstream, server→LLM) and what crosses each.
3. **Tool inventory** — every tool with its **risk tier** (`tool-risk-tiering.md`), inputs (types, limits), outputs (shape, sensitive fields), required caller identity, audit emission.
4. **STRIDE table per tool** — every threat class with its defense and a **status**: `implemented` / `planned` / `accepted-risk (rationale + owner)`. The status column is the point; an all-"planned" table is a backlog, not a defense.
5. **MCP-specific threats** — prompt-injection, output-as-instructions, tool-composition abuse, supply-chain — assessed explicitly (see `prompt-injection-and-output-handling.md`).
6. **Security-test catalog** — the 7-category matrix mapped to the tests that enforce it (`security-test-generation.md`); link to the test files.
7. **Finding log** — open findings with severity, owner, and target date; closed findings with the commit that closed them.
8. **Re-review triggers** — quarterly, every major version, and after any incident. Record the last review date.

## The status discipline

A threat model's value is in the **status column**, not the threat list. Three honest states:

- `implemented` — the defense exists and a test proves it. Cite the test.
- `planned` — named, owned, dated. Anything planned past its date is a finding.
- `accepted-risk` — a named person accepted it, with a rationale and an expiry. Not "we'll get to it."

A table where everything is "implemented" with no test references is aspirational, not evidential — and an auditor will treat it as such.

## As compliance evidence

`soc2-iso27001-controls-mapping` cites this document directly:

| Control | What this doc provides |
|---|---|
| SOC 2 **CC3.2** (risk identification) | Per-system threat model + finding log = the risk register for the server |
| ISO 27001 **A.5.8** (security in project management) | The threat-model-per-project artifact |
| ISO 27001 **A.8.29** (security testing in development) | The security-test catalog (section 6) and its CI wiring |

Keep it current: a threat model dated 14 months ago fails the control it is meant to satisfy.

## Read next

- `threat-modeling.md` — the STRIDE method that fills section 4
- `security-test-generation.md` — the test catalog that fills section 6
- `../../soc2-iso27001-controls-mapping` — the controls this document is evidence for
