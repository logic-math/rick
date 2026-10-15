# Build Domain

**最后更新**: 2026-08-17  **来源 Job**: job_35

项目构建/测试/门禁/提交的事实性命令。

## 事实列表

### 构建

- **构建仓库二进制**: `go build -o bin/rick ./cmd/rick`
  - 验证命令: `ls -la bin/rick`
  - 来源: job_35
  - 状态: ✅ 已确认

- **全量编译检查**: `go build ./...`
  - 验证命令: `go build ./... && echo BUILD_OK`
  - 来源: job_35
  - 状态: ✅ 已确认

- **⚠️ bin/rick 强制暂存（bin/ 在 .gitignore）**: `git add -f bin/rick`（`git add bin/rick` 静默 no-op）
  - 验证命令: `git status --short bin/`（应显示 M）
  - 来源: job_35 / task7
  - 状态: ✅ 已确认

### 测试

- **精确范围单元测试**: `go test ./internal/<pkg>/... -timeout 60s`（禁止无脑 `go test ./...` 全量，会混入依赖真实环境的测试）
  - 来源: job_35 / verify_go_changes_skill
  - 状态: ✅ 已确认

- **全量收敛测试**: `go test ./... -timeout 120s`
  - 来源: job_35
  - 状态: ✅ 已确认

- **集成测试脚本**: `bash tests/tools_integration_test.sh`
  - 来源: job_35 / task12
  - 状态: ✅ 已确认

### 门禁与验收

- **doing 门禁**: `python3 .rick/skills/rick-gates/helper.py .rick/jobs/job_N/doing`（tasks.json 可解析 / 无 zombie / success 有 commit_hash）
  - 来源: job_35 / task8
  - 状态: ✅ 已确认

- **task 验收脚本**: `python3 .rick/jobs/job_N/doing/tests/taskN.py`（输出 `{"pass": true, ...}`）
  - 来源: job_35
  - 状态: ✅ 已确认

## 已知问题与解决方案

### `git add bin/rick` 静默失败（未加 -f 漏提交）

**根因**: `bin/` 在 .gitignore，`git add bin/rick` 无报错但文件未暂存。

**精确解决步骤**:
```bash
go build -o bin/rick ./cmd/rick
git add -f bin/rick
git commit -m "chore(taskX): rebuild bin/rick"
```

**首次发现**: job_35  **验证状态**: ✅ 已修复

## update-pi（v4.1.0 新增）

- **更新 pi runtime**：`rick tools update-pi pi` —— 托管 runtime（`~/.rick/pi/agent/runtime`）走 rick 自己的 `npm install --prefix`（与 init-pi 同语义，绕开 `pi update --self` 对非全局安装的 guard）；无托管 runtime 时委托 `pi update --self`
- **更新全部扩展**：`rick tools update-pi extensions`（`pi update --extensions`，作用于 rick 托管 agent 目录，不碰用户 ~/.pi）
- **更新单个扩展**：`rick tools update-pi pi-subagents`（源名自动解析为注册形态 `npm:pi-subagents`；未注册报错并列出已注册项）
- **刷新模型目录**：`rick tools update-pi models`
- **全量更新**：`rick tools update-pi`（= all：pi → extensions → models，顺序固定）
- **更新后快速自检**（自动执行）：pi 版本 / 必需扩展注册 / rick agents+hooks 落盘 / rick-gates helper.py 语法 / human-loop 提示词渲染冒烟（非空且无 `{{` 残留）
- 注意：裸扩展名是注册行（`npm:pi-subagents`）的子串，解析必须 npm: 前缀形态优先，否则 `pi update pi-subagents` 可能子串误命中传裸名（env.resolveExtensionSource）

## release 布局（job_43 实测）

