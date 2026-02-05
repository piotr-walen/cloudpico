# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
# CGO_ENABLED=1 is required because we use github.com/mattn/go-sqlite3,
# which is a CGO wrapper around the SQLite C library
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o cloudpico-server ./cmd

# Runtime stage
FROM alpine:latest

WORKDIR /app

RUN apk --no-cache add ca-certificates  wget

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /build/cloudpico-server .

# Copy static files
COPY --from=builder /build/static ./static

# Create directory for SQLite database
RUN mkdir -p /app/data

# Expose port
EXPOSE 8080

# Run the server
CMD ["./cloudpico-server"]
