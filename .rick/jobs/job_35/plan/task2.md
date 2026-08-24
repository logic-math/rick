# 依赖关系
task1

# 任务名称
产出 rick 第一份 spec（四层架构 + 5 模块 + env 四职责契约）

# 任务目标
按 task1 定义的 spec 规范，产出 rick 项目第一份 spec（KR1.2），使 rick 拥有这份 spec（信息内核）。spec 覆盖收敛后的最终架构：
- **四层架构（调用逐级往下）**：
  - 第一层 入口：CLI / TUI / WEB-UI（路由命令、解析参数、交互呈现）
  - 第二层 调度聚合：handler（接受入口参数，编排 env/runtime/builder 完成功能）
  - 第三层 执行：env（pi/dsh 及扩展的检查/安装/配置/维护）+ runtime（pi/dsh 调用封装：参数解析+调用）+ builder（按入口拼接 pi/dsh 提示词产物）
  - 第四层 基础设施：pi（当前 runtime）/ dsh（预留 runtime）/ workspace（路径解析）/ config（~/.rick/config.json 加载）
- 调用关系：上层调下层（逐级往下），下层不回调上层；**例外一**：env ↔ dsh 相互调用（dsh 生态交互关系，非纯单向；不单列 dshRuntime/dshBuilder 节点，链接直接连到具体组件 env 与 dsh）；**例外二**：TUI / WEB-UI 跨层直连 pi/dsh（交互界面直接驱动 runtime，绕过 handler/env/runtime/builder）；**例外三**：组合根（cmd 的 RunE 懒加载实例化 piRuntime/piEnv/pibuilder 注入 handler）是 DIP 组合根模式，越级豁免；**例外四**：workspace/config 是跨层叶子基础设施（路径解析/配置加载），可被任意层直接使用，不参与功能调用链的「逐级往下」约束
- 5 模块职责与边界（含 env 四职责、runtime 职责、handler 职责、builder 三件）；`internal/prompt` = builder 三件中 templates 的承载包（L3）；L3 内部（env/runtime/builder）可复用共享路径工具（AgentDir/RuntimeDir/RuntimeBin/AgentEnv 等），不视为越级回调
- builder 三件契约（templates = go embed 内嵌提示词；pibuilder = pi 统一入口组合子 builder；xxxxbuilder = 扩展位）
- runtime 契约：拉起 pi + 内部校验 session 就绪 + 返回 (sessionID, 行为轨迹)
- 删除清单：executor（调度→pi）、parser（读/校验→pi）、actpath（轨迹→runtime）、logging（死代码）、git（→pi 脚本）、agent 接口（失去消费者）
- 验收标准：功能等价 = 通过所有功能验收；rick 做薄（dag 调度与门禁下沉 pi）；**单一 runtime（pi）为当前实现**——为将来 deepseek harness(dsh) 预留三扩展 seam：builder 的 `RuntimeBuilder`（= xxxxbuilder 转义层）、runtime 的 `Runtime`（`Name`/`Run`）、env 的 `RuntimeEnv`（`Ensure`/`DeployCustomizations`/`CheckReady`）+ config `runtime` 字段（默认 `pi`）；当前 pi 是唯一实现，不写 dsh 代码

依据：`.rick/draft/rfc/rfc-rick-三层架构重构与spec信息内核-2026-08-14.md` §4（目标架构）、§6 O1 KR1.2。此 spec 是 task3~11 重构的「契约」。

# 关键结果
1. 新增 `.rick/domain/rick-spec.md`，含四要素结构（模块边界/职责/接口契约/验收标准）且覆盖四层架构 + 5 模块 + 删除清单
2. env 四职责明确写入：①安装/更新 pi agent ②安装/更新 pi 生态扩展/插件/skill ③安装/更新 rick 自有 hook/skill/agent 定制 ④提供 pi 功能点就绪 check 函数（不含 session）
3. runtime 职责明确写入：拉起 pi + 内部校验 session 就绪 + 采集行为轨迹 + 返回 (sessionID, trace)；handler 职责：编排 env→builder→runtime + 持久化 sessionID 到 job 目录；**注入模型明确写入：方法(system prompt) + 技能(skills) + 实例(user prompt) 三层分离——方法走 `--append-system-prompt`（保留 pi 默认骨架）、技能走 pi skills 机制、实例走 prompt 文件**
4. 下沉策略明确写入：dag 调度 → pi workflowScript 编排（await 顺序）、门禁 → pi hook（rick-gates 扩展）+ 确定性脚本；think/research/exporter → pi agent（env 职责 3 落盘）
5. 扩展 seam 明确写入：`RuntimeBuilder`（builder/xxxxbuilder）、`Runtime`（runtime，`Name`/`Run`）、`RuntimeEnv`（env）+ config `runtime` 字段（默认 `pi`）；「新增 dsh 只新增 dshBuilder/dshRuntime/dshEnv 三个实现并注册，cli/handler/方法层 templates 不改」
6. 四层架构图写入 rick-spec.md（含 mermaid/text 图）：第一层 CLI/TUI/WEB-UI → 第二层 handler → 第三层 env/runtime/builder → 第四层 pi/dsh/workspace/config；标注「逐级往下」+ 例外一「env ↔ dsh 相互调用（不单列 dshRuntime/dshBuilder 节点）」+ 例外二「TUI / WEB-UI 跨层直连 pi/dsh」

# 测试方法
（本 task 为文档任务，TDD 红-绿循环不适用，改为「验收断言」：文件存在 + 关键词命中；断言真实文档，不 mock。）

1. 正常路径：前置条件 = `.rick/domain/spec.md`（task1）存在；输入 = rick-spec.md 正文；操作 = 写 `.rick/domain/rick-spec.md` + `git add`；预期 = `test -f .rick/domain/rick-spec.md` 返回 0。
2. 边界（5 模块 + env 四职责 + 四层架构覆盖）：前置条件 = rick-spec.md 已写；输入 = 待写入正文；操作 = `for w in cli handler env builder runtime; do grep -q "$w" .rick/domain/rick-spec.md || exit 1; done` + `for w in 安装 生态扩展 定制 就绪; do grep -q "$w" .rick/domain/rick-spec.md || exit 1; done` + `for w in 第一层 第二层 第三层 第四层 CLI TUI WEB-UI; do grep -q "$w" .rick/domain/rick-spec.md || exit 1; done`；预期 = exit 0（5 模块名 + env 四职责 + 四层架构关键词各自命中）。
3. 异常（与 RFC 一致 + 无变量泄漏 + 扩展 seam）：前置条件 = rick-spec.md 已写；操作 = `for w in dag 门禁 sessionID; do grep -q "$w" .rick/domain/rick-spec.md || exit 1; done` + `for w in RuntimeBuilder RuntimeEnv runtime; do grep -q "$w" .rick/domain/rick-spec.md || exit 1; done` + `grep -c '{{' .rick/domain/rick-spec.md`（=0）；预期 = exit 0（dag/门禁/sessionID + 三 seam 各自命中）且无 `{{`。
