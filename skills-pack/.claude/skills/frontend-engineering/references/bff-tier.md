# The BFF Tier

This reference covers the Backend-for-Frontend — the server-side tier that exists specifically to serve this frontend — why it exists, what it owns, and how to build it as Next.js Route Handlers.

## Why a BFF exists

The browser cannot be trusted with anything privileged, and it should not be coupled to the shape of every backend service. The BFF solves both. It is a thin server tier, owned by the frontend team, that sits between the browser and the system. It exists for three reasons: it is the **trust boundary** (authentication, authorization, secrets live here, not in the browser); it is the **aggregation and shaping layer** (it calls the Python orchestration tier and the Go gateway and returns exactly what the UI needs, in the shape the UI wants); and it **decouples** the frontend from backend churn (a backend response change is absorbed in the BFF, not spread across components). `ai-application-architecture`'s serving topology is where the BFF is placed; this skill builds it.

## What the BFF owns

- **Authentication and session** — it verifies the user and establishes the session; the browser never sees a service credential.
- **Secrets** — model API keys, service-to-service credentials, signing keys are read from the server environment here and never leave it.
- **Calls to the system** — it calls the Python orchestration tier and the Go gateway; the browser calls only the BFF.
- **Response shaping** — it aggregates and trims backend responses to what the UI needs, so the browser is not over-fetching or reshaping data itself.
- **Streaming** — it consumes a model token stream and re-streams it to the browser (see `streaming-llm-responses.md`).

## What the BFF does not own

The BFF is not a general backend. Business logic, the knowledge graph, the orchestration, the model calls — those belong in the Python and Go tiers. A BFF that grows domain logic has become a second backend that the frontend team now maintains by accident. Keep it thin: auth, secrets, call, shape, stream. If logic is being added to the BFF that another client would also need, it belongs in a real service.

## Building it as Route Handlers

In Next.js the BFF is a set of Route Handlers (`route.ts`) and Server Actions. A handler: authenticates the request, reads the secret it needs from the server environment, calls the upstream service with that credential, shapes the result, and returns it. It runs only on the server, so the secret never reaches the client.

```ts
// app/api/review/[docId]/route.ts  — a BFF handler
export async function GET(req: Request, { params }: { params: { docId: string } }) {
  const session = await requireSession(req);          // auth boundary
  const res = await fetch(`${process.env.ORCHESTRATOR_URL}/documents/${params.docId}`, {
    headers: { authorization: `Bearer ${process.env.ORCHESTRATOR_TOKEN}` }, // server-only secret
  });
  if (!res.ok) return new Response("upstream error", { status: 502 });
  const doc = await res.json();
  return Response.json(shapeForReview(doc, session));  // shape for the UI
}
```

## Failure handling

The BFF turns upstream failures into something the UI can render. An orchestration-tier timeout, a 500 from the Go gateway, an auth failure — each becomes a defined status and a typed error body the frontend's error state knows how to display. The BFF never leaks an upstream stack trace to the browser, and it never lets a slow upstream hang the request without a timeout. A clear `502` with a typed message is what lets the frontend render a real error state instead of a blank screen.
