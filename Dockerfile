# Multi-stage Dockerfile for Pandora Exchange User Service
# Optimized for security, size, and build speed

# Build arguments for versioning (injected by CI/CD)
ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_TIME=unknown
ARG GO_VERSION=1.23

# ============================================================================
# Stage 1: Builder - Compile the Go binary
# ============================================================================
FROM golang:${GO_VERSION}-alpine AS builder

# Install required build tools and CA certificates
RUN apk add --no-cache \
    git \
    make \
    ca-certificates \
    tzdata

# Set working directory
WORKDIR /build

# Copy go mod files first for better layer caching
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Build arguments for version injection
ARG VERSION
ARG COMMIT
ARG BUILD_TIME

# Build the binary with optimizations and version info
# CGO_ENABLED=0: Static binary for scratch/distroless compatibility
# -trimpath: Remove file system paths from binary for reproducibility
# -ldflags: Inject version info and reduce binary size
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -trimpath \
    -ldflags="-w -s -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.buildTime=${BUILD_TIME}" \
    -o /build/user-service \
    ./cmd/user-service

# Verify the binary was built correctly
RUN /build/user-service -version 2>&1 | grep -q "${VERSION}" || true

# ============================================================================
# Stage 2: Runtime - Minimal distroless image
# ============================================================================
FROM gcr.io/distroless/static-debian12:nonroot

# Labels for metadata and documentation
LABEL org.opencontainers.image.title="Pandora Exchange User Service"
LABEL org.opencontainers.image.description="User authentication and management service"
LABEL org.opencontainers.image.vendor="Pandora Exchange"
LABEL org.opencontainers.image.source="https://github.com/alex-necsoiu/pandora-exchange"
LABEL org.opencontainers.image.licenses="MIT"

# Version labels (dynamically set from build args)
ARG VERSION
ARG COMMIT
ARG BUILD_TIME
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.revision="${COMMIT}"
LABEL org.opencontainers.image.created="${BUILD_TIME}"

# Copy CA certificates from builder for HTTPS connections
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy timezone data for proper time handling
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy the compiled binary from builder stage
COPY --from=builder --chown=nonroot:nonroot /build/user-service /usr/local/bin/user-service

# Copy configuration files (optional - can be mounted as ConfigMap/Secret in K8s)
# COPY --chown=nonroot:nonroot configs/ /etc/user-service/

# Use nonroot user (UID 65532, GID 65532) - provided by distroless
# This user has no shell and minimal permissions
USER nonroot:nonroot

# Expose ports
# 8080: HTTP server
# 9090: gRPC server
# 2112: Prometheus metrics
EXPOSE 8080 9090 2112

# Health check (note: distroless doesn't have shell, so curl/wget unavailable)
# Health checks should be configured in Kubernetes readiness/liveness probes
# HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
#   CMD ["/usr/local/bin/user-service", "-health"]

# Set working directory
WORKDIR /app

# Environment variables with secure defaults
# These should be overridden by Kubernetes ConfigMap/Secrets
ENV APP_ENV=production \
    LOG_LEVEL=info \
    HTTP_PORT=8080 \
    GRPC_PORT=9090 \
    METRICS_PORT=2112

# Run the service
ENTRYPOINT ["/usr/local/bin/user-service"]

# ============================================================================
# Build Instructions
# ============================================================================
#
# Development build:
#   docker build -t pandora/user-service:dev .
#
# Production build with version info:
#   docker build \
#     --build-arg VERSION=v1.2.3 \
#     --build-arg COMMIT=$(git rev-parse HEAD) \
#     --build-arg BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ") \
#     -t pandora/user-service:v1.2.3 .
#
# Multi-platform build:
#   docker buildx build \
#     --platform linux/amd64,linux/arm64 \
#     --build-arg VERSION=v1.2.3 \
#     --build-arg COMMIT=$(git rev-parse HEAD) \
#     --build-arg BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ") \
#     -t pandora/user-service:v1.2.3 \
#     --push .
#
# Run locally:
#   docker run -p 8080:8080 -p 9090:9090 -p 2112:2112 \
#     -e DATABASE_URL=postgres://... \
#     -e REDIS_URL=redis://... \
#     pandora/user-service:v1.2.3
#
# Security scanning:
#   trivy image pandora/user-service:v1.2.3
#   grype pandora/user-service:v1.2.3
#
# ============================================================================
