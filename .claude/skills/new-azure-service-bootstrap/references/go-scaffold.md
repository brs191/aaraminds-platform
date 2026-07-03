# Go Scaffold (Go 1.25+)

The standard scaffold for a new Go service in the Azure estate. Copy this layout, rename the module path, customize where the domain genuinely requires it.

## Repository layout

```
<service-name>/
├── README.md
├── go.mod
├── go.sum
├── Makefile
├── Dockerfile                           # Multi-stage, distroless
├── .github/workflows/
│   ├── pr.yml
│   ├── main.yml
│   └── release.yml
├── infra/
│   ├── main.tf
│   ├── variables.tf
│   └── outputs.tf
├── cmd/
│   └── server/
│       └── main.go                      # Tiny entry point
├── internal/
│   ├── app/                             # Composition root
│   │   └── app.go
│   ├── domain/                          # Domain types, business rules
│   │   ├── model.go
│   │   └── service.go
│   ├── transport/
│   │   ├── http/                        # HTTP handlers
│   │   │   ├── server.go
│   │   │   ├── handlers.go
│   │   │   └── middleware.go
│   │   └── messaging/                   # Service Bus consumers / producers
│   ├── storage/                         # Postgres / Mongo / Cosmos clients
│   │   └── postgres.go
│   ├── observability/                   # OTel setup, log setup
│   │   ├── otel.go
│   │   └── logging.go
│   └── config/                          # Env / Key Vault config loading
│       └── config.go
├── docs/
│   ├── adr/
│   ├── runbook.md
│   └── slo.md
└── testdata/                            # Test fixtures (if any)
```

**Hard rule:** no `pkg/` directory. Everything under `internal/`. If a downstream repo genuinely needs to import a type, it should call the service's API, not import the source. The Go community moved away from `pkg/` years ago; new services should not adopt it.

## `go.mod`

```go
module github.com/<org>/<svc>

go 1.25.0

require (
    github.com/Azure/azure-sdk-for-go/sdk/azidentity v1.7.0
    github.com/Azure/azure-sdk-for-go/sdk/keyvault/azsecrets v1.3.0
    github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus v1.6.0
    github.com/jackc/pgx/v5 v5.5.0
    go.opentelemetry.io/otel v1.30.0
    go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.30.0
    go.opentelemetry.io/otel/sdk v1.30.0
    go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.55.0
)
```

Re-verify versions quarterly per `../../mcp-go-server-building/references/ecosystem-facts.md`.

## `cmd/server/main.go`

```go
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/<org>/<svc>/internal/app"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	if err := app.Run(ctx, logger); err != nil {
		logger.Error("server failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
}
```

The entry point does three things: build the logger, set up signal-bounded context, call into `internal/app/Run`. That's it. No business logic, no dependency wiring, no HTTP server initialization.

**Note:** this scaffold uses `os.Stdout` because the Go service writes logs to stdout for the Container Apps log driver. **Go MCP servers using stdio transport use `os.Stderr` instead** because stdout is the MCP protocol wire — see `mcp-go-server-building` for that case.

## `internal/app/app.go`

```go
package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/<org>/<svc>/internal/config"
	"github.com/<org>/<svc>/internal/observability"
	"github.com/<org>/<svc>/internal/storage"
	httptransport "github.com/<org>/<svc>/internal/transport/http"
)

func Run(ctx context.Context, logger *slog.Logger) error {
	cfg, err := config.Load(ctx)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	shutdownOtel, err := observability.SetupOTel(ctx, cfg.OTelEndpoint, cfg.ServiceName)
	if err != nil {
		return fmt.Errorf("setup otel: %w", err)
	}
	defer shutdownOtel(context.Background())

	db, err := storage.NewPostgres(ctx, cfg.PostgresURL)
	if err != nil {
		return fmt.Errorf("connect postgres: %w", err)
	}
	defer db.Close()

	srv := httptransport.NewServer(logger, db, cfg)

	httpServer := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_ = httpServer.Shutdown(shutdownCtx)
	}()

	logger.Info("listening", slog.String("addr", cfg.ListenAddr))
	if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("listen: %w", err)
	}
	return nil
}
```

`app.Run` is the composition root: load config, wire observability, connect storage, build the transport layer, start the HTTP server, wait for shutdown. Each `internal/<package>` exposes a constructor that returns its public type. No init functions, no package-level globals.

## `internal/config/config.go` — Managed Identity + Key Vault

```go
package config

import (
	"context"
	"fmt"
	"os"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/keyvault/azsecrets"
)

type Config struct {
	ServiceName  string
	ListenAddr   string
	OTelEndpoint string
	PostgresURL  string
}

func Load(ctx context.Context) (*Config, error) {
	cfg := &Config{
		ServiceName:  envOrDefault("SERVICE_NAME", "<svc>"),
		ListenAddr:   envOrDefault("LISTEN_ADDR", ":8080"),
		OTelEndpoint: os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"),
	}

	kvURL := os.Getenv("AZURE_KEY_VAULT_URL")
	if kvURL == "" {
		return nil, fmt.Errorf("AZURE_KEY_VAULT_URL is required")
	}

	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, fmt.Errorf("azidentity: %w", err)
	}

	client, err := azsecrets.NewClient(kvURL, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("keyvault client: %w", err)
	}

	pgURL, err := client.GetSecret(ctx, "postgres-url", "", nil)
	if err != nil {
		return nil, fmt.Errorf("get postgres-url: %w", err)
	}
	cfg.PostgresURL = *pgURL.Value

	return cfg, nil
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
```

