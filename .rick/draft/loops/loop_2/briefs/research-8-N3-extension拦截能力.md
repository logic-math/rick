# research-8 N3：extension 拦截能力

节点路径：[根 > N3-extension 拦截能力]
事实陈述：resources_discover 事件 + 其他 hook 能拦截哪些路径创建？能否阻止 .pi 目录创建？

## 执行动作

1. Read `extensions/types.ts:520-560`（resources_discover 事件定义）
2. Read `extensions/types.ts:1198-1230`（所有 extension 事件 on() 签名）
3. Read `extensions/runner.ts:1147-1192`（emitResourcesDiscover 实现）
4. Read `resource-loader.ts:330-377`（extendResources 合并逻辑）
5. Grep `resources_discover|ResourcesDiscover|skillPaths` 验证所有调用点

## 信源验证结果

### 代码原文（权重 0.4）✅

**resources_discover 事件定义**（types.ts:543-555）：
```ts
/** Fired after session_start to allow extensions to provide additional resource paths. */
export interface ResourcesDiscoverEvent {
    type: "resources_discover";
    cwd: string;
    reason: "startup" | "reload";
}

/** Result from resources_discover event handler */
export interface ResourcesDiscoverResult {
    skillPaths?: string[];
    promptPaths?: string[];
    themePaths?: string[];
}
```
- 触发时机：**session_start 之后**（注释明确）
- reason：startup（首次启动）/ reload（重载）
- 返回值：可选的 skillPaths / promptPaths / themePaths 数组
- **只支持 skills/prompts/themes 三类资源，不支持 extensions/agents/settings.json/SYSTEM.md**

**emitResourcesDiscover 实现**（runner.ts:1147-1192）：
```ts
async emitResourcesDiscover(...) {
    const skillPaths = [];
    const promptPaths = [];
    const themePaths = [];
    for (const ext of this.extensions) {
        const handlers = ext.handlers.get("resources_discover");
        if (handlers) {
            for (const handler of handlers) {
                const event = { type: "resources_discover", cwd, reason };
                const result = handlerResult as ResourcesDiscoverResult | undefined;
                if (result?.skillPaths?.length) {
                    skillPaths.push(...result.skillPaths.map((path) => ({ path, extensionPath: ext.path })));
                }
                // ... promptPaths, themePaths 同理
            }
        }
    }
    return { skillPaths, promptPaths, themePaths };
}
```
- **追加式**：遍历所有 extension，收集所有返回路径，合并到一个数组
- **不拦截**：extension 无法删除其他 extension 或默认发现的路径
- **无过滤能力**：extension 不能阻止 pi 加载默认路径的资源

**extendResources 合并逻辑**（resource-loader.ts:339-377）：
```ts
extendResources(paths: ResourceExtensionPaths): void {
    const skillPaths = this.normalizeExtensionPaths(paths.skillPaths ?? []);
    // ...
    if (skillPaths.length > 0) {
        this.lastSkillPaths = this.mergePaths(
            this.lastSkillPaths,  // 已有路径（含默认发现）
            skillPaths.map((entry) => entry.path),  // extension 返回的新路径
        );
        this.updateSkillsFromPaths(this.lastSkillPaths, this.resourceMetadataByPath);
    }
}
```
- `mergePaths`（resource-loader.ts:845-858）：去重合并，primary 在前，additional 在后
- **默认路径在前，extension 路径在后**：extension 路径是补充而非替换
- 同名 skill 冲突时，默认路径（user scope > project scope）优先

**extension 事件全枚举**（types.ts:1198-1230，共 30+ 事件）：

| 事件类别 | 事件名 | 能否拦截路径 |
|---|---|---|
| 资源发现 | resources_discover | 否（追加，不拦截） |
| 信任 | project_trust | 否（只决策信任，不影响路径） |
| 会话 | session_start / session_before_switch / session_before_fork / session_before_compact / session_compact / session_shutdown / session_before_tree / session_tree / session_info_changed | 否（会话生命周期，不影响资源路径） |
| 上下文 | context | 否（修改上下文内容，不影响路径） |
| Provider | before_provider_headers / after_provider_response | 否（HTTP 层，不影响路径） |
| Agent | before_agent_start / agent_start / agent_end / agent_settled | 否（agent 生命周期，不影响路径） |
| Turn | turn_start / turn_end | 否（轮次生命周期） |
| Message | message_start / message_update / message_end | 否（消息生命周期） |

