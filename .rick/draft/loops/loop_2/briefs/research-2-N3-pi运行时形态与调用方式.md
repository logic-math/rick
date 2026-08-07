# research-2 N3-pi 运行时形态与调用方式

节点路径:[根 > N3-pi 运行时形态与调用方式]
事实陈述:pi 的运行时形态(Go/Node/Python/Rust)、调用方式(Go binding/CLI/HTTP API/SDK)、是否需嵌入额外 runtime、进程管理开销。

## 执行动作

1. 读取根 `package.json`(engines / packageManager / workspaces)
2. 读取 `packages/coding-agent/package.json`(bin / main / type / dependencies)
3. 读取 `packages/agent/package.json`(exports)
4. 读取 README "Modes" / "Programmatic Usage" / "CLI Reference" 章节
5. 读取 SDK 文档 `sdk.md`
6. 读取 RPC 文档 `rpc.md` 启动方式
7. 检查 standalone binary 构建脚本(scripts/build-binaries.sh)

## 信源验证结果

### 代码原文(权重 0.4)✅

- **运行时**:Node.js( TypeScript),`engines.node >= 22.19.0`
- **package.json type=module**:ESM 模块系统
- **packageManager**:null(未锁定包管理器,但 README 使用 npm install -g)
- **bin**:`{pi: dist/cli.js}` → npm 全局安装后 `pi` 命令指向 `dist/cli.js`(Node shebang)
- **入口**:`packages/coding-agent/src/cli.ts`(编译到 dist/cli.js)
- **关键依赖**(packages/coding-agent):
  - `@earendil-works/pi-agent-core`(agent runtime)
  - `@earendil-works/pi-ai`(多 provider LLM API)
  - `@earendil-works/pi-client` / `pi-protocol` / `pi-tui`
  - `jiti`(运行时 TS 加载 extension)
  - `cross-spawn` / `chalk` / `highlight.js` / `glob` / `hosted-git-info` / `minimatch`
- **exports**(`@earendil-works/pi-agent-core`):`.` / `./node` / `./experimental` / `./experimental/session/testing` / `./package.json`

### 运行时行为(权重 0.3)✅

README + SDK 文档明确 **4 种调用方式**:

1. **CLI(交互式)**:`pi [options] [@files...] [messages...]` — 默认交互式 TUI
2. **CLI(print 模式)**:`pi -p "prompt"` — 一次性输出
3. **CLI(JSON 模式)**:`pi --mode json "prompt"` — 流式 JSON lines 输出
4. **CLI(RPC 模式)**:`pi --mode rpc` — stdin/stdout JSONL 双向通信,**专为非 Node 集成设计**(rpc.md 原文:"RPC mode enables headless operation of the coding agent via a JSON protocol over stdin/stdout. This is useful for embedding the agent in other applications, IDEs, or custom UIs.")
5. **SDK(Node 嵌入)**:`import { createAgentSession } from "@earendil-works/pi-coding-agent"` — 纯 Node.js 程序化嵌入

**Standalone binary**:README "Building standalone binaries from release source" 段:
- `./scripts/build-binaries.sh --offline-model-data --platform linux-x64 --out "$PWD/out"`
- "compiles the Bun executable, and stages its runtime assets" → **使用 Bun 编译独立二进制**(Bun 是 Node.js 兼容的 JS runtime,可将 JS 编译为单文件可执行)
- 这意味着 pi 可分发**无需 Node.js 环境的单二进制**(类似 claude code 的分发形态)

**进程模型**:
- CLI 模式:单进程(交互式/print/json)
- RPC 模式:单进程,stdin/stdout 通信,长生命周期
- SDK 模式:嵌入宿主进程(Node.js)
- bash 工具调用:`cross-spawn` 启动子进程,环境变量传递 `PI_SESSION_ID` / `PI_SESSION_FILE` / `PI_PROVIDER` / `PI_MODEL`

### 文档(权重 0.2)✅

- README "Quick Start":`npm install -g --ignore-scripts @earendil-works/pi-coding-agent` 或 `curl -fsSL https://pi.dev/install.sh | sh`
- README "Programmatic Usage" 章节:SDK + RPC Mode 两种嵌入方式
- rpc.md 开篇:"For Node.js/TypeScript users: If you're building a Node.js application, consider using AgentSession directly... For a subprocess-based TypeScript client, see src/modes/rpc/rpc-client.ts"
- rpc.md 明确:"RPC mode enables headless operation... useful for embedding the agent in other applications, IDEs, or custom UIs"(非 Node 集成首选)
- SDK 文档:"The SDK provides programmatic access to pi's agent capabilities. Use it to embed pi in other applications"

### 反事实(权重 0.1)N/A

- 本节点为外部文档调研,无代码修改

## 还原确认

无 rick 代码修改,无需还原。

## 关键事实

1. **运行时**:Node.js ≥ 22.19.0(ESM/TypeScript),无可选 Go/Rust runtime
2. **调用方式 5 种**:
   - CLI 交互式 / CLI print / CLI json / CLI rpc / SDK 嵌入
3. **rick(Go)集成路径**:
   - ❌ Go binding:不存在(pi 是 Node.js,无 Go FFI)
   - ✅ **CLI 子进程**:`exec.Command("pi", "--mode", "rpc", ...)` 或 `pi -p` —— 与 rick 现有 `exec.Command("claude", "-p", ...)` 模式完全同构
   - ✅ **RPC 模式**:专为非 Node 集成设计,stdin/stdout JSONL 长连接
   - ❌ HTTP API:不内置(但可写 extension 启 HTTP server)
   - ❌ SDK 嵌入:仅限 Node.js 宿主
4. **Standalone binary**:支持,通过 Bun 编译单文件二进制(`scripts/build-binaries.sh`),分发形态可等同 claude code
5. **进程管理开销**:
   - CLI 模式:与 claude code 等同(单子进程)
   - RPC 模式:优于 CLI 模式(一次启动,多次 prompt,避免反复进程启动开销)
   - bash 工具:子进程(cross-spawn),与 claude code 等同
6. **环境变量契约**:`PI_SESSION_ID` / `PI_SESSION_FILE` / `PI_PROVIDER` / `PI_MODEL` / `PI_REASONING_LEVEL` 注入 bash 子进程
7. **配置目录**:`~/.pi/agent/`(可被 `PI_CODING_AGENT_DIR` 覆盖)

## 疑问点

- pi 二进制是否已预编译 macOS arm64?→ 需检查 release 页面(本轮未下载 release artifact,但 install.sh 暗示支持)
- RPC 模式下 rick 如何管理 pi 进程生命周期(启动/心跳/超时/崩溃恢复)?→ N5 语义对齐性讨论

## 置信度评估(由 research 主调度计算)

- 代码原文 ✅ × 0.4 = 0.4
- 运行时行为 ✅ × 0.3 = 0.3
- 文档 ✅ × 0.2 = 0.2
- 反事实 N/A × 0.1 = 0
- 合计 = 0.9(高,≥ 0.8 终止)
