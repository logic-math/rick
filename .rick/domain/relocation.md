# 工作区搬迁（dump / import）事实

**来源 Job**: job_46（2026-10-15）｜ 功能分支: dev/self-evolve（f4278a6 → 0370422 共 6+2 提交，main 尚未 release）

`rick tools dump` / `rick tools import` 的语义速查与机制事实。操作技能见 `skills/cross_machine_migration_skill/`，迭代控制流见 `loops/workspace-relocation-loop.md`。

## 命令语义速查

```bash
# 源机打包（默认：当前目录 + ~/.rick 状态 + pi runtime/npm + 三平台二进制）
rick tools dump [--workspace <path>]... [-o out.tar.gz]
                [--sessions=none|workspaces|all]   # 默认 workspaces（只带被打包工作区转义目录）
                [--slim]                            # 剔 pi runtime/npm（目标机重跑 init-pi）
                [--goos=linux/amd64,darwin/arm64…]  # 内嵌交叉编译二进制（native 失败=error，他平台=warn）
                [--exclude <pattern>]...            # 工作区根相对模式（目录剪枝/glob；非法模式预检报错）
                [--no-exclude]                      # 关闭默认排除（字节级全量）

# 目标机导入（默认合并：已有文件永不覆盖；--force 先备份整个 state 再覆盖）
rick tools import <archive.tar.gz> [--target <dir>] [--state-dir <dir>] [--force]
# --target 落位规则：单工作区 + 空 target → 直接落 target；多工作区/非空 target → target/<name>（重名 -2 去重）
# state 不传 --state-dir 默认 $HOME/.rick（标准布局，推荐——见 bugs.md config 漂移条目）
```

## 归档布局与排除

- 布局：`manifest.json`（首 entry）+ `workspaces/<name>/` + `state/` + `state/pi-agent/` + `bin/<goos>-<goarch>/rick`
- 默认排除 13 条（三根作用域：workspace/state/pi-agent，各自相对）：平台 ELF（bin/rg）、可再生（web/node_modules、bin、dist.prev）、缓存（.pi、.morty、web-search-cache、config.json.bak.*）、运行时（pid/lock/log）、外部（catpaw-proxy 软链——human 裁决 provider 不归 rick 管）
- 权限洞（异属主 700 目录 / ACL 文件）：`OnUnreadable` 钩子跳过 + manifest 记 `kind=unreadable`（不中断打包，不留空 tar entry）
- 每项排除进 manifest 留痕（path/root/kind/reason）——归档始终可审计

## marker 稳定 id 机制（防会话孤儿化）

- 工作区根放 `.rick-workspace.json`（`{"id": "<32hex>", "created_at": …, "rick_version": …}`）；id 用 crypto/rand 生成，与位置永久解耦
- `internal/web/registry.go` 的 `workspaceID`：**marker 优先 → sha1(path)[:8] 回退**（旧数据兼容）；`Add` 新条目自动建 marker，**幂等 re-add 不改 id**；存量 id 升级是 dump/import 回填的职责（避免亲手制造 ghost id）
- **关键语义**：web.json 条目的 id 是 web.json ↔ sessions.json/archived.json/job-names.json 的**不透明连接键**。精确工作区命中 → id 换 marker；**前缀命中非精确（树内子工作区）→ 保留原 id**（两侧都不改即不断链）
- 实测：11 个子工作区迁移后 id 全保留、21 会话全部可读可 resume

## pi 会话移植（最小改写集）

- 目录名转义规则（pi 0.84.x session-manager）：`--` + cwd 去首 `/` 后 `/\:`→`-` + `--`；**下划线不替换**；无截断/冲突处理（不单射：`/a/b` 与 `/a-b` 同名）
- resume 三方式语义：`--continue`=cwd 转义目录内 mtime 最新（不校验 header.cwd）；`--session <id>`=本地精确→全局扫描兜底（cwd 不同会 fork 确认，非交互死路）；**`--session-id <id>`=仅本地精确匹配、无兜底、找不到=静默新建空会话（历史丢失）**——rick 两条链分别用 `--session`（web）与 `--session-id`（CLI）
- 最小改写集（12 组实验矩阵锁定）：① 目录按新 cwd 重转义改名 ② session.jsonl **首行** header.cwd 重写（消息行逐字节不动；parentSession 可选重写）——缺任一即断链
- `--sessions=all` 的孤儿目录（cwd 未随迁）：原样落位 + import 上报「仅可按 id 全局检索」
- 详见 `skills/pi_session_portability_skill/`

## 已知限制与产品改进候选

| # | 项 | 状态 |
|---|---|---|
| 1 | darwin ps/lsof 进程探测真机未实测（release/dev-web 用；表驱动解析器已覆盖） | ⚠ 待 Mac 实测（环境已迁到 Mac，可直接验） |
| 2 | dump 权限预检（preflight）未做——E2E 三跑两败于权限洞，每次失败在 ~1.2GB 已写入后 | 改进候选（Step 2 建议 1） |
| 3 | release `runGates` 内嵌 go test 不清洗 RICK_* env（shell 侧 env -u 规避） | 改进候选（runGates 构造 Cmd.Env 剔除） |
| 4 | config.json 不跟随 `--state-dir`（loader.go:14-30 固定 $HOME/.rick） | 改进候选（影响面待评估） |
| 5 | `.git` gitfile（worktree 单行文件）不重映射 → rick-dev 跨机断链 | 手工修复命令在 bugs.md；dump 预处理候选 |
| 6 | 幽灵 workspace id（源机历史遗留 67250c1c×27、8b549c5f×2）原样保留（ghost id 不猜原则） | 观察项；未来一次性清理工具 |
