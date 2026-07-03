# Traceability Graph — Phase 1 Design

**Status:** Draft for review
**Date:** 2026-05-26
**Owner:** Raja
**Scope:** Phase 1 of the defect-to-HLD traceability capability inside the code intelligence factory.
**Decision record:** This document is both a design doc and an ADR. The ADR log is §13.

---

## Verdict

Build a traceability graph that ingests Confluence (HLD), Jira (work + defects), and GitHub (code), and materializes the chain `Defect → Commit → PullRequest → Story → Epic → Requirement → HLD page` as queryable edges. Backtracing a production defect to its originating HLD requirement then becomes a single graph traversal, and *gap analysis becomes a query* rather than a forensic meeting.

The graph is the right investment because it is an asset, not a tool: the same node-and-edge model also answers impact analysis, test-coverage mapping, change-risk scoring, and release composition. We are not building a defect tracer; we are building the substrate the code intelligence factory was always meant to be.

One hard precondition, stated plainly because skipping it is the failure mode that kills these builds: **a traceability graph over un-traceable data is a graph of holes.** Two enforcement fixes — stable requirement IDs in Confluence, and a mandatory Jira key on every merge to GitHub — are not a separate project. They are Milestone 0 of this build. The graph reports link coverage; it cannot invent links that the data never carried.

---

## 1. Context and problem

When a defect is raised in production today, there is no reliable way to walk it back to the HLD requirement it originated from. The link chain exists in fragments — Jira issue links, commit messages, Confluence pages — but no single system holds the whole chain, and the weak hops (commits without Jira keys, HLD requirements with no stable identifier) break the trace before it reaches the requirement.

The question we actually need to answer for each defect is not just *which requirement* but *which class of gap*:

- **Requirement gap** — the HLD never specified this case, or specified it ambiguously or wrongly.
- **Design gap** — the HLD was correct; the low-level design missed it.
- **Implementation gap** — the design was correct; the code diverged from it.
- **Test gap** — the implementation was plausibly correct; no test caught the regression.
- **Traceability gap** — the link chain itself is broken (orphan commit, Story with no requirement, requirement with no implementing Story). An inability to backtrace is itself a finding.

Phase 1 makes the first four answerable by traversal and makes the fifth visible as a first-class metric.

## 2. Decision

Adopt the automated traceability graph (Option 2 of the gap-analysis options). Reject the alternatives as standalone solutions: the disciplined-link-chain approach (Option 1) is hostage to human hygiene and does not scale to legacy code; the AI-recovery approach (Option 3) is probabilistic and is deferred to a later phase as a *recall* layer that writes into this same graph.

Option 1's enforcement mechanics are not rejected — they are absorbed into this build as Milestone 0, because Option 2 cannot function without them.

## 3. Phase 1 scope

**In scope.** A graph store; an ingestion service for Confluence, Jira, and GitHub; the `backtrace` query and the core gap-detection queries; a `link-coverage` health metric; exposure of `backtrace` as an MCP tool. Proven end to end on **one service and one Epic** before ingestion widens.

**Out of scope for Phase 1.** LLM-assisted link recovery (Phase 3); test-to-code `COVERS` edges beyond a manual seed (Phase 2 — depends on a CI coverage feed); deployment/release topology (Phase 2); a UI beyond the MCP tool and ad-hoc Gremlin queries.

**Anti-goal.** Do not build the universal ingestion layer first. A working backtrace on one service exposes model errors far more cheaply than a half-built ingester across every repo.

## 4. Graph store — decision and rationale

**Use Azure Cosmos DB for Apache Gremlin.** It is the graph store inside the sanctioned stack (Azure-primary; Cosmos DB is already a listed datastore). Introducing Neo4j would be cloud/tool drift against workspace anti-pattern #1.

Honest tradeoff: Neo4j's Cypher is materially nicer to read and write than Gremlin traversals, and Neo4j's tooling ecosystem is richer. We accept Gremlin's worse ergonomics because the Phase 1 queries are shallow, fixed-shape traversals — not open-ended graph analytics where Cypher's expressiveness would earn its keep. Revisit only if query complexity later forces it, and make that a deliberate ADR, not a default.

`[VERIFY]` Confirm at build time that Cosmos DB for Apache Gremlin is still Microsoft's current graph offering and not in deprecation; if it has been superseded, re-open ADR-001 before writing ingestion code.

