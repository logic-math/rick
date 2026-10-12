# research-L4-leaf-1：源码工作树隔离实证（Q3）

## 结论 3 条

1. 推荐 `git worktree add --detach`：2.1s/202M，不复制 .git，只脏自己那份工作树，主仓零污染。
2. 必须 `--detach` 或新分支（`git branch -a` 显示 `+ main`＝已被主仓占用）；裸 `git clone --local` 跨设备失败，须加 `--no-hardlinks`。
3. worktree 内构建不破生产（prod exe inode 7904654 未变、dev 标记未进 prod 二进制）；真陷阱是 `.rick` 180M 副本含 job_36，dev 会读到过期 job 快照。

## 三方式实测（/tmp，已清理）

| 方式 | 命令 | 耗时 | 磁盘 | node_modules | bin/ | .rick | HEAD |
|---|---|---|---|---|---|---|---|
| worktree | `git worktree add --detach /tmp/l4-wt HEAD` | 2.13s | 202M | 无 | 无 | 180M | detached |
| clone | `git clone --local --no-hardlinks <repo> /tmp/l4-clone` | 4.32s | 713M | 无 | 无 | 180M | main（独立仓） |
| cp -a | `cp -a <repo> /tmp/l4-cp` | 8.36s | 958M | **带 185M** | **带 16M** | 200M | 共享 .git 副本 |

- 裸 `git clone --local`：`fatal: failed to create link ... Invalid cross-device link`（`df`：/tmp=`/dev/md0p1`，仓库=`overlay`）。
- worktree `.git` 仅指针 `gitdir: .../.git/worktrees/l4-wt`，`commondir=../..` ⇒ refs/objects 与主仓共享。

## 依赖与产物事实

- `web/dist` **被跟踪 12 文件**；`web/vite.config.ts:54` `outDir:"dist"`；`web/embed.go:31-32` `//go:embed all:dist` ⇒ 产物**编译期进二进制**。提交 dist 与主仓工作区 dist md5 相同（`51a6a649…`）。
- 不改源码 `npm run build`（4.13s）→ hash 不变、status 干净；改 1 可见字节 → `M web/dist/index.html`、`M .vite/manifest.json`、`M sw.js`、`D assets/index-C3ifqTyg.js`、`?? assets/index-D6T850Wu.js` ⇒ **dist 已提交，改前端必脏 git status**（仅该工作树）。
- `web/node_modules` 187M **未跟踪**；worktree `npm ci`=3.55s（`~/.npm` 热缓存 1.4G）→171M。
- `GOCACHE=/home/hadoop-recsys/.cache/go-build`、`GOMODCACHE=/home/hadoop-recsys/go/pkg/mod` **用户级共享** ⇒ worktree `go build -o bin/rick ./cmd/rick`=**0.983s**；缓存内容寻址，无产物串味。

## bin/rick 共享风险

- `bin/` 被 `.gitignore:2` 忽略；prod exe 实为 `ls -l /proc/172761/exe → 主仓 bin/rick`，与磁盘文件 inode 相同（7904654）。
- worktree 内构建写 `/tmp/l4-wt/bin/rick`：prod 二进制 mtime 仍 21:59、`strings` 无 dev 标记（dev 产物含 2 处）⇒ **无影响**。
- 用 `/bin/sleep` 复现 rename 覆盖运行中二进制：`/proc/$P/exe → …/mysleep (deleted)` 且进程存活 ⇒ 主仓直接覆盖 `bin/rick` **不会立刻打断 prod，但下次重启会加载 dev 二进制**。

## 推荐命令

```bash
git worktree add --detach /tmp/rick-dev HEAD     # 或 -b dev/web-self-iterate
cd /tmp/rick-dev/web && npm ci && npm run build
cd /tmp/rick-dev && go build -o bin/rick ./cmd/rick
```

## 遗留陷阱

1. **改前端必带 dist 一起提交**（dist 在库内），否则 baseline 与 dev 分叉。
2. **`.rick` 快照**：worktree 带入 180M/1114 跟踪文件（`jobs` 747、job_36 82）→ dev 若注册该工作区会看到**过期 job 状态**。
3. **ref 共享**：worktree 里 commit/branch 会出现在主仓 ref 列表 → 用独立分支名，勿动 `main`。
4. **命名法无效**：`_dev` 后缀只切 `~/.rick_dev/config.json`（`internal/config/loader.go:22-24`）与 `<cwd>/.rick_dev`（`internal/workspace/paths.go:41-43`），**不改 web 状态/registry**。
5. `/tmp` 跨设备且重启即失；长期 dev 树建议放 `/workdir` 同设备。耗时均为热缓存值。

## 清理验证

`git worktree remove --force /tmp/l4-wt && git worktree prune && rm -rf /tmp/l4-wt /tmp/l4-clone /tmp/l4-cp` → `git worktree list` 仅主仓；`/tmp/l4-*` 不存在；`git diff --cached` 空；受跟踪文件仅基线既有 `design-tree.md`；prod 8413 `http=200`。
