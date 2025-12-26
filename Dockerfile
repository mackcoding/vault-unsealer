FROM golang:1.25.5-alpine AS builder

# Install build dependencies required for CGO (Bitwarden SDK)
RUN apk add --no-cache gcc musl-dev git

WORKDIR /app

# Copy dependency definitions
COPY go.mod ./

# Download dependencies (and generate go.sum if missing)
RUN go mod tidy && go mod download

# Copy source code
COPY . .

# Build the binary with CGO enabled (required for Bitwarden SDK)
# -ldflags "-s -w" strips debug information for a smaller binary
RUN CGO_ENABLED=1 go build -ldflags "-s -w" -o vault-unsealer .

# Final stage
FROM alpine:latest

# Install CA certificates for HTTPS and wget for healthcheck
RUN apk add --no-cache ca-certificates wget

WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/vault-unsealer .

# Expose the health check port
EXPOSE 8080

# Add health check using the built-in health endpoint
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run the binary
ENTRYPOINT ["/app/vault-unsealer"]
