# LLM Gateway API 接口文档

## 认证方式

| 认证类型 | 适用范围 | Header |
|---------|---------|--------|
| API Key | LLM 网关端点 (`/v1/`) | `Authorization: Bearer <api_key>` |
| JWT | 业务管理端点 (`/api/`) | `Authorization: Bearer <jwt_token>` |
| AKSK | 业务管理端点 (`/api/`) | `X-Api-Key: <access_key>` + `X-Api-Time: <unix_timestamp>` + `X-Api-Signature: <md5(time+secret_key)>` |

> AKSK 时间窗口为 5 分钟，签名算法为 `MD5(X-Api-Time + secret_key)`。

## 统一响应格式

所有业务管理端点返回统一 JSON 格式：

```json
{
  "code": 0,
  "msg": "ok",
  "data": {}
}
```

分页响应：

```json
{
  "code": 0,
  "msg": "ok",
  "data": {
    "list": [],
    "total": 100,
    "page": 1,
    "page_size": 20
  }
}
```

LLM 网关端点直接返回 OpenAI/Anthropic 兼容格式，不使用此包装。

---

# 一、LLM 访问端点

基础路径：`/v1`  
认证方式：API Key (`Authorization: Bearer <api_key>`)  
限流：令牌桶 QPM 速率限制（按 API Key 维度）

---

## 1.1 列出可用模型

```
GET /v1/models
```

**认证**：API Key

**请求参数**：无

**返回示例**：

```json
{
  "object": "list",
  "data": [
    {"id": "gpt-4o", "object": "model"},
    {"id": "claude-3-sonnet", "object": "model"}
  ]
}
```

---

## 1.2 Chat Completions

```
POST /v1/chat/completions
```

**认证**：API Key

**功能**：OpenAI 兼容的 Chat Completions，支持流式和非流式响应。

**请求体**：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `model` | string | 是 | 模型名 |
| `messages` | array | 是 | 消息列表 |
| `messages[].role` | string | 是 | `system` / `user` / `assistant` |
| `messages[].content` | string \| array | 是 | 文本内容或多模态内容块数组 |
| `messages[].reasoning_content` | string | 否 | 推理内容（部分模型支持） |
| `messages[].tool_call_id` | string | 否 | 工具调用 ID |
| `messages[].tool_calls` | array | 否 | 工具调用列表 |
| `messages[].name` | string | 否 | 发送者名称 |
| `stream` | bool | 否 | 是否启用 SSE 流式，默认 `false` |
| `max_tokens` | int | 否 | 最大生成 token 数 |
| `temperature` | float | 否 | 采样温度 |
| `top_p` | float | 否 | 核采样概率 |
| `tools` | array | 否 | 工具定义列表 |
| `tool_choice` | string \| object | 否 | 工具选择策略 |

**请求示例**：

```json
{
  "model": "gpt-4o",
  "messages": [
    {"role": "system", "content": "You are a helpful assistant."},
    {"role": "user", "content": "Hello"}
  ],
  "stream": false,
  "max_tokens": 1024,
  "temperature": 0.7
}
```

**非流式返回**：

```json
{
  "id": "chatcmpl-xxx",
  "object": "chat.completion",
  "created": 1234567890,
  "model": "gpt-4o",
  "choices": [
    {
      "index": 0,
      "message": {"role": "assistant", "content": "..."},
      "finish_reason": "stop"
    }
  ],
  "usage": {
    "prompt_tokens": 10,
    "completion_tokens": 20,
    "total_tokens": 30
  }
}
```

**流式返回**（`stream: true`）：SSE 格式，每个 chunk 为 `data: {...}\n\n`，结束时发送 `data: [DONE]\n\n`。

---

## 1.3 Anthropic Messages API

```
POST /v1/messages
```

**认证**：API Key

**功能**：Anthropic 原生 Messages API 兼容端点。内部将 Anthropic 格式请求转换为 OpenAI 格式转发，再将响应转回 Anthropic 格式。

**请求体**：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `model` | string | 是 | 模型名 |
| `messages` | array | 是 | 消息列表 |
| `messages[].role` | string | 是 | `user` / `assistant` |
| `messages[].content` | string | 是 | 消息内容 |
| `max_tokens` | int | 是 | 最大生成 token 数 |
| `stream` | bool | 否 | 是否启用 SSE 流式，默认 `false` |
| `temperature` | float | 否 | 采样温度 |
| `top_p` | float | 否 | 核采样概率 |
| `system` | string | 否 | 系统提示 |

