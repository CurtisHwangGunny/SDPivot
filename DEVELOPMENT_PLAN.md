# SDPivot Sprint 1 开发计划

> 新 API 使用 `/api/v1/sdp`，新部署使用后端实际读取的 `SDP_DB_*`、`SDP_REDIS_*`、`SDP_UPLOAD_DIR`，品牌专属变量使用 `SDP_*`；历史路径与数据标识视为 legacy，不做破坏性重命名。


> **基座项目**: WeKnora (fork at `/home/ubuntu/projects/weknora/`)
> **废弃代码**: `/home/ubuntu/projects/SDPivot-1/` (已标记-1，勿参考)
> **设计文档**: `/home/ubuntu/产品设计文档/` (全套13份)
> **技术栈**: Go 1.26 (Gin + GORM + PostgreSQL + pgvector) + Vue 3 + TDesign React + Vite

## 核心原则
1. **产品设计以 PRD 文档为准** (`3、随越智枢_完整PRD文档_v4.1.md`)
2. **技术架构按文档执行** — 多租户 RLS、双Token、存储抽象层等
3. **前端页面按原型图开发** (`7、随越智枢_完整原型图_v1.0.html`)
4. **WeKnora + fork 方式** — 基于 weknora 代码库修改，不重写

## Sprint 1 分阶段任务

### Phase 1: 基础设施与工程化 (S1-I)
- [x] S1-I-01: WeKnora fork + 项目结构初始化
- [ ] S1-I-02: PostgreSQL 15+ 数据库 + pgvector 扩展
- [ ] S1-I-03: Redis + Asynq 队列初始化
- [ ] S1-I-04: MinIO/COS 存储适配层抽象 (StorageProvider 接口)
- [ ] S1-I-05: Langfuse 可观测性接入
- [ ] S1-I-06: 前端脚手架 (React + TypeScript + Vite + TDesign React)
- [ ] S1-I-07: Docker Compose 本地开发环境编排
- [ ] S1-I-08: 私有化部署配置化开关 (feature flag)

### Phase 2: 用户账号体系 (S1-A)
- [ ] S1-A-01: 用户注册/登录 API (手机号+邮箱)
- [ ] S1-A-02: 双Token认证 (Access JWT 15min + Refresh Opaque 7d)
- [ ] S1-A-03: 个人账号管理 API
- [ ] S1-A-04: 登录/注册前端页面 (TDesign)
- [ ] S1-A-05: 个人设置页面

### Phase 3: Token计量与计费基座 (S1-T)
- [ ] S1-T-01: Token计量数据模型 + 数据库表
- [ ] S1-T-02: API中间件：请求级Token计量埋点
- [ ] S1-T-03: 计量查询API
- [ ] S1-T-04: 计量看板前端组件

### Phase 4: 多租户基座 (S1-M)
- [ ] S1-M-01: 企业/组织数据模型
- [ ] S1-M-02: RLS行级隔离策略定义与实施
- [ ] S1-M-03: 企业创建API
- [ ] S1-M-04: 企业申请加入/邀请机制
- [ ] S1-M-05: 企业认证付费周期
- [ ] S1-M-06: Skip/Check RLS机制 (运营端特权访问)
- [ ] S1-M-07: 前端：企业创建/加入页面
- [ ] S1-M-08: 前端：企业认证状态展示

### Phase 5: 知识空间基础 (S1-K)
- [ ] S1-K-01: KnowledgeSpace 数据模型 + API CRUD
- [ ] S1-K-02: 知识空间成员角色模型
- [ ] S1-K-03: 知识空间分类/标签
- [ ] S1-K-04: 前端：知识空间列表页
- [ ] S1-K-05: 前端：知识空间设置页

## 关键技术决策
- D1: 完整多租户 (RLS行级隔离)
- D4: PostgreSQL RLS + tenant_id 自动注入
- D6: 双Token (Access JWT 15min + Refresh Opaque 7d, Redis存储)
- D7: Token计量埋点
- D8: TDesign React 组件库
- D9: 私有化预留接口抽象
- D10: 微信扫码登录推迟至Phase 2 (Sprint 4)

## WeKnora 代码结构 (关键目录)
- `cmd/server/` - Go API 服务入口
- `cmd/desktop/` - 桌面应用入口
- `config/` - 配置文件
- `frontend/` - Vue 3 前端 (注意: WeKnora原版用Vue，需改为React+TDesign)
- `docreader/` - 文档解析器
- `dataset/` - 数据集处理
- `docker/` - Docker 配置
- `go.mod` - Go 依赖 (module: github.com/Tencent/WeKnora)
