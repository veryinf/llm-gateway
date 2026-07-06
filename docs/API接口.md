# LLM Gateway API 接口文档

## 认证方式

| 认证类型 | 适用范围 | Header |
|---------|---------|--------|
| API Key | LLM 网关端点 (`/v1/`、`/anthropic`) | `Authorization: Bearer <api_key>` |
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

列表响应（`NewDataSet` 格式）：

```json
{
  "code": 0,
  "msg": "ok",
  "data": {
    "list": [],
    "total": 100
  }
}
```

LLM 网关端点直接返回 OpenAI/Anthropic 兼容格式，不使用此包装。

---

# 一、健康检查

基础路径：`/`
认证方式：无

---

## 1.1 健康检查

```
GET /health
```

**认证**：无

**返回**：

```json
{"status": "ok"}
```

---

## 1.2 就绪检查

```
GET /health/ready
```

**认证**：无

**功能**：检查数据库连接是否正常。成功返回 `200`，失败返回 `503`。

**返回**：

```json
{"status": "ready"}
```

---

# 二、LLM 访问端点

基础路径：`/v1` 和 `/anthropic`
认证方式：API Key (`Authorization: Bearer <api_key>`)
限流：令牌桶 QPM 速率限制（按 API Key 维度）

> `/anthropic` 路径下的端点功能与 `/v1` 完全一致，仅 URL 前缀不同，方便 Anthropic SDK 直接对接。

---

## 2.1 列出可用模型

```
GET /v1/models
GET /anthropic/models
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

## 2.2 Chat Completions

```
POST /v1/chat/completions
POST /anthropic/chat/completions
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

## 2.3 Anthropic Messages API

