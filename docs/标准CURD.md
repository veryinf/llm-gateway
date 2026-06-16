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

---

# React 前端 CRUD 页面编写规范

标准化 React 19 + TanStack Router + TanStack React Query 项目的 CRUD 页面编写模式。

## 前端目录结构

```
web/src/
├── routes/
│   └── xxx.tsx              # 页面路由（TanStack Router 文件路由）
├── services/
│   └── xxx.ts               # 数据服务（封装 API 调用）
├── components/
│   ├── full-page.tsx        # Page 组件（核心 CRUD 页面容器）
│   ├── descriptions.tsx     # Descriptions 组件（详情展示）
│   ├── form/
│   │   ├── form-field.tsx   # 表单字段组件
│   │   └── popup-form.tsx   # 弹窗表单
│   ├── data-table/
│   │   └── full-table.tsx   # 表格组件
│   ├── modal.tsx            # Modal 组件
│   ├── confirm.tsx          # 确认对话框
│   └── ui/                  # 基础 UI 组件（Radix + shadcn）
└── typings.d.ts             # 类型定义
```

## 类型定义

### API 命名空间 (`typings.d.ts`)

```typescript
declare namespace API {
  type PrimaryKeyType = number;

  export type Service<T> = {
    primaryKey: (entity: T) => PrimaryKeyType;  // 主键提取函数
    title: (entity: T) => string;               // 实体标题（用于显示）
    search: (params: SearchParams) => Promise<API.DataSet<T>>;
    fetch: (id: PrimaryKeyType) => Promise<API.Data<T>>;
    add: (params: T) => Promise<API.ResponseStruct>;
    update: (id: PrimaryKeyType, params: Partial<T>) => Promise<API.ResponseStruct>;
    delete: (id: PrimaryKeyType) => Promise<API.ResponseStruct>;
  };

  export interface SearchParams {
    kw?: string;
    filters?: { field: string; value: unknown }[];
    pagination?: { pageIndex: number; pageSize: number };
  }

  export interface ResponseStruct {
    errCode: number;
    errMsg: string;
  }

  export interface DataSet<T> {
    errCode: number;
    errMsg: string;
    dataSet: T[];
    total: number;
  }

  export interface Data<T> {
    errCode: number;
    errMsg: string;
    data?: T;
  }
}
```

### ColumnMeta 扩展

```typescript
declare module '@tanstack/react-table' {
  interface ColumnMeta<TData extends RowData, TValue> {
    label?: string;          // 列标题（用于表单字段显示）
    viewDetail?: boolean;    // 点击时是否展示详情弹窗
    className?: string;      // 列宽度样式（如 'w-20'）
    thClassName?: string;    // 表头样式
    tdClassName?: string;    // 单元格样式
    emuns?: OptionsItem[];   // 枚举值
  }
}
```

## Service 编写规范

Service 是数据服务层，封装与后端的 API 通信。

### 模板

```typescript
import { request } from '@/lib';
import type { API } from '@/typings';

// 1. 定义实体接口
export interface User {
  uid: number;
  username: string;
  name: string;
  role: 'admin' | 'user' | 'viewer';
  status: string;
  apiKeyCount: number;
}

// 2. 定义创建参数（可选，当创建和查询结构不同时）
export interface CreateUserParams {
  username: string;
  password: string;
  role?: 'admin' | 'user' | 'viewer';
}

// 3. 实现 Service 接口
export const userService: API.Service<User> = {
  // 主键提取
  primaryKey: (entity) => entity.uid,
  // 实体标题（显示在编辑弹窗标题等处）
  title: (entity) => entity.username,

  // 搜索 - POST 请求，参数为 SearchParams
  async search(params) {
    const res = await request.post<API.DataSet<User>>('/user/search', params);
    return res.data;
  },

  // 获取单条 - POST 请求，参数为 { primaryKey }
  async fetch(uid) {
    const res = await request.post<API.Data<User>>('/user/fetch', { uid });
    return res.data;
  },

  // 新增 - POST 请求，参数为实体
  async add(params) {
    const res = await request.post<API.ResponseStruct>('/user/add', params);
    return res.data;
  },

  // 更新 - POST 请求，参数为主键 + 部分实体
  async update(uid, params) {
    const res = await request.post('/user/update', { uid, ...params });
    return res.data;
  },

  // 删除 - POST 请求，参数为主键
  async delete(id) {
    const res = await request.post('/user/remove', { uid: id });
    return res.data;
  },
};
```

