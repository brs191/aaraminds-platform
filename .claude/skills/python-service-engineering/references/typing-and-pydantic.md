# Typing and Pydantic

This reference covers making a Python service statically checkable: type hints everywhere, Pydantic models at every boundary, a type checker gating CI, and validating untrusted input at the edge. It is the concrete form of the SKILL.md's typed-boundary rule.

## Type hints on every signature

Annotate every function and method — parameters and return type — and every module-level constant whose type is not obvious. Type hints are not decoration: they are what lets a type checker catch a whole class of bug before the code runs, and they are the most reliable documentation a reader has. An un-annotated function is an opaque boundary; the caller has to read the body to know what it takes and returns. In an AI service, where the model boundary is already uncertain, every *other* boundary should be certain.

## Pydantic models at every boundary

Use Pydantic models — not bare `dict`, not `dataclass` — wherever data crosses a boundary: configuration, the incoming request, the model's structured output, the contract between major modules. The reason Pydantic over `dataclass` is **validation**: a Pydantic model validates and coerces its data on construction, so an instance is a *proven-valid* object, not just a typed container. The model's structured output parsed into a Pydantic model is either a valid object or a loud failure — never a half-right dict that fails three calls later.

## Run a type checker in CI as a gate

Type hints do nothing if nothing checks them. Run `mypy` or `pyright` in CI as a **gate** — a failing type check fails the build, the same discipline `pr-review-azure-microservices` applies to other defects. Run it in strict mode for new code: strict mode is what flags the implicit `Any`, the missing return annotation, the unchecked optional. A type checker that runs but does not gate, or runs in lax mode, catches a fraction of what it could and lets the codebase drift untyped.

## Validate at the edge, trust within

Adopt one discipline for untrusted input — an HTTP request body, a config file, a model response, a tool result: **parse it into a Pydantic model once, at the boundary it enters**, and pass the typed, validated object inward. Inside the service, code receives proven-valid objects and does not re-validate. This is the typed-boundary rule's practical shape: validation is concentrated at the edges, the interior is clean, and an invalid input fails at the door with a clear error rather than deep in the call stack as an `AttributeError`.

## The model boundary specifically

The model's output is the highest-value place to apply this. Use the model API's structured-output mode and parse the response straight into a Pydantic model (`ai-application-architecture`, `references/model-and-inference-layer.md`). A schema-validation failure there is reject-and-retry-once-then-fail — never coerce a malformed model response into something that looks usable. The Pydantic model *is* the contract between the non-deterministic model and the deterministic code.

## Verification questions

1. Does every function and method have parameter and return type annotations?
2. Are Pydantic models — not bare dicts or plain dataclasses — used at configuration, request, model-output, and inter-module boundaries?
3. Is `mypy` or `pyright` running in CI as a gate, in strict mode for new code?
4. Is untrusted input parsed into a Pydantic model once at the edge, with the interior trusting typed objects?
5. Is the model's structured output parsed into a Pydantic model, with a validation failure failing loudly rather than being coerced?

## What to read next

- `project-structure-and-packaging.md` — the structure these types live in
- `orchestration-code.md` — typed agents and nodes
- `ai-application-architecture`, `references/model-and-inference-layer.md` — structured model output
- `test-engineering` — testing the validated boundaries
