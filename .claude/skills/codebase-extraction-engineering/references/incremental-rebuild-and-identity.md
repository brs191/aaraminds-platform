# Incremental Rebuild and Identity

A codebase changes constantly, and re-extracting all of it on every commit is waste. This reference covers implementing incremental extraction — change detection, the resolution-dependency ripple, identity resolution in code, and when to fall back to a full rebuild.

## Re-extract the change, not the repo

An incremental extractor re-processes only what changed and what depends on it, reusing everything else from the prior build. The payoff is large — a one-file commit should cost seconds, not a full re-parse — but it is correct only if the dependency ripple (below) is handled. The model carries stable deterministic IDs (`extractor-architecture.md`) so the incremental result merges cleanly into the prior graph and the regeneration diff stays meaningful.

## Change detection

Determine the changed set from the source of truth — a commit range (`git diff --name-status` between the last extracted commit and HEAD) — not file timestamps, which are unreliable across checkouts. Stamp every build with the commit it extracted; the next run diffs against that stamp. Classify each change: a modified file is re-extracted, a deleted file's nodes and edges are removed, an added file is extracted fresh.

## The resolution-dependency ripple

The trap in incremental extraction: changing file A can change the *resolved* model of file B that A never imported directly. Rename a method in a base class and every override and call site resolves differently; change a return type and callers' type resolution shifts. So the re-extraction set is not just the changed files — it is the changed files **plus their resolution dependents**: files whose resolved AST could differ because of the change. Compute that set from the type and call graph of the prior build — who references the changed types. Underestimate it and the incremental model silently diverges from what a full rebuild would produce — the worst kind of bug, because the model still looks fine.

## Identity resolution in code

Deterministic-keyed nodes — the code-layer entities — need no matching: the same key hashes to the same ID across builds, for free. Inferred nodes — clustered components, reconstructed capabilities — have no natural key, so the incremental build assigns new IDs and an **identity-resolution step** matches them against the prior build's inferred nodes by type and attribute similarity, carrying IDs forward where the match holds. Implement it as a real, tested component with a tunable match threshold: false matches fuse distinct components, missed matches make a stable component look new, and both corrupt the diff. This is the hard problem `codebase-comprehension` and `azure-data-tier-design`'s graph reference both flag.

## When to fall back to a full rebuild

Incremental is for source changes. Force a full rebuild when the *extractor itself* changes — a new parser version, a new extraction rule, a new pass — because that invalidates every previously extracted fact, and a model that mixes old-logic and new-logic facts is incoherent. Detect it by versioning the extractor and stamping each build with that version alongside the commit; a version mismatch triggers a full rebuild. Also fall back when the resolution-dependent set grows so large — a change to a widely-used core type — that incremental saves nothing.

## Verification questions

1. Is the changed set computed from a commit range, not file timestamps, and is each build stamped with the commit it extracted?
2. Does re-extraction include the resolution dependents of changed files, not just the changed files themselves?
3. Do deterministic-keyed nodes reuse IDs for free, and is there a tested identity-resolution step for inferred nodes?
4. Is the extractor versioned, with a version change forcing a full rebuild?
5. Does the incremental result merge into the prior graph cleanly enough that the regeneration diff stays meaningful?

## What to read next

- `extractor-architecture.md` — the pipeline being made incremental
- `resolving-the-call-graph.md` — the dependency information the ripple set is computed from
- `codebase-comprehension`, `references/incremental-code-modeling.md` — the model-design view
- `data-access-engineering` — merging the incremental result into the graph
