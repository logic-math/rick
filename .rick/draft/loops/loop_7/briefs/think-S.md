# think-S 简报

> 阶段：S 问题确认（loop_7 第 1 轮）| 主题：「pi vs dsh 谁适合作为 rick runtime」的隐含前提追问 | 输入：briefs/research-S.md + loop_6/judgment.md

## 一、top-5 隐含前提问题（Q1 为元问题，先答）

**Q1 重开已决事项的触发条件**
- 问题：若「loop_7 现在就要判断谁适合做 runtime」成立，那么也假设了「loop_6 之后出现了足以重开已决事项的新条件」——这真的正确吗？dsh 发布（08-13）早于 loop_6 判断（08-14），其后仅 rc 常规迭代，无结构性新材料。
- 依据：research ②-8/③-8；loop_6 已确认单一 runtime=pi、切换前提=更强生态+可定制性。
- 改变判断的证据：dsh 发 1.0 稳定版+插件市场成形，或 pi 停更/bus factor 兑现。
- 性质：判断性（触发条件=human 意图）；事实部分已消解（无新材料）。若答「无新触发」，本轮定位=「校验维持 pi+定义未来切换触发器」而非重选，Q2-Q5 结论预设反转。

**Q2 「短期性能」的指代与杠杆**
- 问题：若「对比 pi/dsh 能回答短期如何提升性能」成立，那么也假设了「短期性能有明确指代，且 runtime 选择是有效杠杆」——这真的正确吗？loop_6 已确认短期抓手是提示词/配置对齐（已落地验收）而非换 runtime；且两者无任何直接性能 benchmark。
- 依据：research ⑤：loop_6 原话「目前确实缺少量化评估，当前都是靠直觉进行优化」；R7-4 无埋点、R7-6 无 benchmark。
- 改变判断的证据：出现可对比执行基准（dsh 显著优于 pi），或证明第三方 pi-subagents 层是触发不确定性的残留根因。
- 性质：判断性为主（哪个维度算「性能」须 human 定：触发确定性/上下文效率/延迟/成本）；基线缺失已消解。建议 research 追加度量方案。

**Q3 长期架构绑定 vs spec 可切换赌注**
- 问题：若「长期架构发展取决于选谁做 runtime」成立，那么也假设了「loop_6 押注的 spec 可切换性（自然语言方法描述→任意等价实现）不足以消解 runtime 锁定风险」——这真的正确吗？若 spec 赌注成立，runtime 选择长期廉价可逆，对比 stakes 大降；若不成立，深定制 pi（不保独立）正使切换成本逐轮复利，对比反而更紧迫。两方向不能同时为真。
- 依据：loop_6 EC 自认「最大假设：方法描述→开发计划→满足测试的程序」；research ①-7：切换 seam 已预留但 dsh 不写代码。
- 改变判断的证据：一次小规模 spec→等价重实现实验（用 spec 重建 rick 某模块并过功能验收）。
- 性质：判断性（取舍必须 human 回答）；可设计实验但不属调研。

**Q4 评判标准集的完备性**
- 问题：若「按生态+可定制性两标准对比即可得出结论」成立，那么也假设了「这两条标准构成的集合是完备的、无遗漏致命维度」——这真的正确吗？成熟度/稳定性（dsh 官方明示 developer preview+breaking changes+已知 bug）与上游维护者风险（pi bus factor=1，badlogic 占 69% commits；rick 又踩在第三方 pi-subagents 上）都不在标准集内，任一条都可能一票否决。
- 依据：research ②-8、③-9、①-4。
- 改变判断的证据：dsh 发 1.0 稳定版且 pi 形成第二梯队核心维护者。
- 性质：判断性（标准加权是价值选择）；所需事实已供齐，无需追加。

**Q5 runtime 边界=薄接口，切换成本 confined 于 seam**
- 问题：若「runtime 是 rick 中可整体替换的薄封装（Runtime 接口 Name/Run），切换只需新增 dshRuntime/dshBuilder 而 cli/handler/templates 不改」成立，那么也假设了「真正的耦合面——提示词模板层的 pi 专有语言——不构成切换障碍」——这真的正确吗？loop_6 已把触发语言迁移为 workflowScript/runs.run 等 pi 专有语法并内嵌 templates；切 dsh 时「templates 不改」与 pi 语法内嵌互相矛盾。
- 依据：research ①-5（sense_loop.md L28 显式 pi 触发语法）+①-7（切换规则声称 templates 不改）。
- 改变判断的证据：盘点 templates 中 pi 专有依赖（触发语法/JSONL 事件流/compaction 行为）；若面广且深，seam 承诺须修订为「templates 分 runtime 版本」。
- 性质：事实性（代码可盘点，建议 research 澄清）——唯一可完全调研消解的架构级假设。

## 二、S 三连总追问

- 现状：你心中的「现状」是「loop_6 修复已落地且效果可感知」，还是「落地但效果未知（无度量）」？若是后者，「性能仍需提升」来自直觉还是有观察？
- 期望：本轮期望产出是（A）重选 runtime 的决定，还是（B）校验维持 pi+定义「未来切换触发器」？A 与 B 所需证据强度完全不同。
- 差距：真正的差距是「对 pi/dsh 的认知差距」，还是「对 rick 自身效果的度量差距」？若是后者，深度学习 dsh 无法消解它，度量方案才是本轮最短缺的交付。

## 三、research 未消解项消化建议

- 值得追加：**R7-4**（最高优先：设计触发概率/性能度量方案，从现有 raw_session_coding.log+Trace 派生统计——消解 Q2 与差距追问）；**R7-5**（条件触发：仅当 Q1 答「确需严肃评估 dsh」时做 dsh headless spike，可同时消解 Q5 模板耦合盘点）；**R7-6**（中等：追查 dsh 是否已进任何 harness benchmark）。
- 建议延后：R7-1/R7-2（dsh 生态规模与 star 含金量：11 天项目一次性调研无解，挂 30 天后复查）。
- 建议放弃：R7-3/R7-7/R7-8（Discord 人数、npm 口径、commit 拆分：对本轮各题均无影响）。
