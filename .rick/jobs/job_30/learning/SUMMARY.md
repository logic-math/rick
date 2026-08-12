APPROVED: true

# Job job_30 执行总结

## 执行概述

**项目目标**: 将 rick 的 agent runtime 从 claude code 迁移到 pi（@earendil-works/pi-coding-agent），全量移除 claude code；并建设 pi 作为受控执行后端的托管能力（init-pi）。

**实际完成**: 迁移主体完成 + 命令级端到端验证通过 + init-pi 托管能力建成。13 个提交，跨 config/piagent/cmd/executor/scripts/docs。

**整体评价**: ⭐⭐⭐⭐⭐ (5/5)

## 关键成就

1. **Phase 1 1:1 迁移完成**：新建 `internal/agent/piagent`（Executor JSONL 解析 + CallCLI 统一抽象 + FindBinary），删除 `internal/agent/claudecode`，config `claude_code_path`→`pi_path`+`pi_extra_args`，接线 13 处调用点，mock_agent.py 适配 pi `--mode json`。

2. **真实 LLM 端到端验证**：用真实 pi v0.84.1 + DeepSeek 校准解析器（修复 FinalMessage bug：user/assistant message_end 区分），`TestRealPi_RealToolCall` 真实工具调用通过。命令级验证 6/7 命令真实跑通（plan/doing/easy/learning/human-loop/ctrl），dream 链路验证但 LLM 量大超时。

3. **`rick tools init-pi` 托管能力**：cobra 子命令，幂等安装 pi + pi-subagents + pi-web-access + Tokyo Night 主题，最终 `pi list` 验证全部生效。主题策略经 3 轮迭代定为"仅 fresh install 才设，尊重用户偏好"。node 作为用户管理依赖，缺失则终止。

4. **pi 托管理念确立**：README 写明 rick 把 pi 当受控执行后端，用户不感知 pi，所有 pi 配置通过 rick 引导完成。演进方向定为配置目录隔离（`~/.rick/pi` + `PI_CODING_AGENT_DIR`，代码待实现）。

## 问题与教训

### 问题1: easy 命令 `--session` vs `--session-id` 语义误判

**根本原因**: loop_2 研究 brief 说 pi `--session` "接受 path 或 id"，但未标"加载 vs 创建"语义。迁移时误把 claude 的 `--session-id`→pi 的 `--session`，而 pi 的 `--session` 是加载已有会话（找不到报错），`--session-id` 才是创建新会话。

**解决方案**: 改回 `--session-id`（pi 同时支持）。见 domain/bugs.md。

**经验教训**: 研究 brief 的 flag 描述必须单独验证语义（加载 vs 创建），不能直接信"接受 id"。验证方法：用不存在的 id 测，看输出是 "No session found"（加载）还是 "creating a new session"（创建）。

### 问题2: subagent 扩展"假成功"安装

**根本原因**: 用 `pi install <本地 .ts 源码目录>` 装 subagent example，settings.json 写了路径但 pi loader 不认（无 package.json）→ `pi list` 假装装上，工具从未注册。

**解决方案**: 改用 `pi install npm:pi-subagents`（标准 npm 包）。见 domain/bugs.md。

**经验教训**: pi 扩展必须用 npm 包；`pi install` 后必须双重验证（`pi list` + 真实工具调用），捕获"假成功"。沉淀为 `pi-extension-install-verification` skill。

### 问题3: 主题策略与 node 安装策略多次反复

**根本原因**: 前期 Grilling 没覆盖"主题何时设"和"node 谁装"的决策点，导致主题策略 3 轮迭代、node 安装方案 2 轮（从"自动装"到"用户管理"）。

**解决方案**: 主题定为"仅 fresh install 设，尊重用户"；node 定为"用户管理，缺失终止"。

**经验教训**: Grilling 阶段要把"环境依赖归属""配置覆盖策略"类决策点问清，避免实现期反复。零重试设计原则不仅适用于 task 分解，也适用于设计决策。

### 问题4: pi 无法复用美团内网 anthropic 网关

**根本原因**: 美团网关（mcli.sankuai.com）做客户端识别，光有 base_url + token + custom headers 直接 curl → 400。claude code 能通是因为有额外认证（疑似 User-Agent/SDK 签名），pi 复制不了。

**解决方案**: 用公网 provider（DeepSeek，已验证可用）。catpaw-pilot 支持 pi-coding-agent 但 enabled:false，启用机制不公开。

**经验教训**: 企业内网网关常做客户端识别，非官方支持的 agent 难以复用。迁移前先测目标 provider 在该环境是否可用，别等迁移完才发现跑不了。

## 知识沉淀清单

- [x] `.rick/skills/pi_runtime_verification_skill/skill.md` - pi runtime 迁移的命令级端到端验证（含 flag 语义验证）
- [x] `.rick/skills/pi_extension_install_verification_skill/skill.md` - pi 扩展安装的真实生效验证（捕获假成功）
- [x] `.rick/loops/agent-runtime-bootstrap-loop.md` - agent runtime 初始化/迁移/重装的 bootstrap 循环
- [x] `.rick/domain/bugs.md` - 追加 3 条（pi 扩展假成功、--session 语义、message_end role 过滤）
- [x] `.rick/domain/pi-runtime.md` - 新建，pi runtime 事实知识（配置、加载机制、网关不可复用、deepseek 配置）

## 遗留与后续

1. **pi 配置目录隔离未实现**：README 写了 `~/.rick/pi` + `PI_CODING_AGENT_DIR` 方向，代码待实现（注入 env 到 piagent.CallCLI/Execute + init-pi 改用隔离目录装扩展/写设置）
2. **dream 命令未完整端到端验证**：LLM 量大（5 job 反思）deepseek 超时，链路已验证但 LLM 完成度未确认
3. **分支未合并**：提交在 `feat/job_30-pi-migration`，未 push/merge 到 main
4. **pi 走美团内网网关**：catpaw pi-coding-agent 启用方式待内部文档