### 要点

- 所有接口使用 `POST` 方法（与后端统一）
- 路由命名：`/{entity}/{action}`，action 为 search/fetch/add/update/remove
- `primaryKey` 函数用于 Page 组件提取实体主键
- `title` 函数用于 Page 组件显示实体标题（如编辑弹窗标题）

## Page 组件

`Page` 是核心 CRUD 页面容器组件，封装了表格、表单、详情弹窗等标准交互。

### Props 定义

```typescript
type PageProps<TEntity> = {
  infomation: PageInformation;           // 页面信息配置
  columns: ColumnDef<TEntity, any>[];    // 表格列定义
  service: API.Service<TEntity>;         // 数据服务
  entityTransfer?: (entity: TEntity) => TEntity;  // 实体转换函数
  formInitialValue?: (formType: FormType, entity?: TEntity) => TEntity | Promise<TEntity>;  // 表单初始值
  formAddValidator?: (entity: TEntity) => boolean | Promise<boolean>;    // 添加表单验证
  formUpdateValidator?: (entity: TEntity, original: TEntity) => boolean | Promise<boolean>;  // 更新表单验证
  onViewDetail?: (entity: TEntity) => void;        // 点击详情回调
  renderViewDetail?: (entity: TEntity) => ReactNode;  // 详情视图渲染
  renderViewForm?: (form: EasyFormApi<TEntity>, entity: TEntity | undefined, formType: FormType) => ReactNode;  // 通用表单渲染
  renderViewAdd?: (form: EasyFormApi<TEntity>) => ReactNode;  // 添加表单渲染
  renderViewUpdate?: (form: EasyFormApi<TEntity>, entity: TEntity) => ReactNode;  // 更新表单渲染
  options?: {
    showSelectColumn?: boolean;      // 是否显示选择列（默认 true）
    showOptionColumn?: boolean;      // 是否显示操作列（默认 true）
    useRefetchDetail?: boolean;      // 详情是否重新获取数据
    useRefetchUpdate?: boolean;      // 编辑时是否重新获取数据
  };
};
```

### PageInformation 结构

```typescript
type PageInformation = {
  name: string;           // 页面唯一标识（英文），用作 React Query Key
  entityName: string;     // 实体名称（中文），用于显示
  page: {
    title: string;        // 页面标题
    description: string;  // 页面描述
  };
  breadcrumbs?: { title: string }[];  // 面包屑导航
};
```

## 完整页面模板

