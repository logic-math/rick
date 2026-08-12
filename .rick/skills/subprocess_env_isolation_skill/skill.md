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

### Step 4：托管/自闭环二进制"优先解析"陷阱（job_34 沉淀，最痛）

**根因**：当代码把解析顺序从"PATH 查找"改成"托管路径优先"（如 `FindBinary`: cfg.PiPath → `RuntimeBin()` → PATH），**所有依赖 PATH fake 二进制的测试会静默命中真实托管二进制**——fake 没生效、甚至触发真实交互/联网（测试从 265s 挂死超时 → 5s 全绿）。

**信号词**：「测试突然挂死/超时」「fake pi 不生效」「走了真实 pi/网络」「timeout after 10m panic 且无具体 --- FAIL」

**精确解决**：测试必须隔离托管路径的解析根：
```go
// Go: 让 AgentDir() 解析到 temp（RuntimeBin 随之不存在 → 回退 PATH fake）
t.Setenv("RICK_PI_AGENT_DIR", t.TempDir())   // piagent 的隔离 env
// 或等价：t.Setenv("HOME", t.TempDir())     // AgentDir() = $HOME/.rick/pi/agent
```
```python
# Python subprocess 同样：env["HOME"] = work_dir（见 Step 2）
```

**排查路径**（快速定位哪个测试被污染）：
1. 全量跑挂死 → 用 `-timeout 60s` + `-run "<可疑测试名>"` 逐个定位（比反复整跑快得多，internal/cmd 全套 6 分钟）
2. 超时 panic 的 "running tests:" 会列出挂死的测试——优先查它
3. 检查该测试是否设置了隔离 env（HOME / RICK_PI_AGENT_DIR / 自定义 agent dir env）

**幂等 patch 的锚点陷阱**（顺带沉淀）：字符串替换式 patch 若锚点在替换后**仍然存在**（如往函数前插 helper，锚点是函数签名本身）→ 二次运行重复插入。解决：用**整函数替换**（old 含完整函数体，替换后被消费）或让 old 包含一段唯一且会被改掉的文本。
