---
name: workspace-relocation-loop
trigger: "当需要把 rick 工作区（含 job 历史/pi 会话/配置状态）迁移到另一台机器并使其正常工作时触发（Linux→Mac 搬迁、A 机分发工作区到 B 机、机器换代整环境搬家）"
scope: "全局（任意承载会话：easy / doing / plan）"
---

# Loop: workspace relocation（工作区跨机搬迁）

## 目标（Goal）

源机上一个或多个工作区 + rick 状态打包为单文件归档，在目标机（可能异构平台）解压落位后：`rick web` 可启动、全部工作区 job 历史可读、pi 会话可 resume 并带历史上下文续答。

- 成功标准：四级验收断言全过（health / workspaces / sessions+resume / prompt 应答）+ 人类 UI 亲验
- 自评命令：本 loop Step 4 的四条 curl（末条模型应答即端到端闭环）

## 上下文管理（Context Management）

- **保留**：迁移范围（工作区清单 + 排除决策 + sessions 策略）、目标机布局约定（`~/workdir` + `~/.rick`）、每轮验收断言结果
- **压缩**：dump/import 的输出只留末尾验证清单；目标机准备步骤一行一条
- **遗忘**：失败的 dump 尝试细节（只留「哪类权限洞被跳过、多少项」结论）、传输耗时

## 可调用工具（Tool Access）

| 命令 | 用途 | 约束 |
|------|------|------|
| `rick tools dump` | 源机打包（`--workspace`/`--exclude`/`--sessions`/`--goos`） | 大目录先 `du` 评估再排除；输出与 `.sha256` 边车成对 |
| `scp` / `rsync` | 传输归档+边车 | 边车漏传 = import 拒绝校验 |
| `rick tools import` | 目标机落位（`--target`/`--state-dir`/`--force`） | 默认合并不覆盖；重布局先停 web |
| `rick web --daemon` | 目标机启动 | 非默认 state-dir 必须成对 `RICK_PI_AGENT_DIR` |
| `curl` | 四级验收 | token 从 `$HOME/.rick/config.json` 读 |
- 权限边界：只读源机（dump 的 marker 回填是唯一幂等副作用）；目标机只写约定落位目录；`pkill -f` 禁用（自匹配自杀，用精确 pid）

## 工作流（Step 0-5）

### Step 0：环境确认 + Domain 搜索

**0.1 依赖准备**（硬约束，缺失则报错停止）：

源机：

| 依赖项 | 确认命令 | 要求 |
|--------|----------|------|
| rick 含 dump/import | `rick tools dump --help` | 能打印用法（main 未 release 时从 dev 树 `go build -o` 构建；改 rick 自身走 rick-rsi-loop） |
| 输出盘空间 | `df -h <out-dir>` | ≥ 源树 gzip 后体积（≈原始的 60-70%） |

目标机（boot 阶段才需要，可先确认后补装）：

| 依赖项 | 确认命令 | 要求 |
|--------|----------|------|
| node ≥22.19 | `node -v` | pi 必需；缺失时代理下载官方 tar.xz 解压即用 |
| 模型 API 可达 | `curl -x <proxy> https://api.deepseek.com` | 401=通（TLS 到达即可） |
| 开放端口 | `ss -lntp \| grep 8413` | 8413 空闲（对外开放约定端口） |

**0.2 Domain 搜索**：`.rick/domain/pi-runtime.md`（pi 配置/运行时事实）、`.rick/domain/env.md`（代理/凭据/机器拓扑）。

### Step 1：parent 确认全局目标

- 目标 + 成功标准（见上）；迁移范围三要素与 human 对齐：工作区清单、大件排除表、sessions 策略（`all` / `workspaces` / `none`）。

### Step 2：parent 读取上下文（压缩策略）

- 从 `doing/debug/` 取上轮摘要：dump 是否遇权限洞（已内建跳过）、import 验证清单哪项红、web 启动哪项 env 缺。
- 跨轮核心事实：归档路径、目标机落位路径、token、四级断言当前状态。

### Step 3：启动 worker child 执行工作流

