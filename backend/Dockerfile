# Build stage
FROM golang:1.23-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git gcc musl-dev make binutils-gold build-base

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Tidy go modules
RUN go mod tidy

# Build the application (try without CGO first for better compatibility)
RUN CGO_ENABLED=0 GOOS=linux go build -a -ldflags '-w -extldflags "-static"' -o mailman ./cmd/mailman

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1000 -S mailman && \
    adduser -u 1000 -S mailman -G mailman

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/mailman .

# Copy any necessary static files if needed
# COPY --from=builder /app/static ./static

# Change ownership
RUN chown -R mailman:mailman /app

# Switch to non-root user
USER mailman

# Expose port
EXPOSE 8080

# Run the application
CMD ["./mailman"]
