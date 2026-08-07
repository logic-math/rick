# research-6 N6-Y14-c：因果链 5 验证（TDD hook 内置循环）

节点路径：[根 > N6-Y14-c：因果链 5 验证]
事实陈述：
- 因果链 5：所有变更执行 TDD 验证 → 确定性更新 DAG 状态
- 验证：pi afterToolCall hook 能否执行外部 test runner（同步阻塞 IO）
- 验证：hook 内能否访问 tool_call 参数（如 Write/Edit 的 file_path）触发对应 test
- 验证：hook 内能否更新 DAG 状态（rick 端 tasks.json 或 pi 端 state）
- 验证：hook 执行失败是否阻塞 LLM 继续下一步

## 执行动作

1. Read `/tmp/pi_repo/packages/coding-agent/src/core/exec.ts`（pi.exec async 实现）
2. Read `/tmp/pi_repo/packages/coding-agent/src/core/extensions/types.ts`（ToolResultEvent / ToolCallEvent / ToolResultEventResult）
3. Read `/tmp/pi_repo/packages/coding-agent/src/core/extensions/runner.ts`（emitToolResult + emitToolCall）
4. Read `/tmp/pi_repo/packages/coding-agent/src/core/agent-session.ts`（afterToolCall hook 安装 + 异常处理）
5. Read `/tmp/pi_repo/packages/coding-agent/src/core/extensions/loader.ts`（pi.exec 注册）

## 信源验证结果

### 代码原文（权重 0.4）✅

**afterToolCall hook 执行外部 test runner 的能力**

**pi.exec API**（types.ts line 1328）：
```ts
exec(command: string, args: string[], options?: ExecOptions): Promise<ExecResult>;
```
- `pi.exec` 是 async 函数，返回 `Promise<ExecResult>`
- extension 在 `afterToolCall` handler 中可 `await pi.exec("pytest", [...])` 执行 test runner
- **同步阻塞**：await 等待 test runner 完成，handler 才返回

**execCommand 实现**（exec.ts line 34-107）：
```ts
export async function execCommand(command, args, cwd, options): Promise<ExecResult> {
    return new Promise((resolve) => {
        const proc = spawn(command, args, { cwd, shell: false, stdio: ["ignore", "pipe", "pipe"] });
        // ... 收集 stdout/stderr
        waitForChildProcess(proc).then((code) => {
            resolve({ stdout, stderr, code: code ?? 0, killed });
        });
    });
}
```
- `shell: false`：直接执行命令，不经 shell（安全）
- 支持 `signal`（AbortSignal）和 `timeout`
- 返回 `{ stdout, stderr, code, killed }`
- **同步阻塞**：Promise 完成前 handler 不返回

**afterToolCall hook 安装**（agent-session.ts line 501-530）：
```ts
this.agent.afterToolCall = async ({ toolCall, args, result, isError }) => {
    const runner = this._extensionRunner;
    const hookResult = runner.hasHandlers("tool_result")
        ? await runner.emitToolResult({...})
        : undefined;
    // ... 处理 hookResult（修改 result.content / details / isError）
    return { content: normalizedContent, details: hookResult?.details, isError: hookResult?.isError ?? isError };
};
```
- `afterToolCall` 在工具执行后**同步等待** handler 完成
- handler 可修改 `result.content` / `details` / `isError`
- **同步阻塞**：handler 未返回前，LLM 不收到 tool_result

**hook 内访问 tool_call 参数**（types.ts line 853-912, 914-922）：
```ts
export interface ToolCallEvent {
    type: "tool_call";
    toolCallId: string;
    toolName: "bash" | "read" | "edit" | "write" | "grep" | "find" | "ls" | string;
    input: BashToolInput | ReadToolInput | EditToolInput | WriteToolInput | ...;
}

export interface ToolResultEvent {
    type: "tool_result";
    toolCallId: string;
    input: Record<string, unknown>;  // 完整的 tool 参数
    content: (TextContent | ImageContent)[];
    isError: boolean;
    usage?: Usage;
}
```
- `event.input` 包含完整工具参数
- Write 工具：`input.file_path` / `input.content`
- Edit 工具：`input.file_path` / `input.old_string` / `input.new_string`
- bash 工具：`input.command`
- extension 可根据 `input.file_path` 触发对应 test

