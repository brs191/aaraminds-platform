# Retrieval and RAG Evaluation

This reference covers evaluating retrieval-augmented and GraphRAG features — the CIF's trust problem made measurable. It is the eval counterpart to `data-access-engineering`'s `graphrag-retrieval.md`: that builds the retriever, this proves it works.

## Evaluate the two surfaces separately

A RAG answer has two independent failure surfaces, and an end-to-end "is the answer good" score cannot tell them apart:

1. **Retrieval** — did the pipeline fetch the right context (the right subgraph)?
2. **Generation** — did the model use that context faithfully, or did it invent?

A good answer from bad context is luck; a bad answer from good context is a generation bug. Score each surface on its own so a regression localizes to the retriever or the prompt, not "somewhere in RAG."

## The four metrics

| Metric | Question | Scorer |
|---|---|---|
| **Context recall** | Did retrieval return the nodes the answer needs? | Deterministic against a labeled relevant set |
| **Context precision** | Was the retrieved set free of noise (relevant ranked first)? | Deterministic against the labeled set |
| **Faithfulness / groundedness** | Is every claim entailed by the retrieved context? | LLM-as-judge, rubric-anchored |
| **Answer relevance** | Does the answer actually address the question? | LLM-as-judge |

Faithfulness is the anti-hallucination metric and the one the CIF lives or dies on: an answer that asserts something the retrieved context does not support is a hallucination, however fluent.

## Citation accuracy — the deterministic win

Because `graphrag-retrieval.md` carries a source id on every node and the answer cites those ids, citation accuracy is largely a **deterministic** check, not a judge call:

1. Does every cited id **resolve** in the graph at the pinned `buildVersion`? (exists check)
2. Does the cited node **support** the claim it is attached to? (the one judge call, and a narrow one)
3. Are there claims with **no** citation? (ungrounded-claim count — should be zero for a grounded answer)

Lead with the deterministic checks (resolve, no-uncited-claims); reserve the judge for "does this node support this sentence." Per `scoring-methods.md`, deterministic-first holds here too.

## GraphRAG-specific: score the subgraph, not just the text

For GraphRAG, the retrieval label is a **golden subgraph** per question — the entities *and edges* a correct answer needs. Score retrieval as set overlap against it: did it fetch the right methods AND the `CALLS`/`DEPENDS_ON` edges between them, or just isolated nodes? A retriever that returns the right nodes with the wrong structure will mislead the model about how the code connects.

## The golden set for RAG

Each case is `(question, golden relevant node-ids / subgraph, golden answer, required citations)`, pinned to a `buildVersion` so retrieval is reproducible (`graphrag-retrieval.md` determinism). Build it from real questions plus hand-labeled relevant context — including questions whose honest answer is "the codebase does not contain this," to catch the retriever that always returns *something* and the model that always obliges.

## Failure modes

- **End-to-end score only** → cannot tell a retrieval miss from a generation hallucination. Score both surfaces.
- **No labeled relevant set** → recall and precision are unmeasurable; you are guessing.
- **Faithfulness judge uncalibrated** → an unvalidated judge is a second opinion of unknown quality (`scoring-methods.md`).
- **Citation check that only verifies id-exists** → a resolvable id attached to an unsupported claim still passes. Check support, not just existence.
- **Eval against a moving graph** → retrieval changes build to build; pin the `buildVersion`.

## Verification questions

1. Are retrieval and generation scored separately, so a regression localizes to one surface?
2. Is there a labeled relevant set (a golden subgraph for GraphRAG) behind context recall/precision?
3. Is faithfulness measured, and is the ungrounded-claim count zero on the golden set?
4. Is citation accuracy checked deterministically (ids resolve, no uncited claims) before any judge call?
5. Is the RAG golden set pinned to a `buildVersion` and grown from real questions, including unanswerable ones?

## What to read next

- `scoring-methods.md` — deterministic checks vs LLM-as-judge, and calibrating the judge
- `golden-datasets-and-fixtures.md` — building and maintaining the labeled set
- `data-access-engineering`, `references/graphrag-retrieval.md` — the retriever this evaluates
- `ai-application-architecture`, `references/evaluation.md` — the per-archetype evaluation view
