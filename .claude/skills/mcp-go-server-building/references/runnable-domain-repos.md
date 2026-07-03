# Skill — MCP-Go Runnable Domain Repositories

## Purpose

Organise example MCP servers as runnable, domain-aligned repositories. The pack ships one (`examples/microservices-system-design-mcp-server/`); future packs may ship more. This skill is about the shape these repositories should take, what they should and shouldn't include, and how they relate to the main skill content.

## What a runnable domain repo is

A runnable domain repo is:

- A self-contained Go module that builds, tests, and runs.
- Focused on one *domain* (microservices design, Azure FinOps, infrastructure review).
- An *exemplar* of the patterns described in the skill files, not a production artifact.
- Maintained alongside the skills it demonstrates; changes track together.

Think of it as a worked example big enough to be useful, small enough to read in an afternoon.

## What it is not

- A reusable framework. Teams should *fork* or *mirror*, not depend on it as a library.
- A complete production system. Operational concerns are demonstrated, not exhaustive.
- A benchmark. Performance isn't the focus.

## Repository structure

```
examples/<domain>-mcp-server/
├── README.md                      # Purpose, build, run, troubleshooting
├── go.mod                         # Module declaration
├── go.sum                         # Dependency checksums
├── Makefile                       # build, test, run, docker-build, docker-run
├── Dockerfile                     # Multi-stage, distroless
├── .github/workflows/ci.yml       # Lint, vet, test, build
├── cmd/server/main.go             # Tiny entry point
├── internal/
│   ├── mcpserver/server.go        # Composition root
│   ├── services/<name>/           # Rule logic + table-driven tests
│   │   ├── service.go
│   │   └── service_test.go
│   └── tools/<name>/              # MCP wiring
│       └── register.go
├── contracts/<area>/implemented/  # Tool contracts as documents
├── testdata/                      # Input + golden output fixtures
└── VERIFICATION_CHECKLIST.md      # Step-by-step verification
```

Properties:

- The `cmd/server/main.go` is small enough to read in one screen.
- Every tool has matching `services/<name>/` and `tools/<name>/` packages.
- Tests are co-located with code.
- Contracts are documents, not source.

## When to spin up a new domain repo

Spin up a new repo when:

- You're authoring skills for a new domain (e.g., security review, cost optimisation).
- The existing example doesn't demonstrate the patterns the new skills require.
- Multiple new tools share a domain and benefit from being grouped.

Don't spin up a new repo just because:

- A tool is slightly different. Add it to an existing example.
- You want a "clean slate" for an experiment. Use a branch; promote to a new repo only when it's stable.

Repo proliferation has a cost: maintenance, CI duplication, dependency drift across examples. Reuse the existing repo until it stops fitting.

## What every domain repo must demonstrate

Each repo, regardless of domain, should demonstrate:

1. **Server skeleton.** Stderr logging, signal-bounded shutdown, transport via env.
2. **At least one tool.** Service-package layering, typed input, table-driven tests, contract document.
3. **Multi-stage Dockerfile.** Distroless, non-root, stripped.
4. **CI workflow.** Lint, vet, race, build.
5. **README.** Build, run, configure, troubleshoot.
6. **Verification checklist.** Step-by-step way for a reviewer to confirm it works.

If a repo lacks any of these, it's not yet a *runnable domain repo*; it's a sketch.

## Versioning and dependency hygiene

Each domain repo:

- Pins its Go version in `go.mod` (`go 1.25.5` or whatever's current).
- Pins SDK versions in `go.sum` (committed).
- Has a clean `go mod tidy` state (no unused dependencies).
- Updates dependencies on the same quarterly cadence as the pack's ecosystem facts (`ecosystem-facts.md`).

If the pack's ecosystem facts say "Go 1.26 is stable", the example repo's `go.mod` should be on Go 1.26 within the next refresh cycle.

## Discoverability

From the pack's root `../../../../README.md`:

```
## Example servers
- examples/microservices-system-design-mcp-server/ — demonstrates Gap 1 patterns
- examples/<future-domain>-mcp-server/             — demonstrates <future>
```

Each example's `README.md` links back to the skill files it demonstrates, so a reader can navigate skill ↔ example freely.

## Maintenance

When skills change in a way that affects examples:

1. Update the example to match the new guidance.
2. Refresh testdata goldens if behaviour changed.
3. Bump example's version; note in the example's README.
4. Run the full verification checklist before tagging.

When examples discover a real issue with the skill content (an anti-pattern that doesn't actually work), update the *skill* and update the example to demonstrate the correction. The two stay in sync.

## Common failure modes

- **Example diverges from skill.** Skill says "use Managed Identity"; example uses connection strings. Detection: skill ↔ example review fails. Fix: update the example; tag a new version.
- **Example becomes a library.** External code imports it. Detection: import-graph analysis. Fix: don't publish as a Go module path other projects import; vendor or fork instead.
- **Repository sprawl.** Three near-identical examples differing in trivial ways. Detection: repos with overlapping purposes. Fix: consolidate; document why each remaining repo exists.
- **CI rot.** Example CI fails for months; nobody noticed. Detection: CI status check on the pack release. Fix: example CI passing is a release-checklist item.
- **Stale go.mod.** Example pinned to a Go version that's no longer supported. Detection: build fails on a current developer machine. Fix: quarterly bump aligned with ecosystem-facts cadence.

## Verification questions

1. Does the example build and test cleanly on a current Go version?
2. Does the example demonstrate every required element (skeleton, tool, Dockerfile, CI, README, checklist)?
3. Is the example's content consistent with the skill files it demonstrates?
4. Is the example's `go.mod` Go version current (within one minor of stable)?
5. Are there real reasons to have multiple example repos, or could they be consolidated?

## What to read next

- `reference-implementation.md` — the canonical walkthrough of the v9.0 example
- `project-structure.md` — the package layout examples should follow
- `../../../../validation/governance/freshness-cadence.md` — when to refresh dependencies and Go version