```tsx
import { createFileRoute } from '@tanstack/react-router';
import { useEffect } from 'react';
import type { ColumnDef } from '@tanstack/react-table';
import { Page, type PageInformation } from '@/components/full-page';
import { Descriptions } from '@/components/descriptions';
import { FormFieldInput, FormFieldSelect } from '@/components/form';
import { Badge } from '@/components/ui/badge';
import { userService, type User } from '@/services/user';
import { useBreadcrumb } from '@/hooks/use-breadcrumb';

// 1. 路由定义
export const Route = createFileRoute('/users')({
  component: UsersPage,
});

// 2. 枚举选项
const roleOptions = [
  { label: '管理员', value: 'admin' },
  { label: '普通用户', value: 'user' },
  { label: '只读', value: 'viewer' },
];

// 3. 页面信息配置
const pageInformation: PageInformation = {
  name: 'users',                    // 唯一标识
  entityName: '用户',               // 中文名称
  page: { title: '用户管理', description: '管理系统用户账号和权限' },
  breadcrumbs: [{ title: '管理' }, { title: '用户管理' }],
};

// 4. 表格列定义
const columns: ColumnDef<User, any>[] = [
  {
    accessorKey: 'username',
    header: '用户名',
    meta: { label: '用户名', viewDetail: true },  // viewDetail: 点击时展示详情
  },
  {
    accessorKey: 'name',
    header: '姓名',
    meta: { label: '姓名', className: 'w-20' },   // className: 控制列宽
  },
  {
    accessorKey: 'role',
    header: '角色',
    meta: { label: '角色', className: 'w-[90px]' },
    cell: ({ row }) => {
      // 自定义单元格渲染
      const role = row.original.role;
      return <Badge>{roleOptions.find(r => r.value === role)?.label ?? role}</Badge>;
    },
  },
];

// 5. 页面组件
function UsersPage() {
  const { setBreadcrumbs } = useBreadcrumb();

  useEffect(() => {
    setBreadcrumbs(pageInformation.breadcrumbs ?? []);
  }, []);

  return (
    <Page<User>
      infomation={pageInformation}
      columns={columns}
      service={userService}
      options={{ showSelectColumn: false }}

      // 表单初始值
      formInitialValue={(_type, entity) => ({
        uid: entity?.uid ?? 0,
        username: entity?.username ?? '',
        password: '',
        name: entity?.name ?? '',
        role: entity?.role ?? 'user',
      })}

      // 详情视图
      renderViewDetail={(entity) => <UserDetail entity={entity} />}

      // 添加表单
      renderViewAdd={(form) => (
        <div className="grid grid-cols-12 gap-4">
          <FormFieldInput
            className="col-span-4"
            form={form}
            name="username"
            title="用户名"
            required
            placeholder="请输入用户名"
          />
          <FormFieldInput
            className="col-span-4"
            form={form}
            name="password"
            title="密码"
            required
            type="password"
          />
          <FormFieldSelect
            className="col-span-4"
            form={form}
            name="role"
            title="角色"
            options={roleOptions}
          />
        </div>
      )}

      // 更新表单
      renderViewUpdate={(form, _entity) => (
        <div className="grid grid-cols-12 gap-4">
          <FormFieldInput className="col-span-4" form={form} name="username" title="用户名" required />
          <FormFieldInput className="col-span-4" form={form} name="password" title="密码" placeholder="留空不修改" type="password" />
          <FormFieldSelect className="col-span-4" form={form} name="role" title="角色" options={roleOptions} />
        </div>
      )}
    />
  );
}
```

## 表单字段组件

### FormFieldInput

```tsx
<FormFieldInput
  form={form}                    // 表单 API 实例（必填）
  name="username"                // 字段名（必填，对应实体属性）
  title="用户名"                 // 字段标题
  required                       // 是否必填
  placeholder="请输入用户名"      // 占位符
  type="text|password|number"    // 输入类型
  className="col-span-4"        // 容器样式（用于 grid 布局）
  description="字段说明"         // 字段描述
  tips="提示信息"                // 提示图标内容
  validators={{                  // 自定义验证器
    onChange: ({ value }) => !value ? '必填项' : undefined,
  }}
/>
```

### FormFieldSelect

```tsx
<FormFieldSelect
  form={form}
  name="status"
  title="状态"
  options={[
    { label: '启用', value: 'active' },
    { label: '禁用', value: 'disabled' },
  ]}
  onCreate={() => {}}    // 新增选项回调（可选）
  onRefresh={() => {}}   // 刷新选项回调（可选）
/>
```

### FormFieldTextarea

```tsx
<FormFieldTextarea
  form={form}
  name="description"
  title="描述"
  rows={3}
/>
```

### FormFieldSwitch / FormFieldCheckbox

```tsx
<FormFieldSwitch
  form={form}
  name="isActive"
  title="启用状态"
  switchLabel="启用"
/>

<FormFieldCheckbox
  form={form}
  name="agreeTerms"
  title="同意条款"
/>
```

### 表单布局规范

使用 12 列网格布局：

```tsx
<div className="grid grid-cols-12 gap-4">
  {/* 1/3 宽度 */}
  <FormFieldInput className="col-span-4" ... />

  {/* 1/2 宽度 */}
  <FormFieldInput className="col-span-6" ... />

  {/* 全宽 */}
  <FormFieldInput className="col-span-12" ... />
</div>
```

