# Spec 规范 — rick 的结构化工程实现契约

> 本文档定义 rick 信息内核「spec」这一 domain 描述方法升级概念：spec 是什么、四要素结构模板、与 domain 其他文档的关系、以及 spec 的验收标准（spec → 开发计划 → 功能等价实现）。
>
> 依据：`.rick/draft/rfc/rfc-rick-三层架构重构与spec信息内核-2026-08-14.md` §5「信息内核 = spec」、§6 O1 KR1.1/KR1.3。

## 1. spec 是什么

- **spec** = 结构化自然语言描述的**工程实现契约**：用无歧义的自然语言，把「模块边界 / 职责 / 接口契约 / 验收标准」四要素写清楚，作为 AI 生成开发计划、进而重构出功能等价实现的唯一信息源。
- spec 是 rick 的**信息内核**，贯彻「方法/实现隔离」：rick = 方法（自然语言描述），pi（或未来 runtime）= 实现（编程语言描述）；方法描述经模型可转化为预期行为完全一致的开发计划。
- **核心目标**：只要 spec 无歧义地描述正确的验收标准，丢弃一切源码，即可完全由 AI coding 出一个**功能等价**的 rick。
- 假设「无歧义自然语言 ⇒ 等价开发计划」是可接受的（human 已确认），其成立的关键 = spec 对验收标准的无歧义表达，最终靠功能验收实测验证。

## 2. spec 四要素结构模板

每份 spec 必须依次包含以下四节，四要素缺一不可，且必须各自独立成节（供检索与验收断言）。占位符使用尖括号 `<...>`，写 spec 时全部替换为具体事实，禁止残留未替换变量。

### 2.1 模块边界

**说明**：回答「有哪些模块、各自边界在哪、模块之间如何关联」。用层级图 + 边界规则刻画系统骨架，明确「谁调用谁、谁不能调谁」，防止越界实现。

**占位示例**（以 rick 三层金字塔重构为例）：

```
模块 = 路由层 <cli> / 处理层 <handler> / 执行层 <env>+<builder>+<runtime>
边界规则 = 上层逐级调用下层，下层不回调上层
例外 = <组合根 DIP 越级豁免 / 跨层叶子基础设施 / 交互界面直连 runtime>，需逐条显式声明
```

### 2.2 职责

**说明**：回答「每个模块做什么、不做什么」。用一句话职责 + 分条职责清单，把模块行为收敛到可判定的边界内；职责清单即后续「职责达成」的验收依据。

**占位示例**（以 rick 重构收敛结论为例）：

```
<env> 四职责：
  ① 安装/更新 pi agent
  ② 安装/更新 pi 生态扩展/插件/skill
  ③ 安装/更新 rick 自有 hook/skill/agent 定制
  ④ 提供 pi 功能点就绪 check 函数（不含 session）
<builder> 三件：templates / pibuilder / xxxxbuilder（扩展位）
<runtime> 职责：拉起 pi + 内部校验 session 就绪 + 采集行为轨迹 + 返回 (sessionID, trace)
<handler> 职责：编排 env→builder→runtime + 持久化 sessionID 到 job 目录
```

### 2.3 接口契约

**说明**：回答「模块之间如何通信、进出参是什么、返回什么数据」。用函数签名 / 返回结构 / 调用顺序刻画交互协议，使实现可替换而行为不变。

**占位示例**（以 rick 重构收敛结论为例）：

```
<runtime>.Run(...) → (sessionID, trace)
<handler> 编排顺序 = env（保证 pi 就绪）→ builder（拼提示词）→ runtime（拉 pi）
<builder>.BuildAndSave(...) → 写入 <job>/doing/prompts/ 的产物文件
```

### 2.4 验收标准

**说明**：回答「怎样算做对」。用可运行、可判定的命令/断言清单，定义「功能等价」的判据；这是 spec 信息内核的核心，验收标准写得越无歧义，AI 重构出的实现越可能等价。

**占位示例**（以 rick 重构收敛结论为例）：

```
功能等价 = 通过所有功能验收（spec → 开发计划 → 功能等价实现）
可操作判据 = go test ./... 全绿 + 集成测试全绿 + 各命令 dry-run 输出正确 + 各 check 命令 pass（详见 §4）
下沉判据 = dag 调度下沉 pi workflowScript、门禁下沉 pi hook/脚本、think/research/exporter 下沉 pi agent
```

