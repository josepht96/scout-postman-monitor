# Stage 1: Node.js dependencies
FROM node:18-alpine AS node-builder

WORKDIR /app/newman

# Copy package files and install Newman
COPY newman/package.json newman/package-lock.json* ./
RUN npm ci --only=production

# Copy Newman executor script
COPY newman/executor.js ./

# Stage 2: Go builder
FROM golang:1.24-alpine AS go-builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY cmd/ cmd/
COPY internal/ internal/
COPY web/ web/

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o scout ./cmd/scout

# Stage 3: Final runtime image
FROM node:18-alpine

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Copy Node.js dependencies from node-builder
COPY --from=node-builder /app/newman /app/newman

# Copy Go binary from go-builder
COPY --from=go-builder /app/scout /app/scout

# Copy web assets (embedded in binary, but keeping for reference)
COPY web/ /app/web/

# Create collections directory
RUN mkdir -p /app/collections

# Set environment variables
ENV COLLECTIONS_DIR=/app/collections
ENV NEWMAN_SCRIPT_PATH=/app/newman/executor.js
ENV PORT=8080
ENV INTERVAL=30s

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run as non-root user
RUN addgroup -g 1001 scout && \
    adduser -D -u 1001 -G scout scout && \
    chown -R scout:scout /app

USER scout

# Run the application
CMD ["/app/scout"]
