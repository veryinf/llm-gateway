# Go CRUD Handler 编写规范

标准化 Go + Echo + GORM 项目的 CRUD Handler 编写模式。

## 目录结构

```
internal/
├── model/xxx.go              # GORM 数据模型
├── web/
│   ├── common/
│   │   ├── base_handler.go   # BaseHandler (DB/分页/JSON解析)
│   │   ├── response.go       # 统一响应格式
│   │   └── search_params.go  # 搜索/分页/过滤参数
│   ├── handlers/
│   │   └── xxx_handler.go    # CRUD Handler
│   └── server.go             # 路由注册
```

## 响应格式

所有接口统一返回 `{errCode, errMsg, data/dataSet}`：

```json
// 成功 - 单条数据
{"errCode": 0, "errMsg": "ok", "data": {...}}

// 成功 - 列表数据
{"errCode": 0, "errMsg": "ok", "dataSet": [...], "total": 100}

// 失败
{"errCode": -11, "errMsg": "用户名和密码不能为空"}
```

对应 Go 函数：

| 场景 | 调用 |
|---|---|
| 成功（无数据） | `h.Success()` |
| 成功（单条） | `common.NewData(entity)` |
| 成功（列表） | `common.NewDataSet(list, total)` |
| 失败 | `h.Error(code, msg)` |

## 错误码约定

| 范围 | 含义 |
|---|---|
| `-1x` | 参数校验错误（必填、格式、唯一性） |
| `-2x` | 业务逻辑错误（加密失败、不存在等） |
| `-20` | 数据库查询/分页错误（Pagination 内部使用） |

## 搜索参数结构

前端统一发送 `SearchParams`：

```json
{
  "index": 1,
  "size": 20,
  "kw": "关键词",
  "filters": [
    {"field": "status", "value": "active"},
    {"field": "role", "value": "admin"}
  ]
}
```

对应 Go 结构体 `common.SearchParams`，内嵌 `Pagination`。

## Handler 模板

以 `User` 为例，完整的 5 接口 CRUD Handler：

### 1. Handler 结构体

```go
package handlers

import (
    "your-module/internal/model"
    "your-module/internal/web/common"

    validation "github.com/go-ozzo/ozzo-validation/v4"
    "github.com/labstack/echo/v4"
)

type UserHandler struct {
    common.BaseHandler
}
```

**要点**：
- 嵌入 `common.BaseHandler`，获得 `h.DB`、`h.Pagination()`、`h.GetJSON()`、`h.Success()`、`h.Error()` 等方法
- 不需要自己持有额外字段（除非需要注入 Service）

### 2. 搜索（Search）— POST /user/search

```go
func (h *UserHandler) SearchUsers(c echo.Context) error {
    input := &common.SearchParams{}
    if err := c.Bind(input); err != nil {
        return err
    }

    // 1. 构建基础查询
    query := h.DB.Model(&model.User{}).Order("uid DESC")

    // 2. 关键词搜索（多字段 LIKE）
    if input.Kw != "" {
        kw := "%" + input.Kw + "%"
        query = query.Where(
            "username LIKE ? OR name LIKE ? OR phone LIKE ?",
            kw, kw, kw,
        )
    }

    // 3. 过滤条件（遍历 filters）
    for _, filter := range input.Filters {
        switch filter.Field {
        case "status":
            if err := validation.Validate(filter.Value,
                validation.Required,
                validation.In("active", "disabled"),
            ); err != nil {
                return h.Error(-11, err.Error())
            }
            query = query.Where("status = ?", filter.Value)
        case "role":
            query = query.Where("role = ?", filter.Value)
        }
    }

    // 4. 分页查询
    var count int64
    var users []model.User
    if err := h.Pagination(&input.Pagination, query, &users, &count); err != nil {
        return err
    }

    // 5. 关联计数（可选）
    // ... 查询关联表数量，组装扩展字段

    // 6. 返回列表
    return common.NewDataSet(users, count)
}
```

**要点**：
- `h.Pagination()` 内部处理 offset/limit/count，失败时返回 `h.Error(-20, ...)`
- 过滤值用 `validation.In()` 做白名单校验
- 关联计数用 `h.DB.Model(&关联Model{}).Select("foreign_key, count(*)").Group("foreign_key").Scan(&counts)` 批量查询，避免 N+1

### 3. 获取单条（Fetch）— POST /user/fetch

```go
func (h *UserHandler) FetchUser(c echo.Context) error {
    input := &struct {
        UID int64 `json:"uid"`
    }{}
    if err := c.Bind(input); err != nil {
        return err
    }
    if err := validation.ValidateStruct(input,
        validation.Field(&input.UID, validation.Required),
    ); err != nil {
        return h.Error(-11, err.Error())
    }

    var user model.User
    if err := h.DB.First(&user, input.UID).Error; err != nil {
        return h.Error(-24, "用户不存在")
    }
    return common.NewData(user)
}
```

**要点**：
- 用匿名 struct 定义入参，json tag 使用**首字母小写的驼峰法**
- `validation.ValidateStruct` 做字段级校验
- `h.DB.First()` 查不到时返回业务错误码（GORM 的 `RecordNotFound` 不要直接暴露）

### 4. 新增（Add）— POST /user/add

