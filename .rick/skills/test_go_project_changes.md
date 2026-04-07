# test_go_project_changes

## 触发场景

当需要验证对 Go 项目代码的修改是否正确时使用，例如：
- 修改了 Go 源文件后需要确认编译通过
- 需要运行单元测试验证逻辑正确性
- 集成测试验证端到端行为

## 使用的 Tools

- `tools/check_go_build.py` — 检查 Go 项目编译是否通过
- `tools/build_and_get_rick_bin.py` — 构建 rick 二进制并获取路径（用于集成测试）

## 执行步骤

1. **检查 Go 编译**
   ```bash
   python3 tools/check_go_build.py
   # 返回 {"pass": true, "errors": []}
   ```

2. **运行单元测试**
   ```bash
   go test ./...
   # 或针对特定包
   go test ./internal/prompt/...
   ```

3. **运行集成测试**（如有）
   ```bash
   python3 tools/build_and_get_rick_bin.py
   # 获取 bin_path 后运行集成测试脚本
   bash tests/tools_integration_test.sh
   ```

4. **验证 dry-run 输出**
   ```bash
   {bin_path} doing {job_id} --dry-run
   # 检查 prompt 中关键变量是否注入正确
   ```

## 示例

```bash
# 修改 internal/prompt/builder.go 后验证
python3 tools/check_go_build.py
# => {"pass": true, "errors": []}

go test ./internal/...
# => ok  rick/internal/prompt  0.012s

python3 tools/build_and_get_rick_bin.py
# => {"pass": true, "bin_path": "/path/to/bin/rick"}

/path/to/bin/rick doing job_12 --dry-run | grep "可用的项目 Tools"
# => 验证 tools section 正确注入
```
