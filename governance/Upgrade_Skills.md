# Upgrade Skills — Course Plan

**Owner:** Raja
**Date:** 2026-05-25 (v2 — now covers Udemy and Educative.io)
**Purpose:** A course plan to deepen the domains that most improve the AaraMinds
skills-pack and the Code Intelligence Factory (CIF). Two platforms were
surveyed; this ends in one final, sequenced list. Courses deepen the author —
a sharper author writes a sharper pack.

---

## How to read this

The constraint is not money — courses are cheap. The constraint is **hours**.
The skills-pack already proves deep command of Azure microservices, Go MCP
servers, and AI architecture, so broad "complete guide" courses in those
domains are sunk time. Leverage sits in the two domains the CIF demands that
are genuinely newer than the rest of the pack: **knowledge graphs / GraphRAG**
and **agentic systems**. Everything below is filtered to those.

---

## Platform comparison — Udemy vs Educative.io

| Dimension | Udemy | Educative.io |
|---|---|---|
| Format | Video lectures | Text + runnable in-browser code |
| Pricing | Per course, ~₹400–600 on sale, owned forever | Annual subscription, unlimited catalogue (verify current price) |
| Course length | 6–50 h of video | 3–7 h of text — far faster to consume |
| Freshness on AI topics | Variable; AI courses can lag | Frequently "updated this week" |
| Best for | Hands-on video code-alongs | Skimming, design-level learning, senior learners |
| Fit for a time-poor director | Lower — video cannot be skimmed | Higher — skim text, skip the known |

**Read:** Educative fits a senior, time-poor practitioner better — text you can
skim, design-framed, fresher on AI, one subscription. Udemy is better for
one-off, hands-on video code-alongs you own forever. The decisive practical
test: an Educative subscription only pays off if you will actually log in
across the year. If you are a "buy it, mean to watch it" person, Udemy's
one-off purchases waste less.

---

## The analysis — what actually moves the needle

Three candidate areas, ranked:

1. **Knowledge graph / GraphRAG — highest leverage.** The CIF's entire
   architecture is a knowledge graph as system-of-record; its M0 deliverable is
   the graph schema. This is the domain where personal depth is furthest behind
   the product's ambition. Both platforms carry a strong, near-twin course.
2. **Agentic systems — second.** The CIF's BA and QA agents, and the pack's
   weakest-rated cluster (the implementation skills). A design-level treatment
   is the right altitude for a director who designs more than codes.
3. **Production AI / MLOps / MCP — considered and cut from the core.**
   `ai-evaluation-harness`, the serving-topology reference, four MCP skills, and
   a 13-tool Go MCP server are already authored. Courses here would mostly
   re-teach known ground. Kept only as a low-priority option, not in the list.

The two GraphRAG courses are near-twins: Educative's *Master Knowledge Graph
RAG with Neo4j* is 3 h, concept-dense, and fast; Udemy's *Java Spring AI +
Neo4j Knowledge Graph RAG* is 6.5 h and written in **Java** — the CIF's actual
backend — so its code transfers directly. They are complementary, not
redundant: concepts first, hands-on-in-stack second.

---

## The final list

### Core — do these (~9 hours, both on Educative)

**1. Master Knowledge Graph RAG with Neo4j** — Educative · Intermediate · 3 h · 4.6★
The single most CIF-shaped course on either platform. GraphRAG with Neo4j and
an LLM, focused on accuracy and reducing hallucination — the exact trust
problem the CIF brief calls existential. Sharpens `data-access-engineering`,
the `azure-data-tier-design` graph reference, and the CIF's M0 graph-schema
work. Start here — it is the smallest and the highest-leverage.

**2. Agentic System Design** — Educative · Advanced · 6 h · 45 lessons · 4.5★
A design-level treatment of autonomous agents — architectures and strategies,
not a code-along. The right altitude for designing the CIF's BA and QA agents.
Sharpens `ai-application-architecture`.

### Extensions — optional, do only when the trigger fires

**3. Master Agentic Design Patterns** — Educative · Advanced · 4 h · 4.7★
*Trigger:* after course 2, if you want the patterns depth. Same subscription.

**4. Java Spring AI, Neo4j & OpenAI for Knowledge Graph RAG** — Udemy ·
Timotius Pamungkas · 6.5 h · 4.5★ · ~₹500
*Trigger:* when you are personally coding the CIF graph layer. It is in Java —
the CIF's real backend — so the code maps directly. Hands-on follow-up to
course 1.

**5. LangChain — Agentic AI Engineering with LangChain & LangGraph** — Udemy ·
Eden Marco · 19 h · 50k ratings · 4.6★ · ~₹400
*Trigger:* when you are personally building the agent orchestration. Deepens
the pack's weakest cluster (`python-service-engineering`). Heavy — 19 hours;
buy it only when you will actually use it.

### Sequence and budget

- **Now:** Core 1 then Core 2 — **9 hours**, one Educative subscription.
- **When a hands-on trigger fires:** add the matching Udemy extension as an
  individual purchase. Do not buy ahead of the trigger.
- **Everything done:** ~38 hours. Realistic target: finish the 9-hour core;
  treat the rest as on-demand.

---

## Do not touch (either platform)

- Anything **Azure, Go-basics, or microservices-design** — the pack already
  commands these; a course re-teaches earned expertise.
- Any **"complete guide" / bootcamp** in a known domain — built for first-time
  learners.
- **Interview prep** — Grokking Coding Interview, ML Interview, Low Level
  Design, and System Design fundamentals. You are not interviewing.
- **Beginner GenAI / LLM "essentials"** courses and **MCP fundamentals** — you
  have shipped MCP servers; fundamentals are beneath you.
- **SOC 2 / ISO 27001** courses — skip unless a personal certification is the
  goal; the pack's compliance skill is sound and this is not a leverage point.

---

## Honest closer

These courses deepen the author, which sharpens what gets written into the
pack. They will **not** fix the pack's actual audit gaps — the missing CIF
document-generation skill, the `mcp-go-threat-modeling` → `guardrails` merge,
the shallow implementation references. Those are authoring tasks waiting to be
done. The courses are input; the pack work is still the output.
