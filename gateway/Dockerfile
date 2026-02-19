# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /build

# Install build dependencies (not needed for CGO but keeping for consistency)
RUN apk add --no-cache gcc musl-dev

# Copy shared module first (needed for replace directive)
COPY shared ./shared

# Copy gateway module files and source code together
COPY gateway ./gateway

# Download dependencies and build
WORKDIR /build/gateway
RUN CGO_ENABLED=0 GOOS=linux go build -a -o cloudpico-gateway ./cmd

# Runtime stage
FROM alpine:latest

WORKDIR /app

RUN apk --no-cache add ca-certificates

# Copy the binary from builder
COPY --from=builder /build/gateway/cloudpico-gateway .

# Run the gateway
CMD ["./cloudpico-gateway"]
