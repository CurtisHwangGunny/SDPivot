# smartKnora (随越·智枢) — WeKnora Fork 侵入修改记录

> **版本**: v1.0  
> **日期**: 2026-07-01  
> **基座**: WeKnora v0.6.3 (commit 974ca359)

---

## 一、侵入修改总览

| # | 文件 | 修改类型 | 行数 | 说明 |
|---|------|---------|:----:|------|
| 1 | `internal/container/container.go` | 新增 1 行 | +1 | 注册 SmartKnoraRouter 到 DI 容器 |
| 2 | `internal/router/router.go` | 新增 6 行 | +6 | RouterParams 增加字段 + RegisterRoutes 调用 |

**总计侵入: 2 个文件, 7 行代码**

---

## 二、详细修改说明

### 2.1 `internal/container/container.go`

**修改位置**: Router configuration 注册段（约第 364 行）

**新增行**:
```go
must(container.Provide(router.NewSmartKnoraRouter)) // smartKnora (随越·智枢) DI registration
```

**作用**: 通过 dig 依赖注入框架注册 `SmartKnoraRouter`，dig 自动解析其依赖：
- `*gorm.DB` ← 由 `initDatabase()` 提供
- `*redis.Client` ← 由 `initRedisClient()` 提供

**风险评估**: 极低。仅新增一个 Provide 调用，不影响现有组件生命周期。

---

### 2.2 `internal/router/router.go`

**修改 1 — RouterParams 结构体** (第 89 行):
```go
type RouterParams struct {
    dig.In
    // ... 原有字段 ...
    WeKnoraCloudHandler  *handler.WeKnoraCloudHandler
    WikiPageHandler      *handler.WikiPageHandler
    SmartKnoraRouter     *SmartKnoraRouter  // ← 新增
}
```

**修改 2 — NewRouter 函数** (第 237 行前):
```go
// Register smartKnora (随越·智枢) routes
if params.SmartKnoraRouter != nil {
    params.SmartKnoraRouter.RegisterRoutes(r)
}
```

**作用**: 
1. 将 `SmartKnoraRouter` 注入到路由参数
2. 在 WeKnora 路由构建完成后，注册 smartKnora 路由

**风险评估**: 极低。`SmartKnoraRouter != nil` 防御性检查确保即使 DI 未注入也不崩溃。smartKnora 路由独立于 WeKnora 原有路由（`/api/v1/smartknora/*`），不会冲突。

---

## 三、新增文件清单（不侵入 WeKnora 原有代码）

### 3.1 数据库迁移（4 个文件）
| 文件 | 说明 |
|------|------|
| `migrations/postgres/000001_smartknora_init.up.sql` | 8 张核心表 + 索引 |
| `migrations/postgres/000001_smartknora_init.down.sql` | 回滚 |
| `migrations/postgres/000002_rls_policies.up.sql` | RLS 策略 + 角色 + 函数 |
| `migrations/postgres/000002_rls_policies.down.sql` | 回滚 |
| `migrations/postgres/000003_smartknora_extensions.up.sql` | 扩展表（user_profiles/org_ext） |
| `migrations/postgres/000003_smartknora_extensions.down.sql` | 回滚 |

### 3.2 Go 数据模型（7 个文件）
| 文件 | 说明 |
|------|------|
| `internal/types/smartknora.go` | **主文件** — 所有 smartKnora 类型 + DTO |
| `internal/types/knowledge_space.go` | 知识空间模型 |
| `internal/types/space_member.go` | 空间成员模型 |
| `internal/types/space_category.go` | 空间分类模型 |
| `internal/types/org_ext.go` | 企业扩展（认证状态） |
| `internal/types/refresh_token.go` | 刷新令牌模型 |
| `internal/types/token_usage.go` | Token 计量模型 |

### 3.3 认证体系（1 个文件）
| 文件 | 说明 |
|------|------|
| `internal/auth/jwt.go` | JWT 双 Token 管理器（HS256 Access + Opaque Refresh） |