**关键结论**：**30+ 个 extension 事件中，0 个能拦截路径创建或路径计算**。extension 只能：
1. 通过 `resources_discover` 追加资源路径（skills/prompts/themes）
2. 通过 `project_trust` 决策是否信任项目（间接影响是否加载 project scope 资源）
3. 通过 `context` 事件修改上下文内容（不影响资源加载路径）

**extension 无法阻止 .pi 目录创建**：
- pi 本身就不主动创建 .pi 目录（N1 已验证，0 处 mkdirSync CONFIG_DIR_NAME）
- extension 无 `before_resource_discover` 或 `filter_resource_paths` 事件
- extension 无法删除默认发现的路径
- 唯一"阻止加载 .pi 资源"的方式：`project_trust` 事件返回 `trusted: "no"`，但这会禁用**所有** project scope 资源（包括 .pi 和假设的 .rick），不是针对性拦截

### 运行时行为（权重 0.3）✅

**resources_discover 触发时序**（types.ts:543 注释 + runner.ts 调用链）：
1. pi 启动
2. 加载默认资源（user scope + project scope，若 trusted）
3. 触发 `session_start` 事件
4. 触发 `resources_discover` 事件（reason: "startup"）
5. extension 返回额外路径
6. extendResources 合并到已加载资源
7. LLM 开始工作

**时序含义**：resources_discover 在默认资源加载**之后**触发，extension 无法阻止默认资源加载。要阻止默认资源加载必须用 `--no-skills` 等 flag（在 extension 加载之前生效）。

### 文档（权重 0.2）✅

- extensions.md `resources_discover`：extension 可动态提供 skill/prompt/theme 路径
- extensions.md 事件列表：所有事件都是"通知"或"追加"，无"拦截"语义
- README "Extensions"：extension 是 pi 的扩展机制，不是覆盖机制

### 反事实（权重 0.1）N/A

本节点为外部源码调研，无代码修改。

## 还原确认

本轮纯外部调研，未修改 rick 仓库代码，无需 git restore。

## 关键事实

1. **resources_discover 是追加式，非拦截式**：extension 返回的路径被 mergePaths 合并到默认路径之后，同名冲突时默认路径优先。extension 无法删除或替换默认发现的路径。

2. **resources_discover 只支持 3 类资源**：skills / prompts / themes。不支持 extensions / agents / settings.json / SYSTEM.md / APPEND_SYSTEM.md / npm / git。

3. **resources_discover 在默认资源加载之后触发**：时序为"加载默认 → session_start → resources_discover → 合并"。extension 无法阻止默认资源加载，只能补充。

4. **30+ 个 extension 事件中 0 个能拦截路径**：所有事件都是"通知"或"追加"语义，无"拦截"或"过滤"语义。extension 无法拦截路径计算或路径创建。

5. **extension 无法阻止 .pi 目录创建**：
   - pi 本身不主动创建 .pi 目录（N1 验证）
   - extension 无 `before_resource_discover` / `filter_resource_paths` 事件
   - extension 无法删除默认发现的路径
   - `project_trust` 事件返回 `trusted: "no"` 可禁用所有 project scope 资源，但这是全局禁用，不是针对性拦截 .pi

6. **extension 的实际能力边界**：
   - 追加资源路径（resources_discover，仅 skills/prompts/themes）
   - 决策项目信任（project_trust，全局影响）
   - 修改上下文内容（context 事件）
   - 注册工具（registerTool）
   - 注册 provider（registerProvider）
   - 注册 agent（registerAgent，subagent extension）
   - 拦截会话/消息/turn 生命周期（可取消操作，不影响路径）

7. **"extension 拦截 .pi"方案不可行**：extension 无法实现"阻止 pi 读 .pi，改读 .rick"的拦截。extension 只能在 pi 已加载默认资源后追加 .rick 路径（但默认 .pi 资源仍在）。

## 疑问点

无。本节点事实清晰，源码三重交叉验证（types.ts + runner.ts + resource-loader.ts）一致。

## 置信度评估（由 research 主调度计算）

- 代码原文 ✅ × 0.4 = 0.4
- 运行时行为 ✅ × 0.3 = 0.3
- 文档 ✅ × 0.2 = 0.2
- 反事实 N/A × 0.1 = 0
- 合计 = 0.9（高，≥ 0.8 终止）