**请求示例**：

```json
{
  "model": "claude-3-sonnet-20240229",
  "messages": [
    {"role": "user", "content": "Hello"}
  ],
  "max_tokens": 1024,
  "system": "You are a helpful assistant."
}
```

**非流式返回**：

```json
{
  "id": "msg_xxx",
  "type": "message",
  "role": "assistant",
  "model": "claude-3-sonnet-20240229",
  "content": [
    {"type": "text", "text": "..."}
  ],
  "usage": {
    "input_tokens": 10,
    "output_tokens": 20
  }
}
```

**流式返回**：Anthropic SSE 格式，事件类型包括：
- `content_block_delta` — 文本增量
- `message_delta` — 停止原因 + 用量
- `message_stop` — 终止事件

---

# 二、业务 API

基础路径：`/api`  
认证方式：JWT 或 AKSK（`/api/admin/login` 除外）

---

## 2.1 认证

### 2.1.1 管理员登录

```
POST /api/admin/login
```

**认证**：无

**请求体**：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `username` | string | 是 | 用户名 |
| `password` | string | 是 | 密码 |

**返回**：

```json
{"code": 0, "msg": "ok", "data": {"token": "eyJhbGci..."}}
```

**错误**：`400` 参数缺失，`401` 凭证错误

---

### 2.1.2 获取当前用户信息

```
GET /api/admin/profile
```

**认证**：JWT / AKSK

**返回**：`User` 对象（不含 `password_hash`、`secret_key`）

```json
{
  "id": 1,
  "username": "admin",
  "name": "管理员",
  "phone": "13800138000",
  "department": "...",
  "role": "admin",
  "is_active": true,
  "access_key": "...",
  "created_at": "...",
  "updated_at": "..."
}
```

---

## 2.2 用户管理

### 2.2.1 用户列表

```
GET /api/admin/users
```

**认证**：JWT / AKSK

**返回**：`User[]` 数组

---

### 2.2.2 创建用户

```
POST /api/admin/users
```

**认证**：JWT / AKSK

**请求体**：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `username` | string | 是 | 用户名 |
| `password` | string | 是 | 密码 |
| `name` | string | 否 | 姓名 |
| `phone` | string | 否 | 手机号 |
| `department` | string | 否 | 部门 |
| `role` | string | 否 | 角色：`admin` / `user` / `viewer`，默认 `user` |

**返回**：新建的 `User` 对象

---

### 2.2.3 更新用户

```
PUT /api/admin/users/:id
```

**认证**：JWT / AKSK

**路径参数**：`id` — 用户 ID

**请求体**：任意字段（部分更新），可选字段包括 `name`、`phone`、`department`、`role`、`is_active` 等

**返回**：`null`

---

### 2.2.4 删除用户

```
DELETE /api/admin/users/:id
```

**认证**：JWT / AKSK

**路径参数**：`id` — 用户 ID

**返回**：`null`

---

## 2.3 API Key 管理

### 2.3.1 用户 API Key 列表

```
GET /api/admin/users/:id/api-keys
```

**认证**：JWT / AKSK

**路径参数**：`id` — 用户 ID

**返回**：`APIKey[]` 数组（`key_hash` 字段不返回）

---

### 2.3.2 创建 API Key

```
POST /api/admin/users/:id/api-keys
```

**认证**：JWT / AKSK

**路径参数**：`id` — 用户 ID

**请求体**：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `name` | string | 是 | 密钥名称 |
| `quota_limit` | int | 否 | 配额限制 |
| `rate_limit_qpm` | int | 否 | QPM 限流 |

**返回**：

```json
{
  "api_key": { "... APIKey 对象 ..." },
  "raw_key": "sk-xxxxxxxxxxxx"
}
```

> ⚠️ `raw_key` 仅在创建时返回一次，之后无法再获取。

---

### 2.3.3 删除用户 API Key

```
DELETE /api/admin/users/:id/api-keys/:kid
```

**认证**：JWT / AKSK

**路径参数**：`id` — 用户 ID，`kid` — API Key ID

**返回**：`null`

---

### 2.3.4 全局 API Key 列表

```
GET /api/admin/api-keys
```

**认证**：JWT / AKSK

**功能**：列出所有用户的 API Key（管理员全局视图），按创建时间倒序

**返回**：`APIKey[]` 数组

---

### 2.3.5 全局删除 API Key

```
DELETE /api/admin/api-keys/:id
```

**认证**：JWT / AKSK

**路径参数**：`id` — API Key ID（不做用户归属校验）

