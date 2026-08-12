# Bugs Domain

**最后更新**: —  **来源 Job**: —

项目已知问题与精确解决方案。由 `rick learning` 和 `rick dream` 自动追加，勿手动覆盖。

## 已知问题与解决方案

### Python 集成测试：subprocess 读取真实 ~/.rick/config.json 导致触发真实 Claude 调用

**根因**: `subprocess.run()` 不显式传 `env=` 时，子进程继承父进程 HOME，`LoadConfig()` 读取真实 `~/.rick/config.json`，触发真实 Claude CLI 调用，测试超时或行为不符合预期。

**精确解决步骤**:
```python
env = os.environ.copy()
env["HOME"] = work_dir  # work_dir 下有 mock ~/.rick/config.json
result = subprocess.run([rick_bin, ...], env=env, cwd=work_dir, timeout=30, ...)
```

**首次发现**: job_26 / task1 / commit d50b255a  **验证状态**: ✅ 已修复

### pi 扩展安装"假成功"：`pi install <本地源码目录>` 写 settings.json 但 loader 不加载

**根因**: pi 的 extension loader（`dist/core/extensions/loader.js`）只认两种入口：`package.json` 含 `pi.extensions` 字段，或 `~/.pi/agent/extensions/<name>/index.ts` 子目录结构。pi 仓库自带的 `examples/extensions/subagent` 是裸 .ts 源码（无 package.json），`pi install <path>` 把路径写进 settings.json 的 packages 数组，但启动时 loader 不加载它 → `pi list` 显示已装，但 LLM 报 "no subagent tool available"。

**精确解决步骤**:
```bash
# 错误（假成功）:
pi install ~/.local/lib/node_modules/@earendil-works/pi-coding-agent/examples/extensions/subagent
# pi list 显示已装, 但 tool 不注册

# 正确（用 npm 包）:
pi install npm:pi-subagents          # subagent 扩展
pi install npm:pi-web-access         # web search 扩展
pi install npm:@wishx127/pi-tokyo-night  # 主题
# 验证真实生效（不只看 pi list）:
pi --mode json -p 'Use the subagent tool...' | grep '"toolName":"subagent"'
```

**首次发现**: job_30 / commit 940b0a9  **验证状态**: ✅ 已修复（改用 npm:pi-subagents）

### pi `--session` vs `--session-id` 语义差异（加载已有 vs 创建新会话）

**根因**: pi 的 `--session <id>` 是**加载已有会话**（找不到报 "No session found matching"），`--session-id <id>` 才是**指定 ID 创建新会话**（"creating a new session with that id"）。loop_2 研究 brief（research-5-N2）说 pi `--session` "接受 path 或 id"，但未标"加载 vs 创建"语义差异。迁移时误把 claude code 的 `--session-id`→pi 的 `--session`，导致 `rick easy` 报错。

**精确解决步骤**:
```bash
# 创建新会话用 --session-id:
piagent.CallCLI(cfg, mainFile, ModeInteractive, "--session-id", sessionID)
# 加载已有会话用 --session (resume):
piagent.CallCLI(cfg, "", ModeInteractive, "--session", sessionID)
# 验证 flag 语义（用一个不存在的 id 测）:
echo OK | pi --<flag> newuuid123 /dev/null 2>&1 | head -3
# "No session found" = 加载语义(错); "creating a new session" = 创建语义(对)
```

**首次发现**: job_30 / commit 098a1fb  **验证状态**: ✅ 已修复

### pi 解析器：pi 对 user 和 assistant 轮次都发 `message_end`，误取 user 输入作 FinalMessage

**根因**: pi 的 `--mode json` 事件流对 user turn（echo prompt）和 assistant turn 都发 `message_end` 事件，内部 message.content schema 相同。迁移时按 claude code 习惯取最后一条 message_end 的 text 作 FinalMessage，会把 user 输入误当 agent 回复。

**精确解决步骤**: 解析 `message_end` 时必须检查 `message.role == "assistant"`，只取 assistant 轮的 text：
```go
case "message_end":
    if ev.Message.Role != "assistant" { continue }  // 跳过 user echo
    for _, c := range ev.Message.Content {
        if c.Type == "text" { sess.finalMessage = truncate(c.Text, 200) }
    }
```

**首次发现**: job_30 / commit a81eda0  **验证状态**: ✅ 已修复


### go:embed 主题文件修改后未重建 → 旧 embed 覆盖磁盘新文件

