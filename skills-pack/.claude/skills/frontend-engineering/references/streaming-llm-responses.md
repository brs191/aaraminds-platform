# Streaming LLM Responses

This reference covers getting an LLM's token-by-token output from the model, through the BFF, to the browser — and rendering it as it arrives.

## Why streaming is not optional for an LLM UI

An LLM response takes seconds to tens of seconds to complete. A UI that waits for the whole response before showing anything looks frozen for that entire time, and a long wait with no feedback reads as broken. Streaming changes the felt experience completely: the first token appears in under a second and text flows in continuously. For a review surface like the CIF Trust Gate — where an engineer watches a reconstructed document being generated — streaming is the core interaction, not a nicety.

## The stream path: model → BFF → browser

The token stream crosses two hops. The model SDK yields tokens to the BFF; the BFF re-streams them to the browser. The BFF does not buffer the whole response and then forward it — that would erase the benefit. It reads each chunk and writes it onward immediately, while still doing its job as the trust boundary: the browser never holds the model key, and the BFF can still authenticate the request before the stream starts.

## Mechanism: SSE or a streamed Response body

Two standard mechanisms. **Server-Sent Events** (`text/event-stream`) is a simple, one-directional server-to-client stream with automatic reconnection — a natural fit. Alternatively a Next.js Route Handler can return a `ReadableStream` as the `Response` body directly. Either way the handler returns a streaming body rather than a finished JSON document:

```ts
export async function POST(req: Request) {
  await requireSession(req);
  const upstream = await callModel(await req.json());   // returns an async iterable of tokens
  const stream = new ReadableStream({
    async start(controller) {
      for await (const token of upstream) {
        controller.enqueue(new TextEncoder().encode(token));
      }
      controller.close();
    },
  });
  return new Response(stream, { headers: { "content-type": "text/plain; charset=utf-8" } });
}
```

On the client, read the response body as a stream and append each chunk to component state, so the text grows on screen as it arrives.

## Rendering the streaming state

Streaming adds a fifth UI state to the four in the SKILL router: a surface is loading (request sent, nothing back yet), then **streaming** (tokens arriving, partial content visible — show the text-so-far plus a cursor or indicator), then success (stream closed, content complete), or error. The streaming state is its own branch: the user can read partial output but actions that need the *complete* document — "accept this reconstruction" — stay disabled until the stream closes. Render the partial text; gate the decision.

## Cancellation, errors, and incomplete streams

A stream can be abandoned (the user navigates away — abort the request and stop the upstream call so tokens are not generated into the void) or can fail mid-flight (the model errors after 200 tokens). Mid-stream failure is the subtle case: the UI already showed partial content, so the error state must make clear the content is *incomplete* — not silently leave half a document looking finished. Carry an explicit terminal status: closed-complete versus closed-error. A review surface must never let a human accept a document that only half-generated.