**返回**：`null`

---

### 2.3.6 切换 API Key 状态

```
PUT /api/admin/api-keys/:id/toggle
```

**认证**：JWT / AKSK

**路径参数**：`id` — API Key ID

**功能**：切换 API Key 的启用/禁用状态

**返回**：

```json
{"is_active": true}
```

---

## 2.4 AKSK 管理

### 2.4.1 生成 AKSK

```
POST /api/admin/users/:id/aksk
```

**认证**：JWT / AKSK

**路径参数**：`id` — 用户 ID

**功能**：为用户生成新的 AKSK 密钥对（覆盖旧的）

**返回**：

```json
{
  "access_key": "ak_xxxxxxxx",
  "secret_key": "sk_xxxxxxxx"
}
```

> ⚠️ `secret_key` 仅在生成时返回一次，之后无法再获取。

---

### 2.4.2 获取 Access Key

```
GET /api/admin/users/:id/aksk
```

**认证**：JWT / AKSK

**路径参数**：`id` — 用户 ID

**返回**：`{"access_key": "ak_xxxxxxxx"}`

---

## 2.5 Provider 管理

### 2.5.1 Provider 列表

```
GET /api/admin/providers
```

**认证**：JWT / AKSK

**返回**：`Provider[]` 数组

```json
{
  "id": 1,
  "name": "openai",
  "type": "openai",
  "base_url": "https://api.openai.com",
  "api_key": "sk-...",
  "is_active": true,
  "priority": 0,
  "rate_limit_qpm": 0,
  "rate_limit_burst": 0,
  "created_at": "...",
  "updated_at": "..."
}
```

> `type` 取值：`openai` / `azure` / `anthropic` / `openai-compatible` / `ollama`

---

### 2.5.2 创建 Provider

```
POST /api/admin/providers
```

**认证**：JWT / AKSK

**请求体**：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `name` | string | 是 | 名称（唯一） |
| `type` | string | 是 | 类型：`openai` / `azure` / `anthropic` / `openai-compatible` / `ollama` |
| `base_url` | string | 是 | API 基础 URL |
| `api_key` | string | 是 | API Key |
| `is_active` | bool | 否 | 是否启用，默认 `true` |
| `priority` | int | 否 | 优先级，默认 `0` |
| `rate_limit_qpm` | int | 否 | QPM 限流，默认 `0`（不限） |
| `rate_limit_burst` | int | 否 | 突发限流，默认 `0`（不限） |

**返回**：新建的 `Provider` 对象

**副作用**：自动重新加载 Provider 注册表

---

### 2.5.3 更新 Provider

```
PUT /api/admin/providers/:id
```

**认证**：JWT / AKSK

**路径参数**：`id` — Provider ID

**请求体**：任意字段（部分更新）

**返回**：`null`

**副作用**：自动重新加载 Provider 注册表

---

### 2.5.4 删除 Provider

```
DELETE /api/admin/providers/:id
```

**认证**：JWT / AKSK

**路径参数**：`id` — Provider ID

**返回**：`null`

**副作用**：自动重新加载 Provider 注册表

---

### 2.5.5 切换 Provider 状态

```
PUT /api/admin/providers/:id/toggle
```

**认证**：JWT / AKSK

**路径参数**：`id` — Provider ID

**返回**：`{"is_active": true/false}`

**副作用**：自动重新加载 Provider 注册表

---

## 2.6 模型路由管理

### 2.6.1 模型路由列表

```
GET /api/admin/models
```

**认证**：JWT / AKSK

**返回**：`Model[]` 数组（关联加载 Provider）

```json
{
  "id": 1,
  "provider_id": 1,
  "name": "gpt-4o",
  "is_active": true,
  "provider": { "... Provider 对象 ..." }
}
```

---

### 2.6.2 创建模型路由

```
POST /api/admin/models
```

**认证**：JWT / AKSK

**请求体**：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `provider_id` | int | 是 | 关联的 Provider ID |
| `name` | string | 是 | 模型名（客户端请求时使用） |
| `is_active` | bool | 否 | 是否启用，默认 `true` |

**返回**：新建的 `Model` 对象

**副作用**：自动重新加载路由表

---

### 2.6.3 更新模型路由

```
PUT /api/admin/models/:id
```

**认证**：JWT / AKSK

**路径参数**：`id` — Model ID

**请求体**：任意字段（部分更新）

**返回**：`null`

**副作用**：自动重新加载路由表

---

## 2.7 系统配置管理

### 2.7.1 配置列表

