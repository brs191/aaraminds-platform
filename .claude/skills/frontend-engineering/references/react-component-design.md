# React Component Design

This reference covers building React components well — composition, the owned-versus-derived state distinction, the server/client split, and when state genuinely needs a manager.

## Compose small, single-purpose components

A component does one thing: it renders a piece of UI, or it owns one piece of behavior. A 400-line component that fetches, transforms, branches, and renders is the frontend equivalent of a god class — split it. The split is by responsibility, not by line count: a data-owning container, presentational children that take props and render, a hook for reusable behavior. Props are the contract; keep them few and typed.

## Owned state versus derived state

The single most common React bug is storing a value that should have been computed. State is for what the component *owns* and cannot recompute — the text in an input, which tab is open. Anything that can be *calculated from existing state or props* is derived, and must be computed during render, not stored in its own `useState` and kept in sync with a `useEffect`. A `useEffect` whose job is to set state from other state is a synchronization bug waiting to happen — the two values drift. If you can derive it, derive it. If you must memoize the derivation because it is expensive, `useMemo` — still not state.

## `useEffect` is for synchronizing with outside systems

`useEffect` exists to synchronize the component with something *outside* React — a subscription, a non-React widget, the document title. It is not a place to react to state changes; that is what render and event handlers are for. An effect that reads state and writes state is almost always wrong. Most "I need an effect" instincts are actually: derive it during render, or handle it in the event handler that caused the change.

## Server Components by default

In the Next.js App Router, a component is a Server Component unless marked `"use client"`. Server Components render on the server, ship no JavaScript for themselves, and can fetch data directly. Make them the default. Reach for `"use client"` only where the component needs interactivity (event handlers, `useState`), a browser API, or a hook that uses them. Push the `"use client"` boundary *down* the tree — a small interactive leaf is a client component; its parent layout stays on the server. A whole page marked `"use client"` for one button has shipped the entire subtree to the browser.

## When state needs a manager

Most state is local, URL, or server state — none of which needs a global store. Local state is `useState`. State that should survive a refresh or be shareable is URL state (search params). Data from the backend is *server state* — owned by a data-fetching layer (React Query / SWR or the framework's own fetching), with its own caching and revalidation, not hand-rolled into `useState` + `useEffect`. Reach for a client state manager (Zustand, Redux) only for genuinely global *client* state that is none of the above and is painful to prop-drill. Adding a global store by default is how simple state becomes hard to trace.

## Lists, keys, and accessibility

A rendered list needs a stable `key` from the data's identity — never the array index, which corrupts state on reorder. Build with semantic HTML — a `button` is a button, not a `div` with an `onClick` — so keyboard and screen-reader behavior is correct for free; add ARIA only to fill genuine gaps. Accessibility is part of writing the component, covered in the SKILL router.
