FROM golang:alpine AS builder

# Install build dependencies
RUN apk add --no-cache gcc musl-dev

WORKDIR /build
COPY unsealer.go .

# Build the binary
RUN go mod init unsealer && \
    go mod tidy && \
    CGO_ENABLED=1 go build -ldflags '-extldflags "-static"' -o unsealer

FROM alpine:latest

# Install Bitwarden CLI
RUN apk add --no-cache curl unzip && \
    curl -L "https://vault.bitwarden.com/download/?app=cli&platform=linux" -o bw.zip && \
    unzip bw.zip && \
    chmod +x bw && \
    mv bw /usr/local/bin/ && \
    rm bw.zip

WORKDIR /app
COPY --from=builder /build/unsealer .

ENTRYPOINT ["/app/unsealer"]
