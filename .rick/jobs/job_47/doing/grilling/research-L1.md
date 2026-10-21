# research-L1：mc --code 账号切换 + token 额度付费

调研时间：2026-10-21 · 来源等级：**[实测]** > **[代码]** > **[官方文档]** > **[二手媒体]** > **[推测]**

---

## 主题 A：mcopilot（mc）账号体系

### A1. 凭据存储结构 [实测]
`~/.config/mcopilot-cli/.config.yaml`（本机 + dev HOME + locked HOME 三处内容一致，均为迁移前的同一账号）：

```yaml
REQUEST_TIMEOUT: 60
DEFAULT_COLOR: magenta
LAST_CHECK_TIME: 1789620618
AUTHORIZATION: <REDACTED-32hex>   # 32 位 hex，单 token
```

结论：**单账号扁平结构**——无 profile / 多账号 / 账号列表字段。切换账号 = 替换这一个 AUTHORIZATION 值。

### A2. mc CLI 本体 [实测 + 推测]
- 本机 **未安装 mc 二进制**（`which mc` 为空；源自 Linux 服务器 `sunquan20`，Mac 迁移后未带）
- 公网检索：`mcopilot cli` / `mc --code` / `mcli.sankuai.com` **无任何公开结果**（内部工具，不公开）
- `mc --code` 行为 [推测，置信中]：由 rick domain 历史记录反推——claude code 走同一网关：`ANTHROPIC_BASE_URL=mcli.sankuai.com` + `ANTHROPIC_AUTH_TOKEN` + `ANTHROPIC_CUSTOM_HEADERS`（动态 X-Timestamp/X-Client-Token）。即 `mc --code` 极可能 = 用 config.yaml 里的 AUTHORIZATION 作 `ANTHROPIC_AUTH_TOKEN` 启动 Claude Code 会话。
- 结论：**无法在本机验证 mc 原生的多账号/切换命令**；从配置结构看不存在账号切换能力，只能换 token 值。

### A3. 两人共用一个 token 的可行性 [实测 + 推断]
- 技术层面：token 是网关 Bearer 凭据，**任何持有者都能用**（本轮已用该 token 成功发起请求，仅在额度耗尽处被拦）——技术上"借用"可行
- 但：额度**按 token 所属人计**，所有消费归因到账号所有者；网关是否做并发会话检测/互踢无公开文档（置信低）
- 已知非公开点：网关做**客户端识别**（job_30 实测：裸 curl 带 base_url+token → 400），须带 `x-anthropic-billing-header` + `X-Working-Dir`；catpaw-proxy 已模拟该指纹

### A4. 额度与提额机制 [实测 — 本轮最硬的证据]
真实拦截报文（2026-10-21，本机有效 token 发起请求）：

```
您的2026-10-21额度已使用完毕，当前剩余额度为 0。额度将于次日 00:00:00自动重置。
如需申请提额：https://credit-hub.sankuai.com/credit-mgmt/credit-apply
```

- **额度按天计**，耗尽后次日 00:00 自动重置 → **"切换账号"的本质 = 借用他人的日额度**
- **官方提额通道**：`credit-hub.sankuai.com`（CatPaw Console，HTTP 200 可达，SPA 需 SSO 登录）→ `/credit-mgmt/credit-apply`
- 内部计费/充值细节公网不可达，需在 CatPaw Console 内查证（建议 human 登录后确认是否有"额度包/购买/审批"入口）

---

## 主题 B：rick 的 token 链路与切换机制（源码级 + A/B 实验）

### B1. 凭据读取链路 [代码]
`~/.rick/pi/agent/extensions/catpaw-proxy.ts`：
- **pi 启动时一次性**读 `.config.yaml` 的 `AUTHORIZATION:` → `pi.registerProvider("catpaw-proxy", { apiKey, baseUrl: "https://mcli.sankuai.com", api: "anthropic-messages", authHeader: true, headers: {"X-Working-Dir": cwd} })`
- 附带 `x-anthropic-billing-header` 注入（否则 mcli 400）
- ⚠️ `session_before_compact` 处理器**直接复用模块级 `apiKey`**（自己 fetch 调 mcli 做摘要）→ 若主请求走覆盖 token，**压缩摘要仍消耗 config.yaml 账号额度**

### B2. pi 鉴权优先级 [代码：pi-ai/dist/auth/resolve.js + core/provider-composer.js]
```
1. 单次调用 apiKey 覆盖
2. stored credential ← RuntimeCredentials：--api-key 运行时覆盖 > auth.json（/login 产物）
3. provider 配置 apiKey（扩展从 config.yaml 读的 token）← 仅当 1、2 都不存在
```
`--api-key` → `setRuntimeApiKey()` → `RuntimeCredentials.overrides` → `read()` 优先返回 override → **绕过扩展的 config.yaml token**

### B3. A/B 实验验证 [实测，决定性证据]
| 命令 | 结果 | 说明 |
|---|---|---|
| `pi --provider catpaw-proxy --model glm-5.3 --api-key bogus_token_test_123 -p 'say ok'` | `401 登录未授权，请先登录` | 网关收到的是**伪造 token** → 覆盖生效 |
| `pi --provider catpaw-proxy --model glm-5.3 -p 'say ok'` | 额度已使用完毕（见 A4） | 未覆盖 → 用 config.yaml 的有效 token |

⇒ **结论：`--api-key <他人token>` 可在完全不动 config.yaml 的前提下消费他人额度。**

