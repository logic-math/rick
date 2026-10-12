# 调研：agent 状态目录（.rick）云端同步机制（C 组）

## 同步方案对比

- **git（bare repo+alias / yadm）**：冲突=文本 3-way 合并需手工解，JSON 冲突标记易碎；两端并发 commit tasks.json → 后 push 者被拒（non-fast-forward），须 rebase 解冲突；频次手动；离线可 commit 后补 push，divergence 累积。chezmoi 是声明式「期望状态」工具，目标端被外部改只告警不合并，不适配运行时状态。
- **Syncthing**：两端并发改同一文件 → 旧 mtime 一方改名 `.sync-conflict-*` 副本，双版本保留人工裁决（mtime 相同按 device ID 大者标冲突）；fs watcher ~10s 近实时；P2P 天然离线容忍，重连即补同步。
- **WebDAV**：协议有 ETag/If-Match 乐观锁（RFC 7232）+ LOCK，但同步语义全靠客户端实现，社区客户端多为 LWW（含「删除后修改→复活」）；频次由客户端定。
- **rclone bisync（S3/R2/WebDAV 通用）**：靠上次运行清单做三方比较；双端都改默认无胜者（`--conflict-resolve` 默认 none），可设 newer/older/path1/loser 留编号副本；定时/手动运行无守护进程；中断后须 `--resync` 重建基线，无真正离线队列。

## 先例

- **Cursor rules**：`.cursor/rules`（.mdc）随项目 git 仓库版本化；团队级规则走 dashboard 云端管理，非文件同步。[官方文档]
- **Continue.dev**：本地 YAML 配置 + Continue Hub 远程配置（控制平面）做组织级共享，云端结构化存储。[官方文档]
- **Claude Code**：会话=本地 JSONL transcript；Web 版会话跑 Anthropic 云端持久化、手机端可监控、teleport 回终端；Agent SDK 的 SessionStore 适配器镜像到 S3/Redis/DB，核心是 append-only 分块文件 `part-{epochMs13}-{rand6}.jsonl`——追加即写、无整文件覆写，天然免冲突。[官方文档+SDK 源码]
- **ufhy/claude-sync**：社区把 `~/.claude` 会话同步到 R2/S3/WebDAV（age 端到端加密），文件级同步先例。[社区]
- **PraisonAI / Genkit**：会话存储可插拔——文件（开发）→ SQLite/PostgreSQL/Firestore（生产并发）；「共享状态并发写应上数据库，文件系统只做接口」。[官方文档+厂商博客]

## 结论

1. 文件级双向同步遇并发写都留冲突：git 要手工解、Syncthing/rclone 留副本（官方文档级）。
2. 先例共识：静态规则走 git，运行态走云端结构化或 append-only 存储（官方文档级）。
3. 建议 .rick 拆层：静态内容留 git，doing/tasks.json 改事件日志或服务端单写（推断，弱信源）。

## 信源链接

- Syncthing 冲突与同步机制：https://docs.syncthing.net/users/syncing.html ；版本控制：https://docs.syncthing.net/users/versioning.html
- rclone bisync（conflict-resolve/loser、resync）：https://rclone.org/bisync/ ；https://rclone.org/commands/rclone_bisync/
- git non-fast-forward push 被拒：https://docs.github.com/en/get-started/using-git/dealing-with-non-fast-forward-errors
- chezmoi 对目标端漂移仅告警：https://github.com/twpayne/chezmoi/issues/4056 ；对比表：https://chezmoi.io/comparison-table/
- WebDAV 冲突（ETag/If-Match、客户端 LWW）：https://manual.davx5.com/technical_information.html ；https://github.com/Ethran/notable/blob/main/docs/webdav-sync-user.md
- Cursor rules：https://cursor.com/docs/rules ；社区同步扩展：https://github.com/kushal-khare-official/cursor-config-sync
- Continue 远程配置：https://docs.continue.dev/guides/configuring-models-rules-tools
- Claude Code on the web：https://code.claude.com/docs/en/claude-code-on-the-web.md
- Claude Agent SDK SessionStore：https://code.claude.com/docs/en/agent-sdk/session-storage ；S3 适配器源码：https://github.com/anthropics/claude-agent-sdk-python/blob/e41cbdd4/examples/session_stores/s3_session_store.py
- ufhy/claude-sync：https://github.com/ufhy/claude-sync
- PraisonAI 存储后端：https://praison.ai/docs/storage/backends ；Genkit session stores：https://genkit.dev/docs/js/agents/session-stores/
- 文件系统 vs 数据库（agent 记忆）：https://blogs.oracle.com/developers/comparing-file-systems-and-databases-for-effective-ai-agent-memory-management
