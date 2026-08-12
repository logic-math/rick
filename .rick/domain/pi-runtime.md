# pi Runtime 事实知识

## pi provider/model/api-key 配置

pi **不读** `PI_PROVIDER` / `PI_MODEL` / `PI_API_KEY` 环境变量。必须通过命令行 flags 配置：

```bash
pi --provider deepseek --model deepseek-v4-flash --api-key sk-... <prompt>
```

rick 通过 config 的 `pi_extra_args` 字段透传这些 flags（`internal/agent/piagent` 在 `CallCLI`/`Execute` 时合并到 pi 命令行）。**不要试图用 env 注入 provider**。

## pi 配置目录隔离（已实现，job_33）

pi 支持 `PI_CODING_AGENT_DIR` 环境变量指定 agent 配置目录（默认 `~/.pi/agent`）。rick 已在**所有** pi 调用入口注入 `PI_CODING_AGENT_DIR=~/.rick/pi/agent`（`piagent.AgentEnv()`，用于 CallCLI 交互入口 + Executor `--mode json` + init-pi 的 install/list/version），rick 的 pi 配置与用户 `~/.pi` 完全隔离：settings.json（含 `hideThinkingBlock: true` 托管默认值）、主题、扩展注册都在 `~/.rick/pi/agent` 下管理。

**注意（pi 内部行为）**：`pi install` 对 user scope 的包在 managed 路径不存在时回退到全局 npm root（`npm root -g`，pi 安装器将其指向 `~/.pi/agent/npm/node_modules`）复用代码——注册（settings.json packages）是隔离的，但包代码可能与用户全局安装共享。用户卸载自己的包可能影响 rick 的扩展（已知限制，非 rick bug）。

## pi 扩展加载机制

pi 启动时 loader 找扩展入口：
1. `package.json` 含 `pi.extensions` 字段 → npm 包（如 `pi-subagents`、`pi-web-access`）
2. `~/.pi/agent/extensions/<name>/index.ts` 子目录结构

`pi install npm:<pkg>` 装到 `~/.pi/agent/npm/node_modules/`，注册到 settings.json packages 数组。`pi list` 列出已装扩展。

## pi `--mode json` 事件流

pi json 模式输出 JSONL（每行一事件）。关键字段：
- session header `{"type":"session","id":"..."}` → session ID
- `agent_settled` → 终止信号（pi 不输出 duration，需自计时）
- `message_end` → user 和 assistant 轮次都发，取 FinalMessage 要过滤 `role=="assistant"`
- `tool_execution_start{toolCallId,toolName,args}` + `tool_execution_end{toolCallId,result,isError}` → 工具调用（result 可能是 JSON 对象非字符串）
- 字段是 camelCase（pi），非 snake_case（claude code）

## 美团内网 anthropic 网关（mcli.sankuai.com）不可被 pi 复用

claude code 用 `ANTHROPIC_BASE_URL=mcli.sankuai.com` + `ANTHROPIC_AUTH_TOKEN` + `ANTHROPIC_CUSTOM_HEADERS`（动态 X-Timestamp/X-Client-Token）。但：
- 光有 base_url + token + headers 直接 curl 网关 → 400 "Request is not allowed"
- claude code 自己能通 → 网关做**客户端识别**（疑似 User-Agent 或 SDK 签名），pi 复制不了
- catpaw-pilot 支持 pi-coding-agent（`enabled: false`），启用机制不公开

**结论**：pi 不能通过 `models.json` 复用美团内网网关。需用公网 provider（deepseek 等）。

## pi 不支持的 claude code flags

迁移时这些 claude code flags 在 pi 无对应，直接删除：
- `--dangerously-skip-permissions`（pi 默认无 permission popup）
- `--output-format stream-json` → 用 `--mode json`（语义对应，flag 不同）
- `--session-id` → pi 也支持 `--session-id`（创建），但 `--session` 是加载（见 bugs.md）

## DeepSeek 配置（已验证可用）

```json
// ~/.rick/config.json
{
  "pi_extra_args": ["--provider", "deepseek", "--model", "deepseek-v4-flash", "--api-key", "sk-..."]
}
```
deepseek-v4-flash 默认带 thinking（思考慢），可用 `--thinking off` 关闭。响应有时不稳定（网关限流），后台运行 + 长超时（600s）+ tee 捕获。

## rick 自闭环运行时（~/.rick/pi/agent/runtime，job_34 起）

- rick 不用全局 pi：`init-pi` 用 `npm install --prefix ~/.rick/pi/agent/runtime @earendil-works/pi-coding-agent@<全局版本>` 装独立副本（全局有则匹配版本，pinned 失败自动降级 latest）
- **解析优先级（全链路一致）**：`cfg.PiPath` → 托管运行时（`piagent.RuntimeBin()` = `~/.rick/pi/agent/runtime/node_modules/.bin/pi`）→ PATH `pi`——FindBinary / piPathOrDefault / piCommand / Executor 默认
- 运行时副本保持 **stock**（与全局逐字节一致），**不做代码级 patch**（用户决策：破坏运行时的做法不引入，后续可能再做）
- 主题等 UI 定制只走主题配置（`rick tools theme` → `~/.rick/pi/agent/themes/rick.json`，custom theme 目录 pi 自动发现 + 热重载）
- 若未来做运行时 patch（diff 反显→加粗/语法高亮）：需整函数替换保幂等；import 相对路径按层级算（diff.js 在 dist/modes/interactive/components/ → utils 需 `../../../`，bash.js 在 dist/core/tools/ → `../../`）
- 测试隔离：RICK_PI_AGENT_DIR 或 HOME 指到 temp，RuntimeBin 不存在 → 回退 PATH fake

## pi 渲染行为 vs 主题 token 边界（job_34 结论）

- 主题只有 51 个颜色 token（fg/bg 颜色映射），**只能决定颜色**
- 渲染行为（diff 变更词反显 `\x1b[7m`、diff/命令行语法高亮）**不可主题化**——pi 无配置口，改行为必须改 dist JS
- 已否决运行时 patch → diff/命令渲染保持 pi 原生行为（反显高亮、平色文本）
