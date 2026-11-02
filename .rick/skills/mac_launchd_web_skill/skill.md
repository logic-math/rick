# skill:mac-launchd-web（macOS 上守护 rick web：开机自启 + 死亡自愈）

## 触发场景

当 rick web 部署在 macOS 上且出现以下任一需求时使用：
- 需要**开机/登录后自动启动** web（不用人工跑 start-web.sh）
- web 进程死后需要**自动复活**（实测：macOS **睡眠/唤醒周期会向后台 daemon 发 SIGTERM**，`--daemon`（setsid+nohup）只能防 SIGHUP 防不了它——job_49 两次同签名死亡实证：worker exit 143 → `graceful shutdown: context deadline exceeded`）
- 信号词：「web 又挂了」「睡眠唤醒后 web 死了」「开机后要手动起 web」

不适用：Linux 机器（用 systemd）；临时前台跑一次（直接 `rick web`）。

## 预期效果

- 开机/登录自动启动；进程死亡 **≤5s 自动复活**（KeepAlive + ThrottleInterval）；睡眠唤醒后自愈
- launchd 环境下 pi worker 正常（node 可达 + 模型网关连通）——这是最容易翻车的两处，本 skill 的两个坑位检查覆盖
- 实战验证方法内建（kill 后看复活 + 纯净 env 起 pi 探针）

## 核心内容

### Step 0：前置事实（launchd 与终端环境的三个差异）

1. **launchd 不继承 shell 的 PATH**：homebrew 的 `/opt/homebrew/bin` 是 shell 配置（zshrc）注入的——plist 里必须显式列出。漏了 → supervisor 拉起 pi worker 报 `env: node: No such file or directory`（exit 127，pi 脚本 shebang `#!/usr/bin/env node` 找不到 node）
2. **proxy 环境不继承**：模型网关连接所需 env（http_proxy/https_proxy/no_proxy）要在 plist 里注入——**no_proxy 必须含 mcli.sankuai.com**（内网直达域名，proxy 转发不了它；见 env_drift_probe_skill）
3. **launchd 直接持有前台进程**：`--daemon` 的 setsid 脱离会让 launchd 追踪失效（KeepAlive 监控不到真实 web 进程）

### Step 1：写 plist（完整可用模板）

`~/Library/LaunchAgents/com.rick.web.plist`：

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.rick.web</string>
    <key>ProgramArguments</key>
    <array>
        <string>/<生产仓库>/bin/rick</string>
        <string>web</string>
        <string>--listen</string>
        <string>0.0.0.0</string>
        <string>--port</string>
        <string>6137</string>
        <string>--state-dir</string>
        <string>/Users/<user>/.rick</string>
        <string>--log-file</string>
        <string>/Users/<user>/.rick/web.log</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>ThrottleInterval</key>
    <integer>5</integer>
    <key>StandardOutPath</key>
    <string>/Users/<user>/.rick/launchd-web.log</string>
    <key>StandardErrorPath</key>
    <string>/Users/<user>/.rick/launchd-web.log</string>
    <key>EnvironmentVariables</key>
    <dict>
        <key>PATH</key>
        <string>/Users/<user>/bin:/opt/homebrew/bin:/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin</string>
        <key>http_proxy</key>
        <string>http://<proxy-host>:<port></string>
        <key>https_proxy</key>
        <string>http://<proxy-host>:<port></string>
        <key>no_proxy</key>
        <string>localhost,127.0.0.1,mcli.sankuai.com</string>
    </dict>
</dict>
</plist>
```

**设计要点（三个坑位，全部实测踩过）**：
- **前台持有，不带 `--daemon`**：`--daemon` setsid 脱离进程组 → launchd 只追到秒退的父进程 → KeepAlive 失效
- **不用 start-web.sh 做入口**：幂等脚本遇到「已在运行」秒退 exit 0 → launchd 把秒退当成功 → 旧实例死透后没人再拉（KeepAlive 重拉也被幂等分支挡掉）
- **`ThrottleInterval: 5`**：KeepAlive 默认重拉间隔 10s，5s 更快自愈

### Step 2：语法校验 + 加载

```bash
plutil -lint ~/Library/LaunchAgents/com.rick.web.plist   # 必须输出 OK
launchctl unload ~/Library/LaunchAgents/com.rick.web.plist 2>/dev/null  # 幂等：先卸再载
launchctl load ~/Library/LaunchAgents/com.rick.web.plist
launchctl list | grep com.rick.web   # 输出 "<pid> 0 com.rick.web" = 正常（第二列 0=上次退出码）
```

⚠️ 加载前**先停掉手动起的实例**（同端口互斥，否则 launchd 起的实例绑不上端口）：
```bash
kill $(cat ~/.rick/web.pid) 2>/dev/null; sleep 2
```

### Step 3：验证三连（每步必须过）

```bash
# ① web 在线且版本正确
curl -s http://127.0.0.1:6137/api/health
# ② KeepAlive 实战：杀掉看 5s 内复活（新 pid ≠ 旧 pid，launchctl list 的 pid 列变化）
kill $(lsof -nP -tiTCP:6137 -sTCP:LISTEN); sleep 8
curl -s http://127.0.0.1:6137/api/health && launchctl list | grep com.rick.web
# ③ pi worker 链路（launchd 纯净 env 模拟——最关键的探针）
env -i HOME=$HOME \
  PATH="/Users/$USER/bin:/opt/homebrew/bin:/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin" \
  http_proxy=http://<proxy-host>:<port> https_proxy=http://<proxy-host>:<port> \
  no_proxy="localhost,127.0.0.1,mcli.sankuai.com" \
  PI_CODING_AGENT_DIR=$HOME/.rick/pi/agent \
  $HOME/.rick/pi/agent/runtime/node_modules/.bin/pi --provider catpaw-proxy --model glm-5.3 --print "回复一个字：通"
# 期望输出「通」；exit 127 = PATH 缺 node；Connection error = no_proxy 缺 mcli.sankuai.com
```

### Step 4：故障排查速查

| 症状 | 原因 | 修法 |
|---|---|---|
| `env: node: No such file or directory`（exit 127） | launchd PATH 缺 homebrew | plist PATH 加 `/opt/homebrew/bin` → unload+load |
| worker 模型 `Connection error` | no_proxy 缺 mcli.sankuai.com | plist no_proxy 补域名 → unload+load |
| launchd list 有 pid 但 6137 无监听 | 端口被手动实例占着 | 停手动实例（kill web.pid），launchd 会在 ThrottleInterval 后重拉 |
| 改了 plist 不生效 | 没重载 | **launchctl unload + load 成对执行**（只 load 不生效） |
| KeepAlive 不复活 | 用了 --daemon 或幂等脚本做入口 | 改为直接前台持有（见 Step 1 设计要点） |
| web.log 无 launchd 实例日志 | 忘了 `--log-file` | ProgramArguments 里带上（launchd 前台模式 stdout 默认进 launchd-web.log，rick 日志仍进 web.log） |

### Step 5：手工兜底（launchd 失效时）

start-web.sh 保留不删（加固版自带架构自检 + 幂等 + health 校验）：
```bash
/Users/<user>/.rick/start-web.sh
```

## 出处

- job_49 追加沉淀（2026-11-02）：web 两次睡眠唤醒死亡（同签名）→ launchd 守护实战验证（kill 复活 77314→77391）→ PATH 缺口翻车修复（pi exit 127 → 补 /opt/homebrew/bin → 纯净 env 探针「通」）
- 关联：env_drift_probe_skill（no_proxy 坑的根因方法论）、debug/bug2-web-sleep-death-launchd.md（死亡根因链）
