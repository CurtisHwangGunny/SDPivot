# 随越·智枢 (smartKnora) 部署报告

> **版本**: v2.0.0  
> **日期**: 2026-07-02  
> **服务器**: 腾讯云 VM (43.133.61.77)

---

## 一、部署概览

| 组件 | 状态 | 端口 | 说明 |
|------|:----:|:----:|------|
| PostgreSQL | ✅ 运行中 | 5432 | WeKnora-postgres (ParadeDB PG17) |
| Redis | ✅ 运行中 | 内部 | WeKnora-redis |
| smartKnora 后端 | ⚠️ 待更新 | 8082 | 需重启加载新版本 |
| smartKnora 前端 | ✅ 运行中 | 3099 | React + TDesign React |
| WeKnora 后端 | ✅ 运行中 | 8080 | 原版 WeKnora |
| WeKnora 前端 | ✅ 运行中 | 81 | 原版 WeKnora |

---

## 二、数据库部署

### 2.1 数据库创建
```sql
CREATE DATABASE smartknora;
CREATE EXTENSION IF NOT EXISTS vector;  -- pgvector 向量扩展
```

### 2.2 迁移执行结果

| 迁移文件 | 状态 | 创建对象 |
|---------|:----:|---------|
| 000001_smartknora_init.up.sql | ✅ | 8 张核心表 + 索引 |
| 000002_rls_policies.up.sql | ✅ | RLS 策略 + 角色 + 函数 |
| 000003_smartknora_extensions.up.sql | ✅ | 2 张扩展表 + RLS |

### 2.3 数据库表清单

| 表名 | 说明 | RLS |
|------|------|:---:|
| users | 用户表 | - |
| organizations | 企业表 | - |
| org_members | 企业成员表 | - |
| refresh_tokens | 刷新令牌表 | - |
| token_usage | Token 计量表 | ✅ |
| knowledge_spaces | 知识空间表 | ✅ |
| space_members | 空间成员表 | - |
| space_categories | 空间分类表 | ✅ |
| smartknora_user_profiles | 用户扩展表 | - |
| org_ext | 企业扩展表 | ✅ |

### 2.4 RLS 角色

| 角色 | 权限 |
|------|------|
| app_user | 普通应用用户，受 RLS 策略约束 |
| ops_admin | 运营管理员，BYPASSRLS 特权 |

---

## 三、后端部署

### 3.1 二进制信息
- **路径**: `/tmp/smartknora-server`
- **大小**: 69MB
- **类型**: ELF 64-bit LSB executable
- **Go 版本**: 1.22.5

### 3.2 环境变量

| 变量 | 值 | 说明 |
|------|---|------|
| SMART_DB_HOST | WeKnora-postgres | PostgreSQL 主机 |
| SMART_DB_PORT | 5432 | PostgreSQL 端口 |
| SMART_DB_USER | postgres | 数据库用户 |
| SMART_DB_PASSWORD | *** | 数据库密码 |
| SMART_DB_NAME | WeKnora | 数据库名（SMK 与 WeKnora 共享数据库，Phase 1 方案 A） |
| SMARTKNORA_JWT_SECRET | *** | JWT 签名密钥 |
| PORT | 8081 | 服务监听端口 |

### 3.3 数据库架构决策（Phase 1 方案 A）

- **设计决策**：SMK 与 WeKnora 共享 PostgreSQL 数据库 `WeKnora`，SMK 不再维护独立 `smartknora` 数据库。
- **隔离规范**：SMK 用户、空间、文档、会话、写作草稿等业务数据必须通过 `tenant_id` 与 WeKnora 用户及数据区分；所有 SMK 查询必须继承认证上下文中的 `tenant_id` 并加租户过滤。
- **环境变量要求**：`SMART_DB_NAME=WeKnora` 是标准配置；不得配置为 `smartknora`。
- **冗余库处理**：历史遗留 `smartknora` 独立库已按方案 A 清理；如需回滚，可使用 `backups/` 下的 drop 前备份。

### 3.3 部署状态

**当前状态**: ⚠️ 待重启

新版本二进制已编译完成并复制到容器，需要重启容器加载新版本。

**重启命令**:
```bash
docker restart smartknora-backend
```

---

## 四、前端部署

### 4.1 构建信息
- **技术栈**: React 18 + TypeScript + Vite + TDesign React
- **构建时间**: 2.31s
- **输出大小**: JS 892KB + CSS 235KB

### 4.2 Docker 镜像
- **镜像名**: smartknora-frontend-v2
- **基础镜像**: nginx:alpine
- **端口映射**: 3100:80