### B4. rick 侧生效路径 [代码]
- 全局配置：`internal/config/config.go` → `PiExtraArgs []string`（json: `pi_extra_args`），落 `~/.rick/config.json`
- 合并点：`internal/runtime/cli.go` `mergeExtraArgs`；`internal/cmd/doing.go:97`；`internal/cmd/web.go:217,300`
- 生效时机（domain job_44 实测）：**CLI 会话每次调用重读 config → 改完下次即生效**；**web daemon 启动时烘焙 ExtraArgs → 必须重启 `rick web`**

---

## 主题 C：切换方案对比（供 L3 追问裁决）

| 方案 | 操作 | 影响面 | 生效 | 风险 |
|---|---|---|---|---|
| **A 改 config.yaml** | 替换 `AUTHORIZATION:` | mc CLI + rick 全部 | 新 pi 进程即生效；web 需重启 | 影响 mc 本体；需备份回滚 |
| **B pi_extra_args + --api-key** | `~/.rick/config.json` 加 `["--api-key","<token>"]` | 仅 rick | CLI 即时；web 重启 | token 出现在**进程参数**（`ps aux` 共享机可见）+ config.json 明文；compact 摘要仍走原账号 |
| **C pi /login** | `pi /login` → 选 catpaw-proxy → 粘贴 token | 仅 rick（auth.json） | 立即、持久 | 需实机验证（扩展 apiKey provider 的 login 提示路径）；compact 同上 |
| **D 官方提额** | credit-hub.sankuai.com 申请 | 本账号 | 审批后 | 无合规风险，但需审批 |

---

## 主题 D：获得 token 额度的付费方式（公网，2026-10 官方价）

### D1. 按量付费 API（可直接接 rick/pi）
| provider | 模型 | 输入 | 输出 | 缓存读 | 备注 |
|---|---|---|---|---|---|
| DeepSeek | deepseek-v4-pro | ¥3/M | ¥6/M | ¥0.025/M | 峰谷价：非高峰约为一半；1M 上下文 [官方 pricing] |
| DeepSeek | deepseek-flash | ¥1/M | ¥2–4/M | ¥0.02/M | 源间不一致（¥2 vs ¥4）[官方+二手] |
| （现有 catpaw-proxy 内价）| GLM-5.3 | $1.40/M | $4.40/M | $0.26/M | 扩展里的参考价，非官方账单 |

### D2. 订阅制（含 coding agent 额度）
| 方案 | 价格 | 额度规则 | 能接 rick/pi? |
|---|---|---|---|
| **Kimi Code** | 月付 Go ¥49 / Plus ¥99 / Pro ¥199 / Max ¥699 / Ultra ¥1,399（年付折 ¥39/79/159/559/1,119） | 仅 Plus 及以上含 coding；**只保留 5 小时滚动窗口，取消周限额**；CLI/IDE/第三方请求共用同一额度 | 官方 K2/K3 API 可接（额度池与套餐是否同一需确认）[官方帮助页] |
| **智谱 GLM Coding Plan** | 历史 Lite/Pro 档，2026-07-30 改「积分制」；**当前价格需在 bigmodel.cn 订阅页确认**（"20元/月"是 2025-09 GLM-4.5 期推广价） | 5 小时 + 周双重限额；耗尽只能等下一周期，不扣资源包；**额度仅在官方支持工具内有效** | ⚠️ 额度可能只对官方工具生效，第三方 API 走按量计费 [官方 docs] |
| **Claude Pro/Max** | Pro $20/月（年付 $17）；Max 5x $100/月；Max 20x $200/月 | Claude Code 含在付费套餐内，受滚动 5 小时 + 周限额，与聊天共用；API 另算（Max 附送 Console 额度，不含 Claude Code） | 订阅不可直接接 pi；需 API key 按量 [官方 pricing] |
| **ChatGPT / Codex** | Plus $20/月；Pro 分档（Pro 200 / Pro 500=$500/月，帮助中心口径）；Pro≈5x Plus 用量 | Codex 含在套餐内；官方不给绝对 token 数 | 订阅不可直接接 pi；OpenAI API 另按量 [官方] |
| 阿里云 Token Plan | Standard ¥139/月、Pro ¥499/月（单一来源，需核实）；团队版 25k/100k/250k Credits/席/月 | Credits 折算输入/输出 | 可接（OpenAI 兼容）[官方 docs + 单源定价] |

### D3. 性价比取向（粗判）
- **每天数百万 token 的重度 coding agent**：按量 API 里 DeepSeek（v4-pro/flash）最低；订阅制里 Kimi Plus ¥99 或 GLM Coding Lite 若额度可被 rick 消费则最划算，但**均存在"额度仅限官方工具"条款**——这是能否落到 rick 的**关键未知项**（需实测/问客服）。
- 纯 API 可接性上最稳的仍是 DeepSeek 按量 + 内网 catpaw（当前组合）。

---

## 待 human 裁决的判断节点（供 L3）
1. 产出形态：纯调研报告 vs 是否要 rick 落地「账号切换」功能
2. 切换方案选型：A / B / C / D（含合规与 token 暴露权衡）
3. 合规边界：借用同事 token 是否接受，报告如何呈现风险
4. 付费调研范围：只列「能直接接 rick」的，还是也列订阅制 coding agent
5. 预算/日均消耗档位 → 决定首推方案