```
GET /api/admin/configs
```

**认证**：JWT / AKSK

**返回**：`Config[]` 数组

```json
[
  {"id": 1, "key": "setting_name", "value": "setting_value", "created_at": "...", "updated_at": "..."}
]
```

---

### 2.7.2 更新配置

```
PUT /api/admin/configs
```

**认证**：JWT / AKSK

**请求体**：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `key` | string | 是 | 配置键 |
| `value` | string | 是 | 配置值 |

**功能**：Upsert 语义（按 key 查找，存在则更新，不存在则创建）

**返回**：`Config` 对象（创建或更新后的完整记录）

---

## 2.8 统计分析

所有统计端点共享通用查询参数：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `start` | string | 否 | 起始日期，格式 `YYYY-MM-DD`，默认 7 天前 |
| `end` | string | 否 | 结束日期，格式 `YYYY-MM-DD`，默认今天 |

### 2.8.1 Token 用量统计

```
GET /api/stats/tokens
```

**认证**：JWT / AKSK

**查询参数**：`start`、`end`、`department`（可选，按部门过滤）

**返回**：按用户 + 模型分组的 Token 用量

```json
[
  {
    "user_id": 1,
    "username": "zhangsan",
    "department": "engineering",
    "model_name": "gpt-4o",
    "prompt_tokens": 5000,
    "completion_tokens": 3000,
    "total_tokens": 8000
  }
]
```

---

### 2.8.2 请求量统计

```
GET /api/stats/requests
```

**认证**：JWT / AKSK

**查询参数**：`start`、`end`、`user_id`（可选）、`model`（可选）

**返回**：按天聚合的请求量

```json
[
  {
    "date": "2026-06-10",
    "request_count": 150,
    "success_count": 145,
    "error_count": 5,
    "avg_latency_ms": 1200
  }
]
```

---

### 2.8.3 费用统计

```
GET /api/stats/costs
```

**认证**：JWT / AKSK

**查询参数**：`start`、`end`

**返回**：按天 + 模型分组的费用

```json
[
  {
    "date": "2026-06-10",
    "model_name": "gpt-4o",
    "total_cost": 12.50
  }
]
```

---

### 2.8.4 用户行为分析

```
GET /api/stats/behavior
```

**认证**：JWT / AKSK

**查询参数**：`start`、`end`、`user_id`（可选）、`model`（可选）

**返回**：按用户 + 模型分组的调用频次 Top 100

```json
[
  {
    "user_id": 1,
    "username": "zhangsan",
    "department": "engineering",
    "model_name": "gpt-4o",
    "count": 320
  }
]
```

---

### 2.8.5 Dashboard 总览

```
GET /api/dashboard/overview
```

**认证**：JWT / AKSK

**查询参数**：`start`、`end`

**返回**：汇总概览数据

```json
{
  "total_requests": 1500,
  "total_tokens": 500000,
  "total_cost": 125.00,
  "avg_latency_ms": 1100,
  "success_rate": 97.5,
  "active_users": 25,
  "top_models": [
    {"model_name": "gpt-4o", "count": 800},
    {"model_name": "claude-3", "count": 400}
  ]
}
```

---

## 2.9 审计日志

### 2.9.1 审计日志查询

```
GET /api/audit/logs
```

**认证**：JWT / AKSK

**查询参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `page` | int | 否 | 页码，默认 `1` |
| `pageSize` | int | 否 | 每页条数，默认 `20`，最大 `100` |
| `user_id` | uint | 否 | 按用户 ID 过滤 |
| `model` | string | 否 | 按模型名精确匹配 |
| `start` | string | 否 | 起始时间，支持 RFC3339 或 `YYYY-MM-DD` |
| `end` | string | 否 | 结束时间，支持 RFC3339 或 `YYYY-MM-DD` |
| `status` | string | 否 | `success` (=200) 或 `error` (=500) |

**返回**：分页格式

```json
{
  "list": [
    {
      "id": 1,
      "trace_id": "uuid-xxx",
      "user_id": 1,
      "api_key_id": 3,
      "provider_id": 1,
      "model_name": "gpt-4o",
      "request_summary": "{...}",
      "response_summary": "{...}",
      "prompt_tokens": 100,
      "completion_tokens": 50,
      "status_code": 200,
      "error_message": "",
      "latency_ms": 1500,
      "cost": 0.0,
      "ip_address": "192.168.1.100",
      "user_agent": "Mozilla/...",
      "created_at": "2026-06-10T10:30:00Z"
    }
  ],
  "total": 150,
  "page": 1,
  "page_size": 20
}
```

