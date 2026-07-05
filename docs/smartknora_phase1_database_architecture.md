# SmartKnora Phase 1 数据库架构决策

> 适用阶段：Phase 1 快速迭代  
> 决策方案：方案 A — 保持当前架构，SMK 与 WeKnora 共享数据库

## 1. 设计决策

SmartKnora（SMK）在 Phase 1 不使用独立数据库，统一连接 WeKnora PostgreSQL 实例中的 `WeKnora` 数据库。

- SMK 后端标准配置：`SMART_DB_NAME=WeKnora`
- WeKnora 主服务标准配置：`DB_NAME=WeKnora`
- 历史遗留的 `smartknora` 独立数据库不再作为运行时依赖，并已从开发服务器清理

该模式表示：**SMK Built on top of WeKnora**，SMK 通过扩展表和扩展字段复用 WeKnora 的用户、租户、组织、模型等基础能力。

## 2. 运行时数据库连接

| 服务 | 数据库主机 | 数据库名 | 说明 |
|------|------------|----------|------|
| WeKnora-app | `WeKnora-postgres` | `WeKnora` | WeKnora 主服务 |
| smartknora-backend | `WeKnora-postgres` | `WeKnora` | SMK API 服务，共享 WeKnora 库 |

SMK 的数据库环境变量必须保持如下语义：

```env
SMART_DB_HOST=WeKnora-postgres
SMART_DB_PORT=5432
SMART_DB_USER=postgres
SMART_DB_NAME=WeKnora
```

> 注意：`SMART_DB_NAME` 不得配置为 `smartknora`。`smartknora` 独立库是历史冗余残留，不属于 Phase 1 运行架构。

## 3. tenant_id 隔离规范

SMK 与 WeKnora 共享 `users`、`organizations`、`models` 等基础表，因此必须通过 `tenant_id` 做逻辑隔离。

### 3.1 用户隔离

- SMK 用户与 WeKnora 用户位于同一个 `users` 表中。
- 用户归属由 `tenant_id` 标识，不再通过独立数据库区分。
- SMK 注册、登录、刷新 token、运营管理等接口必须以用户记录中的 `tenant_id` 作为租户上下文。

### 3.2 业务数据隔离

以下 SMK 业务表/数据访问必须携带 `tenant_id` 过滤：

- `knowledge_spaces`
- `documents`（SMK 文档表）
- `document_chunks`（SMK 文档分块表）
- `qa_sessions`（SMK QA 会话表）
- `qa_messages`（SMK QA 消息表）
- `writing_drafts`（SMK 写作草稿表）
- `token_usage`
- 其他所有租户级 SMK 表

查询规则：

```sql
-- 正确：按 tenant_id 限定
SELECT * FROM documents
WHERE tenant_id = :tenant_id AND deleted_at IS NULL;

-- 错误：缺少 tenant_id，可能跨租户读取
SELECT * FROM documents
WHERE deleted_at IS NULL;
```

### 3.3 运营管理隔离

运营管理接口可以访问共享库中的运营表，但仍需区分：

- 运维账号识别：`users.is_ops_admin = true`
- 企业/用户/空间等租户级数据：必须保留 `tenant_id` 或企业归属过滤
- 系统级配置、公告、敏感词等全局数据：需在接口层明确为系统作用域

## 4. 冗余数据库清理

开发服务器历史存在 `smartknora` 独立数据库，但该库未被 SMK 后端使用，且存在以下问题：

- 与 WeKnora `users.id` 类型不一致
- 缺失 `enterprises`、`billing_plans`、`audit_logs`、`models` 等运营必需表
- 与当前 SMK API 没有运行时连接关系
- 容易造成部署与排障混淆

因此 Phase 1 执行：

```sql
DROP DATABASE smartknora;
```

执行前应保留 `pg_dump` 备份，便于必要时审计或回滚。

## 5. 后续 Phase 2 方向

如 Phase 2 需要 SMK 独立数据库，应重新设计：

1. 独立 SMK schema/数据库与完整运营表结构
2. SMK 用户与 WeKnora 用户映射表
3. 双向同步或 SSO 桥接机制
4. 统一 ID 类型与外键策略
5. 全量迁移与回归测试

在 Phase 2 启动前，不应恢复 `smartknora` 独立库作为运行配置。
