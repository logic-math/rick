# Skill: superopt-json-guard（pi hook 扩展——superopt agent 输出确定性）

## 触发场景

当部署/修改 superopt agent 的 pi hook（JSON 输出保障 / adjust_space schema 结构检 / gates/ 禁读拦截）时加载本 skill。

## 行为（index.ts）

1. **JSON 输出保障**（job_47/task57 原有）：`SUPEROPT_JSON_GUARD=1` 激活；agent_end 时最终回复缺 JSON 块 → `sendUserMessage(deliverAs:"followUp")` 续跑修复轮（上限 `MAX_JSON_REPAIR=2`）。
2. **adjust_space schema 结构检**（job_56 task3）：`SUPEROPT_ROLE=adjust_space` 时，JSON 块之上再校验 proposals schema——`proposals` 数组必填；每项 `type`/`evidence|rationale` 必填；`soft_weight` 必含 `target_cell`/`action`；`new_cell` 必含 `name`/`method`/`orthogonality_rationale`；`modify_cell` 必含 `target_cell`/`method`。违规 → followUp 修复（复用同一上限）。
3. **gates/ 禁读拦截**（job_56 task3）：`SUPEROPT_ROLE ∈ {builder, coding, debug}` 时，`read`/`bash` 工具调用命中 `env/gates/` 路径 → `{block: true, reason}`（tool_call 事件官方拦截通道）。`adjust_*` 角色放行（step7 验证器反思需要读写）。

## 环境变量（PiAgentClient 派发时注入）

- `SUPEROPT_JSON_GUARD=1`：激活本 hook（非 superopt pi 会话完全不注册）
- `SUPEROPT_ROLE=<role>`：当前角色（schema 检与禁读白名单的判定依据）

## 部署

`.rick/skills/superopt-json-guard/` → `<agentDir>/extensions/superopt-json-guard/`（internal/env/customizations.go 自动复制；手动部署时直接 cp index.ts）。

## 验证步骤（手动——node 侧无测试基建）

```bash
# 1. 语法检查（tsc noEmit 或 deno check——agent runtime 内）
cd ~/.rick/pi/agent && npx tsc --noEmit extensions/superopt-json-guard/index.ts 2>&1 | head -5
# 2. 行为验证（fake pi 场景）：
#    a. gates 拦截：SUPEROPT_JSON_GUARD=1 SUPEROPT_ROLE=builder pi --mode json
#       在会话中 read env/gates/gate_check.py → 期望 block 消息
#    b. adjust_space schema：SUPEROPT_ROLE=adjust_space 会话输出缺字段
#       proposals JSON → 期望 followUp 修复提示
#    c. adjust 放行：SUPEROPT_ROLE=adjust_fact read env/gates/... → 正常执行
```
