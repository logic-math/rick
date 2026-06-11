# Job OKR: 实现 RFC-debugging，建立三阶段科学调试体系

## 目标 (Objective)
将 Rick 的调试能力从"盲目重试"升级为基于状态机理论的科学调试——三阶段 SOP（源码推理→增量调试→科学实验）+ review debug agent + 运行时工具指引，消除调试上下文的恶性循环。

## 关键结果 (Key Results)
- KR1: `internal/prompt/templates/skills/debug_skill.md` 存在，包含准备阶段、三阶段 SOP（含回滚约束、循环上限）、review debug agent 协议（两个触发点）、运行时观察工具指引、debug/ 目录文件格式
- KR2: `super-debugging-zh.md` 已删除；`doing.md` 和 `plan.md` 模板中所有 `super_debugging*` 引用替换为 `debug_skill_path`；doing.md 的 debug{N} 调试记录格式替换为 debug_skill 加载指令
- KR3: `doing_prompt.go`、`plan_prompt.go`、`easy_prompt.go` 的 WriteSkillFile/SetVariable 调用全部从 "super-debugging-zh"/"super_debugging_path"/"super_debugging_skill_path" 切换到 "debug_skill"/"debug_skill_path"；`go test ./internal/prompt/...` 全部通过
- KR4: `internal/executor/runner.go` 的重试上下文加载逻辑从仅读 `debug.md` 扩展为同时扫描 `debug/` 目录下所有 `bug*.md` 文件；`go test ./internal/executor/...` 全部通过
