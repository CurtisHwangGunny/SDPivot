# SDP（SDPivot）产品代码库

> 本仓库存放 **SDP / SDPivot** 产品代码。SDP 由开源项目 WeKnora fork 演进而来，曾用名 SmartKnora；重命名后产品品牌统一为 **SDP（SDPivot）**。

## 产品定位

SDP 是面向企业与团队的 **AI 知识库 / 文档智能问答平台**，核心能力：

- 文档 ingestion 与切片（含 cppjieba 中文分词）
- 向量检索（RAG）+ 重排（rerank）
- 多轮对话 Agent 与知识库管理
- 运营管理端（OPS）用于部署、配置与监控

后端基于 Go（`cmd/sdp-server`），前端基于 Vue 3 + Vite（`frontend/sdpivot`），文档解析服务 `docreader`。

> 仓库内仍保留上游 WeKnora 的 `README.md` / `README_CN.md` 等多语言文档作为背景参考；本文件聚焦 SDP 自身的定位与开发流程。

## 仓库结构（关键目录）

| 路径 | 说明 |
|------|------|
| `cmd/sdp-server` | 后端主服务源码（Go），部署时构建为 `sdp-server` 二进制 |
| `internal/` | 后端内部模块（agent / application / auth / database / router …） |
| `frontend/sdpivot` | 前端主应用（Vue3 + Vite），新功能在此开发 |
| `frontend/smartknora` / `smartknora-react-backup` | 历史遗留前端，保留兼容 |
| `docreader/` | 文档解析微服务 |
| `deploy/` | 部署脚本与 compose（`deploy-test.sh`、`docker-compose.test.yml` 等） |
| `migrations/` | 数据库迁移 |
| `docker-compose.yml` / `docker-compose.dev.yml` | 本地 / 开发编排 |
| `config/`、`scripts/`、`tests/` | 配置、脚本、测试 |

## 远程仓库（remote）约定

| remote | 地址 | 用途 |
|--------|------|------|
| `sdp` | `CurtisHwangGunny/SDP-OP`（私有，主仓库） | **SDP 产品代码主远端**，日常推送目标 |
| `fork` | `CurtisHwangGunny/SDPivot` | 旧 fork（SmartKnora 改名前的仓库），保留不动 |
| `origin` | `Tencent/WeKnora` | 上游开源项目，只读参考 |
| `github` | `CurtisHwangGunny/smartKnora` | 旧仓库，保留不动 |

## 开发流程（基于 `sdp` 远端）

```bash
# 1. 克隆主仓库
git clone https://github.com/CurtisHwangGunny/SDP-OP.git
cd SDP-OP

# 2. 确认主远端为 sdp
git remote -v   # 应能看到 sdp -> CurtisHwangGunny/SDP-OP

# 3. 基于默认开发分支建立特性分支
git switch -c feat/your-feature sdp/feature/op-phase0

# 4. 后端构建（产物 sdp-server 不入库，见 .gitignore）
cd cmd/sdp-server && go build -o ../../frontend/sdpivot/sdp-server . && cd ../..

# 5. 前端构建
cd frontend/sdpivot && pnpm install && pnpm build && cd ../..

# 6. 提交并推送
git add -A
git commit -m "feat: ..."
git push sdp feat/your-feature
# 在 GitHub 上对 sdp/feature/op-phase0 发起 PR
```

## 重要约定

- **构建产物不入库**：`sdp-server` / `smartknora-server` 等二进制由 `go build` 在部署阶段生成，已在 `.gitignore` 忽略，切勿 `git add` 这些文件。
- **默认分支为 `feature/op-phase0`**（当前活跃开发分支）。
- **API 路由**：canonical 前缀 `/api/v1/sdp`，兼容旧前缀 `/api/v1/smartknora` 仍保留。
- **共用底座**：数据库 `WeKnora-postgres`、缓存 `WeKnora-redis` 为历史沿用命名，部署时通过 `SDP_DB_HOST` / `SDP_REDIS_ADDR` 指向，无需改名。
