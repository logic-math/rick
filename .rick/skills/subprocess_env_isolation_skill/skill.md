# skill:subprocess-env-isolation（subprocess 测试隔离）

## 触发场景

当集成测试中通过 subprocess 调用 rick CLI，测试本地通过但行为与预期不符时：
- subprocess 读取了真实 `~/.rick/config.json` 而非测试配置
- 测试依赖真实 Claude API key 而意外触发真实调用
- subprocess 的 HOME/env 与父进程不一致导致测试不稳定

信号词：「测试超时」「subprocess 调用了真实 claude」「config 未隔离」

## 预期效果

- subprocess 只读取测试专用 HOME 目录下的配置
- 测试可在无真实 API key 的环境下稳定运行
- 首次失败后无需多轮调试

## 核心内容

### Step 1：创建隔离的测试 HOME 目录

```python
import os, tempfile, json

work_dir = tempfile.mkdtemp()
rick_dir = os.path.join(work_dir, ".rick")
os.makedirs(rick_dir)

# 写入 mock config，禁用真实 Claude 调用
mock_config = {"claude_code_path": "/usr/bin/false", "max_retries": 0}
with open(os.path.join(rick_dir, "config.json"), "w") as f:
    json.dump(mock_config, f)
```

### Step 2：透传 HOME 到 subprocess

```python
env = os.environ.copy()
env["HOME"] = work_dir  # 关键：覆盖 HOME，让 LoadConfig 读 mock 配置

result = subprocess.run(
    [rick_bin, "human-loop", "测试主题"],
    cwd=work_dir,
    env=env,          # 必须显式传 env，不能依赖继承
    timeout=30,
    capture_output=True,
    text=True
)
```

### Step 3：Go 单元测试同样需要隔离

```go
func TestXxx(t *testing.T) {
    dir := t.TempDir()
    t.Setenv("HOME", dir)
    _ = os.MkdirAll(filepath.Join(dir, ".rick"), 0755)
    _ = os.WriteFile(filepath.Join(dir, ".rick", "config.json"),
        []byte(`{"max_retries":0}`), 0644)
    // ... 测试逻辑
}
```

### 常见根因

- `os.environ.copy()` 后未覆盖 HOME → subprocess 继承真实 HOME
- Python `subprocess.run()` 不传 `env=` 参数 → 自动继承父进程全部 env（包括真实 HOME）
