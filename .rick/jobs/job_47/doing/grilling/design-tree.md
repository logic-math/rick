# 设计树：rick 切换 mc --code 账号 + token 额度付费调研

## 根层（L1）OKR

### O（目标）
产出一份**可验证的调研报告**（落盘 doing/）：①「rick 如何切换 mc --code（mcopilot）账号信息、以使用其他人的 token 额度」的可行方案与具体操作路径；②「获得各种 token 额度的付费方式」对比清单（美团内部 + 公网替代），置信度标注来源。

### KR 集（充分性推导：KR1 ∧ KR2 ∧ KR3 ∧ KR4 ⟹ O）
- **KR1 账号体系事实**：摸清 mcopilot CLI（mc）账号体系——登录/token 存储机制（`~/.config/mcopilot-cli/.config.yaml` 结构、token 字段）、`mc --code` 的行为、多账号/切换账号的原生支持情况
- **KR2 rick 集成链路**：摸清 rick → catpaw-proxy → mcopilot 的 token 读取链路，确定「切换账号后 rick 如何生效」（换 config.yaml 即可？需重启 web daemon？）
- **KR3 切换方案**：给出 ≥2 个可行的账号切换方案（含原生命令/手动 token 管理/自动化脚本），含操作步骤与风险（封号/合规）
- **KR4 付费调研**：token 额度获得方式——美团内部 mcopilot 计费/充值/额度申请机制 + 公网 provider（DeepSeek/GLM/Kimi 等）价格对比，含推荐
- **KR5 报告落盘**：调研报告写入 `doing/grilling/` 之外的主产出文件（requirement 追加结论），human 可直接按报告操作

---

## L2/L3：判断节点与 human 裁决（2026-10-21）

**本层 pipeline**：mcopilot 凭据源（codelab/本机 config.yaml）→ 账号仓（token 保险箱）→ 切换脚本（方案 A：改写 config.yaml）→ rick 生效（新 pi 进程 / web 重启）→ 验证探针

| # | 判断节点（隐含前提） | human 裁决 | 落地决策 |
|---|---|---|---|
| Q1 | 产出形态：纯报告 vs rick 内置功能 | **a** | 纯报告 + 独立脚本；**不改 rick 内核**（rick 只需知道脚本路径） |
| Q2 | 切换方案 A/B/C/D | **A** | 改写 `~/.config/mcopilot-cli/.config.yaml` 的 `AUTHORIZATION`（mc CLI 与 rick 同时生效；新 pi 进程即时；web 需重启） |
| Q3 | 合规边界 + 取 key 途径 | 借用同事 codelab 的 key（视为放权）；**必须保留自己的 key 可切回**；要临时脚本 | 脚本含 token 保险箱（save/use/restore）+ 切换前自动快照；报告含风险章节 |
| Q4 | 付费调研范围 | **都列出来** | 按量 API + 订阅制 coding agent 全列，标注「能否直接接 rick」 |
| Q5 | 预算档位 | **B（中度 ¥300–500/月）** | 首推 DeepSeek v4-pro 按量（¥3/¥6 每百万）为公网主选；内网走 credit-hub 提额 |

### 新增事实（本轮实测）
- codelab 机器 `10.128.147.9`：**ping 通（24ms），22/8420 端口不可用**（22 过滤、8420 refused、其他常见端口全闭）；`jumper.sankuai.com:22` 可达但需凭据 → **本机无法直接进入该 codelab**
- 需 human 在 codelab 终端执行取 key 命令（已提供一行命令）

### 叶子层实现（四维度）
1. **代码实现**：新建 `~/.rick/accounts/switch.sh`（bash 3.2 兼容）——子命令 `save|use|list|show|restore|verify`；token 落 `~/.rick/accounts/tokens/<name>`（chmod 600）；每次 `use` 前自动快照当前 token 为 `_prev`
2. **文件结构**：`~/.rick/accounts/{switch.sh, tokens/}`；报告 `doing/research-report.md`；结论追加 `doing/requirement.md`
3. **工具调用**：`switch.sh save me` / `switch.sh use <同事名>` / `switch.sh restore` / `switch.sh verify`；生效：CLI 下次调用即生效、`rick web` 需重启
4. **环境依赖**：bash、`pi`（`~/.rick/pi/agent/runtime/node_modules/.bin/pi` 或 PATH）、`~/.config/mcopilot-cli/.config.yaml`；无需新增依赖