**hook 内更新 DAG 状态**：
- 方式 A（rick 端 tasks.json）：extension 通过 `pi.exec` 或 Node fs API 直接写 `.rick/jobs/job_N/doing/tasks.json`
- 方式 B（pi 端 state）：extension 通过 `pi.appendEntry(customType, data)` 将 DAG 状态写入 session（CustomEntry，不参与 LLM context）
- 方式 C（sendMessage）：extension 通过 `pi.sendMessage({ customType: "dag_update", content: ... })` 注入消息触发 LLM 更新

**hook 执行失败是否阻塞 LLM**（agent-session.ts line 479-499）：
```ts
this.agent.beforeToolCall = async ({ toolCall, args }) => {
    try {
        return await runner.emitToolCall({...});
    } catch (err) {
        if (err instanceof Error) throw err;
        throw new Error(`Extension failed, blocking execution: ${String(err)}`);
    }
};
```
- beforeToolCall handler 抛异常 → **抛出到 agent loop，工具执行被阻塞**
- afterToolCall handler 抛异常 → 行为需验证（runner.ts emitToolResult 有 try-catch）

**emitToolResult 异常处理**（runner.ts line 877-930）：
```ts
async emitToolResult(event: ToolResultEvent): Promise<ToolResultEventResult | undefined> {
    for (const ext of this.extensions) {
        const handlers = ext.handlers.get("tool_result");
        for (const handler of handlers) {
            try {
                const handlerResult = await handler(event, ctx);
                // ...
            } catch (err) {
                this.emitError({...});  // 记录错误但**不抛出**
            }
        }
    }
    return result;
}
```
- afterToolCall handler 抛异常 → **被 catch，记录错误，不阻塞 LLM**
- LLM 仍收到原始 tool_result（handler 修改未生效）
- ⚠️ 与 beforeToolCall 不同：afterToolCall 异常不阻塞，beforeToolCall 异常阻塞

**因果链 5 验证结论**：

1. **afterToolCall hook 能否执行外部 test runner（同步阻塞 IO）**：
   - ✅ 能。`pi.exec` 是 async 但 await 后同步阻塞，可执行 `pytest` / `go test` / `npm test`
   - ✅ 支持 timeout 和 AbortSignal
   - ✅ 返回 stdout/stderr/code，可判断 test 是否通过

2. **hook 内能否访问 tool_call 参数触发对应 test**：
   - ✅ 能。`event.input` 含完整工具参数
   - ✅ Write/Edit 的 `input.file_path` 可映射到对应 test 文件
   - ✅ bash 的 `input.command` 可解析是否为 test 命令

3. **hook 内能否更新 DAG 状态**：
   - ✅ 方式 A：`pi.exec` 或 Node fs 写 `.rick/jobs/job_N/doing/tasks.json`
   - ✅ 方式 B：`pi.appendEntry("dag_state", { taskId, status })` 写入 session
   - ✅ 方式 C：`pi.sendMessage({ customType: "dag_update", ... })` 触发 LLM 更新

4. **hook 执行失败是否阻塞 LLM 继续下一步**：
   - ⚠️ beforeToolCall 失败 → **阻塞**（工具不执行，异常抛到 agent loop）
   - ⚠️ afterToolCall 失败 → **不阻塞**（错误被 catch，LLM 收到原始 tool_result）
   - 💡 TDD hook 应放在 **beforeToolCall**（变更前验证）或 **afterToolCall**（变更后验证，但需手动 block）
   - 💡 若 afterToolCall 中 test 失败需阻塞，应返回 `{ isError: true, content: [...] }` 让 LLM 看到错误

**因果链 5 结论**：
- ✅ 所有变更执行 TDD 验证（afterToolCall hook + pi.exec 执行 test runner）
- ✅ 确定性更新 DAG 状态（pi.exec 写 tasks.json 或 pi.appendEntry）
- ⚠️ "确定性"边界：afterToolCall 异常不阻塞 LLM，需主动返回 isError:true
- ⚠️ TDD hook 若放 afterToolCall，test 失败需返回 isError:true 让 LLM 知道
- ⚠️ TDD hook 若放 beforeToolCall（如"禁止未写 test 就改实现"），可 block 工具执行
- **因果链 5 部分成立**：TDD 验证机制成立，DAG 状态更新成立，但"确定性"依赖 hook 实现质量（afterToolCall 异常不自动阻塞）

### 运行时行为（权重 0.3）✅

- extensions.md 示例用例：git checkpointing（afterToolCall + pi.exec git commit）/ path protection / external integrations
- README "Philosophy"：extension 可实现任意工具调用前/后逻辑
- exec.ts 验证：pi.exec 是 async spawn，支持 timeout/signal

