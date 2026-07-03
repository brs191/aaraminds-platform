# Next.js App Architecture

This reference covers structuring a Next.js application on the App Router — the route tree, where data is fetched, and choosing a rendering strategy per route.

## The App Router structure

The App Router maps the filesystem to routes: a folder is a route segment, `page.tsx` is the page, `layout.tsx` is a shared shell that wraps it and its children, `loading.tsx` and `error.tsx` are the framework-wired loading and error states for that segment. Layouts nest, so shared chrome lives at the level it applies to and is not re-rendered on navigation within it. Co-locate route-specific components inside the route folder; lift genuinely shared components to a top-level `components/`.

## Fetch data on the server

In a Server Component you fetch data directly — `await` a call to the BFF or a service in the component body. There is no `useEffect`-fetch, no client-side loading waterfall for initial data, and no API key exposure because the fetch runs on the server. Fetch at the level that needs the data; let the framework dedupe identical requests within a render. Client-side fetching (React Query / SWR) is for data that changes *after* load in response to user interaction — not for the initial page.

## Choose a rendering strategy per route

Next.js can render a route statically at build time, dynamically per request, or stream it. The strategy is a per-route decision driven by the data, not a global default:

- **Static** — content that is the same for everyone and changes rarely. Rendered once, served from cache.
- **Dynamic** — content that depends on the request: the signed-in user, request-time data. Rendered per request.
- **Streamed** — a dynamic route with a slow part. Wrap the slow component in `<Suspense>`; the shell and fast content render immediately, the slow part streams in. This is how a review surface shows its frame instantly while the reconstructed document loads.

Decide deliberately. A route marked dynamic that could be static wastes render; a route forced static that needs request data is wrong.

## Route Handlers are the BFF

A `route.ts` file is a Route Handler — a server-side HTTP endpoint. This is where the Backend-for-Frontend lives: the handler authenticates the request, reads secrets from the server environment, calls the Python orchestration and Go gateway, and returns a UI-shaped response (or a stream). The browser calls these handlers; the handlers call the system. Detail in `bff-tier.md`.

## Server Actions for mutations

For form submissions and mutations from the UI, Server Actions are the App Router's mechanism — a server function the client can invoke without a hand-written endpoint. Use them for mutations a user triggers (submitting a review decision); keep Route Handlers for the request/response and streaming surfaces. Either way the privileged work runs on the server.

## Environment and configuration

Server-only configuration — API keys, service URLs, secrets — stays in server environment variables, read only in Server Components, Route Handlers, and Server Actions. Only values that are genuinely public may carry the `NEXT_PUBLIC_` prefix, because that prefix *inlines the value into the client bundle*. Treat `NEXT_PUBLIC_` as "this will be visible to every visitor" — because it will be.
