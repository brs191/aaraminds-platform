# The Depth Audit

**Status:** v1.0
**Owner:** Raja
**Date:** 2026-05-24
**Location:** `skills-pack/validation/governance/`
**Purpose:** A diagnostic framework for auditing a layered knowledge system — the
skills pack, the instruction-OS, any system of routers sitting over deeper
references. It names where such systems decay and which layers automated tooling
cannot see, so an audit spends effort where the rot actually is.

---

## Why this exists

`skill_audit.py` answers one question well: is the pack well-formed? Sections
present, links resolve, counts agree, descriptions sized. It cannot answer a
second question that matters more: is the pack *correct*?

The 2026-05-24 internal audit found the gap concretely. The Tier-1 routers had
been rewritten to v1.0 and read clean. The Tier-2 references beneath them had
not — a resilience reference said "retry 4xx" while its router said "never on
4xx"; a service-mapping reference still defaulted to Azure SQL after its router
had moved the default to Postgres. Every one of those skills passed the linter.
Structural green said nothing about whether the depth still agreed with the
surface.

A layered knowledge system rots unevenly. The Depth Audit is how we catch it.

## The framework

Audit in four layers, shallow to deep. Each layer catches what the layer above
it cannot.

### 1. Structure — does it pass the linter?

The cheap, automatable layer: sections present, links resolve, counts agree,
descriptions sized. Run it on every change. It is necessary and never
sufficient — a clean structural pass proves the system is well-formed, not that
it is correct. Treating green here as "healthy" is the core mistake the next
three layers exist to catch.

### 2. Agreement — does the deep content still say what the surface promises?

Vertical drift, inside one unit. A router gets rewritten and looks current; the
reference beneath it does not, and now the two contradict each other. This is
the layer automated linting is blind to — structure checks form, not whether a
reference still agrees with the page that routes into it. Partly automatable: a
term-level lint catches off-stack and renamed-product drift (added to
`skill_audit.py` on 2026-05-24 as the `OFF_STACK_DRIFT` reference scan and the
`STALE_TERM` check). The rest — a reference that contradicts its router's
decision rule — is read-and-compare.

### 3. Coherence — do the parts still fit each other?

Horizontal drift, across siblings. Two skills overlap until they should be one;
scopes collide; the same content is taught twice from two angles, as the four
`mcp-go` skills now do. Not automatable — it needs someone holding the whole set
in view. The signal: you cannot state, in one line each, why two neighbouring
skills are separate.

### 4. Mission — does the whole still match the job it exists for?

The deepest layer: the system against its purpose. A pack can be well-formed,
internally consistent, and still aimed at the wrong target — the pack covers
*building* the Code Intelligence Factory but not its actual product output,
the rendered, versioned, evidence-linked documents. Manual, periodic, and the
easiest layer to skip, because nothing breaks. The system just quietly stops
being the right system.

## Where it applies

- Auditing the skills-pack — routers over references.
- Auditing the instruction-OS — personas over modules, the same layered shape
  and the same rot risk.
- Reviewing any living documentation set before relying on it.
- Deciding what to automate in `skill_audit.py` versus what stays a human review.

## How to use it

1. Run **Structure** on every change — automated, fast, blocking.
2. Run **Agreement** whenever a router is rewritten — the rewrite is the exact
   moment the references fall behind.
3. Run **Coherence** and **Mission** on a cadence, not per-change — they catch
   slow drift, not fresh edits. The quarterly freshness review (see
   `freshness-cadence.md`) is the natural home for both.
4. When a layer fails, reconcile toward the layer that was deliberately
   updated — usually the router. Fix the stale reference up to the router; do
   not loosen the router down to the reference.

## Leadership takeaway

A green linter proves a knowledge system is well-formed. It never proves it is
right. Health is audited by depth — and the deeper the layer, the less your
tooling can see it for you.

## Visual brief

A framework visual for The Depth Audit. **Type:** depth stack. **Canvas:** 4:3,
slide-ready.

- **Layout** — four full-width horizontal bands, stacked shallow-to-deep:
  Structure at top, Mission at the base. Equal band height; colour weight
  deepens downward so the eye reads down.
- **Center of gravity** — a left-margin coverage marker: a solid bracket
  spanning Band 1 only, fading across the top third of Band 2, labelled
  "automated linting reaches here." That marker carries the whole idea.
- **Per band** — band name (1 word), the question (4-8 words), and a small
  right-aligned tag: `automated` / `partly` / `manual` / `manual`.
- **Closing principle** — footer strip: "Green proves well-formed. It never
  proves right."
- **Colour** — navy titles; the four bands in a calm blue progression
  deepening with depth; one orange accent, used only for the coverage marker
  because it marks risk. Thin dividers, not heavy boxes. Keep 45-60% of the
  canvas calm.

## Framework quality record

Run through the Framework Creation System gates before adoption.

- **Decoration Audit — 8/8 (strong).** Improves a real decision (where to look,
  what the linter misses); four distinct relationships (form / vertical /
  horizontal / purpose); changes behaviour; explainable in a minute.
- **Whiteboard Check — 8/8 (ready).** Name self-explains; one-word points;
  survives the worked example; drawable as a stack.

## Version notes

### v1.0 — 2026-05-24

- First version. Extracted from the 2026-05-24 internal skills-pack audit.
- Authored under the AaraMinds Content Strategist persona (Framework Design
  mode); voice is Quiet Authority with Intentional Integrity.
- Known limitation: Layer 2 (Agreement) is only partly automated. The
  `OFF_STACK_DRIFT` reference scan and `STALE_TERM` check cover term-level
  drift; a reference that contradicts its router's *decision rule* is still
  caught only by human read-and-compare.
