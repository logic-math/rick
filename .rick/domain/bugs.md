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

