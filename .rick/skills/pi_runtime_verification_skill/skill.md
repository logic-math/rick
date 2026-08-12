---
name: pi-runtime-verification
description: 迁移/更换 rick 的 agent runtime（如 claude→pi）后，做真实命令级端到端验证
---

# Skill: pi runtime 迁移的命令级端到端验证

## 触发场景

当 rick 迁移或更换底层 agent runtime（如 claude code → pi，或 pi 版本大升级），且改动涉及 rick 调用 agent 的 CLI flags / 事件解析 / prompt 落盘路径时使用。

**问题信号**：
- 只在 piagent 包内部写了单测（mock binary），没跑过真实 `rick <cmd>`
- 研究 brief 说某 flag "接受 id" 但没说"加载 vs 创建"语义
- 改动跨多个 cmd 文件（plan/easy/doing/learning/human-loop/ctrl/dream）

## 预期效果

- 暴露"包内部测试通过但真实命令失败"的隐藏 bug（如 `--session` 加载 vs 创建语义）
- 确保 prompt 落盘到 rick 标准 job 目录（非 /tmp），完全对齐 rick 功能
- 防止"研究 brief 描述正确但语义未验证"导致的迁移 bug 流入提交

## 核心内容

### 1. 每个命令逐个真实跑（非 dry-run）

interactive 命令（plan/easy/learning/human-loop/ctrl）用管道喂 stdin 模拟一轮对话：
```bash
echo "请直接回复: <CMD>_OK" | timeout 200 ./bin/rick <cmd> [args] 2>&1 | tail -8
```
print/json 命令（doing 走 Execute()、dream --background）能自动完成。

### 2. 三项必须验证

| 项 | 验证方法 |
|---|---|
| prompt 落盘 | `find .rick/jobs -name "*_prompt.md" -newer <ref>` 确认在 job_N/doing/prompts/ 下（非 /tmp） |
| pi 真实响应 | 输出含 `<CMD>_OK`（证明 pi 读了 prompt + 真实 LLM 回复） |
| exit code | 0（interactive 命令 pi 处理完应正常退出） |

### 3. flag 语义验证（关键，防 brief 误判）

研究 brief 若说"X flag 接受 id"，**不要直接信**——要单独测其"加载已有 vs 创建新"语义：
```bash
# 测创建语义: 用一个不存在的 id
echo OK | pi --provider <p> --model <m> --<flag> newuuid123 /dev/null 2>&1 | head -3
# 看输出是 "No session found"（加载语义，错）还是 "creating a new session"（创建语义，对）
```

### 4. 用 doing 做全链路（Execute + 解析器 + actpath）

doing 走 `piagent.Execute()`（json 解析），能验证解析器在真实 LLM 工具调用下正确：
```bash
# 给 job 造最小 plan（task1.md 无依赖 + 简单测试方法）
echo "请直接回复: DOING_OK" | ./bin/rick doing <job> 2>&1 | tail
# 确认: pi 真实建文件 + commit + task success
git log --oneline -2  # 应见 pi 的 commit
```

### 5. 验证失败时的处理

发现的 bug 多是"flag 语义误判"或"路径错"——修复后必须**重跑该命令验证**，不能只改代码不验。

### 6. 配置目录隔离验证（PI_CODING_AGENT_DIR，job_33 沉淀）

改动涉及 `piagent.AgentEnv()`（所有 pi 子进程注入 `PI_CODING_AGENT_DIR=~/.rick/pi/agent`）时，四项必须验证：

| 项 | 验证方法 | 通过标准 |
|---|---|---|
| env 注入 | fake pi 脚本 `echo "$PI_CODING_AGENT_DIR"` 断言 | 值 = 托管 agent dir |
| 隔离生效 | `PI_CODING_AGENT_DIR=/tmp/x pi list` + `pi install npm:y` | settings.json 写在 /tmp/x 下，且**保留未知字段**（hideThinkingBlock 等不被重写） |
| 包过滤 | settings.json 用对象形式 `{"source": ..., "extensions": []}` | `pi list` 显示 `(filtered)` 且**不会重写** settings.json |
| 真实加载 | `PI_CODING_AGENT_DIR=$HOME/.rick/pi/agent pi --mode json <provider/model/key flags> -p '列出可用工具'` | 输出含 subagent/web_search（扩展真的加载） |

**已知边界（pi 内部行为）**：
- `pi install` 对 user scope 包在 managed 路径缺失时**回退全局 npm root**（`npm root -g`）复用代码——注册隔离、代码共享
- 隔离后 settings.json 无 provider/model → 直跑 pi 报 "No API key found"，必须带 CLI flags（rick 本就如此）
- `--session-id` 对不存在 id 打 Warning "No project session found...creating a new session" 是**正常创建语义**，不是错误

### 7. 运行时自闭环验证（~/.rick/pi/agent/runtime，job_34 沉淀）

rick 自闭环运行时 = 用 `npm install --prefix` 装**独立的 pi 包副本**，与全局 pi 完全隔离：

| 事实 | 值 |
|------|-----|
| 安装 | `npm install --prefix ~/.rick/pi/agent/runtime @earendil-works/pi-coding-agent@<全局版本>`（全局有则匹配版本，pinned 失败降级 latest） |
| 二进制 | `~/.rick/pi/agent/runtime/node_modules/.bin/pi`（npm bin 软链 → 包内 dist/cli.js） |
| 解析优先级 | `cfg.PiPath` → 托管运行时（`piagent.RuntimeBin()`）→ PATH 的 `pi`（FindBinary / piPathOrDefault / piCommand / Executor 默认全部一致） |
| 版本锁定 | 与全局同版本（0.84.1）→ 行为连续；独立升级互不影响 |
| 测试隔离 | 测试用 `t.Setenv("HOME", t.TempDir())` 或 `t.Setenv("RICK_PI_AGENT_DIR", t.TempDir())` 让 RuntimeBin 解析到不存在路径 → 回退 PATH fake |

验证命令：
```bash
~/.rick/pi/agent/runtime/node_modules/.bin/pi --version   # 托管版本
rick tools theme rick                                     # 主题写入 agent/themes/（配置隔离，job_33）
python3 -c "import filecmp,os; print(filecmp.cmp('$HOME/.local/lib/node_modules/@earendil-works/pi-coding-agent/dist/modes/interactive/components/diff.js','$HOME/.rick/pi/agent/runtime/node_modules/@earendil-works/pi-coding-agent/dist/modes/interactive/components/diff.js',shallow=False))"  # 运行时副本 = 原样（无 patch）
```

**注意**：运行时副本保持 stock，不做代码级修改（用户决策，见 theme skill 第 7 节）。若未来要做运行时 patch（diff 反显→加粗、语法高亮等），需整函数替换保证幂等（锚点被消费），import 相对路径按文件层级算（diff.js 在 `dist/modes/interactive/components/` → utils 需 `../../../`）。
