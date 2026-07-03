# Service Runtime Concerns

This reference covers the operational surface of a Python service — configuration, secrets, logging, telemetry, shutdown. These are not the service's logic, but they determine whether it can be run in production.

## Configuration — one typed Settings object

All configuration goes through one typed object — a Pydantic Settings model — constructed once at startup, sourced from the environment, and *validated on construction*. A missing or malformed setting then fails at startup with a clear message, not at the first request that happens to need it. Scattered `os.environ` reads through the code are untyped, unvalidated, and impossible to inventory; one Settings object is typed, validated, and the single place to see everything the service needs.

## Secrets — Key Vault via managed identity

Secrets — model API keys, search keys, connection strings — live in Azure Key Vault and are read at startup via **managed identity** (`DefaultAzureCredential` from the `azure-identity` library), never from environment variables, never from code or an image layer. A secret in an env var is visible in the process listing, in crash dumps, in logs; a secret baked into the image is in the registry forever. Managed identity means the service authenticates as itself and no secret is stored anywhere the service controls. This is `azure-microservices-security`'s discipline — the Python service follows it, with no exemption for being an AI service.

## Structured logging

Logs are structured JSON, not formatted strings — one event per line with fields (timestamp, level, message, and context such as a request or trace id), so they are queryable in Log Analytics rather than only greppable. Never log a secret, a raw model prompt that might contain one, or PII. A structured log is data an operator can filter and aggregate; a formatted-string log is text they can only search.

## OpenTelemetry — the GenAI conventions

Instrument the service with OpenTelemetry, and instrument the model calls with the **GenAI semantic conventions**, so model latency, token counts, and cost appear as spans in the same trace as the HTTP and tool spans. Propagate trace context across the service's boundaries so one request is one trace end to end (`ai-application-architecture`, `references/serving-topology.md`). The telemetry backbone is `azure-microservices-observability`; this is the Python service's part in it.

## Graceful shutdown and health

The service runs on Container Apps, which stops and restarts it — on scale-in, on deploy, on node movement. Handle the shutdown signal: stop accepting new requests, let in-flight requests finish within a grace window, close connections and flush telemetry, then exit. Expose a health endpoint so the platform knows when the service is ready and when it is live. A service that exits hard on the stop signal drops in-flight work; one with no health check receives traffic before it is ready.

## Verification questions

1. Is all configuration one typed Pydantic Settings object, validated at startup?
2. Are secrets read from Key Vault via managed identity — never environment variables, code, or the image?
3. Are logs structured JSON, with no secret, raw prompt, or PII logged?
4. Is the service instrumented with OpenTelemetry, model calls following the GenAI conventions, one trace per request?
5. Does the service shut down gracefully — drain in-flight work, flush telemetry — and expose a health endpoint?

## What to read next

- `project-structure-and-packaging.md` — the composition root that wires these
- `azure-microservices-security` — Key Vault and managed identity
- `azure-microservices-observability` — the telemetry backbone
- `ai-application-architecture`, `references/serving-topology.md` — tracing across tiers
