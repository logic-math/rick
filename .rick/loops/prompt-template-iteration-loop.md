---
name: prompt-template-iteration-loop
trigger: "当 human 实跑某个交互型会话类型（human-loop/easy/plan 等）后提出体验反馈，且改动对象是 prompt 模板层（internal/prompt/templates/*.md、skills/*.md、agent 清单 internal/env/agents.go）而非 web UI 或业务代码——需要一轮「定向模板增量→build/test→agent 部署三连→新会话继承实测」的迭代时触发"
scope: "全局（easy/doing 会话可加载；反馈对象=模板/agent 层用本 loop，web UI 代码反馈用 web-feedback-iteration-loop，一次性协议形态重构先走 protocol-redesign-loop）"
---

# Loop: prompt 模板实测反馈迭代

把「human 实跑会话后的体验反馈」变成一轮**定向模板增量 + dev 生效 + 新会话继承实测**。job_50 实测 7 轮（human-loop v7→v7.6.2），每轮定向增量到下一轮实测 ≈15 分钟。

**与相邻 loop 的分工**：首轮大形态重构（协议 v2→v3 类）先走 `protocol-redesign-loop`；本 loop 管重构后的**实测反馈轮次**。反馈轮次编排/分级思想参考 `web-feedback-iteration-loop`，但本 loop 的机制是模板层特有的：**agent 部署三连、dev 会话继承迁移、会话输出病灶分析、dry-run 冒烟**（无 gates/rsi_check——模板增量用直接验证）。

**核心事实**（改动生效链）：模板 .md 是 **Go embed**——改后必须 build 才生效；agent 清单（agents.go `rickAgents`）→ `tools init-pi` 部署到 `<HOME>/.rick/pi/agent/agents/`（**deploy 只增不删**，下线 agent 要手动 rm）；会话的 `prompts/` 是**启动时快照**——模板更新后旧会话不继承，须新会话。

## 依赖准备（硬约束，缺失则报错停止）

| 依赖项 | 确认命令 | 要求 |
|--------|----------|------|
| 生产树干净 | `git -C <prod-tree> status --porcelain -uno` | 空（增量期间生产零触碰） |
| dev 工作树 | `git -C <dev-tree> rev-parse --abbrev-ref HEAD` | dev/ 分支 |
| dev 实例 | `cd <prod-tree> && go run ./cmd/rick tools dev-web status` | state=up |
| 网络出口 | `curl --noproxy '*' -sI -m 8 https://www.google.com` | 直连 200（若 403/超时见 domain/env.md——命令须带 `env -u http_proxy -u https_proxy`） |

Domain 搜索：`.rick/domain/env.md`（网络出口/dev 拓扑）、`.rick/domain/commands.md`（dev-web/release/会话 API 契约——steer 字段是 `message`）。

## 目标（Goal）

一轮模板反馈迭代的闭环达成（客观判据）：

| # | 达标条件 | 判据 |
|---|---------|------|
| G1 | 定向增量落地 | `go build ./...` + 改动包测试绿 + dry-run 冒烟 `grep -c '{{'` = 0 |
| G2 | dev 生效 | agent 部署三连完成，`ls <dev-home>/.rick/pi/agent/agents/` 与 rickAgents 清单一致（无过期残留） |
| G3 | 实测收敛 | 新会话按新模板行为从继承点继续；human 实测反馈收敛或明确「阶段性满意」 |

## 上下文管理（Context Management）

- **保留**：反馈原文（追加 requirement.md，编号递增不改历史）；当前 loop 产物进度（mission/zpd/judgment 尾部——决定新会话继承点）；上一轮实测会话的 text 输出统计
- **压缩**：每轮一行（反馈→改动面→提交号→实测 loop_N）；被推翻方案的实现细节只留教训
- **遗忘**：中间试错、重复的部署命令输出、部署三连的中间日志

## 可调用工具（Tool Access）

| 工具 / 命令 | 用途 | 约束 |
|------------|------|------|
| `runs.run`（agent:'worker'） | 派发定向增量 worker | 长任务文本先 write 任务书文件+单行指针 task（pi_orchestration_syntax_skill 派发纪律）；worker 禁 git |
| `go run ./cmd/rick human-loop --dry-run <topic>` | 模板渲染冒烟 | `grep -c '{{'` = 0 |
| `git -C <dev-tree> merge main --no-edit` | dev 源码同步 | 只碰 dev 树 |
| `go run ./cmd/rick tools dev-web restart/down/up` | dev 实例管理 | 带干净 env（`env -u http_proxy -u https_proxy`） |
| `<dev-bin> tools init-pi` | agents 部署到 dev HOME | `HOME=<dev-home>`；`rickAgents` 下线的 agent 手动 rm |
| `POST /api/sessions` + `POST /api/sessions/{id}/steer` | 新会话创建与继承指示 | steer 字段 `message`；先建会话再拷产物 |
| `bin/rick tools release --merge-source --yes` | 收口提升 | release 前先 `dev-web down`（防门禁竞态），后 `up --no-build` 恢复 |

## 子 Agent 工作流（状态机）

每轮一个循环；轮内 A→B→C→D 串行，任一步失败修好后重进该步（不跳步）。