Notes:
- `azidentity.NewDefaultAzureCredential` walks: workload identity → managed identity → environment → Azure CLI. In Container Apps with system-assigned MI, it picks up the managed identity automatically.
- `postgres-url` is the *Key Vault secret name*. The actual connection string lives in Key Vault; the application has no plaintext database credentials.
- This pattern extends to every other secret: Service Bus connection, third-party API keys, signing keys.

## `internal/observability/otel.go`

```go
package observability

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

func SetupOTel(ctx context.Context, endpoint, serviceName string) (func(context.Context) error, error) {
	if endpoint == "" {
		return func(context.Context) error { return nil }, nil
	}

	exp, err := otlptracehttp.New(ctx, otlptracehttp.WithEndpoint(endpoint))
	if err != nil {
		return nil, fmt.Errorf("otlp exporter: %w", err)
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
			semconv.ServiceNamespace("aaraminds"),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("otel resource: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)

	return tp.Shutdown, nil
}
```

OpenTelemetry initialization is small but mandatory. The shutdown function is deferred from `Run` so spans flush before process exit. Without a Shutdown call, the last batch of spans is lost — common cause of "the alert fired but I see no trace."

## `internal/transport/http/server.go` — instrumented HTTP server

```go
package http

import (
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.healthz)
	mux.HandleFunc("GET /readyz", s.readyz)
	mux.HandleFunc("POST /v1/<resource>", s.createResource)
	// ... other routes

	return otelhttp.NewHandler(mux, "<svc>")
}
```

`otelhttp.NewHandler` wraps the router with auto-instrumentation: every incoming request becomes a server span; client calls via `otelhttp.NewTransport` continue the trace.

## `Dockerfile` (multi-stage, distroless)

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
    -o /out/server \
    ./cmd/server

# Runtime stage — distroless
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/server /usr/local/bin/server
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/server"]
```

Image size is ~15-20 MB. Distroless means no shell, no package manager, no extras to exploit. `-ldflags="-s -w"` strips debug info; `-trimpath` removes build-machine paths for reproducibility.

## `Makefile`

```make
.PHONY: build test lint run docker-build docker-run

build:
	go build -o bin/server ./cmd/server

test:
	go test -race -count=1 ./...

lint:
	gofmt -l . | tee /dev/stderr | (! grep .)
	go vet ./...
	govulncheck ./...

run:
	go run ./cmd/server

docker-build:
	docker build -t <svc>:dev .

docker-run:
	docker run --rm -p 8080:8080 \
		-e AZURE_KEY_VAULT_URL=$$AZURE_KEY_VAULT_URL \
		<svc>:dev
```

## What ships in the scaffold but is empty

- `internal/domain/` — package skeleton, no business logic.
- `internal/transport/http/handlers.go` — health and readiness handlers exist; the team adds the rest.
- `docs/adr/0001-initial-architecture.md` — first ADR.
- `docs/runbook.md` — empty headings: "On-call rotation," "Common alerts," "Manual recovery procedures."
- `docs/slo.md` — empty SLO definition; filled in first sprint.

## Anti-patterns (violations of the scaffold)

- **Env-var secrets** — `os.Getenv("PG_PASSWORD")`. Use Key Vault via `azidentity` + `azsecrets`.
- **`pkg/` directory** — never. Use `internal/` for everything.
- **`init()` functions for setup** — initialization happens in `app.Run` explicitly; no hidden init.
- **Package-level globals** — no shared mutable state. Inject dependencies via constructors.
- **Gin / Echo / Fiber for trivial HTTP** — `net/http` plus `http.ServeMux` (Go 1.22+ has path patterns) is sufficient. Reach for a framework only for genuine routing complexity (versioned APIs with rich middleware composition).
- **GORM / sqlboiler** — write SQL; use `pgx`. The ORM you save in writing is paid back in debugging Postgres-specific behavior the ORM abstracted.
- **`log` package or `logrus`** — `log/slog` from stdlib.
- **`os.Stdout` for MCP-server stdio transport** — MCP stdio servers write to `os.Stderr`. This scaffold is for regular services; MCP servers follow `mcp-go-server-building`.

## Verification

A new Go service passes the scaffold check if:

1. Module path matches the repo path
2. `cmd/server/main.go` is tiny (under 30 lines)
3. `internal/app/app.go` is the composition root; no business logic in `main`
4. Zero env-var secrets; Key Vault via `azidentity` + `azsecrets`
5. OpenTelemetry initialized at startup with `Shutdown` deferred
6. `otelhttp.NewHandler` wraps the router; `otelhttp.NewTransport` wraps outbound HTTP clients
7. `log/slog` JSON to stdout; trace and span IDs propagate from context
8. Dockerfile is multi-stage with distroless runtime
9. GitHub Actions workflows use OIDC
10. `docs/adr/`, `docs/runbook.md`, `docs/slo.md` exist
