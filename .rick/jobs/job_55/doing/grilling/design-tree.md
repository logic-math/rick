# 设计树：job_55 —— 10.99.19.123 super 模式「找不到 super loop 文件」排查修复

## 根层（O + KR）

### O（全局目标）
修复 10.99.19.123 上部署的 rick web（源：ssh://git@git.sankuai.com/wm_ai/infer-rsi.git 内网 super 版最新代码，安装目录 /workdir/hpc/build_rick）：用户在 web 使用 super 模式时「找不到中控执行的 super loop 文件」报错消除——super 中控会话可正常创建并运行（可验证：该服务器 web 上创建 super session 不再报错，会话正常加载 super-loop + gen_env skill）。

### KR 集
- **KR1（定位根因）**：在 10.99.19.123 实地复现/取证，确认报错的确切来源（web 创建 super session 的前置检查 `.rick/loops/super-loop.md` 缺失 / super 会话 inline 加载降级 / superopt 调度器路径解析），产出根因记录（含缺失资产清单）。
- **KR2（修复到位）**：把修复实际送达 10.99.19.123 的部署（补齐缺失 super 资产到正确 workspace / 或修复 infer-rsi 代码缺陷后重建），使根因消除。
- **KR3（端到端验证）**：在 10.99.19.123 的 web 上创建 super 中控会话成功（前置检查通过、super-loop + gen_env 正常加载），并验证 super 模式核心链路（`rick super start/status` 或等价中控动作）可用。

### 充分性自检
KR1（知道哪里坏）∧ KR2（修好它）∧ KR3（验证修好且能跑）⟹ O 达成（报错消除 + super 模式可用）。✅

## 第 1 层：排查修复 pipeline

**模块与调用关系**：
- **A 访问通道**（建立 10.99.19.123 的执行通道：ssh / 人工代跑）
- **B 服务器取证**（复现报错 + workspace 注册表 + 部署形态：哪个 workspace 缺哪些资产、二进制形态）——依赖 A
- **C 修复**（按根因：a) 给目标 workspace 部署 9 件资产 / b) 代码层 super 资产自动分发 + sync infer-rsi + 服务器更新）——依赖 B
- **D 端到端验证**（服务器 web 创建 super session 成功 + super 链路可用）——依赖 C

**L1 调研结论**（详见 research-L1.md）：报错 = web 创建 super session 的前置检查 `<ws>/.rick/loops/super-loop.md` 缺失；infer-rsi dev_rick@6a8f906 = 本地 HEAD 全量镜像、9 件资产齐全、含 job_51 修复；同类 bug 在 job_51（MT_INFER）发生过，遗留问题明确记录「无自动分发机制」。

**判断节点（L3 待问 human）**：
- J1 访问通道：ssh 22 超时，如何访问服务器？
- J2 修复范围：最小修复（手动部署 9 件）vs 系统性修复（代码自动分发）vs 两者
- J3 若代码修复：落点与同步发布流（本地改 → sync infer-rsi dev_rick → 服务器 pull + 二进制更新）确认
- J4 验证深度：创建成功即可 vs 跑通 super 全链路（rick super start）

## 范围修订（human 裁决，2026-11-19）

服务器现场已被其他 agent 修复（9 件资产已部署到 /workdir/hpc/project/jiqi_mtfi_opt），用户正在使用。本 job 范围收敛为：
- **KR1（源码确认根因）**：✅ 完成——web 工作区创建/注册只建空 .rick 不分发资产，super 9 件资产只在 rick 树，外部 workspace 首次用 super 必撞缺失（详见 research-L1.md + research-L1-r2.md）
- **KR2（bugfix）**：源码层修复分发缺口（自动分发机制），不发版不重启服务器
- **KR3（源码级验证）**：单测验证修复行为；不做服务器验收

## 第 2 层：bugfix 设计（叶子层，已全部落实）

**human 全部裁决**：6a（super session 创建时 lazy 部署）；7（只补缺失不覆盖，不碰现网用户环境）；8a（静默补齐 + 服务端日志，API 契约不变，前端零改动）；9a（本地仓库修复 + sync infer-rsi dev_rick；服务器不动、不发版）。

