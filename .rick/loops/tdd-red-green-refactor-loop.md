---
name: tdd-red-green-refactor-loop
trigger: "当需要对 Go 代码进行 TDD 迭代直到测试通过时触发"
---

# Loop: Go TDD 迭代循环

## 目标（Goal）

让目标测试从失败状态收敛到通过状态，agent 自己可判断是否达成。

- 成功标准：目标测试全部通过，无 FAIL 输出，exit code 为 0
- 自评命令：`go test ./[package]/... -run [TestName] -v 2>&1`
- 自评输出：最后一行包含 `ok  [package]` 且无 `--- FAIL:`

## 上下文管理（Context Management）

- 保留：每轮的测试失败信息（具体 error message）、已尝试的修改方案摘要、当前失败用例列表
- 压缩：上一轮的完整代码 diff → 只保留"修改了哪个函数、结果如何"一句话
- 遗忘：已回滚的代码改动、临时加入的 fmt.Println 调试语句、通过的测试的详细输出

## 可调用工具（Tool Access）

- `go test`：运行单元测试，判断目标是否达成 —— 约束：只跑目标测试，不跑全量（`-run` 精确匹配）
- `Read / Edit / Write`：读写源码文件 —— 约束：只修改与失败测试直接相关的文件
- `git diff`：确认工作区状态 —— 约束：每轮修改前必须工作区干净
- 权限边界：禁止在迭代过程中 `git commit` 或修改测试文件本身（测试是 spec，不是实现）

## 产出评估（Output Evaluation）

- 评估类型：客观（运行目标测试，结果确定性）
- 评估命令：`go test ./[package]/... -run [TestName] -v 2>&1 | tail -5`
- 进展判断：本轮失败用例数量 < 上轮失败用例数量，视为有实质进展
- 退步判断：本轮失败用例数量 ≥ 上轮，或引入新的编译错误，立即用 `git checkout .` 回滚本轮改动

## 停止标准（Termination Condition）

- **成功退出**：`go test` 输出 `ok` 且 exit code 为 0，目标达成，退出循环
- **失败退出**：连续 3 轮无进展（失败数不减少），或累计迭代超过 5 轮
- **优雅退出**：回滚到本 loop 启动前的 git 状态（`git stash`），将当前失败信息写入 `debug/bug{n}-tdd-stuck.md`，frontmatter status 标为 `"❌ 无法修复"`，等待人工介入
