---
name: web-feedback-iteration-loop
trigger: "当 human 在 web UI（生产或 dev）验收后提出反馈——bug、体验问题、小功能、或对既有设计的裁决性推翻——需要快速组织一轮增量修复并安全送达生产时触发；本 loop 编排反馈轮次，每轮的执行机制（dev 树隔离/层门禁/release/RSI 证据）复用 rick-rsi-loop"
scope: "全局（web 承载的 easy/doing 会话均可加载；与 rick-rsi-loop 叠加使用）"
---

# Loop: web 验收反馈增量迭代

把「human 用过之后的反馈」变成**一轮低成本、可验收、可回滚**的增量迭代。job_43 实测 6 轮（13 task / 10 gate / 6 release），反馈→上线全链路收敛到 ~30 分钟/轮。

**与 rick-rsi-loop 的关系**：本 loop 只管**反馈轮次的编排**（分级/调研/微任务化/验收/呈批节奏）；单轮内的 dev 树隔离、层门禁、S3-S7 提升流程、rsi_check 证据全部**沿用 rick-rsi-loop 原文**，不重复定义。加载本 loop 前先读 rick-rsi-loop.md。

## 依赖准备（硬约束，缺失则报错停止）

| 依赖项 | 确认命令 | 要求 |
|--------|----------|------|
| dev 工作区（rsi-loop 依赖） | `git -C /workdir/sunquan20/rick-dev rev-parse --abbrev-ref HEAD` | 以 `dev/` 开头 |
| dev 实例（8414） | `RICK_DEV_HOME=<dev-home> ./bin/rick tools dev-web status` | `state=up fingerprint=ok` |
| 生产实例（8413）健康 | `curl -s --noproxy '*' http://127.0.0.1:8413/api/health` | `status=ok` |
| 本 job 的证据骨架 | `ls <dev>/.rick/jobs/<job>/doing/rsi/` | 5 文件在（rsi_check --init 产物） |

环境异常处理（休眠/快照后常见）：
- dev HOME / job 目录 ACL mask 被重置（写拒绝）→ 加载 `.rick/skills/acl_permission_recovery_skill/skill.md` 按其流程恢复（rename+checkout 重建，实测零丢失）
- release 提示 `bin/releases` 权限拒绝 → 按报错加 `RICK_RELEASES_DIR=~/.rick/releases`（见 `.rick/skills/rick_release_ops_skill/skill.md`）

## 目标（Goal）

一轮反馈迭代的闭环达成（客观判据）：

| # | 达标条件 | 判据 |
|---|---------|------|
| G1 | 反馈分级正确 + 流程重量匹配 | F1 未重开 grilling；F3 类裁决两案对比上呈 |
| G2 | 修复上线且验收通过 | release 健康（build_id==version）+ human 生产 UI 确认（或明确「先这样」） |
| G3 | RSI 证据完整收口 | `rick tools rsi_check --job <job>` 6/6 绿 |

## 上下文管理（Context Management）

- **保留**：反馈原文（requirement.md 追加条目）、每轮 release 版本链、dev-iterations 每轮一行记录、遗留清单（「暂时放弃」类如 PWA）
- **压缩**：每轮只留「级别→改动→gate→版本」一行；构建输出只留尾部
- **遗忘**：worker 中间试错、被推翻方案的实现细节（白名单拆除后其三层机制细节只需保留教训不需保留实现记忆）

## 可调用工具（Tool Access）

| 工具 / 命令 | 用途 | 约束 |
|------------|------|------|
| `runs.run`（agent:'worker'） | 派发实现 worker | 携带写域边界 + 禁止 git + 只改 dev 树 |
| `python3 <dev>/.rick/jobs/<job>/plan/gates/gate{N}.py` | 层门禁（派发前 RED / 完成后 GREEN） | exit 0 才可 level_complete |
| `level_complete`（rick-gates） | 层检查点 | doing_dir 指 dev 树 job 目录 |
| `rick tools dev-web restart` | 后端改动后重启 dev 实例 | 只碰 dev（8414） |
| `curl http://127.0.0.1:8414/` | 确认新 bundle 生效 | grep index-*.js |
| `rick tools release --dry-run / --merge-source --detach` | S3 演练 / S5 提升 | `RICK_RELEASES_DIR=~/.rick/releases` |
| `rick tools rsi_check --job <job>` | 证据收口校验 | 6/6 绿 |

## 子 Agent 工作流（状态机）

每轮一个循环；轮内步骤串行，任一步失败修好后重进该步（不跳步）。

### 步骤 1：反馈分级（parent）

