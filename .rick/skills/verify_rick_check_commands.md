# verify_rick_check_commands

## 触发场景

当需要验证 `rick tools plan_check` 或 `rick tools doing_check` 命令的行为是否符合预期时使用，例如：
- 验证 check 命令对特定 job 的输出是否正确
- 排查 check 命令报告的错误是否属实
- 确认任务完成后 check 通过

## 使用的 Tools

- `tools/build_and_get_rick_bin.py` — 构建最新 rick 二进制，获取本地路径
- `tools/check_go_build.py` — 确认 Go 项目编译无误

## 执行步骤

1. **构建最新 rick 二进制**
   ```bash
   python3 tools/build_and_get_rick_bin.py
   # 返回 {"pass": true, "bin_path": "..."}
   ```

2. **运行 doing_check 验证当前 job**
   ```bash
   {bin_path} tools doing_check {job_id}
   # 检查输出中的 PASS/FAIL 状态
   ```

3. **运行 plan_check 验证规划阶段产出**
   ```bash
   {bin_path} tools plan_check {job_id}
   ```

4. **解读输出**
   - `✅ PASS` — 检查通过，产出符合规范
   - `❌ FAIL` — 检查失败，根据错误信息修复后重新运行

## 示例

```bash
# 构建并验证 job_12
python3 tools/build_and_get_rick_bin.py
# => {"pass": true, "bin_path": "/path/to/bin/rick"}

/path/to/bin/rick tools doing_check job_12
# => ✅ task1: debug.md exists
# => ✅ task1: commit exists
# => PASS
```