### worker child：A. 定向增量（runs.run agent:'worker'）

- 改动面：`internal/prompt/templates/*.md` + `skills/*.md` + `internal/env/agents.go`/`check.go` + 渲染管线（`human_loop_prompt.go` 等）+ **测试锚词同步**（forbid 旧概念词、required 新锚词）
- 精确命令（自验）：
  ```bash
  go build ./... && go test ./internal/prompt/ ./internal/env/ ./internal/builder/ ./internal/handler/
  go run ./cmd/rick human-loop --dry-run "冒烟" > /tmp/render.md && grep -c '{{' /tmp/render.md   # 期望 0
  ```
- 回执 = 改动清单 + 自测结果（回执可能被传输层截断——parent 一律以产物复核为准）

### parent：B. agent 部署三连（模板/agent 清单改动生效到 dev 的标准动作）

```bash
cd <dev-tree> && git merge main --no-edit                                        # ① 源码同步
cd <prod-tree> && env -u http_proxy -u https_proxy go run ./cmd/rick tools dev-web restart   # ② dev 重启（干净 env）
cd <dev-tree> && HOME=<dev-home> <dev-home>/bin/rick.dev.<new-ver> tools init-pi               # ③ agents 部署
ls <dev-home>/.rick/pi/agent/agents/                                             # ④ 清单核对 + rm 已下线 agent 的 .md
```

### parent：C. 会话继承实测（旧会话 prompts/ 是快照，模板更新必须新会话）

1. `POST /api/sessions`（`{"workspace_id":...,"type":"human-loop","params":{"topic":"... 继承 loop_N 进度 ..."}}`）→ **记下分配的 loop_N**（先建会话再拷产物——预建目录会使 nextLoopID 分配偏移，job_50 实测踩坑）
2. 拷贝上一 loop 产物到新 loop_N：`cp <old>/{mission,zpd,values,syllabus,judgment}.md <new>/ && cp <old>/lessons/*.md <new>/lessons/ && cp <old>/research/*.md <new>/research/`；mission 顶部更新**继承注记**（进度到哪、从哪继续、human 指令原文）
3. streaming 会话用 `POST /api/sessions/{id}/steer`（`{"message":"...直接从 <继承点> 继续..."}`）
4. **病灶定位**（反馈是体验类时）：`ls <dev-home>/.rick/pi/agent/sessions/*/<session_id>*.jsonl` → 分析 assistant text blocks（条数/大小/内容类型）——编排叙述暴露在 text = 输出纪律病灶；thinking 泄漏看 `hideThinkingBlock` 配置

### parent：D. COMMIT

```bash
cd <prod-tree> && git add -A && git commit -m "feat(job_N): vX.Y.Z 增量——<反馈一句话>（改动面）"
```

## 产出评估（Output Evaluation）

评估类型：**客观（机器校验为主 + human 实测验收）**

| 检查项 | 验证方法 | 通过标准 |
|--------|----------|---------|
| build+test | `go build ./... && go test ./internal/...` | 绿（web 包 TempDir 竞态=已知 flaky，重试即过） |
| 渲染冒烟 | `go run ./cmd/rick human-loop --dry-run <topic> \| grep -c '{{'` | 0（无占位符残留） |
| agent 一致 | `ls <dev-home>/.rick/pi/agent/agents/` | 与 rickAgents 清单一致，无过期残留 |
| 继承实测 | 新会话首轮 text 输出 | 按新模板行为 + 从继承点继续（体验类反馈：病灶消失） |
| 生产零触碰 | `git -C <prod-tree> status --porcelain -uno` | 空 |

## 停止标准（Termination Condition）

**成功退出**（human 明确「阶段性满意」）→ 收口三连：

1. **dev 树记录存档**：`cd <dev-tree> && git add -A && git commit`（会话产物防 dev 重建丢失）
2. **release**：`<prod>/bin/rick tools release --merge-source --yes --prod-repo <prod-tree> --dev-tree <dev-tree> --prod-home <real-home> --port <prod-port> --start-script <prod-home>/start-web.sh`——⚠️ **release 前先 `dev-web down`**（门禁 buildEnv 用 dev HOME，与活动 dev 会话竞争 pi sessions 致 TempDir 竞态连挂，job_50 实测 4 连挂；release 后 `up --no-build` 恢复）；生产 agents 部署：`<new-bin> tools init-pi`（真实 HOME）+ rm 过期 agent 文件
3. **生产验证**：`curl --noproxy '*' http://localhost:<prod-port>/api/health`（build_id == 本次版本）+ agents ls + dry-run 冒烟

**优雅退出**（任一触发）：
- human 明确暂停（产物留 dev 树，后续经会话继承继续）
- 同一反馈连续 2 轮仍未改善 → **停止改代码，上呈多方案**（每案附机制/代价/推荐——job_50 探索流先例：混沌学园法→SENSE 状态机→靶向卡点，第 3 案成）
- 反馈实质是协议形态级重构 → 转 `protocol-redesign-loop` 重走 grilling

**退出时**：输出本轮摘要（改动+提交+实测 loop_N+遗留），等待 human 下一条反馈或收口指令。
