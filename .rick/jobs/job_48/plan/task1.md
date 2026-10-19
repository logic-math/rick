# task1：CLI super 工作区解析切换（<工作区>/.rick/super_env）

## 任务目标

super 的运行时工作区从全局 `~/.rick/super/` 切换为**当前工作区内** `<ws>/.rick/super_env/<name>/`——CLI/web 一致（L2-Q1=A 裁决），env 在工作区内生成（J2 修正案）。

## 关键结果

- 修改 `internal/handler/super.go`：
  - `superWsRoot()` 新语义：**优先** `~/.rick/config.json` 的 `super.default_super_dir`（显式覆盖，向后兼容）；否则从**进程 cwd 向上找 `.rick` 目录**（与 `resolveRepoRoot()` 同型遍历）定位工作区根，返回 `<工作区根>/.rick/super_env`；都找不到 → 明确报错（含提示：需在 rick 工作区内运行或配置 default_super_dir）
  - 路径解析链（SuperStart/Stop/Status/Event 的 ws 解析）全部走新 root；`--name` 语义不变
- `internal/super/session.go` BuildSuperPrompt 不动（ws 是参数传入）
- 旧全局目录 `~/.rick/super/` **不迁移不兼容读取**（历史测试工作区留原地）；但在 root 解析到旧目录场景时不受影响（互相独立）
- 测试 `internal/handler/super_test.go` 适配 + 新用例：
  - cwd 在工作区内（含 `.rick`）→ root=`<ws>/.rick/super_env`
  - cwd 无 `.rick` 上溯可达 → 报错信息明确
  - config default_super_dir 显式设置 → 覆盖优先
  - SuperStart 在新 root 下全生命周期（fake superopt）依旧绿
- **注意**：web 会话的 spawn cwd=工作区根 → 新解析天然生效；CLI 在工作区任意子目录也能找到根

## 写域

- internal/handler/super.go
- internal/handler/super_test.go

## 依赖关系

- 无（Kahn L1）

## 实现要点

- 遍历实现照抄 `internal/super/supervisor.go resolveRepoRoot()` 的模式（cwd 向上找 `.rick` 目录；到文件系统根回退报错）
- `superWsRoot()` 依赖 cwd——web spawn 的 cwd 是 ws 根（会话管理器 spawn 时 `cmd.Dir`）；CLI 是用户 shell cwd。两路一致
- e2e（super/tests/e2e_test.py）用 HOME 隔离旧 root 的测试会仍指旧路径——**不动 py 测试**（py 侧 workdir 是显式传参 `--workdir`，与 root 解析无关）；只改 Go 侧

## 测试方法

`go build ./... && go test ./internal/handler/ -run "TestSuper" -timeout 300s` 全绿（含新用例）

## 禁止事项

- 不改 py 调度器（workdir 一直显式）；不动 internal/super/session.go；不做旧数据迁移
