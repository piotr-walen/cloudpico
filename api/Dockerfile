# Build stage
FROM golang:1.25-alpine AS builder

# Install build dependencies for CGO (required for SQLite)
RUN apk add --no-cache gcc musl-dev

WORKDIR /build

# Copy shared and tools modules first (needed for replace directives)
COPY shared ./shared
COPY tools ./tools

# Copy api module files and source code together
COPY api ./api

# Build the application
# CGO_ENABLED=1 is required because we use github.com/mattn/go-sqlite3,
# which is a CGO wrapper around the SQLite C library
WORKDIR /build/api
RUN go mod tidy
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o cloudpico-api ./cmd

# Runtime stage
FROM alpine:latest

WORKDIR /app

RUN apk --no-cache add ca-certificates  wget

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /build/api/cloudpico-api .

# Create directory for SQLite database
RUN mkdir -p /app/data

# Expose port
EXPOSE 8080

# Run the api
CMD ["./cloudpico-api"]
