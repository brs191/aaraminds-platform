# Verification Checklist — v10.0

This is the step-by-step recipe to confirm v10.0 works end-to-end in your environment. Run it once when first adopting the pack; re-run before any forked release.

v10.0 ships content inherited from v9.0 in Claude Skills native format (`.claude/skills/<skill>/SKILL.md` + `references/`). Verification covers two things at once: (a) the v9.0-inherited example server, demo, and validation pack still work; (b) the v10.0 format restructure produced 29 well-formed Tier-1 skills with passing cross-references.

## Prerequisites

- Go 1.25.x or 1.26.x installed (`go version` to confirm). If your local Go is older, use the Docker recipe in Step 2.
- Python 3.8+ for the demo and validation tooling (stdlib only; no pip install).
- Docker (optional but recommended) for reproducible builds.
- Internet access for `go mod tidy` to fetch dependencies.

## Step 1 — Module initialization

```bash
cd examples/microservices-system-design-mcp-server
go mod tidy
```

**Expected:** Downloads `github.com/mark3labs/mcp-go` and transitive dependencies. No errors. `go.sum` is updated.

**If it fails:** Most likely network access to `proxy.golang.org`, or you renamed the module path. Fix the module path or restore network connectivity.

## Step 2 — Compile

```bash
go build ./...
```

**Expected:** Silent success. Every package compiles.

**Fallback for older Go:**

```bash
docker run --rm --network=host \
  -v "$(pwd):/src" -w /src \
  golang:1.26 sh -c \
  "go mod tidy && CGO_ENABLED=0 go build -o ./mcp-server ./cmd/server"
```

This produces a static binary at `./mcp-server` you can use for the demo.

## Step 3 — Run unit tests

```bash
go test ./...
```

**Expected:** All 11 service packages pass. Total runtime under 5 seconds.

```bash
go test -race -count=1 ./...
```

**Expected:** Same packages pass under the race detector. Requires cgo (use `golang:1.26` Docker image, not alpine).

## Step 4 — Static checks

```bash
go vet ./...
gofmt -l .
```

**Expected:** No output from either command.

## Step 5 — Server starts in stdio mode

```bash
go run ./cmd/server
```

**Expected:** Structured log line on stderr:

```json
{"time":"...","level":"INFO","msg":"starting MCP server","transport":"stdio"}
```

Press Ctrl+C to exit. Verifies the server binary works and stdio transport initializes.

## Step 6 — Tool catalog is complete

Start the server and send an MCP `tools/list` request. The tool catalog should return **13 tools**:

- `review_microservice_design`
- `recommend_microservice_patterns`
- `score_well_architected_readiness`
- `generate_service_boundary_canvas`
- `generate_api_contract`
- `detect_architecture_risks`
- `map_patterns_to_azure_services`
- `generate_observability_plan`
- `generate_architecture_decision_record`
- `generate_deployment_topology`
- `generate_event_contract`
- `generate_resilience_plan`
- `generate_diagram_assets`

If you have the MCP Inspector or any MCP client, point it at `./mcp-server` with stdio transport. Otherwise, the demo runner in the next step exercises a representative subset.

## Step 7 — End-to-end demo passes

```bash
# From the pack root:
cd examples/microservices-system-design-mcp-server
go build -o ./mcp-server ./cmd/server

cd ../../demo/architecture-review-demo
export MCP_SERVER_BIN=../../examples/microservices-system-design-mcp-server/mcp-server
make demo
make validate
```

**Expected:**

```
Generated outputs in out for 3 architecture(s).
Validation passed: 3 architecture(s) × 5 tool(s) all match golden fixtures.
```

This is the strongest single signal that the pack works as intended: the example server, the demo runner, the per-tool inputs, and the captured goldens are all internally consistent.

If `make validate` reports mismatches, either the server code changed (intentional → run `make refresh`) or something drifted (investigate before refreshing).

## Step 8 — Pattern-card boilerplate check

```bash
# From the pack root:
grep -r "The problem exists in a real workflow" .claude/skills/*/references/patterns/ | wc -l
grep -r "Define the business capability and boundary" .claude/skills/*/references/patterns/ | wc -l
```

**Expected:** Both return `0`.

This confirms the pattern cards are pattern-specific, not boilerplate. Any non-zero result means a regression to stub content.

## Step 9 — Skill structure is valid

```bash
# From the pack root: every Tier-1 skill must have a SKILL.md
find .claude/skills -mindepth 1 -maxdepth 1 -type d | wc -l         # expect 29
find .claude/skills -mindepth 2 -maxdepth 2 -name SKILL.md | wc -l  # expect 29

# Frontmatter sanity: every SKILL.md must declare name, description, version, last_updated
for f in .claude/skills/*/SKILL.md; do
  head -10 "$f" | grep -qE '^name:' && \
  head -10 "$f" | grep -qE '^description:' && \
  head -10 "$f" | grep -qE '^version:' && \
  head -10 "$f" | grep -qE '^last_updated:' || echo "BROKEN FRONTMATTER: $f"
done

# Agents and hooks (Phase 3) are populated
find .claude/agents -maxdepth 1 -name '*.md' -not -name README.md | wc -l    # expect 3
find .claude/hooks  -maxdepth 1 -name '*.json' | wc -l                       # expect 3

# Hook JSON validates
for f in .claude/hooks/*.json; do python3 -c "import json; json.load(open('$f'))" && echo "  $(basename $f): OK"; done
```

**Expected:** Skill counts return `26`. Agents count is `3`. Hooks count is `3`. All hook JSON parses. The frontmatter loop produces no `BROKEN FRONTMATTER` lines.

## Step 10 — Capability prompts are runnable

Open any prompt file under `validation/prompts/`. Check:

- Front-matter is valid (`id`, `area`, `exercises`, `pass_threshold`, `last_run`, `last_result`).
- Prompt section is present and concrete.
- Rubric has named criteria with a declared pass threshold.
- Reference output is a hand-curated exemplar at the right depth.

Pick one prompt; paste it + the skill files listed in `exercises:` into your LLM of choice; score the response against the rubric. If ≥ the declared pass threshold, the workflow is producing what the prompt expects.

Repeat for a sample of 3–4 prompts (one per area). Track results in each prompt's `last_run` / `last_result` front-matter.

Note: v10.0 trimmed the per-skill evals that v9.0 shipped. The 12 capability prompts under `validation/prompts/` are the only runnable validation artifact now; skill-specific verification lives in each SKILL.md's "Verification questions" section.

## Step 11 — Governance docs reflect your team

Open `validation/governance/freshness-cadence.md`. The ownership table is filled with the personal-pack maintainer model. If you fork to a team setting, split the rows by topic competence and add a named backup per row before relying on the cadence.

Open `validation/governance/release-checklist.md`. Confirm the 7 sections describe your release plumbing. Adapt as needed.

## Reporting failures

When verification fails:

1. **Note which step**.
2. **Copy the exact error output** verbatim — don't summarize.
3. **Note your environment** (`go version`, OS, Docker version).
4. **Note any local changes** that might explain it.

For changes you intend to make to the pack: run the full checklist on your fork before tagging a release.

## When this checklist passes

You have a working, verified v10.0 install. The 35 Tier-1 skills are well-formed and usable from Claude Code; the example server is buildable and deployable; the demo proves the server's output is content-distinct across inputs; the validation pack gives you a way to track regressions.

The pack is now yours to use, adapt, and evolve. Re-run this checklist before any release of your fork.
