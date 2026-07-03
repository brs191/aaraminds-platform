# TypeScript Discipline

This reference covers using TypeScript so it actually prevents bugs — strict mode, the typed API boundary, runtime validation at the edge, and modeling UI state as a discriminated union.

## Strict mode, and `any` is a defect

Turn `strict` on in `tsconfig.json` and leave it on — it is the difference between TypeScript that catches mistakes and TypeScript that decorates them. Treat `any` as a defect: it switches the type checker *off* for everything it touches and the disabling spreads silently through every value derived from it. When a type is genuinely unknown, use `unknown` and narrow it with a check — `unknown` forces you to prove the shape before use; `any` lets you assume it. Ban `any` in review; allow `unknown` plus narrowing.

## The API boundary is a typed contract

Everything the BFF returns to the browser is a contract, and it should be typed once, explicitly, and shared. Define the response types and use them on both sides — the BFF handler's return and the component's consumption — so a change to the contract is a compile error at every call site, not a runtime surprise. Where the backend service and the frontend can share a generated type (from an OpenAPI spec or a shared package), do that; where they cannot, the BFF is the place the external shape is mapped onto the frontend's own types.

## Validate at the edge — types are not runtime checks

A TypeScript type is erased at build time. Annotating a `fetch` result as `ReviewDoc` does not *check* anything at runtime — it just asserts, and if the BFF returns something else the lie propagates until it crashes somewhere distant. So at every boundary where untyped data enters — a `fetch` response, a parsed message, form input — validate it with a runtime schema (Zod is the standard) and derive the static type *from* the schema. One definition produces both the runtime check and the compile-time type:

```ts
import { z } from "zod";

const ReviewDoc = z.object({
  id: z.string(),
  title: z.string(),
  status: z.enum(["draft", "reviewing", "accepted"]),
  sections: z.array(z.object({ heading: z.string(), body: z.string() })),
});
type ReviewDoc = z.infer<typeof ReviewDoc>;

const res = await fetch(`/api/review/${id}`);
const doc = ReviewDoc.parse(await res.json());   // throws here, at the edge, not three components deep
```

Parse at the edge, trust the typed object within.

## Model UI state as a discriminated union

The four-states rule from the SKILL router is enforced, not just remembered, by typing state as a discriminated union — one field discriminates the variant, and each variant carries only the data that state has:

```ts
type ReviewState =
  | { status: "loading" }
  | { status: "streaming"; partial: string }
  | { status: "success"; doc: ReviewDoc }
  | { status: "error"; message: string };
```

Now `state.doc` is reachable only inside the `success` branch — the compiler refuses to read it while loading — and a `switch` over `status` is incomplete until every case is handled. This is strictly better than scattered `isLoading` / `isError` / `data` booleans, where the impossible combination (loading *and* error *and* data) is representable and someone will eventually render it. Make illegal states unrepresentable.

## Prefer precise types

Use `union` types over loose `string` where the values are known (`"draft" | "reviewing" | "accepted"`, not `string`). Type function signatures fully. Avoid non-null assertions (`!`) — they are `any` for null. The payoff of precise types is that a whole class of bug becomes a red squiggle in the editor instead of a production incident.
