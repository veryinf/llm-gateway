# LLM Gateway

统一 LLM API 网关 — 一个入口，调用所有大模型。

为团队提供标准化的 LLM API 接入点，内置用户管理、API Key 分发、用量统计和审计日志。兼容 OpenAI 协议，现有应用只需改一行 `base_url` 即可接入。

## ✨ 核心能力

- **统一接入** — OpenAI 兼容协议，Chat Completions + SSE 流式输出
- **多 Provider** — OpenAI、Azure OpenAI、Anthropic (Claude)、DeepSeek、通义千问、Kimi、Ollama 等
- **模型路由** — 任意模型名绑定到任意 Provider，按需扩展
- **用户 & Key 管理** — 自建用户体系，按用户创建/禁用 API Key
- **限流保护** — 每 Key 独立 QPM 限流，支持全局上限
- **用量统计** — Token 用量、请求量、延迟、费用，一目了然
- **请求审计** — 全量请求日志，按 TraceID 追踪，支持回溯分析
- **管理后台** — Web 界面一站式管理所有配置
- **零外部依赖** — SQLite + 单二进制，开箱即用

## 🚀 快速开始

### 1. 启动服务

```powershell
# 一键构建并运行
.\build.ps1 dev
```

或直接使用 Go：

```bash
go run ./cmd/gateway/
```

服务默认监听 `http://0.0.0.0:3001`。

### 2. 配置 Provider

打开管理后台 `http://localhost:3001`，在 **Provider 管理** 页面添加你的 LLM 服务商：

| Provider 类型 | 适用场景 | 需要的配置 |
|---|---|---|
| `openai` | OpenAI 原生 API | API Key |
| `azure` | Azure OpenAI | API Key + Base URL (含部署名) |
| `anthropic` | Anthropic Claude | API Key |
| `openai-compatible` | DeepSeek / 通义千问 / Kimi 等兼容服务 | API Key + Base URL |
| `ollama` | 本地 Ollama 模型 | Base URL (默认 `http://localhost:11434`) |

### 3. 配置模型路由

在 **模型路由管理** 页面，将模型名映射到对应的 Provider：

```
gpt-4o        → OpenAI Provider
claude-3.5-sonnet → Anthropic Provider
deepseek-chat → DeepSeek Provider
qwen-plus     → 通义千问 Provider
llama3        → Ollama Provider
```

### 4. 创建用户 & API Key

在 **用户管理** 页面创建用户，为每个用户分配 API Key。

### 5. 开始调用

```bash
curl http://localhost:3001/v1/chat/completions \
  -H "Authorization: Bearer sk-your-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o",
    "messages": [{"role": "user", "content": "你好"}]
  }'
```

## 📖 使用示例

### Python (OpenAI SDK)

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

### 流式输出

```python
stream = client.chat.completions.create(
    model="gpt-4o",
    messages=[{"role": "user", "content": "写一首诗"}],
    stream=True
)
for chunk in stream:
    print(chunk.choices[0].delta.content or "", end="")
```

### Node.js (OpenAI SDK)

```javascript
import OpenAI from "openai";

const client = new OpenAI({
  baseURL: "http://localhost:3001/v1",
  apiKey: "sk-your-api-key",
});

const response = await client.chat.completions.create({
  model: "gpt-4o",
  messages: [{ role: "user", content: "Hello!" }],
});
```

### curl

```bash
# 列出可用模型
curl -H "Authorization: Bearer sk-your-api-key" \
  http://localhost:3001/v1/models

# 非流式调用
curl http://localhost:3001/v1/chat/completions \
  -H "Authorization: Bearer sk-your-api-key" \
  -H "Content-Type: application/json" \
  -d '{"model":"gpt-4o","messages":[{"role":"user","content":"Hello"}]}'

# 流式调用
curl http://localhost:3001/v1/chat/completions \
  -H "Authorization: Bearer sk-your-api-key" \
  -H "Content-Type: application/json" \
  -d '{"model":"gpt-4o","messages":[{"role":"user","content":"Hello"}],"stream":true}'
```

## ⚙️ 配置项

通过 CLI 参数或环境变量配置，无需配置文件：

| 参数 | 环境变量 | 默认值 | 说明 |
|---|---|---|---|
| `--http-addr` | `HTTP_ADDR` | `:3001` | 监听地址 |
| `--data-dir` | `DATA_DIR` | `./data` | 数据目录（数据库、日志等） |
| `--admin-password` | `ADMIN_PASSWORD` | 空 | 管理员初始密码（空则不自动创建） |
| `--api-key-prefix` | `API_KEY_PREFIX` | `sk-` | API Key 前缀 |
| `--default-qpm` | `DEFAULT_QPM` | `60` | 每 Key 默认 QPM 限流 |
| `--global-qpm` | `GLOBAL_QPM` | `10000` | 全局 QPM 限流上限 |
| `--stats-buffer-size` | `STATS_BUFFER_SIZE` | `1000` | 统计缓冲区大小 |
| `--stats-flush-interval` | `STATS_FLUSH_INTERVAL` | `5s` | 统计刷盘间隔 |
| `--stats-flush-batch` | `STATS_FLUSH_BATCH` | `100` | 统计刷盘批次 |
| `--request-log-retention-days` | `REQUEST_LOG_RETENTION_DAYS` | `90` | 请求日志保留天数 |
| `--log` | `LOG` | `console` | 日志输出：`console` / `file` / `both` |
| `--log-level` | `LOG_LEVEL` | `info` | 日志级别：`debug` / `info` / `warn` / `error` |

**示例：**

```powershell
.\build.ps1 dev -- --admin-password mypass --http-addr :8080
```

```bash
go run ./cmd/gateway/ --admin-password mypass --http-addr :8080
```

## 🖥️ 管理后台

访问 `http://localhost:3001` 即可进入管理后台：

| 页面 | 功能 |
|---|---|
| **Dashboard** | 总览：请求量、Token 用量、费用、延迟、成功率、趋势图 |
| **Provider 管理** | 添加/编辑/启停 LLM 服务商 |
| **模型路由** | 将模型名映射到 Provider，支持自定义模型名 |
| **用户管理** | 创建/编辑用户，内联管理 API Key |
| **API Key 管理** | 全局视图，按用户筛选，查看/禁用/删除 Key |
| **请求记录** | 分页查看请求日志，按状态/模型筛选，查看请求详情 |
| **系统设置** | 请求详情开关、日志保留天数 |

## 📡 API 端点

### LLM 调用（API Key 认证）

| 方法 | 路径 | 说明 |
|---|---|---|
| `GET` | `/v1/models` | 可用模型列表 |
| `POST` | `/v1/chat/completions` | Chat Completions（支持 SSE 流式） |
| `POST` | `/v1/messages` | Anthropic Messages API 兼容 |

### 健康检查

| 方法 | 路径 | 说明 |
|---|---|---|
| `GET` | `/health` | 服务存活检查 |
| `GET` | `/health/ready` | 服务就绪检查（含数据库） |

## 🔧 构建命令

```powershell
.\build.ps1 setup          # 安装依赖（Go + 前端）
.\build.ps1 dev            # 开发模式运行后端
.\build.ps1 dev-frontend   # 启动前端开发服务器 (localhost:5173)
.\build.ps1 build          # 构建后端二进制 → output/lgw.exe
.\build.ps1 build-all      # 构建前端 + 后端（生产部署用）
```

## License

Internal Use Only