### 3.4 中间件（3 个文件）
| 文件 | 说明 |
|------|------|
| `internal/middleware/smartknora_auth.go` | JWT 认证中间件 |
| `internal/middleware/smartknora_tenant.go` | 租户上下文（RLS） |
| `internal/middleware/smartknora_metering.go` | Token 计量埋点 |

### 3.5 Handler（4 个文件）
| 文件 | 说明 |
|------|------|
| `internal/handler/smartknora_auth.go` | 注册/登录/刷新/登出 |
| `internal/handler/smartknora_org.go` | 企业 CRUD/成员/认证 |
| `internal/handler/smartknora_space.go` | 知识空间 CRUD/成员/分类 |
| `internal/handler/smartknora_token.go` | Token 用量统计 |

### 3.6 路由（1 个文件）
| 文件 | 说明 |
|------|------|
| `internal/router/smartknora.go` | SmartKnoraRouter 定义 + DI 构造函数 |

### 3.7 前端（待创建）
| 目录 | 说明 |
|------|------|
| `frontend/smartknora/` | React + TDesign React 前端应用 |

---

## 四、环境变量

| 变量 | 必填 | 默认值 | 说明 |
|------|:---:|--------|------|
| `SMARTKNORA_JWT_SECRET` | 否 | `smartknora-dev-secret-change-in-production` | JWT 签名密钥 |

---

## 五、API 路由表

所有 smartKnora 路由挂载在 `/api/v1/smartknora/`，独立于 WeKnora 原有 API。

### 公开路由（无需认证）
| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/auth/register` | 用户注册（手机/邮箱） |
| POST | `/auth/login` | 用户登录 |
| POST | `/auth/refresh` | 刷新 Token |
| GET  | `/health` | 健康检查 |

### 受保护路由（需 JWT）
| 方法 | 路径 | 说明 |
|------|------|------|
| GET  | `/profile` | 获取个人资料 |
| PUT  | `/profile` | 更新个人资料 |
| PUT  | `/password` | 修改密码 |
| POST | `/organizations` | 创建企业 |
| GET  | `/organizations` | 企业列表 |
| GET  | `/organizations/:id` | 企业详情 |
| PUT  | `/organizations/:id` | 更新企业 |
| POST | `/organizations/join` | 加入企业（邀请码） |
| GET  | `/organizations/:id/members` | 成员列表 |
| POST | `/organizations/:id/members` | 添加成员 |
| DELETE | `/organizations/:id/members/:userId` | 移除成员 |
| PUT  | `/organizations/:id/members/:userId/role` | 更新角色 |
| POST | `/spaces` | 创建知识空间 |
| GET  | `/spaces` | 空间列表 |
| GET  | `/spaces/:id` | 空间详情 |
| PUT  | `/spaces/:id` | 更新空间 |
| DELETE | `/spaces/:id` | 删除空间 |
| GET  | `/spaces/:id/members` | 空间成员 |
| POST | `/spaces/:id/members` | 添加成员 |
| DELETE | `/spaces/:id/members/:userId` | 移除成员 |
| PUT  | `/spaces/:id/members/:userId` | 更新角色 |
| POST | `/categories` | 创建分类 |
| GET  | `/categories` | 分类列表 |
| DELETE | `/categories/:id` | 删除分类 |
| GET  | `/usage/summary` | Token 用量汇总 |
| GET  | `/usage/history` | 用量历史 |
| GET  | `/usage/by-model` | 按模型统计 |

---

## 六、合并 WeKnora 上游更新时的注意事项

smartKnora 侵入的 2 个文件在 WeKnora 上游更新时需检查：

1. **`internal/container/container.go`** — 若上游新增 handler/service，需要确认 `Provide(router.NewSmartKnoraRouter)` 位置仍在 `Provide(router.NewRouter)` 之前。
2. **`internal/router/router.go`** — 若上游修改 `RouterParams` 结构体或 `NewRouter` 函数体，需同步合并 `SmartKnoraRouter` 字段和 `RegisterRoutes` 调用。

其余新增文件均在独立目录/文件中，不会与上游冲突。
