---
name: agent-runtime-bootstrap-loop
trigger: "当需要初始化/迁移/重装 rick 的 agent runtime（pi）及其扩展时触发（如 rick tools init-pi、版本升级、runtime 迁移）"
scope: "全局"
---

# Loop: agent runtime bootstrap

## 目标（Goal）

初始化/迁移/重装 rick 的 agent runtime（pi）及其扩展，使其端到端可用。
- 成功标准：`rick tools init-pi` 全 ✅（含最终 verification），所有必需扩展真实生效
- 自评命令：`rick tools init-pi`，自评输出：末行 `✅ pi environment ready`

## 上下文管理（Context Management）

- 保留：上次 init-pi 状态（pi 版本、已装扩展、settings.json theme）、当前 `~/.rick/config.json` 的 `pi_extra_args`
- 压缩：研究 brief 的置信度与已验证项摘要（迁移场景）
- 遗忘：已回滚的改动、临时调试输出

## 可调用工具（Tool Access）

- `pi install` / `pi list`：安装/列举扩展（约束：扩展必须用 npm 包，不用本地源码）
- `curl|sh`：安装 pi（官方安装器）
- `rick tools init-pi`：统一引导入口
- 权限边界：不装 node（用户管理）；pi 已存在则不检查 node、不动 theme

## 产出评估（Output Evaluation）

调用验证 skill：`.rick/skills/pi_extension_install_verification_skill/skill.md`

| 检查项 | 验证方法 | 通过标准 |
|--------|----------|----------|
| pi 可用 | `command -v pi && pi --version` | 非空版本 |
| 扩展注册 | `pi list \| grep <pkg>` | 每个必需扩展都有 |
| 扩展真生效 | 真实 LLM 调用 grep `"toolName":"subagent"` 等 | 工具被调用（非 0） |
| init-pi 全绿 | `rick tools init-pi` | 全 ✅ 含 verification |

- 全部通过 → 成功退出
- 扩展"装了但没生效"→ 检查是否用了本地源码而非 npm 包，改用 npm 包重装，重新评估

## 停止标准（Termination Condition）

- **成功退出**：init-pi 全 ✅（含最终 verification），所有必需扩展真实生效
- **失败退出**：连续 3 轮同一扩展装不上 / pi 完全装不上
- **优雅退出**：迭代达上限 3 轮、连续 2 轮同一错误、人类要求停止

---

## 工作流（Step 0-5）

### Step 0：环境确认 + Domain 搜索

**0.1 依赖准备**（硬约束，缺失则报错停止）：

| 依赖项 | 确认命令 | 要求 |
|--------|----------|------|
| node (≥22.19.0) | `command -v node && node --version` | 已安装（用户管理，rick 不装） |
| npm | `command -v npm` | 已安装 |
| pi | `command -v pi` | 已装则跳过安装；未装需 node/npm 就绪 |

node/npm 是用户管理的环境依赖。仅当 pi 未装时检查 node/npm，缺失则终止并提示用户自行装 node（`https://nodejs.org/`）。pi 已装则假定环境就绪。

**0.2 Domain 搜索**：搜索 `.rick/domain/` 下相关文件（bugs.md / pi-runtime.md）。遇到问题优先搜 domain。

### 读取上下文

- 上次 init-pi 的状态（pi 版本、已装扩展、settings.json theme）
- 当前 `~/.rick/config.json` 的 `pi_extra_args`（provider/model/api-key）
- 若是迁移场景：研究 brief（如 loop_2 pi 映射表）的置信度与已验证项

### 启动 Sub Agent 执行工作流

```
[Main Agent]
   ├─ SPAWN Sub Agent: 安装 pi + 扩展（携带：config + 已装状态）
   │     ├─ Step A: 安装 pi（curl|sh，若未装）
   │     ├─ Step B: 安装扩展（pi install npm:<pkg>，每步幂等 check-then-install）
   │     ├─ Step C: 主题（仅 fresh pi install 才激活）
   │     └─ COMMIT
   └─ Main Agent 执行 Step 4 产出评估
```

**Sub Agent：Step A 安装 pi**
- 加载：无（直接调安装器）
- 精确命令：
  ```bash
  curl -fsSL https://pi.dev/install.sh | sh
  command -v pi && pi --version  # 确认
  ```
- 产出：pi 在 PATH

**Sub Agent：Step B 安装扩展**
- 加载 skill：`.rick/skills/pi_extension_install_verification_skill/skill.md`
- 精确命令（每个扩展幂等）：
  ```bash
  pi list | grep <pkg> || pi install npm:<pkg>
  # rick 依赖: pi-subagents, pi-web-access, (主题: @wishx127/pi-tokyo-night)
  ```
- 产出：扩展注册到 settings.json

**Sub Agent：Step C 主题**（仅 Step A 本次新装了 pi 才做）
- 精确命令：
  ```bash
  pi list | grep pi-tokyo-night || pi install npm:@wishx127/pi-tokyo-night
  # 写 settings.json theme=tokyo-night-dark（若当前是默认 dark/light/空）
  ```
- 策略：pi 已存在则不动 theme（尊重用户偏好）

**Sub Agent：COMMIT**
1. `git add` + `git commit`（含 job ID）
2. 运行 `rick tools init-pi`，循环直到全 ✅

## ⚠️ 关键约束

1. **扩展必须用 npm 包**，不用本地 .ts example 源码（无 package.json，loader 不认，pi list 假装装上但工具不注册）
2. **node 是用户管理**，rick 不装，缺失则终止
3. **主题仅 fresh install 才设**，pi 已存在则尊重用户配置
4. **最终必须 verify**：`pi list` + 真实工具调用双重确认，捕获"假成功"
