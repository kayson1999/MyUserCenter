# MyUserCenter 用户中心

一个轻量级的多租户用户中心服务，提供统一的用户注册、登录、Token 管理和租户管理能力，适用于多个前端应用共享同一套用户体系的场景。

## ✨ 特性

- 🏢 **多租户架构** — 一套用户体系，多个应用共享，每个租户独立管理用户角色和扩展数据
- 🔐 **JWT 认证** — 基于 JWT 的无状态认证，支持 Token 刷新和黑名单机制
- ❄️ **雪花算法 ID** — 用户 ID 采用 Snowflake 算法生成，分布式唯一、趋势递增，前端以字符串形式返回避免精度丢失
- 🗄️ **双数据库支持** — 通过环境变量一键切换 MySQL / SQLite
- 🛡️ **安全防护** — 内置请求限流、CORS 跨域、密码 bcrypt 加密、HMAC 签名验证
- 📝 **请求/响应日志** — 中间件自动记录各接口的输入（请求头、请求体）和输出（状态码、响应体），敏感信息自动脱敏
- 📋 **登录日志** — 自动记录用户登录/登出/注册行为
- 📂 **日志滚动** — 日志按天自动滚动，支持配置保留天数，自动清理过期日志
- 🧹 **自动清理** — 定时清理过期 Token 黑名单

## 🛠️ 技术栈

| 组件 | 技术 |
|------|------|
| 语言 | Go 1.21+ |
| Web 框架 | Gin |
| ORM | GORM |
| 数据库 | MySQL / SQLite |
| 认证 | JWT (golang-jwt) |
| 密码加密 | bcrypt |
| ID 生成 | Snowflake 雪花算法 |

## 📁 项目结构

```
MyUserCenter/
├── main.go              # 程序入口，路由注册
├── start.sh             # 启动/停止/重启脚本
├── api.http             # API 测试文件（VS Code REST Client）
├── Dockerfile           # Docker 镜像构建（多阶段）
├── docker-compose.yml   # Docker Compose 编排部署
├── .env                 # 环境变量配置（需自行创建）
├── .env.example         # 环境变量示例模板
├── .dockerignore        # Docker 构建忽略规则
├── .gitignore           # Git 忽略规则
├── go.mod               # Go 模块依赖
├── config/
│   └── config.go        # 配置加载（环境变量）
├── database/
│   └── db.go            # 数据库初始化（MySQL/SQLite）
├── model/
│   └── model.go         # 数据模型定义
├── handler/
│   ├── auth.go          # 认证接口（注册/登录/登出/刷新/验证）
│   ├── user.go          # 用户接口（个人信息/修改密码）
│   ├── tenant.go        # 租户管理接口
│   └── health.go        # 健康检查接口
├── middleware/
│   ├── auth.go          # JWT 认证 & 租户 HMAC 签名校验中间件
│   ├── cors.go          # CORS 跨域中间件
│   ├── logger.go        # 请求/响应日志中间件（记录输入输出）
│   └── ratelimit.go     # 请求限流中间件
├── logger/
│   └── logger.go        # 日志系统（按天滚动、自动清理）
├── util/
│   ├── token.go         # JWT 签发/验证工具
│   └── snowflake.go     # 雪花算法 ID 生成器
├── logs/                # 日志文件目录（自动创建）
└── data/                # SQLite 数据库文件目录
```

## 🚀 快速开始

### 1. 克隆项目

```bash
git clone <your-repo-url>
cd MyUserCenter
```

### 2. 配置环境变量

复制并编辑 `.env` 文件：

```bash
cp .env.example .env
```

