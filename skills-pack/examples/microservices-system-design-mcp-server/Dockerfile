# Multi-stage build for the MCP server.
#
# Stage 1: build with the official Go toolchain.
# Stage 2: distroless runtime — minimal attack surface, no shell, no package manager.
#
# Re-verify base image versions against skills/mcp/00-ecosystem-facts.md and
# the distroless project's current recommendations before production use.

# Build stage
FROM golang:1.26-alpine AS build

WORKDIR /src

# Copy module files first for layer caching.
COPY go.mod go.sum* ./
RUN go mod download

# Copy source and build.
COPY . .

# Build with cgo disabled for a fully static binary.
# Strip debug info and DWARF for a smaller image.
RUN CGO_ENABLED=0 GOOS=linux go build \
    -buildvcs=false \
    -ldflags="-s -w" \
    -trimpath \
    -o /out/mcp-server \
    ./cmd/server

# Runtime stage
FROM gcr.io/distroless/static-debian12:nonroot

# Copy binary from build stage.
COPY --from=build /out/mcp-server /usr/local/bin/mcp-server

# Run as non-root (distroless nonroot UID 65532).
USER nonroot:nonroot

# Default to stdio transport. Override with MCP_TRANSPORT env var.
# When using streamablehttp, expose the port.
EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/mcp-server"]