- **本机 release 固定布局**：`RICK_RELEASES_DIR=/home/hadoop-recsys/.rick/releases`——`<prodRepo>/bin/releases/` 属主 sankuai 且 ACL mask r-x 不可写（mkdir permission denied，release build 阶段失败、生产零影响），必须用 env 重定向（代码预留的 ReleasesRoot 覆盖机制；与 2026-09-25 02:14 起的成功 release 同一布局）
  - 验证命令: `ls -la /home/hadoop-recsys/.rick/releases/current`（软链 → 版本目录）
  - 来源: job_43 / 迭代 1 S5（失败→重试成功）
  - 状态: ✅ 已确认
- **release 版本号语义**：version = `<devHEAD sha7>-<构建时刻 ts>`——S4 批准留痕提交会推进 dev HEAD，实际提升版本 sha7 与 S3 演练版不同属正常（内容=task + 证据提交）
  - 来源: job_43 / 迭代 1-6 六次实测
  - 状态: ✅ 已确认
- **detach 跨休眠**：`--detach` 的 setsid 进程在机器休眠期存活，恢复后继续完成（gate→build→merge→换链→重启全程无人值守）
  - 来源: job_43 / 迭代 2（12:35 启动，机器休眠 4 天后 09-29 02:00 完成）
  - 状态: ✅ 实测确认
- **生产接管**：休眠期外部启动的非 release 实例（如 `~/.rick/bin/rick`，build_id=dev）会被下一次 release 的重启步骤停掉并接管（stopped 列表含外部 PID）
  - 来源: job_43 / 迭代 6（stopped=[6440,46,4895]）
  - 状态: ✅ 实测确认

## dump/import 验证命令集（job_46 新增，功能在 dev/self-evolve 分支）

- **单元测试**: `go test ./internal/env/relocate/... ./internal/env/procfs/... ./internal/env/devweb/... ./internal/env/release/... -count=1 -timeout 600s`（relocate 68 测试含 dump 7 + import 10）
  - 来源: job_46 R1-R3  **状态**: ✅ 全绿
- **darwin 交叉编译代理验证**: `GOOS=darwin GOARCH=arm64 go build ./...` + `GOOS=darwin GOARCH=amd64 go build ./...` + `GOOS=darwin go vet ./internal/env/relocate/... ./internal/cmd/...`
  - 来源: job_46（procfs_darwin 真机行为仍未实测，解析器表驱动测试代理覆盖）  **状态**: ✅ 通过
- **e2e 全链路**: `bash tests/relocate_e2e.sh`（67 断言：dump 结构/排除/回填/边车 + import 落位/重映射/marker 跨机稳定/会话重转义 + 重导入合并幂等；sandbox 零触碰真实 ~/.rick）
  - 来源: job_46 R3  **状态**: ✅ 67/67
- **测试环境警告**: rick 会话内跑上述命令须先 `env -u RICK_STATE_DIR -u RICK_PI_AGENT_DIR -u PI_CODING_AGENT_DIR`（否则 web 单例测试假失败，见 bugs.md job_46 条目）
  - 来源: job_46 S3  **状态**: ✅ 已确认

## dump/import 实测数据点（job_46 E2E，Linux 源机 156G 混合树）

- **打包**：156G 混合树（142G ML 权重经 12 条 `--exclude` 剔除）→ 13.2GB payload / 286,032 文件 → **8.6GB gzip**（9,194,720,964 字节）/ 耗时约 10 分钟（含交叉编译 linux/amd64 内嵌）
- **传输**：scp 8.6GB 内网 47 秒（~190MB/s）
- **导入**：257,255 工作区文件 + 24,997 状态文件 + 3,778 pi 会话文件 / 耗时约 8 分钟（含 sha256 校验 + 重映射 + header 重写）
- **验收**：11 个子工作区 id 全保留（连接键不断）、会话目录全重转义、resume 后 pi 加载 9962 tokens 历史正常应答
- **dump 产物**：`<name>.tar.gz` + `.sha256` 边车；manifest 记录 marker id / origin 主机 / 23 项排除留痕（platform-elf / regenerable / cache / external / custom / unreadable）
