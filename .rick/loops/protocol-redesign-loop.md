---
name: protocol-redesign-loop
description: AI agent 协议重构方法论；当需要重构多阶段协议(如 human-loop/doing/learning)时触发
trigger: "当需要重构 AI agent 的多阶段协议(如人机协作流程、任务执行流程),涉及阶段合并/拆分/反向回流/批判层重设计时触发"
---

# Loop: 协议重构循环

## 目标（Goal）

将现有协议从一种设计形态重构为另一种(如 v2 7 步线性 → v3 5 阶段非线性),agent 自己可判断是否达成。达成标准:
- 新协议文档落地(模板 + skill 同步)
- 配置项写入 config 结构体 + 默认值
- 测试断言新关键字 + 禁止旧概念
- 全量测试 PASS + easy_check PASS + build 成功

## 上下文管理（Context Management）

**输入**(从前序判断获取):
- 现有协议的痛点清单(冗余/缺失/僵化)
- 新设计的核心概念(如反向回流/批判门禁嵌入/系统论描述符)
- RFC 或草稿文档(如有)

**输出**:
- `.rick/skills/{name}_skill/skill.md` — 新 skill(方法论)
- `.rick/loops/{name}-loop.md` — 新 loop(本文件,流程模板)
- `.rick/domain/architecture.md` — 更新架构描述
- `internal/prompt/templates/*.md` — 协议模板
- `internal/config/config.go` + `loader.go` — 配置项
- 测试文件 — 断言更新

## 可调用工具（Tool Access）

- **Read/Write/Edit**:读写模板/skill/loop/config 文件
- **Bash**:运行 `./scripts/build.sh` + `go test ./...` + `./bin/rick tools easy_check`/`learning_check`/`dream_check`
- **Grep/Glob**:搜索现有代码模式
- **Agent**:派发 Plan agent 设计方案(可选)

## 工作流

```
[Step 1] 分析现有协议痛点
  - 列出当前阶段数 + 每阶段职责
  - 识别冗余(如 S1/S2 拆分过细)、缺失(如反向回流)、僵化(如批判层独立)
  - 输出:痛点清单

[Step 2] 设计新阶段框架
  - 合并相关阶段(S1+S2→S, E1+E2→E)
  - 拆分复杂阶段(N→N1+N2)
  - 引入反向回流机制(后续可重启前序)
  - 嵌入批判门禁(每阶段 human 实质性回答后触发)
  - 输出:阶段表 + 流程图

[Step 3] 引入系统论描述符
  - 5 要素:node/input/output/inner/edge
  - 用于 N1 阶段(矛盾生成),替代模糊的概念地图
  - 推演系统稳态迁移:当前 A → 目标 B 所需控制手段
  - 输出:描述符表 + 示例

[Step 4] 设计启发性提问
  - 每阶段核心问题改为启发性表述
  - 禁止确认性句式("是否成立?""对吗?")
  - 每假设 3 启发性问题(信念/前提/反例)
  - 输出:各阶段追问表 + 3 问模板

[Step 5] 配置化所有阈值
  - 重试次数 / 反向回流上限 / top-N / min_assumptions / 权重
  - 写入 config 结构体 + GetDefaultConfig
  - 输出:配置项清单 + 默认值

[Step 6] 验证 + 落地
  - 测试断言新关键字 + 禁止旧概念
  - build + easy_check + try-run
  - 输出:验证报告 + commit

[Step 7] 反思(可选)
  - 评估新协议是否达成"信息量变多 + 不替 human 判断 + 启发性提问"目标
  - 列举仍可改进点
  - 输出:改进方向清单
```

每个 Step 引用的 skill:
- Step 1: 无(直接分析)
- Step 2: `multi_phase_protocol_skill/skill.md`(5 阶段框架设计原则)
- Step 3: `multi_phase_protocol_skill/skill.md`(系统论描述符)
- Step 4: `multi_phase_protocol_skill/skill.md`(启发性提问设计)
- Step 5: 无(直接改 config 结构体)
- Step 6: `verify_go_changes_skill/skill.md`(Go 修改验证)
- Step 7: 无(反思)

## 产出评估（Output Evaluation）

| 检查项 | 判断方法 |
|--------|----------|
| 阶段数精简 | v2 7 步 → v3 5 阶段(或类似) |
| 反向回流机制落地 | 配置项 + 上限 |
| 批判门禁嵌入各阶段 | 非独立步骤 |
| 系统论描述符 5 要素落地 | node/input/output/inner/edge |
| 启发性提问替代确认性 | 各阶段+每假设 3 问 |
| 所有阈值配置化 | 5+ 配置项 |
| 全量测试 PASS | `go test ./...` |
| easy_check PASS | `./bin/rick tools easy_check` |

## 停止标准（Termination Condition）

**完成退出**:Step 6 验证全部通过(测试 PASS + easy_check PASS + build 成功),且协议文档落地。

**优雅退出**:
- 人类明确要求停止
- 反思(Step 7)发现仍需大改 → 重启 Step 2(携带修改意见)

## ⚠️ 不可变约束（硬性,违反即终止 Loop）

1. **避免只改结构不改提问**:阶段合并是表层,提问启发性是深层
2. **避免批判层独立化**:批判门禁嵌入各阶段,不作为独立步骤
3. **避免阈值硬编码**:所有阈值写入 config,不写死在模板
4. **避免测试断言过细**:断言核心关键字,不强约束具体表述(允许 LLM 自由发挥)
5. **避免一次性大改**:分多个 commit(初始重构→强化约束→回退优化→release),每个 commit 可独立 revert

## 来源

- 首次落地:job_28(2026-08-04,human-loop v2.10.9 → v2.11.9 重构)
- 11 commits:初始四文件架构 → research v2 → think v2 → sense v3 → think v3.1 → release