## 详情视图

使用 `Descriptions` 组件展示实体详情。

### Descriptions 组件

```tsx
<Descriptions
  title="用户信息"                    // 标题
  labelClassName="w-20"              // 标签宽度
  column={2}                         // 列数（默认 2）
  items={[
    { label: '用户名', value: entity.username },
    { label: '姓名', value: entity.name || '-' },
    {
      label: '角色',
      value: <Badge>{entity.role}</Badge>,  // value 支持 ReactNode
    },
  ]}
/>
```

### 完整详情视图示例

```tsx
function UserDetail({ entity }: { entity: User }) {
  return (
    <div className="flex flex-col gap-4">
      {/* 基本信息 */}
      <Descriptions
        title="用户信息"
        items={[
          { label: '用户名', value: entity.username },
          { label: '姓名', value: entity.name || '-' },
          {
            label: '状态',
            value: (
              <Badge variant={entity.status === 'active' ? 'default' : 'destructive'}>
                {entity.status === 'active' ? '启用' : '禁用'}
              </Badge>
            ),
          },
        ]}
      />

      {/* 关联数据卡片 */}
      <Card>
        <CardHeader>
          <CardTitle>关联数据</CardTitle>
        </CardHeader>
        <CardContent>
          {/* 使用 useQuery 获取关联数据 */}
          {/* 使用 Table 组件展示 */}
        </CardContent>
      </Card>
    </div>
  );
}
```

## 自定义操作列

默认提供「详情/编辑/删除」操作菜单，可额外添加列：

```tsx
const allColumns: ColumnDef<User, any>[] = [
  ...columns,
  {
    id: 'api_keys',
    header: 'API Keys',
    meta: { label: 'API Keys', className: 'w-24' },
    cell: ({ row }) => (
      <Button variant="ghost" size="sm" asChild>
        <Link to="/app-keys" search={{ user_id: row.original.uid }}>
          {row.original.apiKeyCount}
        </Link>
      </Button>
    ),
  },
];
```

## 使用 useQuery 和 useMutation

在详情视图中，可以使用 React Query 获取/修改关联数据：

```tsx
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';

function UserDetail({ entity }: { entity: User }) {
  const queryClient = useQueryClient();

  // 查询关联数据
  const { data: apiKeys = [], isLoading } = useQuery({
    queryKey: ['user-api-keys', entity.uid],
    queryFn: () => apiKeyService.listByUser(entity.uid),
  });

  // 删除操作
  const deleteMutation = useMutation({
    mutationFn: (keyId: number) => apiKeyService.delete(entity.uid, keyId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['user-api-keys', entity.uid] });
      toast.success('删除成功');
    },
  });

  // 创建操作
  const createMutation = useMutation({
    mutationFn: (params: CreateParams) => apiKeyService.create(entity.uid, params),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['user-api-keys', entity.uid] });
      toast.success('创建成功');
    },
  });
}
```

## 使用 Modal 和 Confirm

```tsx
import { useModal } from '@/components/modal';
import { useConfirm } from '@/components/confirm';

function MyComponent() {
  const { Modal, modalHandler } = useModal();
  const { Confirm, confirmHandler } = useConfirm();

  // 打开弹窗
  const handleOpenModal = () => {
    modalHandler.open('弹窗标题');
  };

  // 确认操作
  const handleDelete = () => {
    confirmHandler.confirmInvoke(
      '确认删除',                        // 标题
      async () => {
        await deleteSomething();
        return true;                     // 返回 true 表示成功
      },
      '确认要删除吗？',                   // 确认消息
      true,                              // 是否显示成功提示
    );
  };

  return (
    <>
      <Button onClick={handleOpenModal}>打开弹窗</Button>
      <Button onClick={handleDelete} variant="destructive">删除</Button>

      {/* 弹窗内容 */}
      <Modal>
        <div>弹窗内容</div>
      </Modal>

      {/* 确认对话框（必须放在最后） */}
      <Confirm />
    </>
  );
}
```

## 验证方式

前端编码结束后运行：

```powershell
cd web && pnpm lint    # ESLint 语法检查
```
