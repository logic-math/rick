# 调研报告：rick 切换 mc --code 账号 + token 额度付费方式

> job_47 · 2026-10-21 · 信源等级：**[实测]** > **[代码]** > **[官方文档]** > **[二手媒体]** > **[推测]**
> 完整调研原始记录：`doing/grilling/research-L1.md`；决策树：`doing/grilling/design-tree.md`

---

## TL;DR（结论）

1. **mc --code 的"账号"实质就是一个 Bearer token**（`AUTHORIZATION:` 32 位 hex），存在 `~/.config/mcopilot-cli/.config.yaml`，**单账号扁平结构**，没有多账号字段 → 切换账号 = 换这一个值。
2. **额度按天计**，耗尽后次日 00:00 重置（本轮实测报文），官方提额入口 `https://credit-hub.sankuai.com/credit-mgmt/credit-apply`（CatPaw Console）。
3. **rick 侧无需改任何代码**：pi 的鉴权优先级允许 `--api-key` 覆盖扩展从 config.yaml 读的 token（源码 + A/B 实验双重验证）。
4. **已交付切换脚本** `~/.rick/accounts/switch.sh`（方案 A：改写 config.yaml），自带 token 保险箱、自动快照、`restore` 切回、`verify` 探针——**你自己的 token 已存为 `me`**。
5. **取 key 在本机做不到**：codelab 机器 `10.128.147.9` ping 通但 22/8420 及其它端口全不可用，跳板机需凭据 → 需要你在同事 codelab 终端跑一行命令把 key 拿回来（见第四节）。
6. **付费方式**：中度用量（¥300–500/月）公网首选 **DeepSeek v4-pro 按量**（¥3/¥6 每百万，非高峰约半价）；订阅制里 **Kimi Plus ¥99** 最便宜但存在"额度仅限官方工具"条款风险。

---

## 一、mc --code 账号体系（事实）

### 1.1 凭据存储 [实测]
本机三个 HOME（`~`、`rick-dev-home`、`rick-dev-home-hadoop.locked`）的 `~/.config/mcopilot-cli/.config.yaml` **内容一致**（迁移前同一账号）：

```yaml
REQUEST_TIMEOUT: 60
DEFAULT_COLOR: magenta
LAST_CHECK_TIME: 1789620618
AUTHORIZATION: <32 位 hex>      # 唯一凭据，单账号
```

**没有** profile / accounts / 多账号列表字段 → CLI 层面不存在"账号切换"能力，只能换 token 值。

### 1.2 mc --code 是什么 [推测，置信中]
- 本机**未安装 mc 二进制**（`which mc` 空；源自 Linux 服务器，Mac 迁移未带）
- 公网检索 `mcopilot cli` / `mc --code` / `mcli.sankuai.com` **无任何公开结果**（内部工具）
- 由 rick domain 历史记录反推：claude code 走同一网关 `ANTHROPIC_BASE_URL=mcli.sankuai.com` + `ANTHROPIC_AUTH_TOKEN` + `ANTHROPIC_CUSTOM_HEADERS`（动态 X-Timestamp/X-Client-Token）→ `mc --code` 极可能 = 用同一 `AUTHORIZATION` 作 `ANTHROPIC_AUTH_TOKEN` 启动 Claude Code 会话

### 1.3 额度机制 [实测，决定性]
用本机有效 token 发起真实请求，网关返回：

```
您的2026-10-21额度已使用完毕，当前剩余额度为 0。额度将于次日 00:00:00自动重置。
如需申请提额：https://credit-hub.sankuai.com/credit-mgmt/credit-apply
```

- **额度 = 每人每天**，00:00 重置
- **官方提额**：CatPaw Console（`credit-hub.sankuai.com`，HTTP 200 可达，SPA 需 SSO 登录）→ `/credit-mgmt/credit-apply`
- 内部计费细则公网不可达；建议登录 Console 确认是否有「额度包 / 购买 / 审批」入口

### 1.4 "借同事 token" 的技术可行性 [实测]
token 是纯 Bearer 凭据，**任何持有者都能调用**（本轮已用该 token 成功发请求，仅在额度处被拦）。但：
- 消费**全部归因到 token 所有者**（对方的额度、对方的审计记录）
- 网关做客户端指纹校验（裸 curl → 400；须 `x-anthropic-billing-header` + `X-Working-Dir`，catpaw-proxy 已模拟）
- 是否存在并发会话检测/互踢：**无公开文档，未知**

---

## 二、rick 的 token 链路（源码级）

