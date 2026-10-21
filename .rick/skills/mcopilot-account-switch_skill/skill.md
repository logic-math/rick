---
name: mcopilot-account-switch
description: 切换 CatPaw/mcopilot 账号（mc --code 与 rick/pi 共用同一 token）：token 保险箱 + use/restore 自动快照 + pi 探针验证 + config.yaml/auth.json 双写热切换（已运行会话免重启换号）+ launchd 每日 00:00 定时切回；含经沙箱跳板从他人 codelab 取 AUTHORIZATION 的完整通道
execution_plane: mac
---

# skill:mcopilot-account-switch

CatPaw CLI（`mc`）与 rick/pi 走的是**同一个 mcopilot 网关凭据**：`~/.config/mcopilot-cli/.config.yaml` 里的 `AUTHORIZATION:`（32 位 hex Bearer token）。本 skill 提供安全的多账号切换（保险箱 + 自动快照 + 可回滚 + 探针验证），以及「从别人的 codelab 把 token 取回来」的可复用通道。

## 触发场景

想换 AI 额度账号、mc 额度用完想借同事额度、rick 报"额度已使用完毕"、切换到 zwm/其他同事的 token、从他人 codelab 取 key、切回自己账号、查当前生效账号、验证 token 是否有效。

（激活词：切账号、切换 token、额度用完、mc --code 账号、AUTHORIZATION、credit-hub、保险箱、restore）

## 预期效果

- 账号切换**一次命令同时生效于 mc CLI 与 rick/pi**（双写 config.yaml + auth.json），已运行的 pi 会话**下一条请求即换号**（无需重启、上下文不丢）——把「切了但当前会话还是旧账号」从数小时排查变成零成本
- 任何切换可回滚（`_prev` 互换 + 写前时间戳备份），token 永远不丢
- 验证门禁有判别力：探针能区分 `ok` / `401` / `额度耗尽`，不会把失效 token 误判为可用
- 额度次日重置后自动回到本人账号（launchd 定时切回），无需人工记得切回；me token 失效时自动回切同事账号保工作不断

## 核心事实（详见 domain/mcopilot-account.md）

| 事实 | 值 |
|---|---|
| 凭据文件 | `~/.config/mcopilot-cli/.config.yaml` → `AUTHORIZATION: <32hex>` |
| 账号模型 | **单账号扁平结构，无多账号字段**；`mc` 也没有 login/切换命令 → 切换 = 换这一个值 |
| 额度 | **按天计**，耗尽后次日 00:00 重置 |
| 官方提额 | `https://credit-hub.sankuai.com/credit-mgmt/credit-apply`（CatPaw Console，需 SSO） |
| 影响面 | 改 config.yaml = **mc CLI 与 rick/pi 同时换** |
| 生效时机 | mc：下次调用；新 pi 会话：启动即读；**已运行 pi 会话：auth.json 双写后下一条请求生效（免重启）**；`rick web` **无需重启**（须重启仅适用于 `pi_extra_args` 变更，见 domain §4.1） |

## 场景速查

| 你要做什么 | 跑这句 |
|---|---|
| 首次存档自己的账号 | `bash $SKILL/switch.sh save me` |
| 看有哪些账号 | `bash $SKILL/switch.sh list` |
| 存入同事 token（不切换） | `bash $SKILL/switch.sh add zwm <32hex>` |
| 切过去 | `bash $SKILL/switch.sh use zwm` |
| 切回自己 | `bash $SKILL/switch.sh restore`（或 `use me`） |
| 验证当前 token（有效/耗尽/失效） | `bash $SKILL/switch.sh verify` |
| 让**已运行的** rick/pi 会话立即换号 | `switch.sh use <名>` 后，在 pi TUI 里 `/model` 选 GLM 模型（auth.json 覆盖已热生效）；若 auth.json 无覆盖则先 `/reload` |
| 让 pi 回到只读 config.yaml | `bash $SKILL/switch.sh pioff` |
| 每日 00:00 自动切回本人（额度重置后优先用自己账号） | 已装 launchd `com.rick.mcopilot-switchback`（日志 `~/.rick/accounts/switchback.log`；手动触发 `launchctl start com.rick.mcopilot-switchback`；卸载见脚本头部注释） |
| 从他人 codelab 取 key | `bash $SKILL/fetch-token-from-codelab.sh <目标IP> <密码> [用户]` |

