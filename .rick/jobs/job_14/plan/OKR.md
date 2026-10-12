# Job OKR: Rick v2.0 核心升级 —— 建立 act-path 进化循环

## 目标 (Objective)

将 Rick 从 v1 的"执行框架"升级为具备"进化能力"的三层正交 AI Coding 控制框架。核心是通过程序性 JSONL 解析建立可靠的 act-path 负反馈机制，使 learning/dream 层能够从真实行为轨迹中提取优化信号，而非依赖 LLM 自觉记录。

## 关键结果 (Key Results)

- KR1: `rick doing` 执行后自动生成 `act-path-{taskID}.md`，包含工具调用轨迹、报错次数、执行时长，`go test ./internal/actpath/...` 全部通过
- KR2: `rick learning` 升级为六步 SOP，`--dry-run` 输出包含 `act-path.md` 加载和 `run_log` 写入指令
- KR3: `rick dream` 命令可运行，`--dry-run` 正常输出完整提示词，`--help` 显示正确说明
- KR4: `rick plan --dry-run` 输出包含 6 subagent 评审 SOP（RFC 一致性/SPEC 合规/skills 利用/代码模拟/测试用例/端到端）
- KR5: `rick doing --dry-run` 输出包含 core-skills 注入内容（sense/tdd/debug 等）
