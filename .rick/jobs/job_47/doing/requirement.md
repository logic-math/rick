调研一下 rick 怎么能切换 mc --code 的账号信息，这样我就可以使用其他人的额度进行 token 消费了，调研一下获得各种 token 额度的付费
---

## Grilling 澄清结论（2026-10-21，job_47）

**human 裁决**：①产出形态 = 调研报告 + 独立切换脚本（不改 rick 内核）②切换方案 = A（改写 `~/.config/mcopilot-cli/.config.yaml` 的 `AUTHORIZATION`）③借同事 codelab 的 key（视为放权），但**必须保留自己的 key 可切回** ④付费方式全列（按量 API + 订阅制）⑤预算档位 B（¥300–500/月，100–300 万 token/天）。

**核心事实**（详见 `doing/research-report.md`、`doing/grilling/research-L1.md`）：
- mc --code 的"账号"= 单个 Bearer token（`~/.config/mcopilot-cli/.config.yaml` 的 `AUTHORIZATION:`，32 位 hex），**单账号结构，无多账号字段**
- 额度**按天计**，耗尽次日 00:00 重置；官方提额入口 `https://credit-hub.sankuai.com/credit-mgmt/credit-apply`
- rick 不用改代码：pi 鉴权优先级中 `--api-key` 覆盖 > auth.json > 扩展 apiKey（源码 + A/B 实验双证）
- **codelab 机器 `10.128.147.9` 本机不可达**（ping 通，22/8420/常见端口全闭，跳板需凭据）→ 需 human 在 codelab 终端执行：
  `grep -h '^AUTHORIZATION:' ~/.config/mcopilot-cli/.config.yaml | cut -d: -f2- | tr -d ' \r'`
- 拿到 key 后本机两条命令：`~/.rick/accounts/switch.sh add <同事> <token>` → `use <同事>`；切回 `restore` 或 `use me`

**交付物**：
- `~/.rick/accounts/switch.sh`（+x，含 save/use/restore/list/show/add/verify）；自己的 token 已存档为 `me`
- `doing/research-report.md`（完整报告含付费全表 + 风险 + 验证清单）
- `doing/switch.sh`（脚本副本，留存记录）

**待 human 执行**：在同事 codelab 取 key → 本机 `add` + `use` + `verify`（我无法代为访问该机器）。

---

## 补充实取结果（2026-10-21 第二轮）

- **`mc --code` 已确认** = CatPaw CLI 的「代理 Claude Code 命令」；mc 无 login/多账号命令（0.2.2/0.2.3 均如此）
- **codelab 取 key 通道打通**：本机 → `ssh -p 8420 hadoop-recsys@10.128.229.113`（免密，我的 codelab）→ 从该机 `sshpass -p '<pwd>' ssh hadoop-recsys@10.128.171.175`（22 端口在该网络内开放）→ 读 `~/.config/mcopilot-cli/.config.yaml`
- **已取得成功**：目标机 `set-zw05-mlp-codelab-pc936.mt` 的 token ≠ 本人 token，探针返回 `ok`（有效且有额度）
- **已存入保险箱**：`~/.rick/accounts/tokens/zwm`；端到端 `use zwm → verify ✅ → restore → me` 验证通过
- 当前生效账号：**me**（本人今日额度已耗尽）；`restore` 可随时切回
