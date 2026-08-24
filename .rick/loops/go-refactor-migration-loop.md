---
name: go-refactor-migration-loop
trigger: "当需要把 Go 包整体迁移/重命名/删除，或大规模改动 import 路径并让 build+test 收敛到绿时触发"
scope: "doing / 全局"
---

# Loop: Go 大重构迁移循环

## 目标（Goal）

把一个或多个 Go 包安全迁移/重命名/删除，最终 `go build ./...` + `go vet ./...` + `go test ./...` 全绿，且无残留旧包引用。

- 成功标准：build 无报错；grep 旧 import 路径 = 0；全量测试通过；门禁通过

## 上下文管理（Context Management）

- 迁移前读取 `.rick/domain/go-patterns.md`（Go 编码规范）、`.rick/domain/architecture.md`（模块划分）、`.rick/domain/bugs.md`（已知坑）
- 从 `doing/debug/`（若有）提取前轮摘要，避免重复踩坑
- 保留「目标包清单 + 旧→新路径映射表」作为本轮核心事实，其余细节压缩

## 可调用工具（Tool Access）

- `cp` / `sed` / `grep -rln` / `rm -rf`：移动与改名 —— 约束：先 grep 确认引用清零，再 rm
- `go build ./...`：找断点 —— 约束：只修 build 报错指向的文件
- `gofmt -w` / `go vet ./...`：格式化与静态检查
- `go test ./... -timeout 120s`：全量验证
- `git add -f bin/rick`：重建后的二进制强制暂存（`bin/` 在 .gitignore）
- 权限边界：迁移是纯代码重构，不新增业务行为；每步修改前工作区干净

## 产出评估（Output Evaluation）

加载 `.rick/skills/go_package_migration_skill/skill.md` + `.rick/skills/verify_go_changes_skill/skill.md`

| 检查项 | 验证方法 | 通过标准 |
|--------|----------|----------|
| build | `go build ./...` | 无报错 |
| 残留引用 | `grep -rln "internal/<oldpkg>" internal/ cmd/ pkg/ --include='*.go'` | 返回空 |
| 静态检查 | `gofmt -l <files>` + `go vet ./...` | 空输出 / 无报错 |
| 全量测试 | `go test ./... -timeout 120s` | 全部 ok |
| 二进制重建 | `go build -o bin/rick ./cmd/rick && git add -f bin/rick` | 暂存成功 |

## 停止标准（Termination Condition）

- 成功退出：build + vet + test 全绿，残留引用为零
- 优雅退出（任意一条触发）：
  - 迭代 3 轮仍未收敛（build 持续报同类错误）
  - 发现迁移范围超出任务边界（涉及协议/业务语义改动）
  - 人类明确要求停止

退出时输出本轮「旧→新路径映射表 + 已删除包清单 + 验证结果」，等待人类决策。

## 工作流（Step 0-5）

- **Step 0 环境确认**：`go version` + `which go`，缺 go toolchain 则报错停止；搜索 `.rick/domain/bugs.md` 已知坑
- **Step 1 parent 确认全局目标**：明确迁移哪些包、目标路径、成功标准（build+test 全绿）
- **Step 2 parent 读上下文**：读 `.rick/domain/` 相关文件 + 前轮摘要，生成「旧→新映射表」
- **Step 3 worker child 执行迁移**：加载 `.rick/skills/go_package_migration_skill/skill.md`，按 Step 1-5（复制→sed 改名→build 找断点→删旧包→gofmt/vet/test）执行
- **Step 4 parent 产出评估**：按上表逐项验证，失败则把 build 报错附加到上下文返回 Step 3
- **Step 5 parent 确认停止标准**：全绿退出；否则按优雅退出条件裁决