**Partitioning.** Cosmos requires a partition key on every vertex. Use `service` (the owning microservice / repo) as the partition key so that most traversals — which stay within one service's commits, PRs, and stories — remain single-partition and cheap. Cross-service edges (a shared-library Requirement, say) are allowed but counted: a high cross-partition edge ratio is a signal the partition key is wrong.

## 5. Graph model

### 5.1 Vertices

| Label | Natural key | Key properties | Partition key |
|---|---|---|---|
| `Requirement` | `req_id` (e.g. `REQ-PAYMENTS-014`) | `text`, `status`, `hld_page_id` | `service` |
| `HLDPage` | `confluence_page_id` | `title`, `space`, `url`, `version` | `service` |
| `Epic` | `jira_key` | `summary`, `status` | `service` |
| `Story` | `jira_key` | `summary`, `status`, `issue_type` | `service` |
| `Defect` | `jira_key` | `summary`, `severity`, `environment`, `incident_id`, `detected_at`, `gap_class` | `service` |
| `PullRequest` | `repo` + `pr_number` | `title`, `merged_at`, `merge_sha`, `author` | `service` |
| `Commit` | `sha` | `message`, `authored_at`, `author` | `service` |
| `File` | `repo` + `path` | `path`, `language` | `service` |
| `TestCase` | `repo` + `test_id` | `name`, `suite` | `service` |
| `Release` | `repo` + `tag` | `tag`, `released_at` | `service` |

`gap_class` on `Defect` is one of the five classes in §1. It is set by the checking template (§9), not inferred — Phase 1 does not auto-classify.

### 5.2 Edges

Directions are chosen so a backtrace from `Defect` flows along `out()` steps.

| Edge | From → To | Meaning |
|---|---|---|
| `REGRESSES` | `Defect → Commit` | this commit introduced the defect |
| `PART_OF` | `Commit → PullRequest` | commit belongs to this PR |
| `DELIVERS` | `PullRequest → Story` | PR implements this story |
| `IMPLEMENTS` | `Story → Requirement` | story implements this requirement |
| `DEFINED_IN` | `Requirement → HLDPage` | requirement is specified on this page |
| `PARENT_OF` | `Epic → Story` | epic owns this story |
| `MODIFIES` | `Commit → File` | commit changed this file (carries `lines` range) |
| `COVERS` | `TestCase → File` | test exercises this file |
| `INCLUDES` | `Release → Commit` | commit shipped in this release |
| `TRACES_TO` | `Defect → Requirement` | derived shortcut, materialized after a successful backtrace |

`TRACES_TO` is a denormalized convenience edge written by the backtrace job once a chain resolves, so dashboards do not re-traverse. It always carries the lowest `confidence` of any edge on the path it summarizes.

### 5.3 Edge provenance contract — mandatory

Every edge, without exception, carries these four properties:

- `source_system` — `jira` | `github` | `git-blame` | `confluence` | `manual` | `llm-recovered`
- `confidence` — `1.0` for a system-of-record link; `< 1.0` for anything inferred
- `observed_at` — ISO-8601 timestamp the edge was last confirmed
- `ingest_run_id` — the ingestion run that wrote it, for auditing and rollback

This costs nothing now and buys two things later: Phase 3's LLM-recovered links live in the *same* graph without contaminating deterministic ones, and any backtrace can be filtered by trust — `show me conclusions that depend on an edge with confidence < 1.0`.

## 6. Milestone 0 — enforcement (precondition, do this first)

Neither of these is optional. They are the difference between a graph and a graph of holes.

**Stable requirement IDs in Confluence.** Every HLD requirement gets an immutable ID like `REQ-PAYMENTS-014` — not a heading that can be renamed. Use the Requirement Yogi or easeRequirements Marketplace app to make requirements first-class objects with IDs and status; do not rely on hand-typed conventions. The HLD page embeds its implementing Jira work via the Jira Issues macro, which also auto-creates the back-link from issue to page.

**Mandatory Jira key on every merge.** A GitHub branch-protection rule plus a PR-title status check (GitHub Actions) that rejects any PR whose title lacks a `PROJ-123` key. Add a `commit-msg` hook in the repo template as a developer-side fast-fail. No key, no merge. This is what makes `Commit → PullRequest → Story` resolvable at all.

Exit criterion for Milestone 0: link coverage (§10) on new merges is climbing toward ≥ 95%.

## 7. Ingestion architecture