**模块与调用关系**：
- **M1 资产部署器**（internal/super/assets.go，新建）
- **M2 挂钩点**（internal/web/sessions.go SessionTypeSuper 分支：前置检查前先尝试自动部署，失败/树无资产 → 落回原显式报错，行为不变）
- **M3 测试**（assets 单测 + web 会话层自动部署测试）

### 叶子层四维度

**1. 关键代码实现**
- `internal/super/assets.go`（新建）：
  - `var SuperAssetPaths = []string{...}` —— 9 件资产相对路径（.rick/ 下，单源清单：super-loop.md + gen_env_skill + 7 件 super_* skill）
  - `func deploySuperAssets(srcRickDir, dstRickDir string) ([]string, error)` —— 纯函数：逐件 dst 存在→跳过（绝不覆盖）；src 缺失→跳过（部分树只部署可部署件）；MkdirAll 父目录 + 复制（0644）；返回实际部署清单；仅 I/O 错误返回 err
  - `func DeploySuperAssets(dstRickDir string) ([]string, error)` —— 复用 resolveRepoRoot()（cwd 上溯 + exe fallback，supervisor.go:507）定位 rick 树，调 deploySuperAssets；树解析失败 → 返回 err
- `internal/web/sessions.go`（修改 SessionTypeSuper）：
  - 前置检查循环之前插入：`if deployed, err := superpkg.DeploySuperAssets(rickDir); err == nil && len(deployed) > 0 { log.Printf("[super] auto-deployed %d assets to %s: %v", ...) }`（std log，web 包零日志惯例下的最小实现）
  - 原前置检查循环**不变**（API 契约字符串保留；部署失败/树无资产 → 行为与现在完全一致）

**2. 文件结构**
- 新建：`internal/super/assets.go`、`internal/super/assets_test.go`
- 修改：`internal/web/sessions.go`（+import log）、`internal/web/sessions_test.go`（新增自动部署测试，位于既有 super 会话测试旁）
- sync：以上 4 文件同步 infer-rsi dev_rick（targeted copy + commit + push，沿用 sync 模式；不重建 checked-in linux 二进制——human 裁决不发版）

**3. 工具调用**
- 验证：`go build ./...` + `go test ./internal/super/ ./internal/web/`（必要时全量）
- sync：`/tmp/infer-rsi-job55` 先 git pull，再复制 4 文件，`git commit -m "sync: rick main@<sha> —— job_55 super 资产自动分发"` + `git push origin dev_rick`

**4. 环境依赖 + 配置**
- 本地 Go 工具链（已有）；infer-rsi git ssh push 权限（已验证 ls-remote/clone 可达）；web 测试依赖真实 rick 树含 .rick 9 件资产（本地仓库满足）；服务器侧零依赖零改动

### 测试断言面（M3）
- assets 单测：假树（t.TempDir 种 9 件）+ 裸 .rick 目标 → 9 件部署且内容一致；目标已有 super-loop.md（定制内容）→ 不覆盖、其余 8 件部署；src 部分缺失 → 只部署可部署件不报错；t.Chdir 假树（含 super/superopt/__main__.py）→ DeploySuperAssets 走 cwd 解析成功
- web 层测试：裸 .rick 工作区（不 seed）→ POST 创建 super session → 成功且 `<ws>/.rick/loops/super-loop.md` 已落盘；既有 seeded 测试不受影响（只补缺失，seeded 即无操作）

## 状态
- [x] 根层 OKR 建立
- [x] L1 调研消解完成（research-L1.md + research-L1-r2.md）
- [x] L1 终止达标（human 裁决后 pipeline 全清：根因已源码确认）
- [x] L2 叶子层四维度全部落实（全部判断节点已由 human 裁决：6a/7/8a/9a）
- [x] **Grilling 完成——设计树已遍历完毕，所有模块已落实到代码实现/文件结构/工具调用/环境配置**