### 4.3 页面清单

| 页面 | 路由 | 状态 |
|------|------|:----:|
| 登录页 | /login | ✅ |
| 注册页 | /register | ✅ |
| 工作台 | /workspace | ✅ |
| 知识空间 | /spaces | ✅ |
| 空间详情 | /spaces/:id | ✅ |
| 企业管理 | /org | ✅ |
| 个人设置 | /settings | ✅ |
| 用量统计 | /usage | ✅ |

---

## 五、API 路由表

### 5.1 公开路由（无需认证）

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /api/v1/smartknora/auth/register | 用户注册 |
| POST | /api/v1/smartknora/auth/login | 用户登录 |
| POST | /api/v1/smartknora/auth/refresh | 刷新 Token |
| GET | /api/v1/smartknora/health | 健康检查 |

### 5.2 受保护路由（需 JWT）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/v1/smartknora/profile | 获取个人资料 |
| PUT | /api/v1/smartknora/profile | 更新个人资料 |
| PUT | /api/v1/smartknora/password | 修改密码 |
| POST | /api/v1/smartknora/organizations | 创建企业 |
| GET | /api/v1/smartknora/organizations | 企业列表 |
| GET | /api/v1/smartknora/organizations/:id | 企业详情 |
| PUT | /api/v1/smartknora/organizations/:id | 更新企业 |
| POST | /api/v1/smartknora/organizations/join | 加入企业 |
| GET | /api/v1/smartknora/organizations/:id/members | 成员列表 |
| POST | /api/v1/smartknora/organizations/:id/members | 添加成员 |
| DELETE | /api/v1/smartknora/organizations/:id/members/:userId | 移除成员 |
| PUT | /api/v1/smartknora/organizations/:id/members/:userId/role | 更新角色 |
| POST | /api/v1/smartknora/spaces | 创建知识空间 |
| GET | /api/v1/smartknora/spaces | 空间列表 |
| GET | /api/v1/smartknora/spaces/:id | 空间详情 |
| PUT | /api/v1/smartknora/spaces/:id | 更新空间 |
| DELETE | /api/v1/smartknora/spaces/:id | 删除空间 |
| GET | /api/v1/smartknora/spaces/:id/members | 空间成员 |
| POST | /api/v1/smartknora/spaces/:id/members | 添加成员 |
| DELETE | /api/v1/smartknora/spaces/:id/members/:userId | 移除成员 |
| PUT | /api/v1/smartknora/spaces/:id/members/:userId | 更新角色 |
| POST | /api/v1/smartknora/categories | 创建分类 |
| GET | /api/v1/smartknora/categories | 分类列表 |
| DELETE | /api/v1/smartknora/categories/:id | 删除分类 |
| GET | /api/v1/smartknora/usage/summary | Token 用量汇总 |
| GET | /api/v1/smartknora/usage/history | 用量历史 |
| GET | /api/v1/smartknora/usage/by-model | 按模型统计 |

---

## 六、访问地址

| 服务 | 地址 | 说明 |
|------|------|------|
| smartKnora 前端 | http://43.133.61.77:3099 | React 应用 |
| smartKnora 后端 | http://43.133.61.77:8082 | API 服务 |
| WeKnora 前端 | http://43.133.61.77:81 | 原版 WeKnora |
| WeKnora 后端 | http://43.133.61.77:8080 | 原版 WeKnora API |

---

## 七、待完成事项

### 7.1 部署待完成
- [ ] 重启 smartknora-backend 容器加载新版本二进制
- [ ] 验证新版本 API 功能正常

### 7.2 Sprint 2-4 开发待完成
- [ ] Sprint 2: 知识导入 + 解析
- [ ] Sprint 3: AI 问答 + 管理后台
- [ ] Sprint 4: AI 写作辅助 + 运营管理端

---

## 八、技术架构

```
┌─────────────────────────────────────────────────────────────┐
│                    smartKnora v2.0.0                         │
├─────────────────────────────────────────────────────────────┤
│  Frontend: React 18 + TypeScript + Vite + TDesign React    │
│  Backend: Go 1.22 + Gin + GORM + JWT (HS256)              │
│  Database: PostgreSQL 17 + pgvector + RLS                  │
│  Cache: Redis 7.0                                          │
│  Auth: Dual Token (Access 15min + Refresh 7d)              │
└─────────────────────────────────────────────────────────────┘
```

---

**报告生成时间**: 2026-07-02 01:15 CST