**Pattern: webhooks for freshness, scheduled reconciliation for correctness.** Webhooks keep the graph near-live; because webhooks drop events, a nightly full reconciliation re-pulls each source and upserts. Every write is idempotent, keyed on the vertex/edge natural key. The ingestion service is a Go service (consistent with the existing Go MCP server) hosted on Azure Container Apps; it owns both ingestion and the query API.

### 7.1 Source contracts

**Jira.** Webhook events for issue create/update/link and reconciliation via the Jira Cloud REST API (`/rest/api/3/search` with JQL, `/issue/{key}` for links). Produces `Epic`, `Story`, `Defect` vertices; `PARENT_OF`, `IMPLEMENTS` edges (from issue links of the configured link types). `source_system = jira`, `confidence = 1.0`.

**GitHub.** Webhook events for `pull_request` and `push`; reconciliation via the GitHub GraphQL API (PRs with commits, files-changed, merge SHA). Produces `PullRequest`, `Commit`, `File` vertices; `PART_OF`, `MODIFIES` edges. The `DELIVERS` edge (`PullRequest → Story`) is resolved by parsing the Jira key out of the PR title — which is exactly why Milestone 0 enforcement exists. `source_system = github`, `confidence = 1.0`.

**Confluence.** No reliable webhooks for content changes — poll the Confluence REST API on the reconciliation schedule. Pull HLD pages, parse `REQ-ID`s (or read them from the Requirement Yogi data) into `Requirement` vertices and `DEFINED_IN` edges. `source_system = confluence`, `confidence = 1.0`.

**The `REGRESSES` edge is human-assisted.** A stack trace yields a file and line; `git blame` on that line gives a *candidate* introducing commit, and blame lies once a line has been moved or touched twice. `git bisect` between release tags is more reliable but needs a reproduction. So `Defect → Commit` is created from the checking template (§9, field 3), not auto-derived. `source_system = manual` or `git-blame`; `confidence < 1.0` when blame-derived without bisect confirmation.

## 8. Queries

### 8.1 Backtrace (the core query)

```groovy
g.V().has('Defect','jira_key','BUG-1234').as('defect')
  .out('REGRESSES').as('commit')
  .out('PART_OF').as('pr')
  .out('DELIVERS').as('story')
  .out('IMPLEMENTS').as('requirement')
  .out('DEFINED_IN').as('hld')
  .select('defect','commit','pr','story','requirement','hld')
  .by(valueMap(true))
```

To recover the owning Epic, add `.select('story').in('PARENT_OF')`. A path that returns no result is itself the answer: the first missing hop names the **traceability gap**.

### 8.2 Gap-detection queries

Orphan commits (untracked changes — traceability gap):

```groovy
g.V().hasLabel('Commit').not(outE('PART_OF')).valueMap('sha','message')
```

Unimplemented requirements (a Requirement with no implementing Story):

```groovy
g.V().hasLabel('Requirement').not(inE('IMPLEMENTS')).valueMap('req_id','text')
```

Untested files in a defect's blast radius (candidate test gap):

```groovy
g.V().has('Defect','jira_key','BUG-1234')
  .out('REGRESSES').out('MODIFIES').dedup()
  .not(inE('COVERS')).valueMap('path')
```

### 8.3 Low-trust backtraces (audit)

```groovy
g.V().hasLabel('Defect').as('d')
  .outE('TRACES_TO').has('confidence', lt(1.0)).inV().as('r')
  .select('d','r').by('jira_key').by('req_id')
```

## 9. The checking template — input to the graph

One record per production defect, used as a Jira Bug screen template or Confluence page template. It is both the human gap-analysis artifact and the *input form* that creates the `REGRESSES` edge and sets `gap_class`.

1. **Defect identity** — Defect ID · severity · prod incident ID · date detected · environment · customer-facing (Y/N).
2. **Symptom** — observed behavior · error signature / stack trace · affected service & endpoint.
3. **Code localization** — file(s) + line(s) from the trace · `git blame` introducing commit (SHA, author, date) · `git bisect` first-bad commit if a regression. → *creates the `REGRESSES` edge.*
4. **Change provenance** — PR # · PR title · merge date · reviewers · Jira key present in commit/PR? If **No**, record a traceability gap immediately.
5. **Requirement linkage** — Story key → Epic key → HLD page + `REQ-ID` + section · how the link was established and its confidence.
6. **Gap classification** — pick one primary class (requirement / design / implementation / test / traceability). → *sets `gap_class` on the `Defect` vertex.*
7. **Evidence** — quote the HLD requirement text · quote the offending code · state precisely where they diverge.
8. **Corrective action** — fix the code *and* the source artifact (update HLD / add requirement / add test / repair link).
9. **Systemic signal** — is this gap class recurring? link related records.

