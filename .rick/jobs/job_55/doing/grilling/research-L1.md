# research-L1 简报：job_55 super loop 文件缺失——代码侧与仓库侧事实

## 报错来源（代码定位，置信度：高）

`internal/web/sessions.go`（SessionTypeSuper 分支，本地 main@9de3618 ≈ infer-rsi dev_rick@6a8f906 同一文件）：

```go
for _, asset := range []struct{ path, hint string }{
    {filepath.Join(rickDir, "loops", "super-loop.md"),
        "super session requires .rick/loops/super-loop.md（super 中控 loop，缺失：请先按 .rick/loops 规范落盘）"},
    {filepath.Join(rickDir, "skills", "gen_env_skill", "skill.md"),
        "super session requires .rick/skills/gen_env_skill/skill.md（env 建设引导 skill，缺失：请先按三段式规范落盘）"},
} {
    if _, err := os.Stat(asset.path); err != nil {
        return nil, newWebError(http.StatusBadRequest, "invalid_params", "%s：缺失文件 %s", asset.hint, asset.path)
    }
}
```

其中 `rickDir = <workspace>/.rick`（**逐工作区**）。用户描述「找不到中控执行的 super loop 文件」与此完全吻合 → 报错 = 目标 workspace 的 `.rick/loops/super-loop.md`（或 gen_env skill）缺失。

## super 资产 9 件清单（web 创建 + 运行所需，均已在 infer-rsi dev_rick git 中）

1. `.rick/loops/super-loop.md`（web 创建前置检查①）
2. `.rick/skills/gen_env_skill/skill.md`（前置检查②）
3. `.rick/skills/super_cell_management/skill.md`
4. `.rick/skills/super_cost_analysis/skill.md`
5. `.rick/skills/super_debug_protocol/skill.md`
6. `.rick/skills/super_event_autopsy_skill/skill.md`
7. `.rick/skills/super_experiment_analysis/skill.md`
8. `.rick/skills/super_experiment_design/skill.md`
9. `.rick/skills/super_sense/skill.md`

## infer-rsi 仓库事实（已实地 clone 验证）

- `ssh://git@git.sankuai.com/wm_ai/infer-rsi.git` 从本 Mac **可达**（git ls-remote / clone 均成功）
- 分支：`master`（几乎空，仅 README.md）、`dev_allmodel2torch`（原 infer 项目）、`dev_rick`（rick 内网版）
- `dev_rick@6a8f906` = "sync: rick main@9de3618" = **本地仓库当前 HEAD 的全量镜像**（含 job_50-53 全部修复）
- dev_rick 树内 **.rick 资产齐全**（9 件全在 git）、supervisor.go 与本地完全一致（含 job_51 的 resolveRepoRoot exe fallback 修复）
- 仓库根目录 checked-in 一个 **Linux x86-64 预编译二进制 `rick`**（24MB ELF，最后提交于 e07011e；其树的 supervisor.go 与当前一致——服务器部署大概率直接用它，无需现场构建）
- `.rick/super/`（运行时工作区）被 gitignore，不在 git

## workspace 注册机制

- web 工作区注册（registry.go Add）只要求路径含 `.rick` 目录（裸 .rick 即可）——**不含 super 资产**也能注册
- web UI 支持「浏览/创建」工作区 → 新建的 workspace 默认无 super 资产 → 第一次用 super 必撞缺失报错

## 先例：job_51（MT_INFER workspace，同类 bug 已修过一次）

- 根因 = super 资产逐 workspace 但只在 rick 树 → 外部 workspace 首次用 super 必报缺失
- 当时修复 = 手动复制 9 件到 MT_INFER + resolveRepoRoot exe fallback（已在此版代码中）
- **遗留（job_51 learning 明确记录）**：「super 资产分发的自动化……现在外部 workspace 上 super 前仍需手动复制 9 件」——无自动分发机制，本 bug 是该类问题第二次发生

## 服务器 10.99.19.123 连通性（本 Mac 实测）

- ping 通（26ms RTT，ICMP 放行）
- ssh 22 / 2222 / 8080 / 9090 / 6137 TCP 全部超时（防火墙拦截或端口未知）
- 无法直连 → 服务器侧事实（部署形态、workspace 列表、确切报错）尚不可自查，需访问通道

## 不可自行消解的剩余未知（需服务器访问或 human 回答）

- A. 服务器访问方式（ssh 用户/端口/跳板？还是人工代跑命令？）
- B. 报错时选择的 workspace 是哪个（推测 ≠ /workdir/hpc/build_rick，而是 web 里注册的另一个工作区）
- C. 服务器上 web 如何运行（checked-in 二进制？如何拉起？注册了哪些 workspace）
- D. 确切报错全文（确认缺的哪一件）
