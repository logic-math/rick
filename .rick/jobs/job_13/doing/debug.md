## task1: 新建三个 sub agent 模板文件（RFC 规范内容）

**分析过程 (Analysis)**:
- 读取 task1.md，内容已包含三个文件的完整 RFC 内容
- 目标目录 `internal/prompt/templates/` 已存在（含 doing.md、human_loop.md 等）
- 三个文件均为纯静态内容，不含 Go 模板占位符（除 human_loop_express.md 末尾保留 `{{rfc_dir}}` 一处，该占位符在 express 模板中作为文档说明使用，符合 RFC 规范）

**实现步骤 (Implementation)**:
1. 创建 `internal/prompt/templates/human_loop_think.md`（苏格拉底追问者，含 SENSE 五步框架）
2. 创建 `internal/prompt/templates/human_loop_learn.md`（调研者，含事实性断言触发逻辑）
3. 创建 `internal/prompt/templates/human_loop_express.md`（书记员，含固定文档结构）

**遇到的问题 (Issues)**:
- 无

**验证结果 (Verification)**:
- 测试命令：`ls internal/prompt/templates/human_loop_*.md && grep -q "如果这个成立其实假设了" internal/prompt/templates/human_loop_think.md && echo "think PASS" && grep -q "事实性的断言" internal/prompt/templates/human_loop_learn.md && echo "learn PASS" && grep -q "澄清问题（Subject）" internal/prompt/templates/human_loop_express.md && echo "express PASS"`
- 测试输出：
  ```
  internal/prompt/templates/human_loop_express.md
  internal/prompt/templates/human_loop_learn.md
  internal/prompt/templates/human_loop_think.md
  think PASS
  learn PASS
  express PASS
  ```
- 结论：✅ 通过

## task2: 重写 human_loop.md 主控模板（注入 sub agent 路径）

**分析过程 (Analysis)**:
- 当前 human_loop.md 仅有 topic/rfc_dir 两个占位符，并直接调用 `/sense-human-loop` 斜杠命令
- 需要新增三个路径占位符 `{{think_agent_path}}`、`{{learn_agent_path}}`、`{{express_agent_path}}`
- 采用渐进式加载：主控模板只写路径，AI 在执行时自行读取文件内容
- 不内联 sub agent 内容，不调用任何斜杠命令

**实现步骤 (Implementation)**:
1. 重写 `internal/prompt/templates/human_loop.md`，包含五个占位符
2. 新增 L1/L2/L3 复杂度判断逻辑
3. 用路径引用替代斜杠命令，明确每阶段对应的 sub agent 文件

**遇到的问题 (Issues)**:
- 无

**验证结果 (Verification)**:
- 测试命令：`grep -q "{{think_agent_path}}" internal/prompt/templates/human_loop.md && grep -q "{{learn_agent_path}}" internal/prompt/templates/human_loop.md && grep -q "{{express_agent_path}}" internal/prompt/templates/human_loop.md && echo "paths PASS" && ! grep -qE "/sense-human-loop|/human-loop" internal/prompt/templates/human_loop.md && echo "no-slash PASS" && grep -q "Level 1" internal/prompt/templates/human_loop.md && echo "complexity PASS"`
- 测试输出：
  ```
  paths PASS
  no-slash PASS
  complexity PASS
  ```
- 结论：✅ 通过