| 级别 | 判定 | 流程 |
|------|------|------|
| **F1 单点 bug/体验**（「收起原文没反应」「间距难受」） | 根因可静态定位 | 跳过 grilling：定位根因 → 单 task + 单 gate 快速通道 |
| **F2 根因不明/涉及面广**（「消息没了」「发烫」） | 需运行时/多文件考古 | 派 research（落 doing/grilling/research-I{N}.md）→ 判断节点上呈一轮（每问附推荐）→ task 化 |
| **F3 产品范式裁决**（「白名单是什么？不要特殊逻辑」/新功能方案） | 涉及安全/部署/交互范式 | **必须两案对比显式上呈**（现状 vs 提议，各附复杂度/安全面/维护成本），禁止单案呈批——job_43 白名单建了又拆的根因是裁决点被「增量扩展」惯性吞没 |

### 步骤 2：上下文读取（parent）

`doing/debug/bug*.md` 摘要（新反馈先对照是否已知问题复发）→ `rsi/dev-iterations.md` 尾部（迭代编号 N / 上轮版本=回滚点）→ 反馈原文追加 requirement.md（编号递增，不改历史）。

### 步骤 3：派发 worker 实现（parent → runs.run）

- 渲染层结构性改动（虚拟化/滚动/事件接管）→ worker 先读 `.rick/skills/virtualization_component_protocol_skill/skill.md`（自定义组件 `...rest` 全量转发 + 引用稳定 + 间距走 Footer）
- worker 自验（dev 树根）：`go build ./... && go test ./internal/...`；前端另跑 `cd web && npm run typecheck && npm run build`

### 步骤 4：层检查点（parent）

```bash
# 派发前验证判别力（应为红）：
python3 <dev>/.rick/jobs/<job>/plan/gates/gate{N}.py; echo $?   # 期望 1
# worker 完成后复跑（应为绿）→ level_complete（gate_cmd + dev 树 doing_dir）
```
gate 断言惯例（job_43 十条 gate 提炼）：存在性断言 + 否定断言（旧模式已移除）+ 构建绿 + **生产零触碰**（`git -C <prod> status --porcelain -uno` 为空）+ **RICK_* 环境净化**（`env = {k:v for k,v in os.environ.items() if not k.startswith("RICK_")}`——RICK_STATE_DIR 穿透 t.Setenv(HOME)，见 subprocess_env_isolation_skill Step 5）。

### 步骤 5：dev UI 真渲染验收（P0 后强制纪律，不可跳过）

```bash
RICK_DEV_HOME=<dev-home> ./bin/rick tools dev-web restart   # 后端改动时
curl -s --noproxy '*' http://127.0.0.1:8414/ | grep -o 'index-[^"]*\.js'
```
渲染层/交互改动 → 请 human 在 dev UI（8414 + DEV_TOKEN）走主路径（**gate 绿 ≠ 可用**：静态断言测不到运行时协议破坏，job_43 P0 实证）。纪律细节见 `.rick/skills/render_layer_acceptance_skill/skill.md`。

### 步骤 6：S3 演练 → S4 呈批 → S5 提升（复用 rick-rsi-loop）

S4 呈批必备：改动表 + 回滚点（上轮版本）+ 影响面 + **F3 类附范式对比说明**。批准留痕 → `release --merge-source --detach` → 健康校验（build_id == version）。

## 产出评估（Output Evaluation）

- 评估类型：**客观（机器校验为主 + human 验收）**

| 检查项 | 验证方法 | 通过标准 |
|--------|----------|---------|
| 层门禁绿 | `python3 .../gates/gate{N}.py` | exit 0（派发前曾为 1——判别力在案） |
| 层检查点 | tasks.json 本层 task | status=success + commit_hash |
| dev 验收（渲染类） | human 在 dev UI 确认 | 明确反馈（「ok 了」级） |
| 生产零触碰 | `git -C <prod> status --porcelain -uno` | 空 |
| S3 演练 | `release --dry-run` | `prod_untouched=true` |
| S5 健康校验 | `curl 127.0.0.1:8413/api/health` | build_id == 本次 version |
| 证据收口 | `rick tools rsi_check --job <job>` | 6/6 绿 |

⚠️ release.md 的头部三行 `version= / rollback_point= / released=` 必须保持在文件**首部**（rsi_check 的 rsiVersionRe 取第一个匹配）——job_43 曾两次重构该文件时切掉头部导致 rsi_check 红。写回时用「**实际生效版本**」（构建时刻 ≠ 演练时刻：批准留痕会推进 dev HEAD）。

## 停止标准（Termination Condition）

**成功退出**：反馈闭环——human 在生产 UI 验收通过（或明确「先这样」），本轮 release 健康且 rsi_check 6/6 绿。

**优雅退出**（任一触发）：
- human 明确「暂时放弃/先这样」（如 job_43 PWA——记遗留，不强行闭环）
- 同一反馈连续 2 轮修复仍复现 → 升级 F2 深度调研或回报 human 重新定级
- 反馈实质是新需求（超出增量范畴）→ 建议开新 job 走完整 grilling

**退出时**：输出本轮摘要（改动 + 版本 + 遗留），等待 human 下一条反馈或「执行 learning」指令。
