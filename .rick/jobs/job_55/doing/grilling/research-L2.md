# research-L2 简报：bugfix 设计层（第 2 层）调研事实

> 第 2 层 = bugfix 设计（M1 资产部署器 / M2 挂钩点 / M3 测试）。本简报记录设计层为把四维度落实到叶子所核实的代码事实（均为本地 grep/read 轻量核实，无外部调研需求）。

## M1 资产部署器——internal/super 现有事实

- **resolveRepoRoot 可复用**（internal/super/supervisor.go:507）：cwd 逐级上溯找 `super/superopt/__main__.py` → 失败后 `os.Executable + EvalSymlinks` 穿二进制 symlink 链回退。job_51 修复，dev_rick 已含
- **测试模式**（test_exe_fallback_test.go / supervisor_test.go:561）：resolveRepoRoot 用 subprocess + 假 repo（含 `super/superopt/__main__.py` marker）测 cwd 路径与 exe 回退；纯函数部分可直接 t.TempDir 测，无需注入 hook
- **包依赖方向无障碍**：internal/web 已 import internal/super（sessions.go 用 `superpkg.BuildSuperPrompt`）——super 包新增导出函数即可被 web 调用
- **资产清单单源**：9 件均为单文件（super-loop.md + gen_env_skill/skill.md + 7 件 super_*/skill.md，见 research-L1.md）；应定义 `SuperAssetPaths` 单源清单供部署器与测试共用

## M2 挂钩点——internal/web 现有事实

- **报错现场**（sessions.go SessionTypeSuper，~2266-2283）：前置检查循环 2 件硬检查（super-loop.md / gen_env_skill），缺失返回 400 + 契约化错误串——**该循环与 API 契约字符串保持不变**（human 8a：静默补齐，前端零改动）
- **web 包零日志惯例**：grep 全包无 log/slog import → 服务端日志用 std `log.Printf`（最小实现，stdout 可见）
- **工作区创建/注册流程**（routes.go:356 createWorkspaceDirs / handleAddWorkspace）：只 MkdirAll 六个空目录、零资产复制——这是「安装时该复制未复制」缺口的确切位置；但按 human 6a 裁决**不在注册时提前部署**（侵入大），挂在 super session 创建时 lazy 部署

## M3 测试——现有模式

- **web 层既有 super 会话测试**（sessions_test.go:1871+）：`seedSuperAssets` 种子 2 件资产 + `createSessionViaHTTP`（fake pi env）；seeded 场景在「只补缺失」语义下零改动（已存在→跳过）
- **新增自动部署测试可行**：裸 .rick 工作区（不 seed）→ web 测试进程 cwd=internal/web → resolveRepoRoot 上溯命中真实 repo 树（本地仓库 .rick 9 件齐全，已核实）→ 自动部署后创建成功；断言 `<ws>/.rick/loops/super-loop.md` 落盘
- **纯函数测试**：`deploySuperAssets(srcRickDir, dstRickDir)` 显式传参直接测（9 件部署/已有不覆盖/部分树只部署可部署件）；wrapper `DeploySuperAssets` 用 `t.Chdir` 假树（含 superopt marker）测 cwd 解析

## infer-rsi sync 事实（M2 发布流）

- 历史模式：整树 sync commit（"sync: rick main@<sha>"）；本次按 human 9a targeted copy 4 文件 + sync commit + push dev_rick
- **不重建 checked-in linux 二进制**（human 4：不发版不重启；服务器侧已临时修复在用）——二进制更新列为遗留事项
