# SmartKnora 新产品设计 v4.2 / 架构评审 v1.1 差异落地报告

## 1. 本次对齐范围

依据 4 份新设计文件完成代码与部署对齐：

- `随越智枢_多租户数据库隔离技术方案_v1.1.md`
- `随越智枢_技术架构评审报告_v1.1.md`
- `随越智枢_完整PRD文档_v4.2.md`
- `随越智枢_Sprint1-4任务拆分计划_v1.3.md`

本次主要落地 Phase 1 明确且低风险的差异：共享 WeKnora 数据库、BIGINT tenant RLS、30/90 天用户生命周期字段、AI 写作知识来源与 DOCX 导出、前端品牌 token 与注册密码校验。

## 2. 已实施代码调整

### 2.1 用户生命周期字段

文件：

- `internal/types/user.go`
- `internal/handler/smartknora_auth.go`
- `migrations/postgres/000010_smartknora_v42_alignment.up.sql`

调整：

- `users` 增加：
  - `trial_started_at`
  - `trial_phase`
  - `authenticated_at`
  - `auth_extended_at`
  - `paid_at`
- 注册新用户时初始化：
  - `trial_started_at = now`
  - `trial_phase = "30day"`
- 默认组织 `org_ext.auth_expires_at` 初始化为注册后 30 天。

### 2.2 RLS 租户上下文修正

文件：

- `internal/middleware/smartknora_tenant.go`
- `internal/middleware/smartknora_auth.go`
- `migrations/postgres/000010_smartknora_v42_alignment.up.sql`

调整：

- 将 `set_tenant_context` 改为 BIGINT tenant_id 版本，匹配当前 `tenant_id uint64/BIGINT` 实现。
- 新增 `app.is_ops_admin` session setting。
- 新增 helper：
  - `get_current_tenant_id()`
  - `is_ops_admin_context()`
- SaaS 普通 API 拒绝 `role="ops_admin"` token，避免运营端 token 混入普通租户 API。
- 对关键 RLS policy 使用 helper 函数，避免 current_setting 缺失时报错。

### 2.3 AI 写作来源与 DOCX 导出

文件：

- `internal/types/smartknora.go`
- `internal/handler/smartknora_writing.go`
- `frontend/smartknora/src/api/writing.ts`
- `frontend/smartknora/src/views/writing/WritingPage.vue`
- `migrations/postgres/000010_smartknora_v42_alignment.up.sql`

调整：

- `writing_drafts` 增加：
  - `source_type`
  - `web_search_enabled`
- 新增企业写作类别映射表：`write_category_config`。
- 前端新建写作增加知识来源切换：
  - 仅知识库
  - 知识库 + 互联网搜索
- 类别对齐 PRD v4.2 示例：通知、公告、技术文档、会议纪要、制度解读、报告等。
- 后端 DOCX 导出已可用；PDF 仍明确返回 501，避免假成功。

说明：互联网搜索当前保留语义字段与提示词入口，实际搜索源后续需要接搜索 provider。

### 2.4 前端品牌与注册校验

文件：

- `frontend/smartknora/src/style.css`
- `frontend/smartknora/src/views/auth/RegisterPage.vue`

调整：

- 全局 accent 从旧紫色改为主色 `#014DB2`。
- 字体族加入 `Inter` 与 `更纱黑体 SC / Sarasa Gothic SC`。
- 注册页密码提示与前端校验从 6 位改为 8 位，与后端 `min=8` 对齐。

## 3. 已执行数据库迁移

已在开发服务器共享数据库 `WeKnora` 执行：

- `migrations/postgres/000010_smartknora_v42_alignment.up.sql`

验证结果：

- `users` 已存在生命周期字段。
- `writing_drafts` 已存在来源字段。
- `write_category_config` 已创建。
- `get_current_tenant_id()` 正常返回。

## 4. 构建、部署与验证

### 4.1 后端

- Go 关键包测试通过：
  - `./cmd/smartknora-server`
  - `./internal/handler`
  - `./internal/middleware`
  - `./internal/types`
- 后端二进制已重新编译。
- `smartknora-backend:latest` 镜像已重新构建。
- `smartknora-backend` 容器已重建并健康运行。
- 健康检查：`http://127.0.0.1:8082/health` 返回 OK。

### 4.2 前端

- `frontend/smartknora` 生产构建通过。
- 3099 预览服务已加载新构建产物。
- 已验证 bundle 包含：
  - `knowledge_plus_web`
  - `密码至少8位`

### 4.3 接口烟测

- 注册接口通过，响应包含：
  - `success=true`
  - `token == access_token`
  - `user.trial_phase = 30day`
- AI 写作 DOCX 导出通过：
  - HTTP 200
  - 文件类型为 Microsoft Word 2007+
  - zip 内容包含 `[Content_Types].xml` 与 `word/document.xml`

## 5. 未完成 / 后续建议

- RLS 当前仍是 session setting 方式。迁移已修正 helper 与 policy，但严格连接池隔离建议下一步改造为 per-request transaction / connection-scoped context。
- AI 写作“互联网搜索”本次完成产品字段、UI 与 prompt 入口，真实 Web Search provider 仍需后续接入。
- PDF 导出仍未实现，当前明确返回 501。
- Redis `NOAUTH Authentication required` 是既有部署配置遗留问题，本次未处理。
- 前端 chunk 仍有超过 500KB 的构建警告，建议后续做动态 import 和代码分包。
