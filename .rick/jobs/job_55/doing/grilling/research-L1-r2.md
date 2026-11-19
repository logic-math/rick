# research-L1-r2 简报：L4 事实回流——human 裁决后的根因源码确认

## human 裁决（2026-11-19）

1. 访问通道：sshpass 沙箱直连可用（本次按裁决不需要服务器操作）
2. 现场：报错 workspace 实为 **/workdir/hpc/project/jiqi_mtfi_opt**（项目工作区，非 build_rick 本身）；**已被其他 agent 修复**（从 infer-rsi/.rick 复制 9 件标准资产到该工作区）——现场已消，用户正在使用
3. 本 job 范围：**源码确认根因 + bugfix**（「是不是安装的时候没有把该复制的文件复制到位」）
4. 修复后**不需要发版/重启服务器**（服务器侧已临时修复）
5. **不需要服务器验收**，源码级验证即可

## 源码确认：安装（工作区创建/注册）时不复制资产的缺口 ✅

**`internal/web/routes.go:356-376` `createWorkspaceDirs`**（web UI「创建工作区」POST /api/workspaces/create 调用）：

```go
func createWorkspaceDirs(path string) (string, error) {
	rickDir := filepath.Join(path, workspace.RickDirName)
	dirs := []string{rickDir, loops/, skills/, domain/, jobs/, dream/, draft/}  // 只 MkdirAll 六个空目录
	...
}
```

- **只建空目录，零资产复制** → 新建工作区 `.rick/loops/`、`.rick/skills/` 为空
- `handleAddWorkspace`（POST /api/workspaces，添加已有路径）只校验 `.rick` 存在，同样**不补任何资产**（裸 .rick 工作区直接注册）
- → 用户在新建/注册的工作区上创建 super session → `sessions.go:2276+` 前置检查 `<ws>/.rick/loops/super-loop.md` 缺失 → **400 报错「super session requires .rick/loops/super-loop.md（super 中控 loop，缺失…）」**＝用户报的「找不到中控执行的 super loop 文件」

**结论**：human 假设完全成立——「安装（创建工作区）时该复制的资产没有复制到位」。super 9 件资产只存在于 rick 树（部署 clone），工作区创建/注册流程从不分发；job_51（MT_INFER）同类问题第一次发生，遗留「无自动分发机制」，本次是第二次发生。

## 修复面源码事实（Layer 2 备料）

- **报错现场**：`internal/web/sessions.go` SessionTypeSuper 前置检查（super-loop.md + gen_env_skill/skill.md 两件硬检查；其余 7 件 super_* skill 为运行时需要）
- **资产源可解析**：`internal/super/supervisor.go:507 resolveRepoRoot()`（cwd 上溯 + os.Executable/EvalSymlinks 回退）可定位 rick 树（含 `.rick/` 9 件资产）；internal/web 已 import internal/super（`superpkg.BuildSuperPrompt`）——包依赖方向无障碍
- **9 件资产清单**（见 research-L1.md）：super-loop.md + gen_env_skill + 7 件 super_* skill，各为单文件
- infer-rsi dev_rick@6a8f906 = 本地 main@9de3618 全量镜像 → 本地修复后按 sync 模式推送即可（本次裁决不发版，但源码修复应落两个仓库）
