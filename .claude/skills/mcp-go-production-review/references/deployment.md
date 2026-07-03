# Skill — MCP-Go Deployment

## Purpose

Package and deploy MCP-Go servers in production-shaped ways. Most MCP servers ship as containers; some ship as local binaries embedded in IDE plugins or workflow tools. This skill is about producing artifacts that are small, secure, observable, and rollback-able — and avoiding the predictable failure modes of each deployment target.

## Deployment targets

| Target | What it is | When to choose |
|---|---|---|
| **Container Apps** | Azure managed serverless container platform | Greenfield Azure MCP services; scale-to-zero useful |
| **AKS** | Managed Kubernetes | Existing K8s investment, complex orchestration |
| **App Service** | Azure managed app platform | Legacy alignment; not preferred for new MCP work |
| **Function App** | Serverless functions | Event-driven MCP servers (rare) |
| **Local binary** | Distributed as a binary for `claude code` etc. | Agent tooling, local execution |
| **Container Apps Job** | One-shot container run | Stdio servers invoked per-job by an orchestrator |

The default for cloud-hosted MCP service work is **Container Apps**. The default for agent-embedded MCP is **local binary**.

## Build: multi-stage Dockerfile

Pin to a specific Go version. Build static. Strip debug. Use distroless runtime.

```dockerfile
# Build stage
FROM golang:1.26-alpine AS build
WORKDIR /src

COPY go.mod go.sum* ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -trimpath \
    -o /out/mcp-server \
    ./cmd/server

# Runtime stage
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/mcp-server /usr/local/bin/mcp-server
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/mcp-server"]
```

Properties of a good MCP container image:
- **Tiny.** Distroless static gets you under 20 MB usually.
- **Non-root.** `nonroot` user, no shell, no package manager. Smaller attack surface.
- **No CGO.** `CGO_ENABLED=0` ensures the binary doesn't link against libc; cross-deployment safe.
- **Stripped.** `-ldflags="-s -w"` removes debug info and DWARF tables.
- **Reproducible.** `-trimpath` removes build-machine paths so the same source produces the same binary.

## Deployment manifest patterns

### Container Apps (preferred)

```yaml
properties:
  configuration:
    activeRevisionsMode: Multiple
    ingress:
      external: true
      targetPort: 8080
      transport: auto
      traffic:
        - revisionName: mcp-server--rev2
          weight: 90
        - revisionName: mcp-server--rev1
          weight: 10
    secrets:
      - name: mcp-jwt-signing-key
        keyVaultUrl: https://...
        identity: system
  template:
    containers:
      - name: mcp-server
        image: <registry>/mcp-server:1.2.0
        env:
          - name: MCP_TRANSPORT
            value: streamablehttp
          - name: PORT
            value: "8080"
        resources:
          cpu: 0.5
          memory: 1Gi
        probes:
          - type: Readiness
            httpGet: { path: /healthz, port: 8080 }
            periodSeconds: 5
          - type: Liveness
            httpGet: { path: /healthz, port: 8080 }
            initialDelaySeconds: 10
    scale:
      minReplicas: 1
      maxReplicas: 10
      rules:
        - name: http-rule
          http:
            metadata:
              concurrentRequests: "50"
```

Notes:
- **Multiple revisions** enable canary / blue-green via traffic weights.
- **System-assigned managed identity** for Key Vault and other Azure resource access.
- **Health probes** on `/healthz`, not on `/`. The MCP wire isn't a health endpoint.
- **Resource limits** are realistic (0.5 vCPU, 1 GiB) — most MCP servers are not CPU-bound.

### AKS

Standard Deployment + Service + HPA + NetworkPolicy. Bind a workload identity to the deployment for Azure resource access. Use a separate namespace per environment.

### Local binary

Build for each target OS/arch:

```bash
GOOS=darwin GOARCH=arm64 go build -o dist/mcp-server-darwin-arm64 ./cmd/server
GOOS=darwin GOARCH=amd64 go build -o dist/mcp-server-darwin-amd64 ./cmd/server
GOOS=linux  GOARCH=amd64 go build -o dist/mcp-server-linux-amd64  ./cmd/server
GOOS=windows GOARCH=amd64 go build -o dist/mcp-server-windows-amd64.exe ./cmd/server
```

Sign binaries on macOS and Windows for distribution. Provide a checksum file (`SHA256SUMS`).

## Configuration and secrets

- **All secrets via Key Vault + Managed Identity.** No secrets in container env vars or config files. Use the CSI driver to mount or fetch via SDK at runtime.
- **Non-secret config via env vars.** Transport choice, log level, port, feature flags.
- **No secrets in logs.** Sanitise structured-log fields if you must log a value derived from a secret.

## Rollout strategy

For Container Apps, AKS, App Service: blue-green via traffic shifting.

1. Deploy new revision (`rev2`) with 0% traffic.
2. Run smoke tests against `rev2`'s direct URL.
3. Shift 10% traffic.
4. Monitor error rate, P99 latency, tool-call success per minute for 10–30 minutes.
5. Shift 50%; monitor.
6. Shift 100%; monitor.
7. Retire `rev1` after a rollback window (24–72 hours).

If any stage shows regression, shift back to the previous revision instantly. Don't keep going "to see if it stabilises".

## Operational health endpoints

Beside the MCP wire, expose:

- `GET /healthz` — liveness/readiness probe. Returns 200 if the server is up and able to handle requests; 503 if shutting down or unhealthy.
- `GET /metrics` (optional) — Prometheus-style metrics. Only if you scrape with Prometheus; Azure Monitor doesn't need this.

Health endpoints are not MCP endpoints. Don't pretend they are.

## Logging in production

- **stdio transport:** logs go to stderr (the orchestrator captures them).
- **HTTP transport:** logs go to stderr (the container captures them); the orchestrator forwards to Log Analytics or equivalent.
- **Structured JSON only.** Levels: Info/Warn/Error. Debug off in production unless investigating.
- **Sample if volume is high.** A high-traffic MCP server can emit hundreds of MB of logs per day; sample at the application or collector layer.

## Common failure modes

- **stdout logging on stdio.** Already covered; this kills the protocol. Detection: client sees malformed messages. Fix: stderr only.
- **Container starts but unhealthy.** Healthcheck returns 200 even though the server can't serve tools. Detection: traffic shifted to the new revision, errors spike. Fix: `/healthz` should actually probe the server's ability to handle a tool call, not just return 200 from a static handler.
- **Image pulled at every cold start.** Image is in a slow registry, container start latency is unacceptable. Detection: scale-to-zero responses take 10+ seconds. Fix: registry geo-redundancy; image pre-pull on the platform.
- **Resource limits too tight.** Sidecar or runtime overhead pushes the container past memory limit and OOM kills it. Detection: pod restart count climbs. Fix: profile memory under load; set limits to P99 usage + 20% headroom.
- **Secrets in image layers.** A debug build leaked an API key into an image. Detection: image scanner flags it. Fix: secrets never written to disk during build; rebuild from clean source.
- **No rollback plan.** "We just push fix-forward." Detection: incident post-mortems mention long restoration times. Fix: keep last 2 revisions, scripted rollback.

## Verification questions

1. Is the production image distroless, non-root, and stripped?
2. Are health probes hitting an actual readiness endpoint, not a static 200?
3. Are secrets in Key Vault, accessed via Managed Identity — no plaintext anywhere?
4. Can you roll back the current production to the previous revision in under one minute?
5. Are resource limits set to actual measured usage plus a margin, not arbitrary defaults?
6. Does the rollout strategy include a soak period at each traffic percentage?

## What to read next

- `../../mcp-go-server-building/references/server-basics.md` — what `main.go` should look like
- `../../mcp-go-server-building/references/transport-selection.md` — transport choice per deployment target
- `../../mcp-go-server-building/references/enterprise-security.md` — Managed Identity, Key Vault, mTLS
- `cicd-quality-gates.md` — CI gates before deploy
