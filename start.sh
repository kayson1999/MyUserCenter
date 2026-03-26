#!/bin/bash

# ============================================
# MyUserCenter 启动脚本
# ============================================

set -e

APP_NAME="myusercenter"
APP_DIR="$(cd "$(dirname "$0")" && pwd)"
PID_FILE="$APP_DIR/$APP_NAME.pid"
LOG_FILE="$APP_DIR/$APP_NAME.log"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # 无颜色

info()  { echo -e "${GREEN}[INFO]${NC} $1"; }
warn()  { echo -e "${YELLOW}[WARN]${NC} $1"; }
error() { echo -e "${RED}[ERROR]${NC} $1"; }

# 编译项目
build() {
    info "正在编译 $APP_NAME ..."
    cd "$APP_DIR"
    go build -o "$APP_NAME" .
    info "编译完成 ✅"
}

# 启动服务
start() {
    if [ -f "$PID_FILE" ]; then
        local pid
        pid=$(cat "$PID_FILE")
        if kill -0 "$pid" 2>/dev/null; then
            warn "$APP_NAME 已在运行中 (PID: $pid)"
            return 1
        else
            rm -f "$PID_FILE"
        fi
    fi

    # 如果可执行文件不存在，先编译
    if [ ! -f "$APP_DIR/$APP_NAME" ]; then
        build
    fi

    info "正在启动 $APP_NAME ..."
    cd "$APP_DIR"
    nohup "./$APP_NAME" > "$LOG_FILE" 2>&1 &
    local pid=$!
    echo "$pid" > "$PID_FILE"
    sleep 1

    if kill -0 "$pid" 2>/dev/null; then
        info "$APP_NAME 启动成功 ✅ (PID: $pid)"
        info "日志文件: $LOG_FILE"
    else
        error "$APP_NAME 启动失败，请查看日志: $LOG_FILE"
        rm -f "$PID_FILE"
        return 1
    fi
}

# 停止服务
stop() {
    if [ ! -f "$PID_FILE" ]; then
        warn "$APP_NAME 未在运行"
        return 0
    fi

    local pid
    pid=$(cat "$PID_FILE")
    if kill -0 "$pid" 2>/dev/null; then
        info "正在停止 $APP_NAME (PID: $pid) ..."
        kill "$pid"
        # 等待进程退出（最多 10 秒）
        local count=0
        while kill -0 "$pid" 2>/dev/null && [ $count -lt 10 ]; do
            sleep 1
            count=$((count + 1))
        done
        if kill -0 "$pid" 2>/dev/null; then
            warn "进程未正常退出，强制终止 ..."
            kill -9 "$pid"
        fi
        info "$APP_NAME 已停止 ✅"
    else
        warn "$APP_NAME 进程不存在 (PID: $pid)"
    fi
    rm -f "$PID_FILE"
}

# 重启服务
restart() {
    stop
    sleep 1
    start
}

# 查看状态
status() {
    if [ -f "$PID_FILE" ]; then
        local pid
        pid=$(cat "$PID_FILE")
        if kill -0 "$pid" 2>/dev/null; then
            info "$APP_NAME 正在运行 (PID: $pid)"
            return 0
        else
            warn "$APP_NAME 进程已退出 (PID 文件残留)"
            rm -f "$PID_FILE"
            return 1
        fi
    else
        warn "$APP_NAME 未在运行"
        return 1
    fi
}

# 查看日志
logs() {
    if [ -f "$LOG_FILE" ]; then
        tail -f "$LOG_FILE"
    else
        warn "日志文件不存在: $LOG_FILE"
    fi
}

# 使用帮助
usage() {
    echo "用法: $0 {build|start|stop|restart|status|logs}"
    echo ""
    echo "  build    - 编译项目"
    echo "  start    - 启动服务（后台运行）"
    echo "  stop     - 停止服务"
    echo "  restart  - 重启服务"
    echo "  status   - 查看运行状态"
    echo "  logs     - 查看实时日志"
}

# 主入口
case "${1}" in
    build)   build   ;;
    start)   start   ;;
    stop)    stop    ;;
    restart) restart ;;
    status)  status  ;;
    logs)    logs    ;;
    *)       usage   ;;
esac
