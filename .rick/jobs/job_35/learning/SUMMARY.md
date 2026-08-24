APPROVED: true

# Job job_35 执行总结

## 执行概述

**项目目标**: 依据 RFC「rick 三层架构重构与 spec 信息内核」把 rick 从 Claude Code 时代迁移到 pi agent runtime，落地四层架构（cli → handler → builder → runtime/env/workspace/prompt）、5 模块、env 四职责契约，并完成「spec → 开发计划 → 功能等价实现」的验收闭环。

**实际完成**: 12 个 task 全部 success、0 次执行器级重试，commit 从 d3cbb3d6 推进到 a080458f。核心成果：
- task1/task2 落地 spec 规范与 rick 第一份 spec（四层架构 + 5 模块 + env 四职责）
- task3~task8 完成四层重构：env 模块、runtime 收口、builder 三件（注入路径而非内容）、handler 调度聚合、cli 变薄、doing 调度与门禁下沉 pi、删除全部冗余 Go 包
- task9~task11 完成 pi 生态对接：注册 think/research/exporter 自定义 agent、pibuilder 单文件内聚、243 处自然语言触发词迁移为 pi 显式触发语法
- task12 完成三个 O 端到端验收 + README/wiki 文档同步

**整体评价**: ⭐⭐⭐⭐⭐ (5 星)。12 任务零重试、门禁全程通过，是「先写 spec 契约再逐层落地」方法论的完整成功案例。

## 关键成就

1. **spec 信息内核落地**: 建立「spec 四要素（模块边界/职责/接口契约/验收标准）→ 开发计划 → 功能等价实现」的验收标准，rick 从此有了可复用的结构化工程契约（`.rick/domain/spec.md` + `.rick/domain/rick-spec.md`）。
2. **四层架构重构一次成型**: 通过「复制包 → sed 改名 → go build 找断点 → 删旧包」的迁移循环，piagent→runtime、能力下沉 env/handler/builder，最终删除 executor/parser/git/agent/actpath 六个冗余包。
3. **doing 门禁下沉 pi**: 用 `rick-gates/helper.py` 确定性脚本替代 Go 侧 doing_check，runtime 解析到 `agent_settled` 后直接校验 tasks.json（可解析/无 zombie/success 有 commit_hash）。
4. **自然语言触发词 → pi 显式语法**: 243 处「派发 subagent/Main Agent/子 Agent」迁移为 `workflowScript + runs.run/runs.all + agent:'name'`，模板真正可被 pi 运行时执行。

## 问题与教训

### 问题1: `git add bin/rick` 静默失败（bin/ 在 .gitignore）

**根本原因**: `bin/` 在项目 `.gitignore` 中，`git add bin/rick` 无报错但静默 no-op，agent 误以为二进制已提交，导致后续用旧二进制跑 check。

**解决方案**: 每次重建后 `git add -f bin/rick`（force），并 `git status --short bin/` 验证确实暂存。

**经验教训**: 已沉淀为 domain/bugs.md + build.md 已知问题，以及 go_package_migration_skill 的陷阱条目。job_35 中 task7/8/10/11 反复踩中，是本 job 最高频的可预防错误。

### 问题2: tasks.json `updated_at` 缺时区导致 Go time.Parse 失败

**根本原因**: 手工编辑 tasks.json 时 `updated_at` 漏 `+08:00` 后缀，Go 按 RFC3339 解析失败，门禁报 `tasks.json not found or invalid`。

**解决方案**: `updated_at` 必须写完整 RFC3339 时区；mark_task_success.py 已用 `timezone(timedelta(hours=8))` 自动修复。

**经验教训**: 已沉淀为 domain/bugs.md 已知问题；时间戳类 JSON 字段一律走辅助脚本而非手工编辑。

### 问题3: Edit oldText 不匹配（批量改 import 块反复失败）

**根本原因**: 凭记忆拼 Go import 块作为 oldText，缩进/行序不符导致 Edit 失败，浪费多轮。

**解决方案**: 先 `Read` 精确行范围，从输出复制目标行作 oldText（global_ref_sync_skill 第 1.5 步）。

**经验教训**: 已由既有 global_ref_sync_skill 覆盖，本次在大规模 import 迁移中再次验证其价值。

## 知识沉淀清单

- [x] skills/go_package_migration_skill/skill.md - Go 包迁移/删除重构（复制→sed 改名→build 找断点→删旧→验证）
- [x] skills/pi_orchestration_syntax_skill/skill.md - pi 子代理编排显式触发语法（自然语言 → workflowScript/runs.run/runs.all/agent）
- [x] loops/go-refactor-migration-loop.md - Go 大重构迁移循环（build+vet+test 收敛到绿）
- [x] domain/bugs.md - 追加 2 个已知问题（bin/rick 静默失败、tasks.json 时区）
- [x] domain/build.md - 新建构建/测试/门禁命令事实
- [x] domain/env.md - 追加 rick 自定义 agent 落盘事实