```env
# 服务端口
PORT=4000

# JWT 配置
JWT_SECRET=your_jwt_secret_here
JWT_EXPIRES_IN=7d

# 内部通信密钥（服务间调用）
INTERNAL_SECRET=your_internal_secret_here

# 数据库类型：mysql 或 sqlite（默认 mysql）
DB_TYPE=sqlite

# MySQL 配置（DB_TYPE=mysql 时生效）
DB_HOST=127.0.0.1
DB_PORT=3306
DB_USER=root
DB_PASSWORD=root
DB_NAME=usercenter

# SQLite 配置（DB_TYPE=sqlite 时生效）
DB_PATH=./data/usercenter.db

# 日志配置
LOG_TO_FILE=true
LOG_DIR=./logs
LOG_FILE_PREFIX=usercenter
LOG_MAX_DAYS=30

# 雪花算法节点 ID（0~1023，多实例部署时每个实例设置不同值）
SNOWFLAKE_NODE_ID=1
```

### 3. 启动服务

**使用启动脚本（推荐）：**

```bash
chmod +x start.sh

# 编译
./start.sh build

# 启动（后台运行）
./start.sh start

# 查看状态
./start.sh status

# 查看实时日志
./start.sh logs

# 重启
./start.sh restart

# 停止
./start.sh stop
```

**直接运行：**

```bash
go build -o myusercenter .
./myusercenter
```

服务启动后访问 `http://localhost:4000/health` 验证是否正常。

## 🐳 Docker 部署

### SQLite 模式（轻量，无需额外数据库）

```bash
# 1. 准备配置
cp .env.example .env
# 编辑 .env，确保 DB_TYPE=sqlite

# 2. 构建并启动
docker compose up -d usercenter

# 3. 查看日志
docker compose logs -f usercenter
```

### MySQL 模式（生产推荐）

```bash
# 1. 准备配置
cp .env.example .env
# 编辑 .env，设置：
#   DB_TYPE=mysql
#   DB_HOST=mysql（容器内使用服务名）
#   DB_PASSWORD=your_password

# 2. 启动应用 + MySQL
docker compose --profile mysql up -d

# 3. 查看日志
docker compose logs -f
```

### 常用命令

```bash
# 查看运行状态
docker compose ps

# 停止服务
docker compose down

# 停止并清除数据卷（⚠️ 会删除数据库数据）
docker compose down -v

# 重新构建镜像
docker compose build --no-cache
```

> **说明**：MySQL 服务使用 `profiles` 机制，仅在 `--profile mysql` 时启动，SQLite 模式下不会拉取 MySQL 镜像。数据通过 Docker Volume 持久化存储。

## 📡 API 文档

### 认证接口 `/auth`

| 方法 | 路径 | 说明 | 认证 | 租户 |
|------|------|------|------|------|
| POST | `/auth/register` | 用户注册 | ❌ | ✅ 必须 |
| POST | `/auth/login` | 用户登录 | ❌ | ✅ 必须 |
| GET | `/auth/verify` | 验证 Token | ✅ | 可选 |
| POST | `/auth/logout` | 用户登出 | ✅ | ❌ |
| POST | `/auth/refresh` | 刷新 Token | ✅ | ❌ |

### 用户接口 `/user`

| 方法 | 路径 | 说明 | 认证 |
|------|------|------|------|
| GET | `/user/profile` | 获取个人信息 | ✅ |
| PUT | `/user/profile` | 更新个人信息 | ✅ |
| PUT | `/user/password` | 修改密码 | ✅ |

### 租户管理接口 `/tenant`（需内部密钥）

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/tenant/register` | 注册新租户 |
| GET | `/tenant/list` | 查询所有租户 |
| GET | `/tenant/:appId/secret` | 查询租户密钥 |
| PUT | `/tenant/:appId/status` | 启用/禁用租户 |
| PUT | `/tenant/:appId/user/:userId/role` | 设置用户角色 |
| PUT | `/tenant/:appId/user/:userId/extra` | 更新用户扩展数据 |
| GET | `/tenant/:appId/users` | 查询租户下用户列表 |

### 健康检查

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/health` | 基础健康检查 |
| GET | `/health/stats` | 详细统计信息 |

### 请求头说明