**根因**: rick 内置主题（`internal/cmd/themes/*.json`）通过 go:embed 打进二进制。修改 json 后若只跑 `rick tools theme <name>` 而未先 `go build`，theme 命令用**旧二进制里的旧 embed** 重写磁盘主题文件 —— 表现是"改了没生效"，且再次激活时旧内容继续覆盖新文件。

**精确解决步骤**:
```bash
# 修改 themes/*.json 后必须：先重建，再激活
go build -o bin/rick ./cmd/rick
./bin/rick tools theme <name>
# 验证安装副本（非源码）确实是新内容：
python3 -c "import json; print(json.load(open('$HOME/.rick/pi/agent/themes/rick.json'))['colors']['mdHeading'])"
```

**首次发现**: job_33 / commit dac5370  **验证状态**: ✅ 已修复

### fake pi 脚本在 PATH 替换测试中静默失败（command not found 被吞）

**根因**: Go 测试用 `t.Setenv("PATH", tmp)` 只留 fake bin 目录后，fake 脚本里 `cat`/`ls` 等**外部命令**找不到（`command not found`），若脚本再 `2>/dev/null` 或测试用 `cmd.Output()`（吞 stderr），错误完全不可见——表现为"某分支静默返回空"。`echo` 是 shell 内建所以该分支正常，造成"部分命令行、部分不行"的迷惑现象。

**精确解决步骤**:
```bash
# fake 脚本开头恢复系统 PATH（首选）
#!/bin/sh
export PATH=/usr/bin:/bin:/usr/sbin:/sbin:$PATH
# 或只用内建：while IFS= read -r line; do echo "$line"; done < "$FILE"
# 调试：去掉 2>/dev/null、加 >&2 诊断、sh -x ./fake 独立跑
```

**首次发现**: job_33 / commit b018a41  **验证状态**: ✅ 已修复

### pi 解析"托管运行时优先"后，PATH-fake 测试静默命中真实 pi（挂死/联网）

**根因**: `FindBinary`/`piCommand` 等把解析顺序改为 `cfg.PiPath → RuntimeBin()（~/.rick/pi/agent/runtime）→ PATH` 后，测试只 `t.Setenv("PATH", fakeDir)` 不再够——真实托管 pi 存在时优先命中，fake 不生效；部分测试（plan/learning workflow）甚至拉起真实 pi 交互会话 → 全套件从 265s 挂到 10 分钟超时（panic 无具体 --- FAIL）。

**精确解决步骤**:
```go
// 所有依赖 PATH fake pi 的测试，隔离托管运行时解析根：
t.Setenv("RICK_PI_AGENT_DIR", t.TempDir())  // piagent.AgentDir() → temp，RuntimeBin() 不存在 → 回退 PATH
// 或等价：t.Setenv("HOME", t.TempDir())    // AgentDir() = $HOME/.rick/pi/agent
// 配置类测试用 setupPiSettings（HOME 隔离）模式
```
排查：`go test -timeout 60s -run <可疑测试>` 逐个定位（全量 6 分钟太慢）；超时 panic 的 "running tests:" 直接点名。

**首次发现**: job_34 / commit c4812dc  **验证状态**: ✅ 已修复

### Go raw string 里嵌 JS 模板字符串（反引号）会截断字符串

**根因**: Go raw string（`` ` ``...`` ` ``）无法包含反引号——测试 fixture 要复刻含 `${...}` 模板字符串的 JS 代码时，直接写 `` \`-${x} ` `` 会在第一个反引号处截断，编译报错。

**精确解决步骤**:
```go
// 用"raw 段 + 解释型段"拼接，反引号放解释型段里：
const fixture = `result.push(theme.fg("toolDiffRemoved", ` + "`-${removed.lineNum} ${removedLine}`" + `));`
```

**首次发现**: job_34 / commit 740756d  **验证状态**: ✅ 已修复

### 幂等字符串 patch 的"锚点仍存在"陷阱（helper 插入重复）

**根因**: 字符串替换式 patch 若 old 锚点在 new 里**仍然存在**（如往 `function formatBashCall(args) {` 前插 helper，锚点本身未变）→ 二次运行 old 再次命中 → 重复插入。

**精确解决步骤**: 用**整函数替换**——old 包含完整函数体（含会被改掉的行），替换后被消费；或让 old 含一段唯一且必被修改的文本。验证：同一命令跑两遍，第二遍应报 no-op，文件逐字节不变。

**首次发现**: job_34 / commit 740756d  **验证状态**: ✅ 已修复
