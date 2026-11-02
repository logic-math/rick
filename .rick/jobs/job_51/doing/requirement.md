# 需求（job_51）

用户报告：web UI 创建 super session（MT_INFER 工作区）报错「缺失 .rick/loops/super-loop.md」。

执行中发现并修复三层问题：
1. **super 资产未部署到外部 workspace**：super session 资产检查是逐工作区的（<ws>/.rick/loops/super-loop.md + gen_env_skill 等），只存在于 rick 树 → 部署 9 个资产到 MT_INFER
2. **superopt 调度器跨树解析失败**：resolveRepoRoot 只从 cwd 上溯找 super/superopt——pi worker cwd=外部 workspace → No module named superopt（实测复现）→ 修复：cwd 上溯失败后回退到 rick 二进制自身位置（os.Executable + EvalSymlinks 穿 bin/rick→releases/current/rick 链）；附带修 parseScriptFlag 对「PORT=6137 + --port $PORT」写法的解析（字面量正则命中 $PORT 提前返回 → 变量赋值回退）→ 65c2cd1 已发布生产
3. **生产实例 proxy 杀不死之谜**：kill web 后 launchd 5 秒内拉回带 proxy 的新实例——真源头是 ~/Library/LaunchAgents/com.rick.web.plist 的 EnvironmentVariables **硬编码** http_proxy/https_proxy（KeepAlive=true）→ 删变量 + bootout/bootstrap 重载；顺带修 release prodEnv() 不再透传 *_proxy（08b5782 已推 GitHub）

后续：git push GitHub（104+105 条积压提交，proxy 403 根因=plist 灌进会话 env，env -u 绕过后直连 200 成功推送）。