| Header | 说明 | 必须 | 示例 |
|--------|------|------|------|
| `Authorization` | JWT Token（Bearer 格式） | 认证接口必须 | `Bearer <token>` |
| `X-App-Id` | 租户应用 ID | 租户接口必须 | `worker` |
| `X-App-Sign` | HMAC-SHA256 签名（可选，提供则验证） | 可选 | `a1b2c3...` |
| `X-Timestamp` | 请求时间戳（秒），与签名配合使用，5 分钟内有效 | 签名时必须 | `1711526400` |
| `X-Internal-Secret` | 内部通信密钥（租户管理接口） | 管理接口必须 | `your_internal_secret` |

> **签名算法**：`HMAC-SHA256(app_secret, timestamp + app_id + request_body)`
>
> 签名验证为可选项，仅在同时提供 `X-App-Sign` 和 `X-Timestamp` 时触发。不提供签名时，仅验证 `X-App-Id` 对应的租户是否存在且启用。

### 请求示例

**注册用户：**

```bash
curl -X POST http://localhost:4000/auth/register \
  -H "Content-Type: application/json" \
  -H "X-App-Id: worker" \
  -d '{"username": "testuser", "password": "123456", "nickname": "测试用户"}'
```

**用户登录：**

```bash
curl -X POST http://localhost:4000/auth/login \
  -H "Content-Type: application/json" \
  -H "X-App-Id: worker" \
  -d '{"username": "testuser", "password": "123456"}'
```

**获取个人信息：**

```bash
curl http://localhost:4000/user/profile \
  -H "Authorization: Bearer <token>"
```

**注册新租户（内部接口）：**

```bash
curl -X POST http://localhost:4000/tenant/register \
  -H "Content-Type: application/json" \
  -H "X-Internal-Secret: your_internal_secret" \
  -d '{"app_id": "myapp", "name": "我的应用", "description": "应用描述", "allowed_origins": ["http://localhost:3000"]}'
```

## 📋 数据模型

- **Tenant（租户）** — 接入的应用服务，包含 app_id、app_secret、允许的跨域源
- **User（用户）** — 全局用户，ID 为雪花算法生成的 `int64`（JSON 中以字符串形式返回），包含用户名、密码哈希、昵称、头像等
- **TenantUser（租户用户关联）** — 用户在各租户下的角色、状态、扩展数据
- **TokenBlacklist（Token 黑名单）** — 已登出/已刷新的 Token 记录
- **LoginLog（登录日志）** — 用户行为日志（注册/登录/登出）

> **关于用户 ID**：用户 ID 采用雪花算法生成（如 `"162911976203227136"`），在 JSON 响应中以**字符串**形式返回，避免 JavaScript `Number.MAX_SAFE_INTEGER`（2^53 - 1）精度丢失问题。

## 📊 日志系统

### 日志文件

- 日志按天自动滚动，文件名格式：`{prefix}-{date}.log`（如 `usercenter-2026-03-27.log`）
- 同时输出到控制台和文件
- 支持配置保留天数，自动清理过期日志

### 请求/响应日志

每次接口调用自动记录输入输出，格式示例：

```
────────────────────────────────────────
  ▶ 请求: POST /auth/login
  ▶ 来源IP: 127.0.0.1
  ▶ 请求头: X-App-Id=worker; Content-Type=application/json; Authorization=***
  ▶ 请求体: {"username":"testuser","password":"123456"}
  ◀ 状态码: 200 ✅
  ◀ 响应体: {"token":"eyJhbGciOi...","user":{...}}
  ⏱ 耗时: 12.345ms
────────────────────────────────────────
```

**特性：**
- 敏感请求头自动脱敏（`Authorization`、`X-Internal-Secret`、`X-App-Sign` 显示为 `***`）
- 请求体/响应体超过 4KB 自动截断
- 仅记录 `application/json`、`application/x-www-form-urlencoded`、`text/plain` 类型的请求体
- 健康检查接口（`/health`、`/health/stats`）不记录日志

## 📄 License

MIT
