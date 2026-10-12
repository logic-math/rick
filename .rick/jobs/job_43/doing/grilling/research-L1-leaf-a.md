# research-L1-leaf-a：rick CLI 功能面现状（代码原文级）

## 1. 命令与 flags 全集（root.go:67-75 注册 9 命令）

| 命令 | 专有 flags | 全局 flag 依赖 |
|---|---|---|
| plan [requirement] | 无（requirement 为位置参数） | --job/--resume/--dry-run |
| doing [job_id] | --job、--easy、--ctx（doing.go:107-109） | --resume/--dry-run |
| learning [job_id] | --job（learning.go:60） | --dry-run |
| easy | -r/--requirement、--ctx、--resume（easy.go:51-53） | --dry-run |
| dream | --job_num(默认5)、-p/--background（dream.go:31-32） | --dry-run |
| ctrl | 无（必传全局 --job） | --job/--dry-run |
| human-loop [topic] | 无 | --resume loop_N/--dry-run |
| web | --port(6137)/--listen/--token/--state-dir/--daemon/--log-file（web.go:148-155）；子命令 customize(--yes)/reset | 无 dry-run |
| tools | 见下 | — |

全局持久 flags（root.go:57-60）：-v/--verbose、--dry-run、--job、--resume；-V/--version。

**tools 子命令（tools.go:36-44，共 9 个）**：init-pi、update-pi [target]、theme [name]、learning_check <job_id>、dream_check、loops_check（--dir/--json）、dev-web {init,build,up,restart,status,down}、release（--yes/--rollback/--dry-run/--detach/--prod-repo/--dev-tree/--prod-home/--state-dir/--start-script/--port/--keep/--no-gates，tools_release.go:148-159）、rsi_check（--job/--prod-url/--init/--json，tools_rsi_check.go:548-551）。

## 2. dry-run 覆盖情况

plan/doing/easy/dream/ctrl/human-loop/learning 全部支持（各文件均有 GetDryRun 分支，doing.go:39/62/73 等）；web 无（服务型，合理）；tools release 有 --dry-run；check 类天然只读。

## 3. --resume 覆盖情况

支持：plan（job）、doing（job，doing.go:59-70）、easy（job，easy.go:29-34）、human-loop（loop_N，human_loop.go:25-32）。不支持：learning、dream、ctrl（learning.go 无 GetResume 调用）。

## 4. 体验缺口清单（附证据）

1. **无 job 列表/状态查看命令**：root.go:67-75 无 jobs/status/list 类命令；用户看进度只能 `ls .rick/jobs` + 手动 cat tasks.json。
2. **无 job 清理/归档命令**：同上；.rick/jobs 已积至 job_43 + job_N 模板，无 CLI 归档入口。
3. **domain 文档与代码漂移**：commands.md「Tools 子命令体系」表列 plan_check/doing_check，实际未注册（tools.go:36-44 无）；architecture.md 列 internal/executor、internal/agent、internal/actpath、callcli、parser、git 包，实际 internal/ 仅 9 包（builder/cmd/config/env/handler/prompt/runtime/web/workspace）——文档失真影响 agent 依据 domain 做决策。
4. **dream 无法定向单 job**：dream.go:31-32 仅 --job_num/--background，无 --job 过滤。
5. **easy 双入口冗余**：`rick easy`（easy.go）与 `rick doing --easy`（doing.go:108）同为 handler.Easy 入口，帮助文案/维护成本翻倍。
6. **learning 无 --resume**：learning 是长交互会话但中断后无法恢复（plan/doing/easy/human-loop 均可）。

## 5. 结论

CLI 主体命令（9 个）+ tools 子命令（9 个）功能面完整、dry-run 覆盖良好；主要缺口集中在「job 生命周期管理（列表/状态/归档）」「文档-代码同步」「dream 定向」三处，均为功能级可改动（新增 cobra 命令 + 读 .rick/jobs 目录，不触碰架构）。
