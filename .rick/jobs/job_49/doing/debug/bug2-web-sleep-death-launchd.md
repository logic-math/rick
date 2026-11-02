# bug2：web 二次死亡（睡眠唤醒周期）→ launchd 守护根治（2026-11-02 15:30）

## 症状
11-2 13:54 web 死亡（与 10-30 17:00 死亡**完全同签名**：worker 143 → graceful shutdown: context deadline exceeded）。机器未重启（boot 10-14）。

## 证据链
- 13:33 promote 65c2cd1 成功（job_50 super 修复——job_49 的 release 防线在真实自改进中零事故实战验证）
- 13:33-13:54 期间 human 恢复过两个挂起会话（worker 6f4c2b1f/76160f6a 活跃后被杀）
- 13:50-13:51 系统日志：睡眠/唤醒周期证据（WakeOnWireless 翻转、PreventUserIdleSystemSleep 释放）
- 13:54 web 收到 SIGTERM 死亡（推断：睡眠/唤醒周期中 macOS 对后台 daemon 的信号——两次死亡同签名支持此判定）

## 根因
macOS 睡眠/唤醒周期会向后台进程发信号（SIGTERM），--daemon（setsid+nohup）只能防 SIGHUP 防不了这个；web 无守护者 → 死后无人拉起。

## 修复（根治级）
launchd 守护（P2-2 遗留项转刚需）：
- plist：~/Library/LaunchAgents/com.rick.web.plist
- 设计要点：launchd **直接前台持有** rick web（不用 --daemon——daemon setsid 脱离会让 launchd 追踪失效；也不用 start-web.sh 幂等入口——脚本秒退会让 KeepAlive 误判，实测踩过）
- KeepAlive=true + ThrottleInterval=5：死亡 5s 内自动复活
- RunAtLoad=true：开机/登录自动启动
- env 注入：proxy 三件套 + no_proxy 含 mcli.sankuai.com（bug1 的教训固化）
- 实战验证：kill web → launchd 自动拉起（77314→77391）→ health ok ✅

## 关联
- start-web.sh 保留作为**手工兜底**（launchd 失效时用）
- 首次死亡（10-30 17:00）推断同因（当时未深挖——J3 裁决「不深挖」依据是修好 P0 后可自愈，本次 launchd 落地后彻底自愈）
