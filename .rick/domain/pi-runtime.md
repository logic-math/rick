# pi Runtime 事实知识

## pi provider/model/api-key 配置

pi **不读** `PI_PROVIDER` / `PI_MODEL` / `PI_API_KEY` 环境变量。必须通过命令行 flags 配置：

```bash
pi --provider deepseek --model deepseek-v4-flash --api-key sk-... <prompt>
```

rick 通过 config 的 `pi_extra_args` 字段透传这些 flags（`internal/agent/piagent` 在 `CallCLI`/`Execute` 时合并到 pi 命令行）。**不要试图用 env 注入 provider**。

## pi 配置目录隔离（规划中，未实现）

pi 支持 `PI_CODING_AGENT_DIR` 环境变量指定 agent 配置目录（默认 `~/.pi/agent`）。源码 `dist/config.js:396-417`：`getAgentDir()` 优先读此 env。未来 rick 会设 `PI_CODING_AGENT_DIR=~/.rick/pi/agent`，让 rick 的 pi 配置与用户 `~/.pi` 完全隔离。

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