---

### 2.9.2 按 Trace ID 查询

```
GET /api/audit/logs/:trace_id
```

**认证**：JWT / AKSK

**路径参数**：`trace_id` — UUID 格式追踪 ID

**功能**：查询同一次请求的完整审计日志链（流式传输可能产生多条记录）

**返回**：`AuditLog[]` 数组（按 `created_at` 正序排列）

---

# 端点总览

| # | 方法 | 路径 | 认证 | 分组 | 描述 |
|---|------|------|------|------|------|
| 1 | GET | `/v1/models` | API Key | LLM | 可用模型列表 |
| 2 | POST | `/v1/chat/completions` | API Key | LLM | Chat Completions（OpenAI 兼容） |
| 3 | POST | `/v1/messages` | API Key | LLM | Messages API（Anthropic 兼容） |
| 4 | POST | `/api/admin/login` | 无 | 认证 | 管理员登录 |
| 5 | GET | `/api/admin/profile` | JWT/AKSK | 认证 | 当前用户信息 |
| 6 | GET | `/api/admin/users` | JWT/AKSK | 用户 | 用户列表 |
| 7 | POST | `/api/admin/users` | JWT/AKSK | 用户 | 创建用户 |
| 8 | PUT | `/api/admin/users/:id` | JWT/AKSK | 用户 | 更新用户 |
| 9 | DELETE | `/api/admin/users/:id` | JWT/AKSK | 用户 | 删除用户 |
| 10 | GET | `/api/admin/users/:id/api-keys` | JWT/AKSK | 密钥 | 用户 API Key 列表 |
| 11 | POST | `/api/admin/users/:id/api-keys` | JWT/AKSK | 密钥 | 创建 API Key |
| 12 | DELETE | `/api/admin/users/:id/api-keys/:kid` | JWT/AKSK | 密钥 | 删除 API Key |
| 13 | POST | `/api/admin/users/:id/aksk` | JWT/AKSK | 密钥 | 生成 AKSK |
| 14 | GET | `/api/admin/users/:id/aksk` | JWT/AKSK | 密钥 | 获取 Access Key |
| 15 | GET | `/api/admin/api-keys` | JWT/AKSK | 密钥 | 全局 API Key 列表 |
| 16 | DELETE | `/api/admin/api-keys/:id` | JWT/AKSK | 密钥 | 全局删除 API Key |
| 17 | PUT | `/api/admin/api-keys/:id/toggle` | JWT/AKSK | 密钥 | 切换 API Key 状态 |
| 18 | GET | `/api/admin/providers` | JWT/AKSK | Provider | Provider 列表 |
| 19 | POST | `/api/admin/providers` | JWT/AKSK | Provider | 创建 Provider |
| 20 | PUT | `/api/admin/providers/:id` | JWT/AKSK | Provider | 更新 Provider |
| 21 | DELETE | `/api/admin/providers/:id` | JWT/AKSK | Provider | 删除 Provider |
| 22 | PUT | `/api/admin/providers/:id/toggle` | JWT/AKSK | Provider | 切换 Provider 状态 |
| 23 | GET | `/api/admin/models` | JWT/AKSK | 模型路由 | 模型路由列表 |
| 24 | POST | `/api/admin/models` | JWT/AKSK | 模型路由 | 创建模型路由 |
| 25 | PUT | `/api/admin/models/:id` | JWT/AKSK | 模型路由 | 更新模型路由 |
| 26 | GET | `/api/admin/configs` | JWT/AKSK | 配置 | 配置列表 |
| 27 | PUT | `/api/admin/configs` | JWT/AKSK | 配置 | 更新配置 |
| 28 | GET | `/api/stats/tokens` | JWT/AKSK | 统计 | Token 用量统计 |
| 29 | GET | `/api/stats/requests` | JWT/AKSK | 统计 | 请求量统计 |
| 30 | GET | `/api/stats/costs` | JWT/AKSK | 统计 | 费用统计 |
| 31 | GET | `/api/stats/behavior` | JWT/AKSK | 统计 | 用户行为分析 |
| 32 | GET | `/api/dashboard/overview` | JWT/AKSK | 统计 | Dashboard 总览 |
| 33 | GET | `/api/audit/logs` | JWT/AKSK | 审计 | 审计日志分页查询 |
| 34 | GET | `/api/audit/logs/:trace_id` | JWT/AKSK | 审计 | 按 Trace ID 查询 |