```go
func (h *UserHandler) AddUser(c echo.Context) error {
    input := &model.User{}
    if err := c.Bind(input); err != nil {
        return err
    }
    if err := validation.ValidateStruct(input,
        validation.Field(&input.Username, validation.Required),
        validation.Field(&input.Password, validation.Required),
        validation.Field(&input.Role, validation.Required,
            validation.In(string(model.RoleAdmin), string(model.RoleUser))),
    ); err != nil {
        return h.Error(-11, err.Error())
    }

    // 唯一性检查
    var exist int64
    h.DB.Model(&model.User{}).Where("username = ?", input.Username).Count(&exist)
    if exist > 0 {
        return h.Error(-12, "用户名已存在")
    }

    // 业务处理（密码加密、设置默认值等）
    hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
    if err != nil {
        return h.Error(-21, "密码加密失败")
    }
    input.Password = string(hash)
    input.Status = "active"

    if err := h.DB.Create(input).Error; err != nil {
        return h.Error(-21, err.Error())
    }
    return common.NewData(input)
}
```

**要点**：
- 直接 Bind 到 Model struct，利用 `gorm` tag 自动映射
- 唯一性检查：`COUNT` 查询 + 手动判断（不要依赖 DB 唯一索引报错）
- 业务处理在 `Create` 之前完成（加密、默认值）
- 创建后返回完整对象

### 5. 更新（Update）— POST /user/update

```go
func (h *UserHandler) UpdateUser(c echo.Context) error {
    input, err := h.GetJSON(c)
    if err != nil {
        return err
    }

    // 1. 提取主键
    uid := input.Get("uid")
    if !uid.Exists() || uid.Uint() == 0 {
        return h.Error(-23, "uid is required")
    }

    // 2. 逐字段提取，构建更新 map
    newState := map[string]any{}
    if input.Get("username").Exists() {
        newState["username"] = input.Get("username").String()
    }
    if input.Get("role").Exists() {
        role := input.Get("role").String()
        if err := validation.Validate(role,
            validation.In(string(model.RoleAdmin), string(model.RoleUser)),
        ); err != nil {
            return h.Error(-11, err.Error())
        }
        newState["role"] = role
    }
    // ... 其他字段同理

    // 3. 空更新检查
    if len(newState) == 0 {
        return h.Success()
    }

    // 4. 执行更新
    if err := h.DB.Model(&model.User{}).Where("uid = ?", uid.Uint()).Updates(newState).Error; err != nil {
        return h.Error(-22, err.Error())
    }
    return h.Success()
}
```

**要点**：
- 使用 `h.GetJSON()` + `gjson` 逐字段提取，**只更新前端传了的字段**
- 主键字段（uid/id）必须校验存在且非零
- 敏感字段（如 uid、accessKey）不应出现在 `newState` 中，防止被覆盖
- `validation.In()` 校验枚举值白名单
- 空更新直接返回成功，不执行 DB 操作

### 6. 删除（Remove）— POST /user/remove

```go
func (h *UserHandler) RemoveUser(c echo.Context) error {
    input := &struct {
        UID int64 `json:"uid"`
    }{}
    if err := c.Bind(input); err != nil {
        return err
    }
    if err := validation.ValidateStruct(input,
        validation.Field(&input.UID, validation.Required),
    ); err != nil {
        return h.Error(-11, err.Error())
    }

    if err := h.DB.Delete(&model.User{}, input.UID).Error; err != nil {
        return h.Error(-23, err.Error())
    }
    return h.Success()
}
```

**要点**：
- 删除前校验主键存在
- 级联删除在 Handler 中显式处理（先删关联表，再删主表）

### 7. 路由注册

```go
func (h *UserHandler) RegisterRoutes(g *echo.Group) {
    g.POST("/user/search", h.SearchUsers)
    g.POST("/user/fetch", h.FetchUser)
    g.POST("/user/add", h.AddUser)
    g.POST("/user/update", h.UpdateUser)
    g.POST("/user/remove", h.RemoveUser)
}
```

在 `server.go` 中注册：

```go
base := common.BaseHandler{DB: db, Store: store, TokenManager: tokenManager, Config: cfg}
(&handlers.UserHandler{BaseHandler: base}).RegisterRoutes(bizApi)
```

**要点**：
- 所有接口使用 `POST` 方法（与前端统一）
- 路由命名：`/{entity}/{action}`，action 为 search/fetch/add/update/remove
- Handler 通过结构体字面量注入 `BaseHandler`

## JSON Key 命名规范

所有 JSON key 使用**首字母小写的驼峰法**（camelCase）：

```go
// ✓ 正确
APIKeyCount int `json:"apiKeyCount"`
AccessKey   string `json:"accessKey"`

// ✗ 错误
APIKeyCount int `json:"api_key_count"`
AccessKey   string `json:"access_key"`
```

Model struct 中的 json tag 遵循同一规则，Handler 中的匿名 struct 和 map key 同理。

## GORM 主键命名约定

Model 的主键字段名决定数据库列名（GORM snake_case 转换）：

| 字段名 | GORM 列名 | 说明 |
|---|---|---|
| `ID` | `id` | 默认主键 |
| `UID` | `uid` | 自定义主键（如用户表） |
| `Key` | `key` | 字符串主键（如配置表） |

**注意**：修改主键字段名后，已有数据库的列名不会自动变更，需要手动 `ALTER TABLE RENAME COLUMN` 或添加 `gorm:"column:old_name"` tag 映射。

## 验证方式

编码结束后只运行 `go vet ./...` 做语法检查，不进行构建测试。