Fields 4 and 6 are the two that turn a pile of defect tickets into a gap analysis: field 4 measures whether traceability is leaking, field 6 measures whether the HLD process is leaking.

## 10. Health metric

**Link coverage** — the percentage of commits merged in the trailing 90 days that have a resolvable path to a `Requirement`. This single number is both the graph's trustworthiness gauge and the Milestone 0 hygiene scorecard. Track it from day one. If it is not climbing toward ~95%, the enforcement is not real and no query will compensate.

Secondary: cross-partition edge ratio (validates the partition key); reconciliation drift (edges added/removed by the nightly pass vs. webhook state — high drift means webhook handling is buggy).

## 11. MCP exposure

Expose `backtrace(defect_key)` as a tool on the existing Go MCP server so agents can query the graph directly. Phase 1 ships one read tool; writes stay with the ingestion service. Returns the resolved chain plus, on a broken chain, the first missing hop and the recommended gap class.

## 12. Milestones

| # | Milestone | Exit criterion |
|---|---|---|
| M0 | Enforcement: REQ-IDs in Confluence; Jira-key gate on merge | Link coverage on new merges climbing toward ≥ 95% |
| M1 | Cosmos Gremlin store stood up; vertex/edge schema + provenance contract applied | Schema validated; a hand-built sample chain traverses |
| M2 | Ingestion for one service: Jira + GitHub + Confluence, webhook + reconciliation | One service's graph builds and reconciles cleanly |
| M3 | `backtrace` query + gap-detection queries; checking template wired to `REGRESSES`/`gap_class` | A real recent production defect backtraces end to end |
| M4 | `link-coverage` metric + MCP `backtrace` tool | Metric dashboarded; tool callable from an agent |
| M5 | Widen ingestion service by service, watching coverage | Coverage holds ≥ 95% as repos are added |

Sequencing is strict: M0 before M1, and M3's end-to-end proof on one service before M5 widens. Effort is deliberately not quoted in weeks until M0 scope is confirmed against the actual Confluence/Jira estate.

## 13. ADR log

- **ADR-001 — Graph store.** Cosmos DB for Apache Gremlin, not Neo4j. Rationale §4. Carries a `[VERIFY]` on Cosmos Gremlin's current support status.
- **ADR-002 — Partition key.** `service`. Rationale §4.
- **ADR-003 — Edge provenance is mandatory on every edge.** Rationale §5.3. Enables Phase 3 coexistence and trust filtering.
- **ADR-004 — `REGRESSES` is human-asserted, not auto-derived.** Rationale §7.1. `git blame` is a candidate, not proof.
- **ADR-005 — LLM link recovery deferred to Phase 3.** It writes into this graph as `llm-recovered` edges with `confidence < 1.0`; it is recall, never audit evidence on its own.

## 14. Risks

- **Garbage in.** If Milestone 0 enforcement slips, the graph faithfully records the holes. Mitigation: link-coverage gate as M0 exit criterion; do not start M2 until it holds.
- **Code-side brittleness.** `git blame` mis-attributes across refactors and renames. Mitigation: prefer `git bisect` for regressions; mark blame-only edges `confidence < 1.0`; for stack-frame-to-symbol resolution, buy a code-intelligence tool (e.g. Sourcegraph) rather than building a symbol resolver — a tooling choice, not stack drift.
- **Graph staleness.** Dropped webhooks. Mitigation: nightly reconciliation; monitor reconciliation drift.
- **Scope creep into a universal ingester.** Mitigation: M3 must prove one service end to end before M5 widens.
- **Cosmos Gremlin support risk.** Mitigation: the `[VERIFY]` on ADR-001; the provenance-bearing model is store-agnostic enough to migrate if forced.

## 15. Out of scope — later phases

- **Phase 2** — CI-fed `COVERS` edges (real test-coverage mapping); `Release`/`Deployment` topology; change-risk scoring.
- **Phase 3** — LLM-assisted traceability link recovery for legacy and keyless history, writing `llm-recovered` edges with confidence scores and human-in-the-loop confirmation.