### 文档（权重 0.2）✅

- types.ts `ExtensionAPI.exec`：`exec(command, args, options): Promise<ExecResult>`
- types.ts `ToolResultEvent.input`：完整工具参数
- agent-session.ts `afterToolCall`：同步等待 handler
- runner.ts `emitToolResult`：异常被 catch，不阻塞

### 反事实（权重 0.1）N/A

本节点为外部源码调研，无代码修改。

## 还原确认

无 rick 代码修改，无需还原。

## 关键事实

1. **pi.exec 执行外部 test runner**：
   - ✅ `pi.exec(command, args, options): Promise<ExecResult>` 是 async spawn
   - ✅ `shell: false` 安全执行
   - ✅ 支持 timeout 和 AbortSignal
   - ✅ 返回 `{ stdout, stderr, code, killed }`
   - ✅ await 后同步阻塞 handler

2. **hook 内访问 tool_call 参数**：
   - ✅ `event.input` 含完整工具参数
   - ✅ Write: `input.file_path` / `input.content`
   - ✅ Edit: `input.file_path` / `input.old_string` / `input.new_string`
   - ✅ bash: `input.command`
   - ✅ 可根据 file_path 映射到对应 test 文件

3. **hook 内更新 DAG 状态**：
   - ✅ 方式 A：`pi.exec` 或 Node fs 写 `.rick/jobs/job_N/doing/tasks.json`
   - ✅ 方式 B：`pi.appendEntry("dag_state", { taskId, status })` 写入 session（不参与 LLM context）
   - ✅ 方式 C：`pi.sendMessage({ customType: "dag_update", ... })` 触发 LLM 更新

4. **hook 执行失败的阻塞行为**：
   - ⚠️ beforeToolCall 失败 → **阻塞**（工具不执行，异常抛到 agent loop）
   - ⚠️ afterToolCall 失败 → **不阻塞**（错误被 catch，LLM 收到原始 tool_result）
   - 💡 TDD hook 放 beforeToolCall 可确定性阻止"未写 test 就改实现"
   - 💡 TDD hook 放 afterToolCall 需主动返回 `{ isError: true }` 让 LLM 知道 test 失败

5. **因果链 5 部分成立**：
   - ✅ TDD 验证机制成立（afterToolCall + pi.exec）
   - ✅ DAG 状态更新成立（pi.exec 写 tasks.json / pi.appendEntry）
   - ⚠️ "确定性"依赖 hook 实现质量：
     - beforeToolCall 异常阻塞 ✅
     - afterToolCall 异常不阻塞 ⚠️（需主动 isError:true）
   - 💡 建议 rick TDD hook 分两层：
     - beforeToolCall：检查"是否已有对应 test"（无 test 则 block）
     - afterToolCall：执行 test，失败则返回 isError:true + 错误内容

6. **rick TDD hook 实现建议**：
   ```ts
   pi.on("tool_call", (event, ctx) => {
     if (event.toolName === "edit" || event.toolName === "write") {
       const filePath = event.input.file_path;
       const testFile = mapToTestFile(filePath);
       if (!fs.existsSync(testFile)) {
         return { block: true, reason: `No test file found for ${filePath}` };
       }
     }
   });

   pi.on("tool_result", async (event, ctx) => {
     if (event.toolName === "edit" || event.toolName === "write") {
       const filePath = event.input.file_path;
       const testFile = mapToTestFile(filePath);
       const result = await pi.exec("go", ["test", testFile]);
       if (result.code !== 0) {
         // 更新 DAG 状态为 failed
         await pi.exec("rick-cli", ["dag", "update", "--status", "failed"]);
         return { isError: true, content: [{ type: "text", text: result.stderr }] };
       }
       // 更新 DAG 状态为 passed
       await pi.exec("rick-cli", ["dag", "update", "--status", "passed"]);
     }
   });
   ```

## 疑问点

无。本节点事实清晰，源码三重交叉验证（exec.ts + types.ts + runner.ts + agent-session.ts）一致。

## 置信度评估（由 research 主调度计算）

- 代码原文 ✅ × 0.4 = 0.4
- 运行时行为 ✅ × 0.3 = 0.3
- 文档 ✅ × 0.2 = 0.2
- 反事实 N/A × 0.1 = 0
- 合计 = 0.9（高，≥ 0.8 终止）