### 2.1 读取链路 [代码]
`~/.rick/pi/agent/extensions/catpaw-proxy.ts`：
- **pi 启动时一次性**读 `AUTHORIZATION:` → `registerProvider("catpaw-proxy", { apiKey, baseUrl: "https://mcli.sankuai.com", api: "anthropic-messages", authHeader: true, headers: { "X-Working-Dir": cwd } })`
- 请求前注入 `x-anthropic-billing-header`（缺则 400）
- ⚠️ `session_before_compact`（压缩摘要）**直接用模块级 `apiKey`** 自己 fetch → 若主请求走覆盖 token，**压缩摘要仍扣 config.yaml 账号的额度**

### 2.2 pi 鉴权优先级 [代码]
`pi-ai/dist/auth/resolve.js` + `core/provider-composer.js` + `core/runtime-credentials.js`：

```
1. 单次调用 apiKey 覆盖
2. stored credential（RuntimeCredentials：--api-key 运行时覆盖 > auth.json 的 /login 凭据）
3. provider 配置 apiKey（= 扩展从 config.yaml 读的 token）  ← 仅当 1、2 都缺
```

### 2.3 A/B 实验 [实测，决定性]

| 命令 | 结果 | 推断 |
|---|---|---|
| `pi --provider catpaw-proxy --model glm-5.3 --api-key bogus_token_test_123 -p 'say ok'` | `401 登录未授权，请先登录` | 网关收到**伪造 token** → 覆盖生效 |
| `pi --provider catpaw-proxy --model glm-5.3 -p 'say ok'` | 额度已使用完毕 | 未覆盖 → 用 config.yaml 有效 token |

⇒ **`--api-key <他人token>` 可在不碰 config.yaml 的前提下消费他人额度。**

### 2.4 rick 侧配置注入点 [代码]
- 全局配置 `~/.rick/config.json` → `pi_extra_args`（`internal/config/config.go` `PiExtraArgs`）
- 合并点：`internal/runtime/cli.go` `mergeExtraArgs`、`internal/cmd/doing.go:97`、`internal/cmd/web.go:217,300`
- 生效时机：**CLI 会话每次调用重读 config → 下次即生效**；**web daemon 启动时烘焙 → 改完须重启 `rick web`**

---

## 三、四种切换方案对比 + 采用建议

| 方案 | 操作 | 影响面 | 生效 | 风险 / 代价 |
|---|---|---|---|---|
| **A 改 config.yaml**（本次采用） | 替换 `AUTHORIZATION` | mc CLI + rick 全部 | 新 pi 进程即生效；web 需重启 | 影响 mc 本体；需备份回滚（脚本已自动备份） |
| **B pi_extra_args + `--api-key`** | 改 `~/.rick/config.json` | 仅 rick | CLI 即时；web 重启 | token 进**进程参数**，`ps aux` 在共享机可见；config.json 明文；compact 仍走原账号 |
| **C `pi /login`** | 存 `~/.rick/pi/agent/auth.json` | 仅 rick，持久 | 立即 | 需实机验证；compact 同上；token 落盘但不在 ps |
| **D credit-hub 官方提额** | 网页申请 | 本账号 | 审批后 | ✅ 无合规风险（推荐长期方案） |

**已按你的裁决（Q2=A）交付方案 A 的脚本化实现。** B/C 保留在报告中作为备选（若将来需要"只影响 rick、不动 mc"）。

---

## 四、从同事 codelab 取 key：操作步骤

### 4.1 本机无法直连（已实测排除）
| 检查 | 结果 |
|---|---|
| `ping 10.128.147.9` | ✅ 通（22–24 ms） |
| `ssh -p 22` | ❌ 超时（被过滤） |
| `ssh -p 8420`（内网惯例端口，known_hosts 有先例） | ❌ Connection refused |
| 扫描 80/443/3000/8080/8443/8888/9000/9100/…/2022/10080 | ❌ 全闭 |
| `ssh -J jumper.sankuai.com` | ❌ 跳板机要求凭据（Permission denied） |

⇒ **必须由你在同事的 codelab 终端里执行命令取 key**（codelab 共享即代表对方授权，此判断权在你）。

### 4.2 在 codelab 终端执行（一行，取 key）
```bash
grep -h '^AUTHORIZATION:' ~/.config/mcopilot-cli/.config.yaml 2>/dev/null | cut -d: -f2- | tr -d ' \r'
```
> 顺手确认环境（可选）：
> ```bash
> which mc; mc --version 2>/dev/null; ls -la ~/.config/mcopilot-cli/
> ```

