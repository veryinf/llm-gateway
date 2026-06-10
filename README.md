# LLM Gateway

统一 LLM API 网关服务，为公司员工提供标准化的 LLM API 接入点，同时集成统计和审计功能。

## 功能特性

- **统一 API 入口**: OpenAI 兼容的 API 端点，直接替换 base_url 即可接入
- **多 Provider 支持**: OpenAI, Azure, Anthropic, DeepSeek, 通义千问, Kimi, Ollama 等
- **SSE 流式响应**: 原生支持 Chat Completions SSE 流式传输
- **API Key 管理**: 自建用户体系，可创建多个 API Key，支持配额和速率限制
- **速率限制**: 内存令牌桶实现，支持 QPM 级别限流
- **统计系统**: Token 用量、请求量/延迟、费用核算、用户行为分析
- **审计日志**: 全量记录请求信息，按 trace_id 追踪，自动过期清理
- **管理后台**: Web 管理界面，用户/Provider/模型/统计可视化管理
- **零依赖部署**: SQLite + 单二进制文件，裸机直接运行

## 快速开始

### 环境变量 (Windows PowerShell)

```powershell
$env:OPENAI_API_KEY = "sk-xxx"
$env:DEEPSEEK_API_KEY = "sk-xxx"
```

### 编译运行

```powershell
# 启动后端
.\build.ps1 dev

# 启动前端 (另一个终端)
.\build.ps1 dev-frontend

# 生产构建 (前端 + 后端)
.\build.ps1 build-all

# 运行测试
.\build.ps1 test

# 安装依赖
.\build.ps1 setup
```

或直接使用 go 命令:

```bash
go run ./cmd/gateway/
```

服务默认监听 `http://0.0.0.0:3001`

## 配置说明

所有配置通过 CLI 参数传入，无配置文件：

```bash
go run ./cmd/gateway/ \
  -port 3001 \
  -admin-password admin123 \
  -jwt-secret your-secret-key \
  -db-path ./data/llm_gateway.db
```

| Flag | 默认值 | 说明 |
|------|--------|------|
| `-port` | `3001` | 服务端口 |
| `-host` | `0.0.0.0` | 监听地址 |
| `-db-path` | `./data/llm_gateway.db` | SQLite 数据库路径 |
| `-admin-password` | `""` | 管理员密码（空=不创建） |
| `-jwt-secret` | `change-me-in-production` | JWT 签名密钥 |
| `-api-key-prefix` | `sk-` | API Key 前缀 |
| `-default-qpm` | `60` | 默认 QPM 限流 |
| `-global-qpm` | `10000` | 全局 QPM 限流 |
| `-stats-buffer-size` | `1000` | 统计缓冲区大小 |
| `-stats-flush-interval` | `5s` | 统计刷新间隔 |
| `-stats-flush-batch` | `100` | 统计刷新批次大小 |
| `-audit-retention-days` | `90` | 审计日志保留天数 |

## 使用方式

### 1. 通过 API Key 调用 LLM

```bash
# 获取模型列表
curl -H "Authorization: Bearer sk-your-api-key" \
  http://localhost:3001/v1/models

# Chat Completions (非流式)
curl -H "Authorization: Bearer sk-your-api-key" \
  -H "Content-Type: application/json" \
  -d '{"model":"gpt-4o","messages":[{"role":"user","content":"Hello"}]}' \
  http://localhost:3001/v1/chat/completions

# Chat Completions (流式)
curl -H "Authorization: Bearer sk-your-api-key" \
  -H "Content-Type: application/json" \
  -d '{"model":"gpt-4o","messages":[{"role":"user","content":"Hello"}],"stream":true}' \
  http://localhost:3001/v1/chat/completions
```

### 2. 使用 OpenAI SDK

```python
from openai import OpenAI

client = OpenAI(
    base_url="http://localhost:3001/v1",
    api_key="sk-your-api-key"
)

response = client.chat.completions.create(
    model="gpt-4o",
    messages=[{"role": "user", "content": "Hello!"}]
)
```

## 管理后台

管理员登录 `http://localhost:3001/admin` 后可以:

- **Dashboard**: 总览 Token 用量、请求量、费用、活跃用户
- **用户管理**: 创建/编辑用户、分配角色
- **API Key**: 为用户创建/禁用/删除 API Key
- **Provider**: 配置 LLM 服务商
- **模型路由**: 管理模型到 Provider 的路由映射
- **统计报表**: Token 用量、请求量、费用、用户行为
- **审计日志**: 查询请求记录，按用户/模型/时间筛选

## API 文档

### LLM Gateway API (API Key 认证)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/v1/models` | 可用模型列表 |
| POST | `/v1/chat/completions` | Chat Completions (支持 SSE Stream) |

### 管理 API (JWT 认证)

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/admin/login` | 管理员登录 |
| GET | `/api/admin/users` | 用户列表 |
| POST | `/api/admin/users` | 创建用户 |
| PUT | `/api/admin/users/:id` | 更新用户 |
| POST | `/api/admin/users/:id/api-keys` | 创建 API Key |
| DELETE | `/api/admin/users/:id/api-keys/:kid` | 删除 API Key |
| GET | `/api/admin/providers` | Provider 列表 |
| POST | `/api/admin/providers` | 添加 Provider |
| PUT | `/api/admin/providers/:id` | 更新 Provider |
| PUT | `/api/admin/providers/:id/toggle` | 启用/禁用 |
| GET | `/api/admin/models` | 模型路由列表 |
| POST | `/api/admin/models` | 添加模型路由 |

### 统计 API (JWT 认证)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/stats/tokens` | Token 用量统计 |
| GET | `/api/stats/requests` | 请求量/延迟统计 |
| GET | `/api/stats/costs` | 费用统计 |
| GET | `/api/stats/behavior` | 用户行为分析 |
| GET | `/api/dashboard/overview` | 仪表盘总览 |

### 审计 API (JWT 认证)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/audit/logs` | 审计日志分页查询 |
| GET | `/api/audit/logs/:trace_id` | 按 TraceID 查询 |

## 项目结构

```
llm-gateway/
├── cmd/gateway/main.go          # 服务入口
├── internal/
│   ├── cache/                   # 内存缓存层
│   ├── config/config.go         # CLI 参数解析
│   ├── database/                # 数据库初始化
│   ├── handler/                 # HTTP 处理器
│   ├── middleware/              # 中间件
│   ├── model/                   # 数据模型
│   ├── provider/                # LLM Provider 适配器
│   ├── router/                  # 模型路由
│   ├── service/                 # 业务逻辑层
│   └── worker/                  # 后台 Worker
├── pkg/                         # 公共库
├── web/                         # 前端 (React)
├── data/                        # 运行时数据
└── README.md
```

## License

Internal Use Only
