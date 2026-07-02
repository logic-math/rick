# skill:verify-go-changes（验证 Go 代码修改）

## 触发场景

修改了 Go 源文件后，需要验证编译通过、单元测试和集成测试通过时使用。

## 预期效果

- 编译无报错
- 单元测试全绿（精确匹配改动包，不跑全量）
- 集成测试通过
- dry-run 输出包含正确注入内容

## 核心内容

### Step 1：编译检查

```bash
go build ./...
# 或使用辅助脚本：
python3 .rick/skills/mark_task_success_skill/build_rick.py
```

### Step 2：单元测试（精确范围，禁止 ./internal/...）

```bash
# 只跑改动的包，避免混入依赖真实 API key 的无关测试
go test ./internal/prompt/... -v
go test ./internal/executor/... -v
go test ./internal/cmd/... -timeout 60s -v
```

**禁止**：`go test ./internal/...` 全量，会混入依赖真实环境的无关测试。

### Step 3：集成测试（如有）

```bash
bash tests/tools_integration_test.sh
```

### Step 4：dry-run 验证注入效果

```bash
./bin/rick doing job_N --dry-run | grep "关键词"
./bin/rick plan --dry-run | grep -c '{{未替换变量}}'  # 应为 0
```

### Go test 超时问题

如果测试卡住 > 30s（卡在 retry sleep），检查是否有全局 `~/.rick/config.json` 设置了高 `max_retries`：

```go
// 在测试函数开头添加：
dir := t.TempDir()
t.Setenv("HOME", dir)
_ = os.MkdirAll(filepath.Join(dir, ".rick"), 0755)
_ = os.WriteFile(filepath.Join(dir, ".rick", "config.json"), []byte(`{"max_retries":2}`), 0644)
```

### DIP 验证命令

修改 internal/ 模块后验证依赖倒置是否保持：

```bash
# 确认 executor/actpath 不直接依赖 claudecode
grep -r "claudecode" internal/executor/ internal/actpath/ 2>/dev/null
# 应无输出（仅 doing.go 可引用 claudecode）
```

### embed.FS 变更后必须重新构建

模板文件（`internal/prompt/templates/`）通过 embed.FS 编译进二进制，改后必须：

```bash
./scripts/build.sh  # 重新构建才能生效
./bin/rick doing job_N --dry-run  # 验证新内容
```