```
POST /v1/messages
POST /anthropic/messages
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

# 三、业务 API

基础路径：`/api`
认证方式：JWT 或 AKSK（`/api/admin/login` 除外）

---

## 3.1 认证

### 3.1.1 管理员登录

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

**错误**：`-11` 参数缺失或凭证错误

---

### 3.1.2 注销

```
POST /api/admin/logout
```

**认证**：JWT / AKSK

**功能**：注销当前 token，使其失效。

**返回**：

```json
{"code": 0, "msg": "ok"}
```

---

## 3.2 个人信息

### 3.2.1 获取当前用户信息

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

### 3.2.2 更新个人信息

```
POST /api/admin/profile/update
```

**认证**：JWT / AKSK

**功能**：当前登录用户更新自己的个人信息。

**请求体**：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `name` | string | 否 | 姓名 |
| `phone` | string | 否 | 手机号 |
| `department` | string | 否 | 部门 |

**返回**：更新后的 `User` 对象

---

## 3.3 用户管理

### 3.3.1 用户列表

```
GET /api/admin/users
```

**认证**：JWT / AKSK

**返回**：`UserWithCount[]` 列表，每项包含 `User` 字段 + `api_key_count`（该用户的 API Key 数量）

```json
[
  {
    "id": 1,
    "username": "zhangsan",
    "name": "张三",
    "phone": "13800138000",
    "department": "engineering",
    "role": "user",
    "is_active": true,
    "access_key": "ak_xxx",
    "created_at": "...",
    "updated_at": "...",
    "api_key_count": 3
  }
]
```

---

### 3.3.2 创建用户

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

### 3.3.3 更新用户

```
PUT /api/admin/users/:id
```

**认证**：JWT / AKSK

**路径参数**：`id` — 用户 ID

**请求体**：任意字段（部分更新）。可更新字段包括 `name`、`phone`、`department`、`role`、`is_active`、`password`（传入 `password` 时会自动加密后存储为 `password_hash`）。`id`、`password_hash`、`created_at`、`updated_at` 会被忽略。

**返回**：

```json
{"code": 0, "msg": "ok"}
```

---

### 3.3.4 删除用户

```
DELETE /api/admin/users/:id
```

**认证**：JWT / AKSK

**路径参数**：`id` — 用户 ID

**返回**：

```json
{"code": 0, "msg": "ok"}
```

---

## 3.4 API Key 管理

### 3.4.1 用户 API Key 列表

```
GET /api/admin/users/:id/api-keys
```

**认证**：JWT / AKSK

**路径参数**：`id` — 用户 ID

**返回**：`APIKey[]` 列表

```json
[
  {
    "id": 1,
    "user_id": 1,
    "key": "sk-xxxxxxxxxxxx",
    "name": "生产环境",
    "quota_limit": 10000,
    "quota_used": 500,
    "rate_limit_qpm": 60,
    "expires_at": null,
    "is_active": true,
    "last_used_at": "2026-06-10T10:00:00Z",
    "created_at": "2026-06-01T00:00:00Z"
  }
]
```

---

### 3.4.2 创建 API Key

```
POST /api/admin/users/:id/api-keys
```

**认证**：JWT / AKSK

**路径参数**：`id` — 用户 ID

**请求体**：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `name` | string | 是 | 密钥名称 |
| `quota_limit` | int | 否 | 配额限制，默认 `0`（不限） |
| `rate_limit_qpm` | int | 否 | QPM 限流，默认 `60` |

**返回**：

```json
{
  "code": 0,
  "msg": "ok",
  "data": {
    "api_key": { "... APIKey 对象 ..." },
    "raw_key": "sk-xxxxxxxxxxxx"
  }
}
```

> ⚠️ `raw_key` 仅在创建时返回一次，之后无法再获取。

---

### 3.4.3 删除用户 API Key

```
DELETE /api/admin/users/:id/api-keys/:kid
```

**认证**：JWT / AKSK

**路径参数**：`id` — 用户 ID，`kid` — API Key ID

**功能**：删除指定用户的 API Key（会校验归属关系）

**返回**：

```json
{"code": 0, "msg": "ok"}
```

---

### 3.4.4 全局 API Key 列表

```
GET /api/admin/api-keys
```

**认证**：JWT / AKSK

**功能**：列出所有用户的 API Key（管理员全局视图），按创建时间倒序

**返回**：`APIKey[]` 列表

---

### 3.4.5 全局删除 API Key

```
DELETE /api/admin/api-keys/:id
```

**认证**：JWT / AKSK

**路径参数**：`id` — API Key ID（不做用户归属校验）

**返回**：

```json
{"code": 0, "msg": "ok"}
```

---

### 3.4.6 切换 API Key 状态

```
PUT /api/admin/api-keys/:id/toggle
```

**认证**：JWT / AKSK

**路径参数**：`id` — API Key ID

**功能**：切换 API Key 的启用/禁用状态

**返回**：

```json
{"code": 0, "msg": "ok", "data": {"is_active": true}}
```

---

## 3.5 AKSK 管理

### 3.5.1 生成 AKSK

```
POST /api/admin/users/:id/aksk
```

**认证**：JWT / AKSK

**路径参数**：`id` — 用户 ID

**功能**：为用户生成新的 AKSK 密钥对（覆盖旧的）

**返回**：

```json
{
  "code": 0,
  "msg": "ok",
  "data": {
    "access_key": "ak_xxxxxxxx",
    "secret_key": "sk_xxxxxxxx"
  }
}
```

> ⚠️ `secret_key` 仅在生成时返回一次，之后无法再获取。

---

### 3.5.2 获取 Access Key

```
GET /api/admin/users/:id/aksk
```

**认证**：JWT / AKSK

**路径参数**：`id` — 用户 ID

**返回**：

```json
{"code": 0, "msg": "ok", "data": {"access_key": "ak_xxxxxxxx"}}
```

---

## 3.6 Provider 管理

### 3.6.1 Provider 列表

```
GET /api/admin/providers
```

**认证**：JWT / AKSK

**返回**：`ProviderWithCount[]` 列表，每项包含 `Provider` 字段 + `model_count`（该 Provider 下的模型数量）

```json
[
  {
    "id": 1,
    "name": "openai",
    "base_url": "https://api.openai.com",
    "api_key": "sk-...",
    "support_openai": true,
    "openai_base_url": "",
    "support_anthropic": false,
    "anthropic_base_url": "",
    "preferred_api": "openai",
    "is_active": true,
    "created_at": "...",
    "updated_at": "...",
    "model_count": 15
  }
]
```

> **多协议架构**：`support_openai` / `support_anthropic` 表示 Provider 支持的 API 协议，`preferred_api` 为首选协议，`openai_base_url` / `anthropic_base_url` 为各协议的基础 URL（为空时使用 `base_url`）。

---

### 3.6.2 创建 Provider

```
POST /api/admin/providers
```

**认证**：JWT / AKSK

**请求体**：

```json
{
  "provider": {
    "name": "openai",
    "base_url": "https://api.openai.com",
    "api_key": "sk-xxx",
    "support_openai": true,
    "openai_base_url": "",
    "support_anthropic": false,
    "anthropic_base_url": "",
    "preferred_api": "openai",
    "is_active": true
  },
  "models": [
    {
      "name": "gpt-4o",
      "display_name": "GPT-4o",
      "api_type": "openai",
      "is_active": true
    }
  ]
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `provider.name` | string | 是 | 名称（唯一） |
| `provider.base_url` | string | 是 | API 基础 URL |
| `provider.api_key` | string | 是 | API Key |
| `provider.support_openai` | bool | 否 | 是否支持 OpenAI 协议，默认 `false` |
| `provider.openai_base_url` | string | 否 | OpenAI 协议基础 URL（为空则用 `base_url`） |
| `provider.support_anthropic` | bool | 否 | 是否支持 Anthropic 协议，默认 `false` |
| `provider.anthropic_base_url` | string | 否 | Anthropic 协议基础 URL（为空则用 `base_url`） |
| `provider.preferred_api` | string | 否 | 首选协议：`openai` / `anthropic`，默认 `openai` |
| `provider.is_active` | bool | 否 | 是否启用，默认 `true` |
| `models` | array | 否 | 关联的上游模型列表，创建时批量插入 |

**返回**：新建的 `Provider` 对象

**副作用**：自动重新加载 Provider 注册表

---

### 3.6.3 更新 Provider

```
PUT /api/admin/providers/:id
```

**认证**：JWT / AKSK

**路径参数**：`id` — Provider ID

**请求体**：

```json
{
  "provider": {
    "name": "openai",
    "base_url": "https://api.openai.com",
    "api_key": "",
    "support_openai": true,
    "preferred_api": "openai",
    "is_active": true
  },
  "models": [
    {"id": 1, "name": "gpt-4o", "api_type": "openai", "is_active": true},
    {"name": "gpt-4o-mini", "api_type": "openai", "is_active": true}
  ]
}
```

**功能**：更新 Provider 及其关联的上游模型。`api_key` 为空字符串表示不修改。`models` 数组提供时会执行 reconcile：
- 有 `id` 的项更新已有记录
- 无 `id` 的项创建新记录
- 已有但不在列表中的记录会被删除

**返回**：

```json
{"code": 0, "msg": "ok"}
```

**副作用**：自动重新加载 Provider 注册表

---

### 3.6.4 删除 Provider

```
DELETE /api/admin/providers/:id
```

**认证**：JWT / AKSK

**路径参数**：`id` — Provider ID

**功能**：删除 Provider，同时级联删除关联的上游模型和引用这些上游模型的下游模型

**返回**：

```json
{"code": 0, "msg": "ok"}
```

**副作用**：自动重新加载 Provider 注册表

---

### 3.6.5 切换 Provider 状态

```
PUT /api/admin/providers/:id/toggle
```

**认证**：JWT / AKSK

**路径参数**：`id` — Provider ID

**返回**：

```json
{"code": 0, "msg": "ok", "data": {"is_active": true/false}}
```

**副作用**：自动重新加载 Provider 注册表

---

### 3.6.6 获取上游 Provider 模型列表

```
POST /api/admin/providers/fetch-models
```

**认证**：JWT / AKSK

**功能**：从上游 Provider 的 `/v1/models` 端点自动发现可用模型。Anthropic 类型的 Provider 因无标准模型列表端点，直接返回空列表。

**请求体**：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `base_url` | string | 是 | Provider 基础 URL |
| `api_key` | string | 否 | API Key（部分上游需要） |
| `api_type` | string | 否 | `openai` 或 `anthropic`，默认 `openai` |

**返回**：

```json
{
  "code": 0,
  "msg": "ok",
  "data": [
    {"id": "gpt-4o"},
    {"id": "gpt-4o-mini"}
  ]
}
```

---

### 3.6.7 批量导入上游模型

```
POST /api/admin/providers/batch-import-models
```

**认证**：JWT / AKSK

**功能**：将模型名批量导入为指定 Provider 的上游模型，已存在的模型名会被跳过。

**请求体**：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `provider_id` | int | 是 | Provider ID |
| `model_names` | string[] | 是 | 模型名列表 |

**返回**：

```json
{
  "code": 0,
  "msg": "ok",
  "data": {"created": 10, "skipped": 2}
}
```

**副作用**：自动重新加载 Provider 注册表

---

## 3.7 上游模型管理

### 3.7.1 上游模型列表

```
GET /api/admin/models
```

**认证**：JWT / AKSK

**查询参数**：`provider_id`（可选，按 Provider 过滤）

**返回**：`Model[]` 列表（关联加载 Provider）

```json
[
  {
    "id": 1,
    "provider_id": 1,
    "name": "gpt-4o",
    "api_type": "openai",
    "display_name": "GPT-4o",
    "description": "...",
    "max_context_tokens": 128000,
    "max_output_tokens": 16384,
    "input_price": 2.5,
    "output_price": 10.0,
    "tpm": 0,
    "qpm": 0,
    "is_chat": true,
    "is_completion": false,
    "is_vision": true,
    "is_embedding": false,
    "is_active": true,
    "created_at": "...",
    "updated_at": "...",
    "provider": { "... Provider 对象 ..." }
  }
]
```

---

### 3.7.2 创建上游模型

```
POST /api/admin/models
```

**认证**：JWT / AKSK

**请求体**：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `provider_id` | int | 是 | 关联的 Provider ID |
| `name` | string | 是 | 模型名（上游模型 ID） |
| `api_type` | string | 否 | `openai` / `anthropic`，默认 `openai` |
| `display_name` | string | 否 | 显示名称 |
| `description` | string | 否 | 模型描述 |
| `max_context_tokens` | int | 否 | 最大上下文 token 数 |
| `max_output_tokens` | int | 否 | 最大输出 token 数 |
| `input_price` | float | 否 | 输入价格（元/百万 token） |
| `output_price` | float | 否 | 输出价格（元/百万 token） |
| `tpm` | int | 否 | TPM 限流，`0` 不限 |
| `qpm` | int | 否 | QPM 限流，`0` 不限 |
| `is_chat` | bool | 否 | 是否支持 Chat，默认 `true` |
| `is_completion` | bool | 否 | 是否支持 Completion，默认 `false` |
| `is_vision` | bool | 否 | 是否支持视觉，默认 `false` |
| `is_embedding` | bool | 否 | 是否为 Embedding 模型，默认 `false` |
| `is_active` | bool | 否 | 是否启用，默认 `true` |

**返回**：新建的 `Model` 对象

**副作用**：自动重新加载路由表

---

### 3.7.3 更新上游模型

```
PUT /api/admin/models/:id
```

**认证**：JWT / AKSK

**路径参数**：`id` — Model ID

**请求体**：任意字段（部分更新）。`provider_id` 和 `created_at` 不可修改。

**返回**：

```json
{"code": 0, "msg": "ok"}
```

**副作用**：自动重新加载路由表

---

### 3.7.4 删除上游模型

```
DELETE /api/admin/models/:id
```

**认证**：JWT / AKSK

**路径参数**：`id` — Model ID

**返回**：

```json
{"code": 0, "msg": "ok"}
```

**副作用**：自动重新加载路由表

---

## 3.8 下游模型管理

下游模型（Downstream Model）是对上游模型的别名映射，客户端使用下游模型名发起请求，网关自动路由到对应的上游模型。

### 3.8.1 下游模型列表

```
GET /api/admin/downstream-models
```

**认证**：JWT / AKSK

**查询参数**：`upstream_model_id`（可选，按上游模型 ID 过滤）

**返回**：`DownstreamModel[]` 列表（关联加载 UpstreamModel）

```json
[
  {
    "id": 1,
    "name": "gpt4",
    "display_name": "GPT-4",
    "upstream_model_id": 1,
    "description": "...",
    "is_active": true,
    "created_at": "...",
    "updated_at": "...",
    "upstream_model": { "... 上游 Model 对象 ..." }
  }
]
```

---

### 3.8.2 创建下游模型

```
POST /api/admin/downstream-models
```

**认证**：JWT / AKSK

**请求体**：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `name` | string | 是 | 下游模型名（客户端请求时使用） |
| `display_name` | string | 否 | 显示名称 |
| `upstream_model_id` | int | 是 | 关联的上游模型 ID |
| `description` | string | 否 | 模型描述 |
| `is_active` | bool | 否 | 是否启用，默认 `true` |

**返回**：新建的 `DownstreamModel` 对象

**副作用**：自动重新加载路由表

---

### 3.8.3 更新下游模型

```
PUT /api/admin/downstream-models/:id
```

**认证**：JWT / AKSK

**路径参数**：`id` — Downstream Model ID

**请求体**：任意字段（部分更新）

**返回**：

```json
{"code": 0, "msg": "ok"}
```

**副作用**：自动重新加载路由表

---

### 3.8.4 删除下游模型

```
DELETE /api/admin/downstream-models/:id
```

**认证**：JWT / AKSK

**路径参数**：`id` — Downstream Model ID

**返回**：

```json
{"code": 0, "msg": "ok"}
```

**副作用**：自动重新加载路由表

---

## 3.9 系统配置管理

### 3.9.1 配置列表

```
GET /api/admin/configs
```

**认证**：JWT / AKSK

**返回**：`Config[]` 列表

```json
[
  {"id": 1, "key": "setting_name", "value": "setting_value", "created_at": "...", "updated_at": "..."}
]
```

---

### 3.9.2 更新配置

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

## 3.10 统计分析

所有统计端点共享通用查询参数：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `start` | string | 否 | 起始日期，格式 `YYYY-MM-DD`，默认 7 天前 |
| `end` | string | 否 | 结束日期，格式 `YYYY-MM-DD`，默认今天 |

### 3.10.1 Token 用量统计

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

### 3.10.2 请求量统计

```
GET /api/stats/requests
```

**认证**：JWT / AKSK

**查询参数**：`start`、`end`、`user_id`（可选）、`model`（可选）

**返回**：按小时聚合的请求量（`date` 格式 `YYYY-MM-DD HH:00`）

```json
[
  {
    "date": "2026-06-10 14:00",
    "requestCount": 18,
    "successCount": 17,
    "errorCount": 1,
    "avgLatencyMs": 1200
  }
]
```

---

### 3.10.3 费用统计

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

### 3.10.4 用户行为分析

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

### 3.10.4.1 用户维度分页统计

```
GET /api/stats/users
```

**认证**：JWT / AKSK

**查询参数**：
- `start`、`end`（可选，缺省近 7 天）
- `keyword`（可选，模糊匹配 `users.username` 或 `users.department`）
- `page`（可选，默认 1）
- `size`（可选，默认 20，上限 200）
- `sort`（可选，默认 `totalTokens`，白名单：`requestCount` / `promptTokens` / `completionTokens` / `reasoningTokens` / `totalTokens` / `modelCount` / `lastCallAt` / `username` / `department`）
- `order`（可选，`asc` / `desc`，默认 `desc`）

**返回**：分页列表

```json
{
  "errCode": 0,
  "errMsg": "ok",
  "dataSet": [
    {
      "userId": 2,
      "username": "zhangsan",
      "department": "engineering",
      "requestCount": 320,
      "promptTokens": 80000,
      "completionTokens": 40000,
      "reasoningTokens": 0,
      "totalTokens": 120000,
      "modelCount": 5,
      "lastCallAt": "2026-07-03 14:23:11"
    }
  ],
  "total": 25,
  "hasMore": true
}
```

### 3.10.4.2 服务商模型维度分页统计

```
GET /api/stats/provider-models
```

**认证**：JWT / AKSK

**查询参数**：
- `start`、`end`（同上）
- `keyword`（可选，模糊匹配 `provider_model`）
- `page`、`size`（同上）
- `sort`（可选，默认 `totalTokens`，白名单：`requestCount` / `promptTokens` / `completionTokens` / `totalTokens` / `userCount` / `avgLatencyMs` / `providerTitle` / `providerModel`）
- `order`（同上）

**返回**：分页列表

```json
{
  "errCode": 0,
  "errMsg": "ok",
  "dataSet": [
    {
      "providerId": 1,
      "providerTitle": "OpenAI",
      "providerModel": "gpt-4o",
      "requestCount": 800,
      "promptTokens": 200000,
      "completionTokens": 100000,
      "totalTokens": 300000,
      "userCount": 18,
      "avgLatencyMs": 1200.5
    }
  ],
  "total": 8,
  "hasMore": false
}
```

---

### 3.10.5 Dashboard 总览

```
GET /api/dashboard/overview
```

**认证**：JWT / AKSK

**查询参数**：`start`、`end`

**返回**：汇总概览数据

```json
{
  "totalRequests": 1500,
  "totalTokens": 500000,
  "totalCost": 0,
  "avgLatencyMs": 1100,
  "successRate": 97.5,
  "activeUsers": 25,
  "topModels": [
    {"userModel": "gpt-4o", "providerModel": "gpt-4o", "count": 800},
    {"userModel": "claude-3", "providerModel": "claude-3.5-sonnet", "count": 400}
  ],
  "topProviders": [
    {
      "providerId": 1,
      "providerTitle": "OpenAI",
      "providerModel": "gpt-4o",
      "requestCount": 800,
      "promptTokens": 200000,
      "completionTokens": 100000,
      "totalTokens": 300000,
      "userCount": 18
    }
  ],
  "topUsers": [
    {
      "userId": 2,
      "username": "zhangsan",
      "department": "engineering",
      "requestCount": 320,
      "promptTokens": 80000,
      "completionTokens": 40000,
      "totalTokens": 120000,
      "modelCount": 5
    }
  ]
}
```

> 注：本期不计算费用，`totalCost` 始终为 `0`；`topModels` 按 `(userModel, providerModel)` 聚合；`topProviders` 按 `providerModel` 聚合并附带 `providerTitle`；`topUsers` 按 `userId` 聚合并附带 `username` / `department`。三者均 `LIMIT 10`。

---

## 3.11 请求日志

### 3.11.1 请求日志分页查询

```
GET /api/request-logs
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
      "trace_id": "uuid-xxx",
      "user_id": 1,
      "api_key_id": 3,
      "model_name": "gpt-4o",
      "is_stream": false,
      "prompt_tokens": 100,
      "completion_tokens": 50,
      "total_tokens": 150,
      "request_body": "{...}",
      "response_body": "{...}",
      "is_detail": true,
      "status_code": 200,
      "error_message": "",
      "latency_ms": 1500,
      "cost": 0.0,
      "ip_address": "192.168.1.100",
      "user_agent": "Mozilla/...",
      "created_at": "2026-06-10T10:30:00Z"
    }
  ],
  "total": 150
}
```

---

### 3.11.2 按 Trace ID 查询

```
GET /api/request-logs/:trace_id
```

**认证**：JWT / AKSK

**路径参数**：`trace_id` — UUID 格式追踪 ID

**功能**：查询同一次请求的完整日志链（流式传输可能产生多条记录）

**返回**：`RequestLog[]` 数组（按 `created_at` 正序排列）

---

### 3.11.3 获取请求 Stream Chunks

```
GET /api/request-logs/:trace_id/chunks
```

**认证**：JWT / AKSK

**路径参数**：`trace_id` — UUID 格式追踪 ID

**功能**：获取指定请求的 SSE stream chunks（按 chunk_index 正序排列）

**返回**：`RequestChunk[]` 数组

```json
[
  {
    "id": 1,
    "trace_id": "uuid-xxx",
    "chunk_index": 0,
    "chunk_data": "{\"id\":\"chatcmpl-xxx\",\"choices\":[...]}",
    "created_at": "2026-06-10T10:30:00Z"
  }
]
```

---

# 端点总览

| # | 方法 | 路径 | 认证 | 分组 | 描述 |
|---|------|------|------|------|------|
| 1 | GET | `/health` | 无 | 系统 | 健康检查 |
| 2 | GET | `/health/ready` | 无 | 系统 | 就绪检查 |
| 3 | GET | `/v1/models` | API Key | LLM | 可用模型列表 |
| 4 | POST | `/v1/chat/completions` | API Key | LLM | Chat Completions（OpenAI 兼容） |
| 5 | POST | `/v1/messages` | API Key | LLM | Messages API（Anthropic 兼容） |
| 6 | GET | `/anthropic/models` | API Key | LLM | 可用模型列表（Anthropic 路径） |
| 7 | POST | `/anthropic/chat/completions` | API Key | LLM | Chat Completions（Anthropic 路径） |
| 8 | POST | `/anthropic/messages` | API Key | LLM | Messages API（Anthropic 路径） |
| 9 | POST | `/api/admin/login` | 无 | 认证 | 管理员登录 |
| 10 | POST | `/api/admin/logout` | JWT/AKSK | 认证 | 注销 |
| 11 | GET | `/api/admin/profile` | JWT/AKSK | 个人信息 | 获取当前用户信息 |
| 12 | POST | `/api/admin/profile/update` | JWT/AKSK | 个人信息 | 更新个人信息 |
| 13 | GET | `/api/admin/users` | JWT/AKSK | 用户 | 用户列表 |
| 14 | POST | `/api/admin/users` | JWT/AKSK | 用户 | 创建用户 |
| 15 | PUT | `/api/admin/users/:id` | JWT/AKSK | 用户 | 更新用户 |
| 16 | DELETE | `/api/admin/users/:id` | JWT/AKSK | 用户 | 删除用户 |
| 17 | GET | `/api/admin/users/:id/api-keys` | JWT/AKSK | API Key | 用户 API Key 列表 |
| 18 | POST | `/api/admin/users/:id/api-keys` | JWT/AKSK | API Key | 创建 API Key |
| 19 | DELETE | `/api/admin/users/:id/api-keys/:kid` | JWT/AKSK | API Key | 删除用户 API Key |
| 20 | GET | `/api/admin/api-keys` | JWT/AKSK | API Key | 全局 API Key 列表 |
| 21 | DELETE | `/api/admin/api-keys/:id` | JWT/AKSK | API Key | 全局删除 API Key |
| 22 | PUT | `/api/admin/api-keys/:id/toggle` | JWT/AKSK | API Key | 切换 API Key 状态 |
| 23 | POST | `/api/admin/users/:id/aksk` | JWT/AKSK | AKSK | 生成 AKSK |
| 24 | GET | `/api/admin/users/:id/aksk` | JWT/AKSK | AKSK | 获取 Access Key |
| 25 | GET | `/api/admin/providers` | JWT/AKSK | Provider | Provider 列表 |
| 26 | POST | `/api/admin/providers` | JWT/AKSK | Provider | 创建 Provider |
| 27 | PUT | `/api/admin/providers/:id` | JWT/AKSK | Provider | 更新 Provider |
| 28 | DELETE | `/api/admin/providers/:id` | JWT/AKSK | Provider | 删除 Provider |
| 29 | PUT | `/api/admin/providers/:id/toggle` | JWT/AKSK | Provider | 切换 Provider 状态 |
| 30 | POST | `/api/admin/providers/fetch-models` | JWT/AKSK | Provider | 获取上游模型列表 |
| 31 | POST | `/api/admin/providers/batch-import-models` | JWT/AKSK | Provider | 批量导入上游模型 |
| 32 | GET | `/api/admin/models` | JWT/AKSK | 上游模型 | 上游模型列表 |
| 33 | POST | `/api/admin/models` | JWT/AKSK | 上游模型 | 创建上游模型 |
| 34 | PUT | `/api/admin/models/:id` | JWT/AKSK | 上游模型 | 更新上游模型 |
| 35 | DELETE | `/api/admin/models/:id` | JWT/AKSK | 上游模型 | 删除上游模型 |
| 36 | GET | `/api/admin/downstream-models` | JWT/AKSK | 下游模型 | 下游模型列表 |
| 37 | POST | `/api/admin/downstream-models` | JWT/AKSK | 下游模型 | 创建下游模型 |
| 38 | PUT | `/api/admin/downstream-models/:id` | JWT/AKSK | 下游模型 | 更新下游模型 |
| 39 | DELETE | `/api/admin/downstream-models/:id` | JWT/AKSK | 下游模型 | 删除下游模型 |
| 40 | GET | `/api/admin/configs` | JWT/AKSK | 配置 | 配置列表 |
| 41 | PUT | `/api/admin/configs` | JWT/AKSK | 配置 | 更新配置 |
| 42 | GET | `/api/stats/tokens` | JWT/AKSK | 统计 | Token 用量统计 |
| 43 | GET | `/api/stats/requests` | JWT/AKSK | 统计 | 请求量统计 |
| 44 | GET | `/api/stats/costs` | JWT/AKSK | 统计 | 费用统计 |
| 45 | GET | `/api/stats/behavior` | JWT/AKSK | 统计 | 用户行为分析 |
| 46 | GET | `/api/dashboard/overview` | JWT/AKSK | 统计 | Dashboard 总览 |
| 47 | GET | `/api/request-logs` | JWT/AKSK | 请求日志 | 请求日志分页查询 |
| 48 | GET | `/api/request-logs/:trace_id` | JWT/AKSK | 请求日志 | 按 Trace ID 查询 |
| 49 | GET | `/api/request-logs/:trace_id/chunks` | JWT/AKSK | 请求日志 | 获取 Stream Chunks |
