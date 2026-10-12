# Learning Step 5 产出：Domain 事实知识（job_43）

> 依据：gen-domain 惯例（无独立 skill 文件——按 learning_loop 声明的兜底惯例执行：事实性知识、追加不重复、精确命令）| 写路径：全部 dev 树（RSI 纪律）

## 追加清单（4 文件 / 9 事实条目，先读后写零重复）

### bugs.md（+4 条，583→601 行）

| # | 条目 | 验证状态 |
|---|------|---------|
| 1 | RICK_* env 穿透 t.Setenv(HOME) 测试隔离（gate7 实录 + python 剥离 RICK_* 前缀精确步骤） | ✅ 已修复 |
| 2 | 环境休眠/快照重置 ACL mask 与属主（两次实测 + rename+git checkout 重建法 + 沙盒重建法） | ✅ 已恢复 |
| 3 | ResumeEasyIn 无条件覆盖任意 job 的 tasks.json（bug1，git checkout 应急 + 条件写入修复方向） | ❌ 未修复（遗留下轮） |
| 4 | rsi/release.md 头部三行误删（两次实测 + 头部契约格式） | ✅ 已恢复 |

### commands.md（+1 节，136→143 行）

- 新增 `## rick web（job_43 起：会话持久持有 + 工具箱）`：web_max_active 配置链（config→组合根→DefaultMaxActive 64→IdleDisabled 持有语义）；工具箱入口与 FileReaderPanel 全局挂载；**GET /api/workspaces/{ws}/file 裁决语义**（human 2026-10-01「不做根限制」，含迭代 5→6 白名单往返史）；curl 验证命令

### architecture.md（+1 节，174→182 行）

- 新增 `## web 前端关键结构（job_43 六轮增量，5.0.12-5.0.13）`：MessageList 虚拟化要点（双通道贴底/前置防跳/Scroller ...rest P0/Footer spacer）；FileReaderProvider 全局挂载；starfieldMode 移动端静态化；Enter 固定换行输入框；窄屏节流结构

### build.md（+1 节 4 条，73→89 行）

- 新增 `## release 布局（job_43 实测）`：RICK_RELEASES_DIR 固定布局（bin/releases 属主坑）；版本号=构建时刻语义；detach 跨休眠；生产接管外部实例

## 排除项（核对后无需追加）

- env.md：无新环境事实（RICK_RELEASES_DIR/RICK_DEV_TREE 均为既有机制，已记 build.md/architecture.md 的使用事实）
- go-patterns.md / testing-conventions.md：本次改动遵循既有模式，无新规范提炼（测试相关教训已进 bugs.md + subprocess_env_isolation skill）

## 验证

- 追加前 grep 四文件确认 0 重复（web_max_active/工具箱/virtuoso/FileReader/RICK_RELEASES_DIR 均不存在）
- 追加后关键词唯一性复核：新条目各出现 1 次（bugs.md 第 1 条标题+正文各 1 次属同条目非重复）
- 全部追加（>> 追加写入，未覆盖任何既有内容；各文件既有行数只增不减）
