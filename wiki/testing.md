# Rick CLI 测试与验证

## 测试策略

Rick 采用多层次测试策略：

```
         ┌─────────────┐
         │  集成测试    │  ← bash tests/tools_integration_test.sh
         └─────────────┘
       ┌───────────────────┐
       │  Go 单元测试       │  ← go test（精确匹配改动包）
       └───────────────────┘
```

## 单元测试

**范围精确性**：禁止跑全量 `go test ./internal/...`（会混入依赖真实 API key 的无关测试）。精确匹配改动包：

```bash
go test ./internal/config/... ./internal/env/... ./internal/runtime/... \
  ./internal/builder/... ./internal/prompt/... ./internal/handler/... \
  ./internal/cmd/... ./internal/workspace/... -timeout 60s
```

## 集成测试

```bash
bash tests/tools_integration_test.sh
```

覆盖：
- **pi 侧门禁脚本**（`.rick/skills/rick-gates/helper.py`）：tasks.json 可解析 / 无 zombie / success 有 commit_hash
- **learning_check**：SUMMARY.md + loops/skills 格式
- **dry-run**：skills 注入、命令可用
- **mock_agent** 替代真实 pi 调用

## 关键约定

- **配置污染防护**：测试开头 `t.Setenv("HOME", t.TempDir())`，避免读真实 `~/.rick/config.json`。
- **fake pi 脚本**：PATH 替换测试中 fake 脚本开头恢复系统 PATH，或用 shell 内建。
- **托管运行时隔离**：依赖 PATH fake pi 的测试须 `t.Setenv("RICK_PI_AGENT_DIR", t.TempDir())`，避免命中真实托管 pi。
- **JSON 输出**：`json.dumps(..., ensure_ascii=False)`，中文不转义。
- **dry-run 断言**：先定位 section 再检查内容，避免全文误判。

## 门禁（rick-gates）

doing 门禁已下沉为确定性脚本 `.rick/skills/rick-gates/helper.py`，在 pi `agent_settled` 后由 runtime 调用，exit 非 0 = 门禁失败。`plan_check`/`doing_check` 命令已删除。
