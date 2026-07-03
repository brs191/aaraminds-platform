# Rollback / kill-switch runbook — aara-business-analyst

**Tested 2026-06-18** (disable + restore demonstrated). An unrelated operator can restore cold from this.

## Kill-switch (disable now)
- This runtime: remove the wired link `.claude/agents/aara-business-analyst.md` → the agent is no longer
  dispatchable. (Demonstrated: removal + restore both work.)
- Production runtime: disable the agent registration / set its feature flag off.

## Rollback (restore prior version)
- The agent definition is a single versioned file (git-trackable). Roll back =
  `git checkout <prev-tag> -- skills-pack/.claude/agents/aara-business-analyst.md` then re-wire.
- Config (model, tools, permissionMode, maxTurns) is in the file's frontmatter — versioned with it.

## Triggers (when to roll back / kill)
- Injection-refusal miss · hallucinated-requirement rate breach · an authoritative write reaching a
  system of record (should be impossible — draft-only) · reviewer-override spike · security incident.

## After rollback
- Capture the failing transcript as a new regression case (eval-plan), fix, re-run the suite, re-gate.
