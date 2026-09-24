# Leaf C：loop/skill 与 tools 校验体系现状（job_43 L1）

## 1. Loops（.rick/loops/，6 活动+1 弃用）
rick-rsi（自进化主线，改 rick 必走）、agent-runtime-bootstrap、go-refactor-migration、protocol-redesign、readme-wiki-sync、tdd-red-green-refactor、deprecated/do-check-mark-success。
发现机制：LoadLoopsContext 扫 *.md 取 frontmatter（prompt/context_helpers.go:12），注入 plan/easy/dream/pibuilder ~15 处，覆盖广。

## 2. Skills（.rick/skills/，19 目录+rick-gates）
流程计划（zero_retry/dag_decomposition/multi_phase_protocol）、校验门禁（check_mechanism/mark_task_success/verify_go_changes）、同步注入（global_ref_sync/template_injection/command_registration_verification）、测试（test_script/fake_binary/subprocess_env）、pi 四件套、Go（go_package_migration）、反馈（failure_feedback）。rick-gates 为 pi 扩展（doing hook level_complete/pipeline_gate，env/customizations.go 部署），非 skill 目录。

## 3. Tools 子命令（tools.go:36-44，9 个）
init-pi、update-pi [target]、theme [name]、learning_check <job_id>（--auto-fix）、dream_check、loops_check（--dir --json）、dev-web {init,build,up,restart,status,down}、release（--dry-run/--rollback/--merge-source/--detach）、rsi_check（6 机器契约，--init/--json）——均成熟。
**已移除**：plan_check/doing_check 不在注册表，domain/commands.md、architecture.md 仍记载（漂移）。

## 4. 缺口（附证据）
- **G1（高）skills 格式校验空转**：checkMarkdownDir 跳过 IsDir（tools_loops_skills_check.go:50），skills 实为 `_skill/` 目录 → 三个 check 从不校验 skill.md；且要求英文章节（:23-28）与实际中文三段式冲突（verify_go_changes_skill/skill.md）
- **G2（高）skill 生命周期无机器化**：README 称"3 次 dream 未引用→deprecated/"，dream.go 无实现、无 deprecated/ 目录
- **G3（中）skills README 索引漂移**：索引 9 个 vs 实际 19 目录
- **G4（中）gen-loop/gen-skill 无 CLI 命令**：仅 easy prompt 注入模板（easy_prompt.go:34-49）
- **G5（低）README.md 无 frontmatter 每次注入打 warn**（context_helpers.go:38）
- **G6（低）architecture.md 模块清单漂移**：internal/ 实为 builder/cmd/config/env/handler/prompt/runtime/web/workspace，无 executor/agent/actpath
