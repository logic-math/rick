# Job OKR: 强化 Check 机制的正确性与强制集成，提升 Doing 失败反馈质量

## 目标 (Objective)
让 plan/doing/learning 三个阶段的 check 工具准确反映产出格式要求，并强制集成到各阶段 Agent 提示词中；同时确保 doing 重试循环中测试失败信息完整传递给下一轮 Agent。

## 关键结果 (Key Results)
- KR1: plan_check 新增检查 OKR.md 存在性；doing_check 增强 debug.md 内容检查；learning_check 增强 SUMMARY.md 内容检查（消除现有 check 与实际产出格式的不一致）
- KR2: plan.md 模板新增强制 plan_check 步骤；doing.md 模板新增强制 doing_check 步骤；learning.md 模板 Step 3 措辞强化为"必须通过才能继续"
- KR3: plan_prompt.go 和 doing_prompt.go 正确注入 rick_bin_path 和 job_id 变量，使模板中的 check 命令可被正确替换
- KR4: doing 重试循环将完整的 test 脚本输出（含 stderr/traceback）注入给下一轮 doing agent，移除 500 字符硬截断
