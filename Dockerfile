# ============================================
# 阶段一：编译
# 使用 CGO 支持 SQLite（mattn/go-sqlite3 需要 CGO）
# ============================================
FROM golang:1.21-bullseye AS builder

WORKDIR /build

# 先复制依赖文件，利用 Docker 缓存层
COPY go.mod go.sum ./
RUN go mod download

# 复制源码并编译
COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -o myusercenter .

# ============================================
# 阶段二：运行
# ============================================
FROM debian:bullseye-slim

# 安装运行时依赖（SQLite 需要 libc）
RUN apt-get update && \
    apt-get install -y --no-install-recommends ca-certificates && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /app

# 从编译阶段复制二进制文件
COPY --from=builder /build/myusercenter .

# 创建数据目录（SQLite 模式使用）
RUN mkdir -p /app/data

# 暴露端口
EXPOSE 4000

# 健康检查
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:4000/health || exit 1

# 启动服务
CMD ["./myusercenter"]
