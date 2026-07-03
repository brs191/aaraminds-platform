# Incremental Code Modeling

A codebase changes constantly, so its model is not built once — it is rebuilt, repeatedly, and the rebuilds must produce stable, comparable results. This reference covers stable identity, identity resolution for inferred entities, regeneration diffs, and incremental rebuilds.

## The model is rebuilt, not built once

A comprehension model is only useful if it tracks the code, and the code keeps moving. So the model is regenerated — on a schedule, or per commit — and the central requirement of every rebuild is that **the same real-world entity gets the same identity every time**. Without that, every rebuild looks like a wholesale change, no two builds can be compared, and the model cannot answer "what changed."

## Stable IDs for deterministic entities

Code-layer entities have natural keys, so their IDs are deterministic hashes of those keys (`ast-extraction-and-parsing.md`): a type by fully-qualified name, a method by type plus name plus ordered parameter-type list, an endpoint by HTTP method plus path. Same code in, same ID out, every build, with no cross-build matching needed. This is why the identity rules are not a detail — they are the precondition for the model being rebuildable at all. A method keyed by name alone changes identity the moment an overload is added; keyed by full signature, it is stable.

## Identity resolution for inferred entities

Inferred entities — a component clustered from un-annotated classes, a reconstructed capability — have no natural key. A rebuild assigns them fresh IDs, and a separate **identity-resolution step** matches the new build's inferred entities against the previous build's by type and attribute similarity, carrying the ID forward where the match holds. This is a genuinely hard problem: the matcher has false positives (two different components fused across builds) and false negatives (one component that looks new because its members shifted), and both corrupt the diff. Treat identity resolution as a real component — tune it against known rebuilds, and keep deterministic-keyed and resolved entities distinguishable. The same problem and treatment appear in `azure-data-tier-design`, `references/graph-databases.md`.

## Regeneration diffs

Once IDs are stable, a rebuild can be diffed against the prior build: the set of entities and edges added, removed, and changed. The diff is the highest-value output of incremental modeling — it answers "what changed in the architecture since the last release" and lets a consumer re-examine only the delta instead of the whole model. Stamp every build with the source version it was built from — the commit SHA — so any two builds are precisely comparable and the diff is anchored to a real range of source history.

## Incremental rebuild — scope the work to the change

A full re-parse of a large codebase on every commit is wasteful. An incremental rebuild re-extracts only the changed files and the entities whose resolution depends on them, then re-runs the affected slice of the graph and the inference. Full rebuilds remain the right move when the extraction logic itself changes — a new parser version, a new rule — because that invalidates every prior fact. Incremental for a code change; full for an extractor change.

## Worked example — the Code Intelligence Factory

The CIF schema makes stable identity a design principle — "regeneration diffs are only possible if the same real-world entity gets the same ID on every build" — and dedicates a section to it: deterministic hashed IDs for code-layer nodes, UUIDs plus an identity-resolution step for inferred nodes, every node stamped with `firstSeenCommit` / `lastSeenCommit`. The regeneration diff is what the product's document-snapshot model rests on — "here is what changed in the HLD since the last release" — so a reviewer re-reviews only the delta. The schema also flags identity resolution for inferred nodes as a known hard open problem, which is the honest framing.

## Verification questions

1. Do code-layer entities get deterministic IDs hashed from natural keys, identical across rebuilds?
2. Is there an identity-resolution step matching inferred entities across rebuilds, tuned against known cases?
3. Is every build stamped with the source commit it was built from?
4. Does a rebuild produce a diff — entities and edges added, removed, and changed?
5. Is incremental rebuild scoped to changed files, with a full rebuild reserved for when the extractor logic changes?

## What to read next

- `ast-extraction-and-parsing.md` — the natural keys IDs are hashed from
- `call-and-dependency-graphs.md` — re-resolving the affected graph slice
- `azure-data-tier-design`, `references/graph-databases.md` — identity resolution and graph storage