## 3. spec 与 domain 其他文档的关系

spec 是 domain 描述方法的**升级**：它把「散落在各文档里的模块事实」收敛为「一份可被 AI 转化为等价开发计划的契约」。关系如下：

| 文档 | 与 spec 的关系 |
|------|----------------|
| `spec.md`（本文档） | 定义「spec 是什么 + 四要素模板 + 验收标准」的**规范层**；是其它 spec 的元规范 |
| `rick-spec.md` | rick 项目的**第一份 spec 实例**（按本文档四要素模板填写的具体契约） |
| `architecture.md` | 技术栈、模块划分、DIP 组合根等**事实描述**；是 spec「模块边界 / 接口契约」要素的事实来源 |
| `commands.md` | 各命令行为规范；是 spec「验收标准」要素中命令判据的事实来源 |
| `go-patterns.md` / `testing-conventions.md` / `project-conventions.md` | 编码/测试/工程约定；约束「开发计划 → 功能等价实现」这一落地的实现方式 |
| `env.md` / `pi-runtime.md` | 环境与 pi 运行时事实；是 spec「职责 / 接口契约」中 env/runtime 契约的事实来源 |
| `bugs.md` | 已知问题与精确解决命令；spec 重构落地时的避坑依据 |

**层级关系**：`spec.md`（规范）→ 具体 spec 实例（如 `rick-spec.md`）→ 实现（Go 源码 + 提示词产物）。domain 其它文档是「事实库」，spec 是「契约」，实现是「对契约的等价兑现」。

## 4. spec 的验收标准（spec → 开发计划 → 功能等价实现）

**功能等价 = 通过所有功能验收**：近似测试通过了所有的功能验收，就算是一致的；只要功能等价，就认为是效果等价。

「丢弃源码 → AI 重构出功能等价 rick」的可操作判据（以下命令全部通过，即判定重构出的 rick 与原 rick 功能等价）：

| 判据 | 命令 | 通过标准 |
|------|------|----------|
| 单元测试 | `go test ./...` | 全部 `ok`，exit code 0 |
| 集成测试 | `bash tests/tools_integration_test.sh` | 全绿（CLI 命令 + mock_agent 全链路） |
| 构建 | `./scripts/build.sh` | 成功产出 `./bin/rick` |
| dry-run | `./bin/rick <cmd> --dry-run`（doing/plan/learning/easy/dream/ctrl/human-loop） | 输出完整 prompt，且无未替换模板变量残留 |
| check | `./bin/rick tools plan_check/doing_check/learning_check/dream_check <job_id>` | 均 `✅ PASS`，exit code 0 |
| 功能验收 | 按 spec「验收标准」要素逐条断言 | 全部命中 |

判据约定：任何一条判据失败，即判定「功能不等价」，重构未完成。

## 5. 本次重构收敛结论的承载

本规范（四要素模板）须能承载本次三层金字塔重构的收敛结论，作为 `rick-spec.md` 的直接输入：

- **rick = 引导程序**：env 保证 pi 就绪 → builder 拼提示词 → runtime 拉 pi；dag 调度与门禁不再由 rick 维护，而是利用 pi 能力直接实现（rick 做薄）。
- **env 四职责**：①安装/更新 pi agent ②安装/更新 pi 生态扩展/插件/skill ③安装/更新 rick 自有 hook/skill/agent 定制 ④提供 pi 功能点就绪 check（不含 session）。
- **runtime 职责**：拉起 pi + 内部校验 session 就绪 + 采集行为轨迹 + 返回 `(sessionID, trace)`。
- **handler 职责**：编排 env→builder→runtime + 持久化 sessionID 到 job 目录。
- **下沉策略**：dag 调度 → pi workflowScript 编排；门禁 → pi hook/脚本；think/research/exporter → pi agent。
- **单一 runtime（pi）**：当前 pi 是唯一实现，为将来 dsh 预留扩展 seam（builder 的 `RuntimeBuilder`、runtime 的 `Runtime`、env 的 `RuntimeEnv` + config `runtime` 字段），切换 runtime 的前提 = 新 runtime 带来更强的生态与可定制性。
