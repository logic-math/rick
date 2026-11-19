用户 bug fix 解决 10.99.19.123 这台服务里部署了 rick 的 web 服务，但是用户使用 super 模式的时候，报错，说找不到中控执行的 super loop 文件。
你来排查修复一下，它使用的是 ssh://git@git.sankuai.com/wm_ai/infer-rsi.git 
内网的 super 版本的最新代码
---

## Grilling 澄清结论（2026-11-19，详见 grilling/design-tree.md + research-L1.md / research-L1-r2.md / research-L2.md）

**根因（源码确认）**：web 工作区创建/注册（routes.go createWorkspaceDirs/handleAddWorkspace）只建空 .rick 目录、零资产复制；super 9 件资产只存在于 rick 树。用户在项目工作区（服务器实测为 /workdir/hpc/project/jiqi_mtfi_opt）创建 super session → sessions.go 前置检查 `<ws>/.rick/loops/super-loop.md` 缺失 → 400「找不到中控执行的 super loop 文件」。job_51（MT_INFER）同类第一次，遗留「无自动分发机制」，本次第二次。

**human 全部裁决**：
1. 服务器访问：sshpass 沙箱直连可用（本次不需要）
2. 现场：已被其他 agent 修复（9 件资产已复制到位），用户正在使用——**不碰现网用户环境**
3. 范围：源码确认根因 + bugfix（super session 创建时从 rick 树自动分发缺失资产）
4. 修复后**不发版、不重启服务器**
5. **不需要服务器验收**，源码级验证（单测）即可
6. hook 位置：super session 创建时 lazy 部署（不在工作区注册时提前部署）
7. 复制策略：只补缺失、绝不覆盖已存在文件
8. 静默补齐 + std log 服务端日志，API 契约不变、前端零改动
9. 落点：本地仓库修复+单测 → sync infer-rsi dev_rick（targeted copy + sync commit + push）；不重建 checked-in linux 二进制（遗留）

**修复设计（叶子层，四维度落实）**：
- M1 `internal/super/assets.go`（新建）：`SuperAssetPaths` 9 件单源清单 + 纯函数 `deploySuperAssets(srcRickDir, dstRickDir) ([]string, error)`（只补缺失不覆盖，部分树只部署可部署件）+ `DeploySuperAssets(dstRickDir)`（resolveRepoRoot 定位 rick 树）
- M2 `internal/web/sessions.go` SessionTypeSuper：前置检查前调 `superpkg.DeploySuperAssets(rickDir)`，成功且部署>0 → `log.Printf`；失败/树无资产 → 落回原显式报错（行为不变）
- M3 测试：`internal/super/assets_test.go`（新建）+ `internal/web/sessions_test.go`（新增裸工作区自动部署测试）
- 发布：go build/test 绿 → commit → infer-rsi dev_rick targeted sync push