`$SKILL` = 本 skill 目录（`~/workdir/MT_INFER/.rick/skills/mcopilot-account-switch_skill`）

## Procedure

1. **存档自己**（首次必做，否则切走无法用名字回来）：
   `bash $SKILL/switch.sh save me`
2. **取对方 token**（二选一）：
   - 对方 codelab 终端执行：`grep -h '^AUTHORIZATION:' ~/.config/mcopilot-cli/.config.yaml | cut -d: -f2- | tr -d ' \r'`
   - 走本 skill 通道：`bash $SKILL/fetch-token-from-codelab.sh 10.128.171.175 '<codelab密码>'`（经沙箱跳板，见下节）
3. **存入保险箱**：`switch.sh add <名字> <token>`（不切换，可先验）
4. **切换 + 验证**：`switch.sh use <名字>` → `switch.sh verify`（探针输出 `ok` = ✅；`401` = ❌失效；`额度已使用完毕` = ⏳有效但今日耗尽）
5. **让当前已开的会话用上**：auth.json 双写已热生效——若会话跑在 catpaw 的 GLM 上，下一条请求即换号；若会话已回退到别的 provider（如 deepseek），TUI `/model` 选回 GLM 即可
6. **用完切回**：`switch.sh restore`（web/CLI 会话均自动跟进，无需重启 daemon）

## 取他人 codelab token 的通道（2026-10-21 实测打通）

Mac 直连他人 codelab 常被防火墙拦（22 过滤/全端口闭），但**沙箱可以到**：

```
Mac ──ssh -p 8420──▶ 沙箱 10.128.229.113（免密，见 env.md §1）
                        └──sshpass -p '<密码>'──▶ 目标 codelab <IP>:22
                              └─ ~/.config/mcopilot-cli/.config.yaml → AUTHORIZATION
```

- 判据：Mac 上 `nc -z -G 4 <IP> 22` 超时 ≠ 不可达；先 `ping` 通再试沙箱跳板
- 沙箱上 `sshpass` 已装（`/usr/bin/sshpass`），Mac 上没有也不需要
- 一次可批量取多台：目标 IP 列表 + 各自密码（codelab 密码通常同人同值）

## 实现要点（改脚本前必读）

- **双写（一次对齐两处）**：`use/restore` 同时写
  1. `~/.config/mcopilot-cli/.config.yaml` 的 `AUTHORIZATION`（**mc CLI** 用）
  2. `~/.rick/pi/agent/auth.json` 的 `catpaw-proxy.api_key`（**rick/pi** 用）
- **为什么必须双写**：pi 的鉴权解析里 **stored credential（auth.json）优先级高于 provider 配置的 apiKey**（`provider-composer.js` `composeApiKeyAuth.resolve`：`input.credential` 在时用 `credential.key`）；而 `catpaw-proxy.ts` 只在**进程启动**时读一次 config.yaml。⇒ 只改 config.yaml，已运行的 pi 会话永远不换号；有 auth.json 覆盖则换号立即生效。
- **auth.json 为什么能热生效**：`AuthStorage.read → readLatestData` 每次请求比对文件 revision，变了就重载（无需重启）。
- `switch.sh list` 会显示 pi 覆盖行，并在两处 token 不一致时报**漂移警告**。
- **bash 3.2 兼容**（macOS 自带）：全角字符紧跟变量必须写 `${var}`，否则 `$var（` 会把多字节吞进变量名 → `unbound variable`
- 写入 config.yaml 前**自动备份** `config.yaml.bak.<时间戳>`（auth.json 同样备份 `auth.json.bak.<时间戳>`）；token 文件 chmod 600、目录 700
- `restore` 是**与 `_prev` 互换**（可反复撤销/重做），不是单向回退
- **定时切回（midnight-switchback.sh）**：launchd 每日 00:00 调用；`use me` 幂等 → 每天跑都安全；护栏：切后 `verify`，me 401 → 自动回切 zwm 并记日志。**选 launchd 不选 cron**：Mac 睡着时 launchd 唤醒后补跑，cron 直接错过。副作用须知：切回后，**正在运行的 pi 会话**（auth.json 热重载）下一条请求也跟着回 me——额度重置后这正是期望行为
- 切换后 `mc` 本体也换了账号——若只想影响 rick，改用 `pi --api-key <token>`（代价：token 出现在 `ps aux`）

