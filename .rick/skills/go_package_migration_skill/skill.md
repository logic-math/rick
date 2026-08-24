# skill:go-package-migration（Go 包迁移/删除重构）

## 触发场景

需要把 Go 包整体移动/重命名/删除，或大规模改动 import 路径时使用，特别是：

- 迁移包（如 `internal/agent/piagent` → `internal/runtime`）
- 下沉能力收口到新包（如 `internal/env`、`internal/handler`）
- 删除整批冗余包（如 `internal/executor`、`internal/parser`、`internal/git`）

信号词：「迁移 X 到 Y」「收口」「删除冗余包」「做薄」「下沉」

## 预期效果

- 一次完成迁移，`go build ./...` + `go vet ./...` + `go test ./...` 全绿
- 无残留旧包引用（grep 旧 import 路径 = 0）
- 不反复踩 `git add bin/rick` 静默失败、Edit oldText 不匹配等坑

## 核心内容

### Step 1：复制 + 改 package 声明（迁移场景）

```bash
mkdir -p internal/<newpkg> && cp internal/<oldpkg>/*.go internal/<newpkg>/
sed -i 's/^package <oldpkg>$/package <newpkg>/' internal/<newpkg>/*.go
sed -i 's#internal/<oldpkg>#internal/<newpkg>#g' internal/<newpkg>/*.go
```

### Step 2：全局改调用方 import（先 grep 找全）

配合 [global_ref_sync_skill](../global_ref_sync_skill/skill.md) 第 0 步：

```bash
grep -rln "internal/<oldpkg>" internal/ cmd/ pkg/ --include='*.go'
# 批量替换 import 路径 + 限定符：
sed -i 's#"github.com/sunquan/rick/internal/<oldpkg>"#"github.com/sunquan/rick/internal/<newpkg>"#g; s/<oldpkg>\./<newpkg>./g' <files>
```

### Step 3：go build 找断点，逐批修复

```bash
go build ./... 2>&1 | head -40
# 只修 build 报错指向的文件，再 build，循环到无报错
```

### Step 4：删除旧包 + 残留引用清零

```bash
rm -rf internal/<oldpkg>
grep -rn "internal/<oldpkg>\|<oldpkg>." internal/ cmd/ pkg/ --include='*.go' || echo CLEAN
```

### Step 5：格式化 + vet + 全量测试

```bash
gofmt -w <改动的 .go 文件>
go vet ./...
go test ./... -timeout 120s
```

### 陷阱

- **`git add bin/rick` 静默失败**：`bin/` 在 `.gitignore` 里，`git add bin/rick` 无输出但未暂存 → 必须 `git add -f bin/rick`（force）。验证：`git status --short bin/`
- **Edit oldText 不匹配**：批量改 import 块时先 `Read` 精确行再粘贴为 oldText（见 global_ref_sync 第 1.5 步），不要凭记忆拼 import 块
- **删除包前先确认引用清零**：`grep -rln "internal/<oldpkg>"` 返回空再 `rm -rf`，否则 build 全红
