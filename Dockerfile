# ============================================
# Stage 1: Build with CGO for SQLite support
# ============================================

FROM golang:1.21-alpine AS builder

# Configure Alpine package mirror for faster downloads
RUN sed -i "s/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g" /etc/apk/repositories

# Configure Go proxy for faster dependency downloads
ENV GOPROXY=https://goproxy.cn,direct

# Install CGO dependencies (sqlite3 requires CGO)
RUN apk update
RUN apk add --no-cache gcc musl-dev build-base sqlite-dev

# Set SQLite3 compilation environment variables
ENV CGO_CFLAGS="-DUSE_PREAD64=0 -DHAVE_PREAD64=0 -DHAVE_PWRITE64=0 -D_LARGEFILE64_SOURCE=1"

WORKDIR /build

# Copy dependency files first to leverage Docker cache
COPY go.mod go.sum ./
RUN go mod download

# Copy source code and build
COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -tags musl -o myusercenter .

# ============================================
# Stage 2: Runtime
# ============================================

FROM alpine:3.19

# Configure Alpine package mirror for faster downloads
RUN sed -i "s/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g" /etc/apk/repositories

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache ca-certificates curl

# Copy binary from build stage
COPY --from=builder /build/myusercenter .

# Create data directory (for SQLite mode)
RUN mkdir -p /app/data

# Expose port
EXPOSE 4000

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:4000/health || exit 1

# Start service
CMD ["./myusercenter"]
