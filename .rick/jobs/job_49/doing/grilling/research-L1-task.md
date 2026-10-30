# research-L1 任务简报（job_49 恢复 rick web 生产功能）

你是 job_49 的 L1 代码调研员。仓库：/Users/logic/workdir/AI_CODING/rick（HEAD=02d144d，只读调研，不改任何文件）。

背景：wiki/handover-2026-10-30-web-start-macos.md 交接单已给出根因链（错平台 ELF 投放 + stopProd 漏杀 skipped=[11135] + 回滚无校验 + health 诊断盲区），并给了代码锚点。你的任务：在当前 HEAD 逐条核实/补全下列事实，标注 file:line 证据与置信度，供后续追问与修复设计使用。

## 调研清单（10 条，逐条澄清到高置信度）

1. internal/env/release/release.go：核实这些函数的现状与逻辑——Restart、waitHealthy、Rollback、stopProd、ownedByProd（约 L1276）、prodEnv（约 L865）、findProdPids（如存在）。具体回答：
   a) promote/restart 链路里是否存在任何「目标二进制可执行性」预校验（magic/exec 冒烟）？在哪一步（build/gate/promote/health）失败才会被发现？
   b) Rollback() 是否无条件 current→.last？有无校验？
   c) ownedByProd 精确判断什么（proc.Cwd/proc.Exe/HOME 兜底）？darwin 上取 Cwd/Exe 失败时走什么分支？「skipped=[11135]」是哪段代码产生的（宁可不杀的所有权断言）？
   d) prodEnv() 白名单透传哪些变量？HOME 在不在？对 ownedByProd 的 HOME 兜底意味着什么？
   e) waitHealthy 如何读 build_id（调 /api/health？）？失败信息现在包含什么、缺什么？
2. internal/env/relocate/import.go 的 wrongPlatformBinary()：精确逻辑（检查什么 magic、支持哪些平台、返回什么）？被哪些调用方使用？~/.rick/bin/rick 写入路径为什么没覆盖到（F6）？
3. internal/cmd/web.go：--daemon 的实现（setsid+nohup 细节）；web.lock 单实例逻辑（只认持有者存活？）；重点——「graceful shutdown: context deadline exceeded」这个错误字符串在哪个文件哪行产生？SIGTERM 处理链路是什么（supervisor 先杀 worker 再关 http？超时多少）？为什么 worker exit 143 后 web 会跟着死？
4. internal/cmd/root.go:26 build_id 默认 "dev" 的机制（-X cmd.BuildID 注入）——health 里 build_id="dev" 意味着什么（哪个入口起的进程）？
5. internal/env/release/release_test.go：列出现有测试名（特别是回滚/工具链相关），新增 P0-1/P0-2 用例放同文件是否合适？
6. internal/env/procfs（或等价包）：进程枚举用什么（ps/lsof/syscall）？darwin 上 proc.Cwd/proc.Exe 的实现与已知失败模式（lsof 权限/TCC）？
7. scripts/self-evolve-e2e.sh 与 scripts/rsi-loop-e2e.sh：对 release/promote/build_id 断言了什么？P0 修复后这些 e2e 是否需要适配？
8. 全仓搜索：除了 import.go 的 wrongPlatformBinary，还有没有别的「平台/架构校验」「exec format」「cannot execute」处理？start-web.sh 和 ~/.rick/deploy-web.sh 是 rick 源码生成的吗（搜 "start-web" "deploy-web"）？若是，生成点在哪（修脚本=修源码）？
9. web customize（internal/cmd/web.go customize 子命令）：~/.rick/web/dist 覆盖层机制——覆盖层与内嵌 dist 的优先级、reset 命令行为（对 P1-5 前端对齐的处置选项）。
10. 顺带：~/.rick/config.json 里 web_token 之外还有什么与本任务相关的配置（不要输出 token 值）。

## 产出要求

- 简报落盘：/Users/logic/workdir/AI_CODING/rick/.rick/jobs/job_49/doing/grilling/research-L1.md（write 首块 + bash 分批追加；按上述 10 条编号组织，每条给结论 + file:line 证据 + 置信度；末尾给「交接单锚点核实结论表」：哪些锚点仍准确/哪些漂移）
- 最终回复=回执（路径 + 要点 ≤300 字）
