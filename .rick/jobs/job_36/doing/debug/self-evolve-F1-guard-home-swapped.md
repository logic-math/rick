# F1：隔离守卫在「换 HOME 形态」下未安装（dev 可注册生产工作区）

- 发现方式：task22 的端到端验收脚本（`scripts/self-evolve-e2e.sh`）用**真实启动形态**跑守卫用例
- 严重度：高（绕过 human 裁决的 L4-6 fail-fast 纪律：dev 会话可写生产工作区 `.rick`）
- 状态：已修复（gate7 新增第 ⑨ 条断言永久钉死）

## 现象（修复前，实测）
```
$ HOME=<dev-home> RICK_PI_AGENT_DIR=<agent> rick web --listen 127.0.0.1 --port 18999 --token f1tok
$ curl -X POST :18999/api/workspaces -d '{"path":"/workdir/sunquan20/BERT_KEETA","name":"f1-probe"}'
{"id":"3afaa742","path":"/workdir/sunquan20/BERT_KEETA",...}   [HTTP 201]   ← 本应 4xx
```

## 根因
守卫启用条件 = `IsDevStateDir(resolved)`，其定义是 `resolved != $HOME/.rick`。
而 `rick tools dev-web` 启动 dev 实例的形态是**换 HOME**（`HOME=<dev-home>`），状态目录仍是默认的 `$HOME/.rick`（即 `<dev-home>/.rick`）→ 条件恒为 false → **守卫与 `RICK_PI_AGENT_DIR` 强制项都没生效**。

两种 dev 形态的差异（修好后明确区分）：

| 形态 | 状态目录 | pi 沙盒 | 工作区守卫 | `RICK_PI_AGENT_DIR` 强制 |
|---|---|---|---|---|
| 生产 | 真实用户 `~/.rick` | 生产 `~/.rick/pi/agent` | 不装 | 不要求 |
| **home-swapped**（dev-web 实际形态） | `<dev-home>/.rick` | 随 HOME 已隔离 | **装** | 不要求 |
| state-dir-only | `--state-dir` | **仍指生产** | 装 | **强制** |

## 修复
新增 `web.DevModeInfo(resolved) (isDev, prodState, homeSwapped)`：
- `prodState` = `RICK_PROD_STATE_DIR` > 真实用户家目录（`user.Current()`）的 `.rick`
- `isDev` = `Clean(resolved) != Clean(prodState)`（两种形态都算 dev）
- `homeSwapped` = 当前 `$HOME` ≠ 真实用户家目录

`internal/cmd/web.go`：`isDev` 时一律装守卫；`RICK_PI_AGENT_DIR` 强制仅在 `!homeSwapped` 时生效。

## 教训（门禁设计）
只按「实现者以为的形态」构造门禁会漏掉**运行时真实形态**：gate7 原先只覆盖 `--state-dir` 形态，而 dev-web 用的是换 HOME 形态，于是缺口一路通过 3 个门禁直到 E2E 才暴露。
→ 门禁必须按**用户/产品实际调用路径**构造（E2E 用真实命令与真实启动形态），而不是按内部实现的分支。