每轮由 parent 派发 worker child（agent:'worker'），携带任务目标 + 上轮断言结果：

**worker child：A. dump（源机）**
- 加载 skill：`.rick/skills/cross_machine_migration_skill/skill.md`（① 节）
- 精确命令：
  ```bash
  du -sh <ws>/*/ | sort -rh | head -15
  rick tools dump --workspace <ws> -o /tmp/dump.tar.gz --sessions all --goos <目标平台> --exclude <big-dir>...
  ```
- 产出：`dump.tar.gz` + `.sha256`（权限洞跳过为正常留痕，非失败）

**worker child：B. transfer + import（目标机）**
- 加载 skill：同上（② 节）
- 精确命令：
  ```bash
  scp /tmp/dump.tar.gz* <user>@<target>:~/
  mkdir -p boot && tar xzf ~/dump.tar.gz -C boot bin/<plat>/rick && chmod +x boot/bin/<plat>/rick
  boot/bin/<plat>/rick tools import ~/dump.tar.gz --target ~/workdir
  ```
- 产出：import 十项验证清单全 ✅（marker 一致 / web.json 可解析 / 会话目录就位）

**worker child：C. boot + verify（目标机）**
- 加载 skill：同上（③④ 节）
- 精确命令：
  ```bash
  export PATH=<node>/bin:$PATH
  rick web --listen 0.0.0.0 --port 8413 --daemon
  curl -s http://127.0.0.1:8413/api/health
  curl -s -H "Authorization: Bearer $T" http://127.0.0.1:8413/api/workspaces
  curl -s -X POST -H "Authorization: Bearer $T" http://127.0.0.1:8413/api/sessions/<id>/resume
  curl -s -X POST -H "Authorization: Bearer $T" -H "Content-Type: application/json" \
       -d '{"message":"只回复两个字：成功"}' http://127.0.0.1:8413/api/sessions/<id>/prompt
  ```
- 产出：四级断言结果（含模型应答原文）

**worker child：COMMIT**（若本迁移伴随 rick 功能改动/修复）：`git add` + commit + `go test ./internal/env/relocate/...` 循环到绿（走 rick-rsi-loop 时按其纪律）。

### Step 4：parent 产出评估

调用验证 skill：`.rick/skills/cross_machine_migration_skill/skill.md`（④ 验收断言节）；断言结果对照下节「产出评估」表逐项过。

### Step 5：parent 确认停止标准

对照下节「停止标准」判定；收尾清理目标机测试残留（boot/ 旧 state/），保留归档+边车（回滚与二次分发用）；源机不动。

## 产出评估（Output Evaluation）

调用验证 skill：`.rick/skills/cross_machine_migration_skill/skill.md`（④ 验收断言节）

| 检查项 | 验证方法 | 通过标准 |
|--------|----------|---------|
| health | `curl .../api/health` | `status=ok` |
| 工作区完整 | `GET /api/workspaces` | 数量=源机、路径已重映射、id 未变 |
| 会话历史 | `GET /api/sessions` + `POST resume` | 会话可读、resumed=true → active |
| 端到端 | `POST prompt` | 模型带历史上下文应答 |
| 人类亲验 | 外网可达链接 + token 呈给 human | UI 确认功能正常 |

- 全部通过 → 成功退出
- 存在失败 → 原因附上下文回 Step 3 迭代（典型：dump 补 `--exclude`；web 补 env 配对；token 换 `$HOME` 里的真实值；端口改 8413）

## 停止标准（Termination Condition）

**成功退出**：四级断言全过 + 人类 UI 亲验（外网可达链接 + token 呈给 human）。

**优雅退出**（任一触发）：
- 迭代达上限 3 轮
- 连续 2 轮同一断言失败且无新线索
- 人类明确要求停止

## 与相邻 loop 的关系

- `agent-runtime-bootstrap-loop`：dump 用了 `--slim` 时，目标机需先走它跑 `rick tools init-pi` 再 boot。
- `rick-rsi-loop`：迁移过程中若发现并修复 rick 产品缺陷（本 loop 的 dump/import 自身），修 rick 必须走 RSI。
