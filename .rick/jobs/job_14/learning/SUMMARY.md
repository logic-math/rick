APPROVED: true

# Job job_14 执行总结

## 执行概述

**项目目标**: Rick v2.0 核心升级 —— 建立 act-path 进化循环，将 Rick 从"执行框架"升级为具备"进化能力"的三层正交 AI Coding 控制框架
**实际完成**: 9/9 任务全部成功，零重试
**整体评价**: ⭐⭐⭐⭐⭐

## 关键成就

1. **act-path 负反馈机制**: 通过程序性 NDJSON 解析，`rick doing` 执行后自动生成 `act-path-{taskID}.md`，包含工具调用轨迹、报错次数、执行时长，为 learning/dream 层提供可靠优化信号（KR1 ✅）

2. **DIP 全链路架构**: 建立 `agent.AgentSession / AgentExecutor` 稳定接口契约，`claudecode` 适配器实现接口，`doing.go` 作为唯一组合根注入——runner/executor/actpath 无任何具体实现依赖（KR1 ✅）

3. **dream 命令落地**: 新增 `rick dream` 命令，支持 `--dry-run`，完整 8 步 SOP，精准注入 sense + evolve-skills（KR3 ✅）

4. **Core-skills 精准注入**: 8 个 core-skill 文件通过 `embed.FS` 编译进二进制，按 SOP 阶段精准注入（plan→sense+tc / doing→tdd+debug / learning→gen-skill / dream→sense+evolve-skills）（KR5 ✅）

5. **红绿 TDD SOP**: testing agent（红阶段）生成失败测试后立即验证 RED；若意外通过（pass=true）则重试最多 2 次；coding agent（绿阶段）强制 debug skill 声明（KR5 ✅）

6. **v2 E2E 全覆盖**: 4 阶段 mock 端到端测试（plan/doing/learning/dream），基于真实 NDJSON 格式验证 act-path 生成和 raw_session.log 内容（KR1 ✅）

## 问题与教训

### 问题1: task.md 描述结构 vs 测试期望不一致

**根本原因**: task3.md 描述 nested 结构（`skills/sense/skill.md`），但 test3.py 期望 flat 结构（`skills/sense.md`）。计划文档与测试脚本不同步
**解决方案**: 以测试脚本为准，采用 flat 结构
**经验教训**: **测试脚本是规格的黄金来源**，与 task.md 描述冲突时以测试脚本为准；plan 阶段应确保 task.md 描述与测试断言结构一致

### 问题2: AgentExecutor 接口签名与实现不匹配

**根本原因**: task1 定义接口时使用 `Execute(ctx context.Context, prompt string)`，task2 实现时基于实际需求用 `Execute(promptFile, taskID string)`——两个 task 并行规划，接口未对齐
**解决方案**: task6 接线时以实现为准，更新接口签名，移除 `context` 依赖
**经验教训**: 并行 task 定义接口时需要"接口所有者"预先协商签名；或接口 task 先完成后其他 task 才基于此实现

### 问题3: 同包 mock struct 命名冲突

**根本原因**: `runner_test.go` 和 `executor_test.go` 同属 `executor_test` 包，各自定义 `mockAgentExecutor` 导致编译冲突
**解决方案**: `runner_test.go` 改名为 `mockAgentExecutorWithSession`
**经验教训**: 同包测试文件共享命名空间；team mock 应集中定义或用不同命名前缀区分

### 问题4: check_variadic_api.py 不支持 Go methods

**根本原因**: 工具使用 `func\s+{func_name}\s*\(` 正则，不匹配 Go method 形式 `func (tr *T) methodName(...)`
**解决方案**: 直接 grep 验证
**经验教训**: 将此局限记录到工具文档；或升级工具支持 method 形式

### 问题5: json.dumps 中文字符被 unicode 转义

**根本原因**: `json.dumps` 默认 `ensure_ascii=True`，中文字符变为 `\uXXXX`，导致字符串匹配失败
**解决方案**: 所有 `json.dumps` 调用统一加 `ensure_ascii=False`
**经验教训**: Rick 工具链中所有 JSON 输出应统一用 `ensure_ascii=False`；这是全局工程约定

### 问题6: mock self-test stdout 污染

**根本原因**: `mock_agent.py` 的 `run_self_test()` 中 `scenario_doing_v2_success` 将 NDJSON 输出到 stdout，污染 self-test 框架的输出判断
**解决方案**: self-test 开始时重定向 `sys.stdout` 到 `/dev/null`，结束时恢复
**经验教训**: self-test 框架应隔离被测函数的 stdout；或被测函数接受 output 参数而非直接 print

### 问题7: build_and_get_rick_bin.py 输出 JSON vs 纯路径

**根本原因**: 脚本输出 `{"pass": true, "bin_path": "..."}` JSON，但 task5.py 期望纯路径字符串
**解决方案**: task5.py 中先 `json.loads()` 提取 `bin_path`，失败则 fallback
**经验教训**: 共享脚本的输出格式应在 SPEC 中明确规定；涉及跨 task 调用的工具需要接口文档

## 技术总结

### 关键技术决策

- **DIP 组合根模式**: `doing.go` 是唯一知道 `claudecode` 包的地方；runner/executor/actpath 依赖 `agent` 接口——这是 testability 的关键，让单元测试无需真实 Claude 即可验证全流程
- **embed.FS vs 独立 string 变量**: 现有模板用 `//go:embed file` 绑定 string，新增目录嵌入必须用 `embed.FS`；两种方式可在同一文件共存，`_ "embed"` 改为 `"embed"` 暴露包
- **NDJSON 双写模式**: `bufio.Scanner` 逐行读取，每行先写 `raw_session.log`（追加，含非 JSON 行），再解析——这保证了原始数据完整性，解析失败不丢数据
- **RED 验证 maxRetries=2**: 意外 pass 最多重试 2 次后继续执行（记录 warn），避免无限循环阻塞 CI
- **variadic TestGenContext**: 让 `buildTestGenerationPromptFile` 的新参数可选，零修改现有测试——Go variadic 改造保持接口向后兼容的标准模式

### 知识沉淀清单

- [x] wiki/act_path_mechanism.md - act-path 生成机制与 DIP 全链路
- [x] wiki/dream_command.md - dream 命令工作原理与 SOP
- [x] wiki/core_skills_injection.md - core-skills embed + 精准注入机制
- [x] OKR.md - 新增 O4 进化循环目标，更新 KR
- [x] SPEC.md - 新增 act-path/dream/DIP 规范，更新路径约定