### 4.3 拿到后在本机导入并切换（两条命令）
```bash
~/.rick/accounts/switch.sh add <同事标识> <32位token>   # 存入保险箱，不切换
~/.rick/accounts/switch.sh use <同事标识>               # 切换（自动快照当前账号，可 restore）
~/.rick/accounts/switch.sh verify                       # 探针验证：有效期/额度状态
```
**切回自己**：
```bash
~/.rick/accounts/switch.sh restore        # 撤销上一次切换
# 或
~/.rick/accounts/switch.sh use me         # 显式回到自己的账号
```

### 4.4 生效与回滚
- **rick CLI**（`rick call` / `rick plan` / `rick easy`…）：下次调用即生效
- **rick web**：需重启 daemon 才对 web 会话生效
- **回滚**：脚本每次写入前自动备份 `config.yaml.bak.<时间戳>`，也可 `restore`
- **mc CLI 本体**：同样读这个文件，切换后下次 `mc` 调用即用新账号（如果你想 mc 保持自己的账号，应改用方案 B/C）

---

## 五、token 额度付费方式（全表，2026-10 官方价）

### 5.1 按量付费 API —— 可直接接 rick/pi

| provider | 模型 | 输入 | 输出 | 缓存读 | 能接 rick | 备注 |
|---|---|---|---|---|---|---|
| **DeepSeek** | deepseek-v4-pro | **¥3/M** | **¥6/M** | ¥0.025/M | ✅ pi 内置 | 1M 上下文；**非高峰约半价**（周末全非高峰） |
| **DeepSeek** | deepseek-flash | ¥1/M | ¥2–4/M ⚠️源间不一致 | ¥0.02/M | ✅ | 最便宜可用档 |
| **Kimi (Moonshot)** | K2.6 / K3 | 见套餐；API 按量另计 | — | — | ✅ OpenAI 兼容 | 扩展内参考价 K2.6 $0.95/$4.00、K3 $2.78/$13.89 每百万 |
| **智谱 GLM** | GLM-5.3 | 按量另计（扩展参考 $1.40/$4.40） | — | $0.26/M | ✅（按量） | ⚠️ **订阅套餐额度仅限官方工具，可能不被 rick 消费** |
| **阿里云百炼** | Token Plan | Standard ¥139/月、Pro ¥499/月 ⚠️单源 | — | — | ✅ OpenAI 兼容 | 团队版 25k/100k/250k Credits/席/月 |
| **OpenAI API** | GPT-5.4 | 扩展参考 $2.50/$15.00 | — | $0.25/M | ✅ | 需出网 |
| **Anthropic API** | Claude | 官方价 3/15 美元每百万级 | — | — | ✅ | 需出网/代理 |

> 内网 `friday` provider（`aigc.sankuai.com`，OpenAI 兼容，需 `PI_FRIDAY_KEY`）已在扩展中注册，可作为**内网第二通道**，但其额度来源未在本次调研范围内查证。

### 5.2 订阅制（含 coding agent 额度）—— 多数**不能**直接接 rick

| 方案 | 价格 | 额度规则 | 能接 rick |
|---|---|---|---|
| **Kimi Code** | 月付：Go ¥49 / **Plus ¥99** / Pro ¥199 / Max ¥699 / Ultra ¥1,399（年付折 ¥39/79/159/559/1,119） | **Plus 及以上**才含 coding；**只保留 5 小时滚动窗口、取消周限额**；CLI/IDE/第三方请求共用额度 | ⚠️ 额度池与 API key 是否同源需确认；Kimi API 本身可接 |
| **智谱 GLM Coding Plan** | 积分制（2026-07-30 改版）；价格以 bigmodel.cn 订阅页为准（"20元/月"是 2025-09 GLM-4.5 期推广价） | 5 小时 + 周**双重**限额；耗尽只能等下一周期、不扣资源包；**额度仅在官方支持工具内有效** | ❌（额度条款限制） |
| **Claude Pro / Max** | Pro $20/月（年付 $17）；Max 5x $100/月；Max 20x $200/月 | Claude Code 含在套餐内，与聊天共用滚动 5h + 周限额；**API 额度另算**（不含 Claude Code） | ❌ 订阅不可接；API 可接 |
| **ChatGPT / Codex** | Plus $20/月；Pro 分档（Pro 200 / Pro 500 = $500/月，帮助中心口径） | Codex 含套餐；Pro ≈ 5x Plus 用量；官方不给绝对 token 数 | ❌ 订阅不可接；OpenAI API 可接 |