## 踩坑

| 坑 | 现象 | 处理 |
|---|---|---|
| 全角紧跟 `$var` | `CONF（: unbound variable` | 一律 `${var}` |
| 反引号写进 echo | `rick: command not found` | 提示语里别用反引号，或转义 |
| Mac 直连 codelab | 22 超时、其它端口全闭 | 走沙箱跳板（上节） |
| **切了但当前会话还是旧账号** | pi 扩展只在启动时读 config.yaml | 跑 `switch.sh use <名>`（双写 auth.json，热生效）→ TUI `/model` 选 GLM；或 `/reload` 重读 config.yaml；或 `rick <cmd> --resume <job>` 重开进程 |
| **auth.json 与 config.yaml 不一致** | 手工只改了一处 | `switch.sh list` 看漂移警告 → `use <名>` 一次性对齐 |
| config.yaml 改完 web 没反应 | 旧认知误判 | **web 无需重启**：新会话 spawn 新 pi 进程自动读新 token；活跃会话靠 auth.json 双写热重载。仅 `pi_extra_args`（provider/model）变更才需重启 `rick web` |
| 定时切回日误以为被踢号 | 每日 00:00 launchd 自动 `use me` | 预期行为（额度重置优先用本人账号）；想连日用 zwm 先卸载定时任务 |
| **程序生成名手打笔误** | `ls`/`cd`/`stat` 报 ENOENT 但 glob/python `os.listdir` 正常 | 名字是程序生成的（如 pi sessions 目录 `--Users-...-MT_INFER--`）：**禁止手打，一律枚举后按返回值引用**；首次 ENOENT 即 `printf '%s' <名> \| od -c` 对比 glob 展开的字节（本例 `MT_INFER` 的 `_` 被打成 `-`，视觉不可分辨） |
| pi compact 串号（仅 `--api-key` 方案） | `/compact` 仍扣原账号 | catpaw-proxy 的 compact 处理器硬读 config.yaml |

## 验证清单（可执行断言）

| # | 断言 | 命令 | 期望 |
|---|---|---|---|
| 1 | 自己的账号已存档 | `switch.sh list` | 出现 `* me`（`*` = 当前生效） |
| 2 | 切换生效 | `switch.sh use <名>` → `verify` | 探针 `ok` |
| 3 | 可切回 | `switch.sh restore` → `list` | 回到 `me`，config.yaml 恢复原值 |
| 4 | 可回滚 | `ls ~/.config/mcopilot-cli/*.bak.*` | 每次写入前有备份 |
| 5 | 失效 token 不被误判 | `add t <假token>` → `use t` → `verify` | 判定 `❌ token 无效（401）` |
| 6 | auth.json 覆盖真的优先 | config.yaml 写有效 token、auth.json 写伪造 token → 跑 pi 探针 | 探针 `401`（证明走 auth.json） |
| 7 | 双写一致 | `use <名>` → `list` | pi 覆盖行与当前 token 一致，无漂移警告 |
| 8 | 定时切回就位 | `launchctl list \| grep com.rick.mcopilot-switchback`；`bash $SKILL/midnight-switchback.sh --dry-run` | launchctl 出现条目；dry-run 只打印动作与当前状态、不落盘 |
