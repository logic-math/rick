# Bugs Domain

**最后更新**: —  **来源 Job**: —

项目已知问题与精确解决方案。由 `rick learning` 和 `rick dream` 自动追加，勿手动覆盖。

## 已知问题与解决方案

### Python 集成测试：subprocess 读取真实 ~/.rick/config.json 导致触发真实 Claude 调用

**根因**: `subprocess.run()` 不显式传 `env=` 时，子进程继承父进程 HOME，`LoadConfig()` 读取真实 `~/.rick/config.json`，触发真实 Claude CLI 调用，测试超时或行为不符合预期。

**精确解决步骤**:
```python
env = os.environ.copy()
env["HOME"] = work_dir  # work_dir 下有 mock ~/.rick/config.json
result = subprocess.run([rick_bin, ...], env=env, cwd=work_dir, timeout=30, ...)
```

**首次发现**: job_26 / task1 / commit d50b255a  **验证状态**: ✅ 已修复