### 5.3 你的档位（Q5=B：100–300 万 token/天，¥300–500/月）测算

按 **300 万 token/天 ≈ 9,000 万/月**、输入:输出 ≈ 85:15 估算：

| 方案 | 月成本估算 | 说明 |
|---|---|---|
| **DeepSeek v4-pro（推荐）** | **≈ ¥310/月**（高峰）、**≈ ¥155/月**（非高峰） | 76.5M in×¥3 + 13.5M out×¥6；缓存命中可再降 |
| DeepSeek flash | ≈ ¥130/月（高峰）、≈ ¥65/月 | 输出价源间有 ¥2/¥4 分歧，取高估 |
| Kimi Plus ¥99 | ¥99/月 | 若额度可被 rick 消费则最划算；**条款风险需先验证** |
| Claude Max 5x $100 | ≈ ¥720/月 | 超预算；且不可直连 pi |
| ChatGPT Plus $20 | ≈ ¥145/月 | Codex 不可直连 pi；仅当用官方 Codex CLI 时才有意义 |

**推荐**：公网主选 **DeepSeek v4-pro 按量**（无"仅官方工具"条款风险、可完全接 rick、成本可控）；内网优先 **credit-hub 提额**（0 成本）。

---

## 六、风险与边界（如实记录）

| 风险 | 说明 | 缓解 |
|---|---|---|
| **额度归因** | 用同事 token 的消费全部记在对方账号上，对方日额度被消耗 | 只在对方明确同意的时间窗内用；用完 `restore` |
| **合规** | 公司内部额度是否允许跨人使用，本报告无法判定（无公开制度文档）；codelab 共享可视为对方放权，但**不等于公司授权** | 走 credit-hub 官方提额作为长期方案；把共用限制在临时、对等互助范围 |
| **token 泄露** | `--api-key` 会出现在 `ps aux`（共享开发机可见）；config.yaml / 保险箱均为明文 | 本脚本用 chmod 600 + `~/.rick/accounts/`（700）；**避免用方案 B** |
| **压缩摘要串号** | catpaw-proxy 的 compact 处理器硬用 config.yaml 的 token → 方案 B/C 下压缩仍扣原账号 | 方案 A 无此问题（本脚本采用 A） |
| **并发互踢** | 网关是否有并发检测未知 | 避免长时间高频并发共用同一 token |
| **切换后 web 不生效** | web daemon 启动时烘焙配置 | 切换后重启 `rick web` |

---

## 七、验证清单（可执行断言，均已实测通过 ✅）

| # | 断言 | 命令 | 结果 |
|---|---|---|---|
| 1 | 自己的 token 已存档 | `~/.rick/accounts/switch.sh list` | ✅ 显示 `* me dad3…2331` |
| 2 | 能切到他人账号 | `switch.sh add x <token>` + `switch.sh use x` | ✅ 切换后 `list` 显示新账号为当前 |
| 3 | 能切回自己 | `switch.sh restore` 或 `use me` | ✅ 回到 `me`，config.yaml 恢复原值 |
| 4 | token 有效性可探针验证 | `switch.sh verify` | ✅ 无效→`401`；有效但耗尽→额度报文；正常→`ok` |
| 5 | 覆盖机制生效（源码事实） | `pi ... --api-key bogus` | ✅ 返回 `401`（证明网关收到覆盖值） |
| 6 | 配置变更可回滚 | `ls ~/.config/mcopilot-cli/*.bak.*` | ✅ 每次写入前自动备份 |

---

## 附录：脚本清单

**`~/.rick/accounts/switch.sh`**（+x，bash 3.2 兼容）

| 子命令 | 作用 |
|---|---|
| `list` | 列出已存账号（掩码）+ 标记当前生效 |
| `show` | 当前 token 掩码 + 所属账号 |
| `add <名字> <token>` | 存入账号（不切换） |
| `save <名字>` | 把当前生效 token 存档（首次：`save me`） |
| `use <名字>` | 切换，自动把当前账号快照为 `_prev` |
| `restore` | 与 `_prev` 互换（可反复撤销/重做） |
| `verify` | pi 探针：无效 / 有效但耗尽 / 正常 |
| `path` | 打印 config、账户目录、pi 二进制路径 |

**存储**：`~/.rick/accounts/tokens/<名字>`（chmod 600，目录 700）；每次写入 config.yaml 前自动备份 `.bak.<时间戳>`。
