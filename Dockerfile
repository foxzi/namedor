# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /build

# Install build dependencies
RUN apk add --no-cache git ca-certificates gcc musl-dev

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build arguments for version info
ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_DATE=unknown

# Build binary
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo \
    -ldflags="-w -s -X main.Version=${VERSION} -X main.GitCommit=${COMMIT} -X main.BuildDate=${BUILD_DATE}" \
    -o namedot ./cmd/namedot

# Runtime stage
FROM alpine:latest

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata wget

# Copy binary from builder
COPY --from=builder /build/namedot .

# Copy GeoIP databases
COPY geoipdb ./geoipdb

# Copy default config (can be overridden via volume mount)
COPY examples/config.yaml ./config.yaml

# Create directory for SQLite database
RUN mkdir -p /data

# Expose DNS and REST API ports
EXPOSE 53/udp 53/tcp 8080/tcp

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -q --spider http://localhost:8080/health || exit 1

# Run as non-root user
RUN addgroup -g 1000 geodns && \
    adduser -D -u 1000 -G geodns geodns && \
    chown -R geodns:geodns /app /data

USER geodns

ENTRYPOINT ["/app/namedot"]
